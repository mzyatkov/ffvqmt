<script setup lang="ts">
import { useSession } from "../stores/session";

defineProps<{ ffmpegBadge: string }>();
const session = useSession();
</script>

<template>
  <div class="statusbar">
    <span>{{ ffmpegBadge }}</span>
    <span class="spacer"/>
    <span v-if="session.running">Running… {{ session.progress }}%</span>
    <span v-else-if="session.progress >= 100">Done</span>
    <span v-else>Idle</span>
  </div>
</template>
