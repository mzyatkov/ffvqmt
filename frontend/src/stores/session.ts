import { defineStore } from "pinia";
import { ref, computed, reactive } from "vue";
import { API, type Metric, type ProbeInfo, type MediaInfo, type RunRequest, type RawFormat } from "../api/wails";

export interface DistFile {
  path: string;
  active: boolean;
  exists: boolean;
  info?: MediaInfo;
  error?: string;
  raw?: RawFormat | null;     // user-provided raw-format params (.yuv etc.)
  needsRaw?: boolean;         // true when the file is a raw container
  thumb?: string | null;      // data URI of preview
}

export function isRawExt(p: string): boolean {
  const ext = p.split(/[\\\/]/).pop()?.split(".").pop()?.toLowerCase() || "";
  return ["yuv", "raw", "rgb", "bgr", "gray", "y"].includes(ext);
}

export function defaultRawFor(_path: string): RawFormat {
  return { format: "rawvideo", pixFmt: "yuv420p", width: 1920, height: 1080, frameRate: 30 };
}

function rawComplete(r: RawFormat | null | undefined): r is RawFormat {
  return !!r && !!r.format && !!r.pixFmt && r.width > 0 && r.height > 0 && r.frameRate > 0;
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
  const refRaw = ref<RawFormat | null>(null);
  const refNeedsRaw = ref(false);

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
    startFrame: 0,
    endFrame: 0,
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
    refNeedsRaw.value = isRawExt(path);
    if (refNeedsRaw.value && !refRaw.value) {
      refRaw.value = defaultRawFor(path);
    }
    await refreshReferenceInfo();
  }

  async function refreshReferenceInfo() {
    const path = refPath.value;
    if (!path) return;
    const raw = refNeedsRaw.value ? refRaw.value : null;
    if (refNeedsRaw.value && !rawComplete(raw)) {
      // wait for the user to fill in the form
      return;
    }
    try {
      refInfo.value = await API.mediaInfo(path, raw);
    } catch (e: any) {
      lastError.value = `ref media info: ${e?.message || e}`;
    }
    // Clear first so Vue sees a real src change even if the new data URI
    // happens to be identical to the previous one.
    refThumb.value = null;
    try {
      refThumb.value = await API.makeThumbnail(path, raw);
    } catch (e: any) {
      // surface raw-input errors (wrong size / pix_fmt) instead of just
      // dropping the thumbnail silently.
      lastError.value = `ref thumbnail: ${e?.message || e}`;
    }
  }

  async function setRefRaw(r: RawFormat) {
    refRaw.value = { ...r };
    await refreshReferenceInfo();
  }

  async function addDistFiles(paths: string[]) {
    for (const p of paths) {
      if (distFiles.value.some((d) => d.path === p)) continue;
      const needsRaw = isRawExt(p);
      const entry: DistFile = {
        path: p,
        active: true,
        exists: true,
        needsRaw,
        raw: needsRaw ? defaultRawFor(p) : null,
      };
      distFiles.value.push(entry);
      await refreshDistInfo(entry);
    }
  }

  async function refreshDistInfo(entry: DistFile) {
    const raw = entry.needsRaw ? entry.raw || null : null;
    if (entry.needsRaw && !rawComplete(raw)) {
      // Probe still returns a stub so the user sees the row
      try {
        entry.info = await API.mediaInfo(entry.path, raw);
      } catch (e: any) {
        entry.error = e?.message || String(e);
      }
      entry.thumb = null;
      return;
    }
    try {
      entry.info = await API.mediaInfo(entry.path, raw);
      entry.error = undefined;
      entry.exists = true;
    } catch (e: any) {
      entry.error = e?.message || String(e);
      entry.exists = false;
    }
    // Regenerate the thumbnail with the new params; clear first so Vue
    // always sees a real src change.
    entry.thumb = null;
    try {
      entry.thumb = await API.makeThumbnail(entry.path, raw);
    } catch (e: any) {
      lastError.value = `${entry.path} thumbnail: ${e?.message || e}`;
    }
  }

  async function setDistRaw(path: string, r: RawFormat) {
    const entry = distFiles.value.find((d) => d.path === path);
    if (!entry) return;
    entry.raw = { ...r };
    await refreshDistInfo(entry);
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
      (!refNeedsRaw.value || rawComplete(refRaw.value)) &&
      distFiles.value.some(
        (d) => d.active && d.exists && (!d.needsRaw || rawComplete(d.raw)),
      ) &&
      enabledMetrics.value.length > 0 &&
      !running.value,
  );

  async function start() {
    if (!canStart.value) return;
    clearSeries();
    progress.value = 0;
    lastError.value = null;
    running.value = true;

    const activeDist = distFiles.value.filter(
      (d) => d.active && d.exists && (!d.needsRaw || rawComplete(d.raw)),
    );
    const distRaw: Record<string, RawFormat> = {};
    for (const d of activeDist) {
      if (d.needsRaw && rawComplete(d.raw)) distRaw[d.path] = d.raw;
    }
    const req: RunRequest = {
      refPath: refPath.value,
      distPaths: activeDist.map((d) => d.path),
      refRaw: refNeedsRaw.value && rawComplete(refRaw.value) ? refRaw.value : null,
      distRaw: Object.keys(distRaw).length ? distRaw : undefined,
      metrics: enabledMetrics.value,
      skip: options.skip,
      duration: options.duration,
      startFrame: options.startFrame,
      endFrame: options.endFrame,
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
    refPath, refInfo, refThumb, refRaw, refNeedsRaw,
    distFiles, metrics, options,
    running, progress, lastError,
    series, enabledMetrics, canStart,
    detect, setReference, setRefRaw, setDistRaw, refreshDistInfo,
    addDistFiles, removeDist, clearDist,
    start, cancel, bindEvents, loadInitialFromCLI, handleDroppedFiles,
  };
});
