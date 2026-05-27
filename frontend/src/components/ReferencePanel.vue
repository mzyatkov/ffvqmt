<script setup lang="ts">
import { ref } from "vue";
import { useSession } from "../stores/session";
import { API, type RawFormat } from "../api/wails";
import RawFormatForm from "./RawFormatForm.vue";

const session = useSession();
const dragging = ref(false);

function applyRefRaw(r: RawFormat) {
  void session.setRefRaw(r);
}

async function pick() {
  const p = await API.selectFiles(false, "Select reference");
  if (p?.[0]) await session.setReference(p[0]);
}

function onDragOver(e: DragEvent) {
  e.preventDefault();
  dragging.value = true;
}
function onDragLeave() { dragging.value = false; }
async function onDrop(e: DragEvent) {
  e.preventDefault();
  dragging.value = false;
  const files = e.dataTransfer?.files;
  if (files && files.length > 0) {
    // Browsers expose path via `webkitRelativePath` only sometimes; Wails injects via OnFileDrop.
    // We fall back to file name and rely on Wails's native drop handler for the real path.
    const path = (files[0] as any).path || files[0].name;
    if (path && path !== files[0].name) {
      await session.setReference(path);
    }
  }
}

function fmtRes(w: number, h: number) {
  if (!w || !h) return "—";
  return `${w}×${h}`;
}
function fmtDur(d: number) {
  if (!d || isNaN(d)) return "—";
  const h = Math.floor(d / 3600);
  const m = Math.floor((d % 3600) / 60);
  const s = Math.floor(d % 60);
  return h > 0 ? `${h}h ${m}m ${s}s` : `${m}m ${s}s`;
}
function fmtBitrate(b: number) {
  if (!b) return "—";
  return (b / 1000).toFixed(0) + " kbps";
}
</script>

<template>
  <div class="col">
    <div class="row"><b>Reference</b><span class="spacer"/><button @click="pick">…</button></div>
    <div
      class="drop-zone"
      :class="{ dragging }"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <template v-if="session.refPath">
        <div style="font-size:12px; word-break:break-all">{{ session.refPath }}</div>
      </template>
      <template v-else>
        Drop reference file here<br/>or click <b>…</b> to choose
      </template>
    </div>

    <RawFormatForm
      v-if="session.refNeedsRaw"
      :model-value="session.refRaw"
      label="Reference raw video parameters"
      @apply="applyRefRaw"
    />

    <img v-if="session.refThumb"
         :src="session.refThumb"
         alt="thumbnail"
         style="max-width:100%; max-height:160px; object-fit:contain; background:#000; border-radius:4px"/>

    <div v-if="session.refInfo" class="col" style="gap:2px; font-size:12px">
      <div><span class="muted">Resolution:</span> {{ fmtRes(session.refInfo.width, session.refInfo.height) }}</div>
      <div><span class="muted">Pixel format:</span> {{ session.refInfo.pixFmt || "—" }} ({{ session.refInfo.colorRange || "unspecified range" }})</div>
      <div><span class="muted">Duration:</span> {{ fmtDur(session.refInfo.duration) }}</div>
      <div><span class="muted">Frame rate:</span> {{ session.refInfo.frameRate ? session.refInfo.frameRate.toFixed(3) + " fps" : "—" }}</div>
      <div><span class="muted">Codec:</span> {{ session.refInfo.videoCodec || "—" }}</div>
      <div><span class="muted">Bitrate:</span> {{ fmtBitrate(session.refInfo.bitrate) }}</div>
    </div>
  </div>
</template>
