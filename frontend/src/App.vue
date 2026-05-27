<script setup lang="ts">
import { onMounted, ref, computed } from "vue";
import { useSession } from "./stores/session";
import MenuBar from "./components/MenuBar.vue";
import ReferencePanel from "./components/ReferencePanel.vue";
import FileList from "./components/FileList.vue";
import OptionsPanel from "./components/OptionsPanel.vue";
import ChartsArea from "./components/ChartsArea.vue";
import StatusBar from "./components/StatusBar.vue";

const session = useSession();

const showOptions = ref(false);

onMounted(async () => {
  // Add platform class for OS-specific styling (e.g. macOS traffic-light inset).
  const ua = navigator.userAgent || "";
  const platform = (navigator as any).platform || "";
  if (/Mac|iPhone|iPad|iPod/i.test(platform) || /Mac OS X/i.test(ua)) {
    document.body.classList.add("platform-mac");
  }
  session.bindEvents();
  await session.detect();
  await session.loadInitialFromCLI();
});

const ffmpegBadge = computed(() => {
  if (session.detecting) return "detecting ffmpeg…";
  if (session.detectError) return "ffmpeg: " + session.detectError;
  if (!session.probe) return "ffmpeg not detected";
  const p = session.probe;
  const feats = [
    p.hasPSNR ? "PSNR" : null,
    p.hasSSIM ? "SSIM" : null,
    p.hasVMAF ? "VMAF" : null,
    p.hasXPSNR ? "XPSNR" : null,
  ].filter(Boolean).join("/");
  return `${p.version} • ${feats}`;
});
</script>

<template>
  <div class="app-shell">
    <MenuBar :show-options="showOptions" @toggle-options="showOptions = !showOptions" />

    <div class="main">
      <aside class="sidebar">
        <section>
          <ReferencePanel />
        </section>
        <section class="flex">
          <FileList />
        </section>
        <section>
          <div class="row" style="gap:6px">
            <button class="primary" :disabled="!session.canStart" @click="session.start()">
              ▶ Start
            </button>
            <button class="danger" :disabled="!session.running" @click="session.cancel()">
              ■ Stop
            </button>
            <div class="spacer" />
            <button @click="showOptions = !showOptions">⚙ Options</button>
          </div>
        </section>
      </aside>

      <main class="workarea">
        <OptionsPanel v-if="showOptions" />
        <ChartsArea v-else />
      </main>
    </div>

    <StatusBar :ffmpeg-badge="ffmpegBadge" />

    <div v-if="session.lastError" class="toast error" @click="session.lastError = null">
      {{ session.lastError }}
    </div>
  </div>
</template>
