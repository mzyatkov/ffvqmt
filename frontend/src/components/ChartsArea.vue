<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useSession } from "../stores/session";
import MetricChart from "./MetricChart.vue";
import type { Metric } from "../api/wails";

const session = useSession();

const tabs = computed<Metric[]>(() => session.enabledMetrics);
const active = ref<Metric | null>(null);

watch(tabs, (t) => {
  if (!active.value || !t.includes(active.value)) {
    active.value = t[0] || null;
  }
}, { immediate: true });
</script>

<template>
  <div style="display:flex; flex-direction:column; min-height:0; flex:1">
    <div class="tabs">
      <div
        v-for="m in tabs"
        :key="m"
        class="tab"
        :class="{ active: active === m }"
        @click="active = m"
      >
        {{ m }}
      </div>
      <div v-if="!tabs.length" class="muted" style="padding:6px 12px; font-size:12px">
        Select at least one metric in Options.
      </div>
    </div>
    <div class="chart-wrap">
      <MetricChart v-if="active" :metric="active" />
    </div>

    <div v-if="active && session.series[active]" style="border-top:1px solid var(--border); padding:8px 14px; max-height:160px; overflow:auto">
      <table class="summary">
        <thead>
          <tr><th>File</th><th>Frames</th><th>Min</th><th>Max</th><th>Average</th><th>Harmonic</th></tr>
        </thead>
        <tbody>
          <tr v-for="(s, file) in session.series[active].summaryByFile" :key="file">
            <td :title="file">{{ file.split(/[\\/]/).pop() }}</td>
            <td>{{ s.frames }}</td>
            <td>{{ s.min.toFixed(4) }}</td>
            <td>{{ s.max.toFixed(4) }}</td>
            <td>{{ s.average.toFixed(4) }}</td>
            <td>{{ s.harmonic.toFixed(4) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
