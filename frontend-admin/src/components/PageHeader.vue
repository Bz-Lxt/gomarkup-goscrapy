<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<template>
  <header class="mb-5 flex w-full flex-col gap-3 tablet:gap-3 md:flex-row md:items-end md:justify-between">
    <div>
      <h1 class="font-display text-[28px] tracking-wide text-ink">{{ title }}</h1>
      <p v-if="subtitle" class="mt-1 text-sm text-muted">{{ subtitle }}</p>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <slot />
    </div>
  </header>
</template>
