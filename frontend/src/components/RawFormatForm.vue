<script setup lang="ts">
import { reactive, watch } from "vue";
import type { RawFormat } from "../api/wails";

const props = defineProps<{
  modelValue: RawFormat | null | undefined;
  label?: string;
}>();
const emit = defineEmits<{
  (e: "update:modelValue", v: RawFormat): void;
  (e: "apply", v: RawFormat): void;
}>();

const local = reactive<RawFormat>({
  format: props.modelValue?.format || "rawvideo",
  pixFmt: props.modelValue?.pixFmt || "yuv420p",
  width: props.modelValue?.width || 1920,
  height: props.modelValue?.height || 1080,
  frameRate: props.modelValue?.frameRate || 30,
});

watch(
  () => props.modelValue,
  (v) => {
    if (!v) return;
    local.format = v.format || local.format;
    local.pixFmt = v.pixFmt || local.pixFmt;
    local.width = v.width || local.width;
    local.height = v.height || local.height;
    local.frameRate = v.frameRate || local.frameRate;
  },
);

const FORMATS = [
  "rawvideo",
];
const PIX_FMTS = [
  "yuv420p", "yuv422p", "yuv444p",
  "yuv420p10le", "yuv422p10le", "yuv444p10le",
  "yuv420p12le", "yuv422p12le", "yuv444p12le",
  "nv12", "nv21",
  "yuyv422", "uyvy422",
  "gray", "gray10le", "gray12le", "gray16le",
  "rgb24", "bgr24", "rgba", "bgra",
];
const PRESETS = [
  { name: "QCIF",         w: 176,  h: 144 },
  { name: "CIF",          w: 352,  h: 288 },
  { name: "VGA",          w: 640,  h: 480 },
  { name: "480p (SD)",    w: 854,  h: 480 },
  { name: "720p (HD)",    w: 1280, h: 720 },
  { name: "1080p (FHD)",  w: 1920, h: 1080 },
  { name: "1440p (QHD)",  w: 2560, h: 1440 },
  { name: "2160p (4K)",   w: 3840, h: 2160 },
];

function pickPreset(e: Event) {
  const v = (e.target as HTMLSelectElement).value;
  const p = PRESETS.find((x) => x.name === v);
  if (p) {
    local.width = p.w;
    local.height = p.h;
  }
}

function apply() {
  emit("update:modelValue", { ...local });
  emit("apply", { ...local });
}
</script>

<template>
  <div class="raw-form col" style="gap:6px; padding:8px; border:1px dashed var(--border, #888); border-radius:4px; background: rgba(255,200,80,0.05)">
    <div class="row" style="gap:6px; align-items:center">
      <b>{{ label || "Raw video parameters" }}</b>
      <span class="muted" style="font-size:11px">(file has no header — please specify)</span>
    </div>
    <div class="row" style="gap:8px; flex-wrap:wrap">
      <label>Format
        <select v-model="local.format">
          <option v-for="f in FORMATS" :key="f" :value="f">{{ f }}</option>
        </select>
      </label>
      <label>Pixel format
        <select v-model="local.pixFmt">
          <option v-for="f in PIX_FMTS" :key="f" :value="f">{{ f }}</option>
        </select>
      </label>
      <label>Preset
        <select @change="pickPreset" :value="''">
          <option value="">—</option>
          <option v-for="p in PRESETS" :key="p.name" :value="p.name">{{ p.name }} ({{ p.w }}×{{ p.h }})</option>
        </select>
      </label>
      <label>Width <input type="number" v-model.number="local.width" min="1" style="width:80px"/></label>
      <label>Height <input type="number" v-model.number="local.height" min="1" style="width:80px"/></label>
      <label>FPS <input type="number" v-model.number="local.frameRate" min="0.001" step="0.001" style="width:80px"/></label>
      <button @click="apply" style="align-self:flex-end">Apply</button>
    </div>
  </div>
</template>
