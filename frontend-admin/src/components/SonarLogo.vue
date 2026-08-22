<script setup lang="ts">
withDefaults(defineProps<{ compact?: boolean }>(), { compact: false })
</script>

<template>
  <div class="flex items-center gap-3">
    <svg class="sonar-ring" :width="compact ? 28 : 36" :height="compact ? 28 : 36" viewBox="0 0 64 64" aria-hidden="true">
      <circle cx="32" cy="32" r="28" fill="none" stroke="#1E3A47" stroke-width="2" />
      <circle cx="32" cy="32" r="18" fill="none" stroke="#3EE0C5" stroke-opacity="0.45" stroke-width="1.5" />
      <circle cx="32" cy="32" r="8" fill="none" stroke="#3EE0C5" stroke-width="1.6" />
      <circle cx="32" cy="32" r="2.6" fill="#3EE0C5" />
      <path d="M32 32 L54 18" stroke="#3EE0C5" stroke-width="1.4" stroke-linecap="round" class="sweep" />
    </svg>
    <div v-if="!compact" class="leading-none">
      <div class="font-display tracking-[0.22em] text-[18px] text-ink">GOSCRAPY</div>
      <div class="mt-1 text-[10px] uppercase tracking-[0.28em] text-muted">Sonar Ops</div>
    </div>
  </div>
</template>

<style scoped>
.sonar-ring {
  filter: drop-shadow(0 0 6px rgba(62, 224, 197, 0.35));
}
.sweep {
  transform-origin: 32px 32px;
  animation: sweep 6s linear infinite;
}
@keyframes sweep {
  to {
    transform: rotate(360deg);
  }
}
</style>
