import { computed, onMounted, onUnmounted, ref } from 'vue'

export function useBreakpoint() {
  const width = ref(typeof window === 'undefined' ? 1440 : window.innerWidth)

  function onResize() {
    width.value = window.innerWidth
  }

  onMounted(() => {
    width.value = window.innerWidth
    window.addEventListener('resize', onResize)
  })
  onUnmounted(() => window.removeEventListener('resize', onResize))

  const isTablet = computed(() => width.value < 768)
  const isPhone = computed(() => width.value < 480)
  return { width, isTablet, isPhone }
}
