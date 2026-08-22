<script setup lang="ts">
import { useDialog, useMessage } from 'naive-ui'
import { registerFeedback } from '@/utils/feedback'

registerFeedback(useMessage(), useDialog())
</script>

<template>
  <router-view />
</template>
