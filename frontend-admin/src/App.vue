<script setup lang="ts">
import { darkTheme, dateZhCN, NConfigProvider, NDialogProvider, NMessageProvider, zhCN } from 'naive-ui'
import { themeOverrides } from '@/theme/naive'
import FeedbackHost from '@/components/FeedbackHost.vue'
</script>

<template>
  <n-config-provider
    :theme="darkTheme"
    :theme-overrides="themeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <n-message-provider :duration="5000" :closable="true" :max="4" :keep-alive-on-hover="true">
      <n-dialog-provider>
        <FeedbackHost />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>
