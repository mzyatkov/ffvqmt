<script setup lang="ts">
import { ref } from "vue";
import { useSession } from "../stores/session";
import { API } from "../api/wails";

const session = useSession();
const dragging = ref(false);

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
      <div v-for="d in session.distFiles" :key="d.path" class="file-row" :class="{ missing: !d.exists }">
        <input type="checkbox" v-model="d.active" :disabled="!d.exists"/>
        <div class="name" :title="d.path">
          {{ baseName(d.path) }}
          <div class="badge" v-if="d.info">{{ d.info.width }}×{{ d.info.height }} · {{ d.info.videoCodec }}</div>
          <div class="badge" v-if="d.error" style="color:var(--danger)">{{ d.error }}</div>
        </div>
        <button @click="session.removeDist(d.path)" style="padding:2px 6px">✕</button>
      </div>
    </div>
  </div>
</template>
