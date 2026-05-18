import { defineStore } from "pinia";
import { ref, computed, reactive } from "vue";
import { API, type Metric, type ProbeInfo, type MediaInfo, type RunRequest } from "../api/wails";

export interface DistFile {
  path: string;
  active: boolean;
  exists: boolean;
  info?: MediaInfo;
  error?: string;
}

export interface Sample {
  frame: number;
  value: number;
  extra?: Record<string, number>;
}

export interface MetricSeries {
  // key: dist path -> samples
  byFile: Record<string, Sample[]>;
  summaryByFile: Record<string, { min: number; max: number; average: number; harmonic: number; frames: number }>;
}

export const useSession = defineStore("session", () => {
  const probe = ref<ProbeInfo | null>(null);
  const detecting = ref(false);
  const detectError = ref<string | null>(null);

  const refPath = ref<string>("");
  const refInfo = ref<MediaInfo | null>(null);
  const refThumb = ref<string | null>(null);

  const distFiles = ref<DistFile[]>([]);

  const metrics = reactive<Record<Metric, boolean>>({
    PSNR: true,
    SSIM: true,
    VMAF: true,
    XPSNR: false,
  });

  const options = reactive({
    skip: 0,
    duration: 0,
    scaling: "BICUBIC",
    vmafModel: "",
    vmafPool: "MEAN",
    vmafSubsample: 1,
    vmafPhone: false,
    vmafUpscale: true,
    logCommands: false,
    logFrames: false,
    logFramesDir: "",
    tempDir: "",
    nThreads: 0,
    autoSaveResults: false,
    autoSaveResultsFile: "",
    plotUpdateSpeed: "NORMAL" as "HIGH" | "NORMAL" | "LOW" | "OFF",
  });

  const running = ref(false);
  const progress = ref(0);
  const lastError = ref<string | null>(null);

  // Per-metric series storage
  const series: Record<Metric, MetricSeries> = reactive({
    PSNR: { byFile: {}, summaryByFile: {} },
    SSIM: { byFile: {}, summaryByFile: {} },
    VMAF: { byFile: {}, summaryByFile: {} },
    XPSNR: { byFile: {}, summaryByFile: {} },
  });

  function clearSeries() {
    (["PSNR", "SSIM", "VMAF", "XPSNR"] as Metric[]).forEach((m) => {
      series[m].byFile = {};
      series[m].summaryByFile = {};
    });
  }

  async function detect() {
    detecting.value = true;
    detectError.value = null;
    try {
      probe.value = await API.detectFFmpeg();
      // disable metrics that ffmpeg doesn't support
      if (!probe.value.hasPSNR) metrics.PSNR = false;
      if (!probe.value.hasSSIM) metrics.SSIM = false;
      if (!probe.value.hasVMAF) metrics.VMAF = false;
      if (!probe.value.hasXPSNR) metrics.XPSNR = false;
    } catch (e: any) {
      detectError.value = e?.message || String(e);
    } finally {
      detecting.value = false;
    }
  }

  async function setReference(path: string) {
    refPath.value = path;
    refInfo.value = null;
    refThumb.value = null;
    try {
      refInfo.value = await API.mediaInfo(path);
    } catch (e: any) {
      lastError.value = `ref media info: ${e?.message || e}`;
    }
    try {
      refThumb.value = await API.makeThumbnail(path);
    } catch {
      /* thumbnail is best-effort */
    }
  }

  async function addDistFiles(paths: string[]) {
    for (const p of paths) {
      if (distFiles.value.some((d) => d.path === p)) continue;
      const entry: DistFile = { path: p, active: true, exists: true };
      distFiles.value.push(entry);
      try {
        entry.info = await API.mediaInfo(p);
      } catch (e: any) {
        entry.error = e?.message || String(e);
        entry.exists = false;
      }
    }
  }

  function removeDist(path: string) {
    distFiles.value = distFiles.value.filter((d) => d.path !== path);
  }

  function clearDist() {
    distFiles.value = [];
  }

  const enabledMetrics = computed<Metric[]>(() =>
    (Object.keys(metrics) as Metric[]).filter((m) => metrics[m]),
  );

  const canStart = computed(
    () =>
      !!probe.value &&
      !!refPath.value &&
      refInfo.value !== null &&
      distFiles.value.some((d) => d.active && d.exists) &&
      enabledMetrics.value.length > 0 &&
      !running.value,
  );

  async function start() {
    if (!canStart.value) return;
    clearSeries();
    progress.value = 0;
    lastError.value = null;
    running.value = true;

    const req: RunRequest = {
      refPath: refPath.value,
      distPaths: distFiles.value.filter((d) => d.active && d.exists).map((d) => d.path),
      metrics: enabledMetrics.value,
      skip: options.skip,
      duration: options.duration,
      scaling: options.scaling,
      vmafModel: options.vmafModel,
      vmafPool: options.vmafPool,
      vmafSubsample: options.vmafSubsample,
      vmafPhone: options.vmafPhone,
      vmafUpscale: options.vmafUpscale,
      logCommands: options.logCommands,
      logFrames: options.logFrames,
      logFramesDir: options.logFramesDir,
      tempDir: options.tempDir,
      nThreads: options.nThreads,
      autoSaveResults: options.autoSaveResults,
      autoSaveResultsFile: options.autoSaveResultsFile,
    };
    try {
      await API.startRun(req);
    } catch (e: any) {
      lastError.value = e?.message || String(e);
      running.value = false;
    }
  }

  async function cancel() {
    try {
      await API.cancelRun();
    } catch {
      /* ignore */
    }
  }

  // Wire events
  let frameBuffer: Record<string, { last: number }> = {};
  function throttleFrame(file: string, frame: number): boolean {
    if (options.plotUpdateSpeed === "OFF") return false;
    const minStep =
      options.plotUpdateSpeed === "HIGH" ? 1 :
      options.plotUpdateSpeed === "NORMAL" ? 5 :
      25;
    const b = frameBuffer[file] || { last: -minStep };
    if (frame - b.last >= minStep) {
      b.last = frame;
      frameBuffer[file] = b;
      return true;
    }
    return false;
  }

  function bindEvents() {
    API.on("metric:frame", (d: any) => {
      const m = String(d.metric).toUpperCase() as Metric;
      if (!series[m]) return;
      const arr = (series[m].byFile[d.file] ||= []);
      arr.push({ frame: d.frame, value: d.value, extra: d.extra });
      // throttle UI: only emit reactive change at a cadence
      if (throttleFrame(d.file + ":" + m, d.frame)) {
        // trigger reactivity
        series[m].byFile = { ...series[m].byFile };
      }
    });
    API.on("metric:summary", (d: any) => {
      const m = String(d.metric).toUpperCase() as Metric;
      if (!series[m]) return;
      series[m].summaryByFile[d.file] = d.summary;
      // force a final reactive update
      series[m].byFile = { ...series[m].byFile };
    });
    API.on("run:progress", (d: any) => {
      progress.value = d.percent ?? 0;
    });
    API.on("run:done", (d: any) => {
      running.value = false;
      progress.value = 100;
      if (!d.ok && d.error) lastError.value = d.error;
      frameBuffer = {};
    });
    API.on("file:error", (d: any) => {
      lastError.value = `${d.file}: ${d.error}`;
    });
    API.on("log:error", (msg: any) => {
      lastError.value = String(msg);
    });
    API.onFileDrop((paths) => {
      handleDroppedFiles(paths);
    });
  }

  async function handleDroppedFiles(paths: string[]) {
    if (paths.length === 0) return;
    if (!refPath.value) {
      await setReference(paths[0]);
      if (paths.length > 1) await addDistFiles(paths.slice(1));
    } else {
      await addDistFiles(paths);
    }
  }

  async function loadInitialFromCLI() {
    try {
      const opts = await API.getInitialOptions();
      if (opts.refFile) await setReference(opts.refFile);
      if (opts.distFiles?.length) await addDistFiles(opts.distFiles);
      // metrics from CLI
      const cli = (opts.metrics || []).map((m) => m.toUpperCase());
      (["PSNR", "SSIM", "VMAF", "XPSNR"] as Metric[]).forEach((m) => {
        metrics[m] = cli.includes(m);
      });
      // copy remaining options
      options.skip = opts.skip;
      options.duration = opts.duration;
      options.scaling = opts.scalingMethod;
      options.vmafModel = opts.vmafModel;
      options.vmafPool = opts.vmafPool;
      options.vmafSubsample = opts.vmafSubsample || 1;
      options.vmafPhone = opts.vmafPhoneModel;
      options.vmafUpscale = opts.vmafUpscaleToModel;
      options.logCommands = opts.logCommands;
      options.logFrames = opts.logFrames;
      options.logFramesDir = opts.logFramesDir;
      options.tempDir = opts.tempDir;
      options.autoSaveResults = opts.autoSaveResults;
      options.autoSaveResultsFile = opts.autoSaveResultsFile;
      options.plotUpdateSpeed = opts.plotUpdateSpeed as any;

      if (opts.run) {
        // Defer start until detection is done
        const tryStart = setInterval(() => {
          if (probe.value) {
            clearInterval(tryStart);
            void start();
          }
        }, 100);
      }
    } catch {
      /* not in wails */
    }
  }

  return {
    probe, detecting, detectError,
    refPath, refInfo, refThumb,
    distFiles, metrics, options,
    running, progress, lastError,
    series, enabledMetrics, canStart,
    detect, setReference, addDistFiles, removeDist, clearDist,
    start, cancel, bindEvents, loadInitialFromCLI, handleDroppedFiles,
  };
});
