<script setup lang="ts">
import { ref } from "vue";
import { useSession } from "../stores/session";
import { API, type RawFormat } from "../api/wails";
import RawFormatForm from "./RawFormatForm.vue";

const session = useSession();
const dragging = ref(false);
const expanded = ref<Record<string, boolean>>({});

function toggleRaw(path: string) {
  expanded.value[path] = !expanded.value[path];
}
function applyRaw(path: string, r: RawFormat) {
  void session.setDistRaw(path, r);
  // keep the panel open so the user can see the refreshed thumbnail and tweak again
}

async function pick() {
  const paths = await API.selectFiles(true, "Add distorted files");
  if (paths?.length) await session.addDistFiles(paths);
}
function baseName(p: string) {
  const m = p.replace(/\\/g, "/").split("/");
  return m[m.length - 1] || p;
}
function onDragOver(e: DragEvent) { e.preventDefault(); dragging.value = true; }
function onDragLeave() { dragging.value = false; }
async function onDrop(e: DragEvent) {
  e.preventDefault();
  dragging.value = false;
  const files = e.dataTransfer?.files;
  if (!files) return;
  const paths: string[] = [];
  for (let i = 0; i < files.length; i++) {
    const p = (files[i] as any).path;
    if (p) paths.push(p);
  }
  if (paths.length) await session.addDistFiles(paths);
}
</script>

<template>
  <div class="col" style="flex:1; min-height:0">
    <div class="row">
      <b>Distorted files ({{ session.distFiles.length }})</b>
      <span class="spacer"/>
      <button @click="pick">＋</button>
      <button @click="session.clearDist()" :disabled="!session.distFiles.length">🗑</button>
    </div>
    <div
      class="file-list"
      :class="{ dragging }"
      @dragover="onDragOver"
      @dragleave="onDragLeave"
      @drop="onDrop"
    >
      <div v-if="!session.distFiles.length" class="drop-zone" style="margin:8px 4px">
        Drop video files here<br/>or click <b>＋</b>
      </div>
      <template v-for="d in session.distFiles" :key="d.path">
        <div class="file-row" :class="{ missing: !d.exists }">
          <input type="checkbox" v-model="d.active" :disabled="!d.exists || (d.needsRaw && !d.raw)"/>
          <div class="name" :title="d.path">
            {{ baseName(d.path) }}
            <div class="badge" v-if="d.needsRaw" style="background:#c87f00; color:#fff">
              RAW {{ d.raw ? `${d.raw.pixFmt} ${d.raw.width}×${d.raw.height}@${d.raw.frameRate}` : "needs params" }}
            </div>
            <div class="badge" v-else-if="d.info">{{ d.info.width }}×{{ d.info.height }} · {{ d.info.videoCodec }}</div>
            <div class="badge" v-if="d.error" style="color:var(--danger)">{{ d.error }}</div>
          </div>
          <button v-if="d.needsRaw" @click="toggleRaw(d.path)" style="padding:2px 6px" title="Raw parameters">⚙</button>
          <button @click="session.removeDist(d.path)" style="padding:2px 6px">✕</button>
        </div>
        <div v-if="d.needsRaw && expanded[d.path]" style="margin:4px 24px 8px" class="col" :style="{gap:'6px'}">
          <RawFormatForm
            :model-value="d.raw"
            :label="'Raw parameters for ' + baseName(d.path)"
            @apply="(r: RawFormat) => applyRaw(d.path, r)"
          />
          <img v-if="d.thumb"
               :src="d.thumb"
               alt="preview"
               style="max-width:240px; max-height:120px; object-fit:contain; background:#000; border-radius:4px"/>
        </div>
      </template>
    </div>
  </div>
</template>
