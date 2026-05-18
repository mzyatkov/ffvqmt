<script setup lang="ts">
import { useSession } from "../stores/session";
import { API } from "../api/wails";

const session = useSession();

async function pickDir(target: "logFramesDir" | "tempDir") {
  const d = await API.selectDir("Choose directory");
  if (d) session.options[target] = d;
}
async function pickResultsFile() {
  const p = await API.saveFile("Choose results CSV", "FFvqmt.results.csv");
  if (p) session.options.autoSaveResultsFile = p;
}
</script>

<template>
  <div class="scroll" style="padding:16px 18px">
    <h2 style="margin-top:0">Options</h2>

    <h3>Metrics</h3>
    <div class="metrics">
      <label v-for="m in (['PSNR','SSIM','VMAF','XPSNR'] as const)" :key="m">
        <input type="checkbox" v-model="session.metrics[m]"
               :disabled="!session.probe || !session.probe['has' + m as 'hasPSNR']"/>
        {{ m }}
      </label>
    </div>

    <h3>Sampling</h3>
    <div class="row" style="gap:16px; flex-wrap:wrap">
      <label>Skip (s) <input type="number" v-model.number="session.options.skip" min="0" step="0.1"/></label>
      <label>Duration (s) <input type="number" v-model.number="session.options.duration" min="0" step="0.1"/></label>
      <label>Scaling
        <select v-model="session.options.scaling">
          <option v-for="s in ['NEIGHBOR','GAUSS','BILINEAR','BICUBIC','LANCZOS','SINC','SPLINE']" :key="s" :value="s">{{ s }}</option>
        </select>
      </label>
      <label>Plot speed
        <select v-model="session.options.plotUpdateSpeed">
          <option v-for="s in ['HIGH','NORMAL','LOW','OFF']" :key="s" :value="s">{{ s }}</option>
        </select>
      </label>
      <label>Threads <input type="number" v-model.number="session.options.nThreads" min="0"/></label>
    </div>

    <h3>VMAF</h3>
    <div class="row" style="gap:16px; flex-wrap:wrap">
      <label>Model
        <select v-model="session.options.vmafModel" style="min-width:240px">
          <option value="">auto-detect</option>
          <option v-for="m in (session.probe?.vmafModels || [])" :key="m" :value="m">{{ m }}</option>
          <option v-if="session.probe?.inBuildVMAF" value="vmaf_v0.6.1">in-build: vmaf_v0.6.1</option>
          <option v-if="session.probe?.inBuildVMAF" value="vmaf_4k_v0.6.1">in-build: vmaf_4k_v0.6.1</option>
        </select>
      </label>
      <label>Pool
        <select v-model="session.options.vmafPool">
          <option value="MEAN">MEAN</option>
          <option value="HARMONIC_MEAN">HARMONIC_MEAN</option>
        </select>
      </label>
      <label>Subsample <input type="number" v-model.number="session.options.vmafSubsample" min="1"/></label>
      <label><input type="checkbox" v-model="session.options.vmafPhone"/> Phone model</label>
      <label><input type="checkbox" v-model="session.options.vmafUpscale"/> Upscale to model size</label>
    </div>

    <h3>Logging</h3>
    <div class="col" style="gap:6px">
      <label><input type="checkbox" v-model="session.options.logCommands"/> Write ffmpeg commands to FFvqmt.log</label>
      <label><input type="checkbox" v-model="session.options.logFrames"/> Save per-frame metrics to CSV</label>
      <div class="row" style="gap:6px">
        <label style="flex:1">Per-frame CSV directory
          <input type="text" v-model="session.options.logFramesDir" style="width:100%"/>
        </label>
        <button style="margin-top:18px" @click="pickDir('logFramesDir')">…</button>
      </div>
      <div class="row" style="gap:6px">
        <label style="flex:1">Temp directory
          <input type="text" v-model="session.options.tempDir" style="width:100%" placeholder="default user temp"/>
        </label>
        <button style="margin-top:18px" @click="pickDir('tempDir')">…</button>
      </div>
      <label><input type="checkbox" v-model="session.options.autoSaveResults"/> Auto-save aggregate results</label>
      <div class="row" style="gap:6px">
        <label style="flex:1">Aggregate CSV file
          <input type="text" v-model="session.options.autoSaveResultsFile" style="width:100%"/>
        </label>
        <button style="margin-top:18px" @click="pickResultsFile">…</button>
      </div>
    </div>
  </div>
</template>
