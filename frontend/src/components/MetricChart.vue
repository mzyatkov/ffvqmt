<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch, computed } from "vue";
import Plotly from "plotly.js-dist-min";
import { useSession } from "../stores/session";
import type { Metric } from "../api/wails";

const props = defineProps<{ metric: Metric }>();
const session = useSession();

const root = ref<HTMLDivElement | null>(null);

const palette = [
  "#4f9eef", "#f0ad4e", "#6dd3a8", "#e85a5a", "#c084fc", "#22d3ee",
  "#facc15", "#fb7185", "#34d399", "#60a5fa", "#a78bfa", "#fcd34d",
  "#f97316", "#10b981", "#3b82f6", "#ec4899", "#8b5cf6", "#06b6d4",
  "#eab308", "#ef4444", "#14b8a6", "#0ea5e9", "#d946ef", "#84cc16",
];

function baseName(p: string) {
  return p.replace(/\\/g, "/").split("/").pop() || p;
}

function buildTraces(): Plotly.Data[] {
  const byFile = session.series[props.metric].byFile;
  const files = Object.keys(byFile);
  return files.map((file, i) => {
    const samples = byFile[file];
    return {
      x: samples.map((s) => s.frame),
      y: samples.map((s) => s.value),
      mode: "lines",
      type: "scattergl",
      name: baseName(file),
      hovertemplate: `<b>${baseName(file)}</b><br>frame %{x}<br>${props.metric}=%{y:.4f}<extra></extra>`,
      line: { color: palette[i % palette.length], width: 1 },
    } as Plotly.Data;
  });
}

function yAxisTitle() {
  switch (props.metric) {
    case "PSNR":
    case "XPSNR":
      return props.metric + " (dB)";
    case "SSIM":
      return "SSIM (0..1)";
    case "VMAF":
      return "VMAF (0..100)";
  }
  return props.metric;
}

const layout = computed<Partial<Plotly.Layout>>(() => ({
  paper_bgcolor: "#1b2636",
  plot_bgcolor: "#243044",
  font: { color: "#e9eef7", family: "system-ui, sans-serif", size: 12 },
  margin: { l: 60, r: 20, t: 20, b: 40 },
  xaxis: {
    title: { text: "frame" },
    gridcolor: "#344057",
    zerolinecolor: "#344057",
    showspikes: true,
    spikemode: "across",
    spikethickness: 1,
    spikecolor: "#4f9eef",
  },
  yaxis: {
    title: { text: yAxisTitle() },
    gridcolor: "#344057",
    zerolinecolor: "#344057",
  },
  legend: { orientation: "h", x: 0, y: 1.08, bgcolor: "rgba(0,0,0,0)" },
  hovermode: "closest",
  dragmode: "pan",
}));

const config: Partial<Plotly.Config> = {
  displaylogo: false,
  responsive: true,
  scrollZoom: true,
  toImageButtonOptions: {
    format: "png",
    filename: "ffvqmt-chart",
    height: 720,
    width: 1280,
    scale: 2,
  },
  modeBarButtonsToAdd: ["toggleSpikelines"],
  modeBarButtonsToRemove: ["lasso2d", "select2d"],
};

let resizeObs: ResizeObserver | null = null;

onMounted(async () => {
  if (!root.value) return;
  await Plotly.newPlot(root.value, buildTraces(), layout.value, config);
  resizeObs = new ResizeObserver(() => {
    if (root.value) Plotly.Plots.resize(root.value);
  });
  resizeObs.observe(root.value);
});

onBeforeUnmount(() => {
  if (root.value) Plotly.purge(root.value);
  resizeObs?.disconnect();
});

watch(
  () => [session.series[props.metric].byFile, props.metric],
  async () => {
    if (!root.value) return;
    await Plotly.react(root.value, buildTraces(), layout.value, config);
  },
  { deep: false },
);
</script>

<template>
  <div ref="root" class="plot"></div>
</template>
