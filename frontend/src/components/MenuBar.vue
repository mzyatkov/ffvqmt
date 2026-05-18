<script setup lang="ts">
import { useSession } from "../stores/session";
import { API } from "../api/wails";

defineProps<{ showOptions: boolean }>();
const emit = defineEmits<{ (e: "toggleOptions"): void }>();

const session = useSession();

async function chooseRef() {
  const paths = await API.selectFiles(false, "Select reference file");
  if (paths?.[0]) await session.setReference(paths[0]);
}
async function addDist() {
  const paths = await API.selectFiles(true, "Add distorted files");
  if (paths?.length) await session.addDistFiles(paths);
}
async function loadProject() {
  const paths = await API.selectFiles(false, "Open project");
  if (!paths?.[0]) return;
  try {
    const p = await API.loadProject(paths[0]);
    if (p.refFile) await session.setReference(p.refFile);
    if (p.distFiles?.length) {
      await session.addDistFiles(p.distFiles.map((d: any) => d.path));
    }
    if (p.metrics) {
      (["PSNR", "SSIM", "VMAF", "XPSNR"] as const).forEach((m) => {
        session.metrics[m] = p.metrics.includes(m);
      });
    }
  } catch (e: any) {
    session.lastError = e?.message || String(e);
  }
}
async function saveProject() {
  const path = await API.saveFile("Save project", "project.ffvqmtproj");
  if (!path) return;
  try {
    await API.saveProject(path, {
      version: 1,
      refFile: session.refPath,
      distFiles: session.distFiles.map((d) => ({ path: d.path, active: d.active })),
      metrics: session.enabledMetrics,
      skip: session.options.skip,
      duration: session.options.duration,
      scaling: session.options.scaling,
      vmafModel: session.options.vmafModel,
      vmafPool: session.options.vmafPool,
      vmafSubsample: session.options.vmafSubsample,
      vmafPhone: session.options.vmafPhone,
      vmafUpscale: session.options.vmafUpscale,
    });
  } catch (e: any) {
    session.lastError = e?.message || String(e);
  }
}
function clearAll() {
  session.clearDist();
}
</script>

<template>
  <div class="menu-bar">
    <div class="item" @click="chooseRef">📄 Reference…</div>
    <div class="item" @click="addDist">➕ Add files…</div>
    <div class="item" @click="clearAll">🗑 Clear</div>
    <span style="width:12px" />
    <div class="item" @click="loadProject">📂 Open project</div>
    <div class="item" @click="saveProject">💾 Save project</div>
    <div class="spacer" />
    <div class="item" :class="{ active: showOptions }" @click="emit('toggleOptions')">⚙ Options</div>
  </div>
</template>
