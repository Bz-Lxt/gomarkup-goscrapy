<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NForm, NFormItem, NInput } from 'naive-ui'
import SonarLogo from '@/components/SonarLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { hasErrors, maxLen, required, type ErrorMap } from '@/utils/validate'
import { toastError, toastSuccess } from '@/utils/feedback'

const auth = useAuthStore()
const router = useRouter()
const route = useRoute()
const loading = ref(false)
const form = reactive({ username: 'admin', password: 'Admin@12345' })
const errors = reactive<ErrorMap>({})

function validate(): boolean {
  errors.username = required(form.username, '用户名') || maxLen(form.username, 64, '用户名')
  errors.password = required(form.password, '密码') || maxLen(form.password, 128, '密码')
  return !hasErrors(errors)
}

async function onSubmit() {
  if (!validate()) {
    toastError('请先修正登录表单')
    return
  }
  loading.value = true
  try {
    await auth.login(form.username.trim(), form.password)
    toastSuccess('声呐已解锁')
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.replace(redirect)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-screen">
    <div class="login-card">
      <SonarLogo />
      <h1 class="font-display mt-6 text-[32px] tracking-wide">深海声呐作战室</h1>
      <p class="mt-2 text-sm text-muted">输入值班凭证，接入 Master 控制面。</p>

      <n-form class="mt-8" @submit.prevent="onSubmit">
        <n-form-item label="用户名">
          <n-input v-model:value="form.username" placeholder="用户名" @keydown.enter="onSubmit" />
          <div v-if="errors.username" class="field-error">{{ errors.username }}</div>
        </n-form-item>
        <n-form-item label="密码">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="密码"
            @keydown.enter="onSubmit"
          />
          <div v-if="errors.password" class="field-error">{{ errors.password }}</div>
        </n-form-item>
        <n-button
          class="sonar-btn mt-2 w-full"
          type="primary"
          size="large"
          :loading="loading"
          attr-type="submit"
          @click="onSubmit"
        >
          接入集群
        </n-button>
      </n-form>
      <p class="mt-5 font-mono text-[11px] text-muted">测试账号 admin / Admin@12345</p>
    </div>
  </div>
</template>

<style scoped>
.login-screen {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.login-card {
  width: 100%;
  max-width: 420px;
  padding: 36px 32px 28px;
  border-radius: 18px;
  background: rgba(12, 26, 36, 0.66);
  border: 1px solid rgba(62, 224, 197, 0.22);
  backdrop-filter: blur(18px);
  box-shadow: 0 0 48px rgba(62, 224, 197, 0.08);
}
</style>
