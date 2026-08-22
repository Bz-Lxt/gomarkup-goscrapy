<script setup lang="ts">
import { computed, h, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NButton, NDrawer, NDrawerContent, NLayout, NLayoutSider, NMenu } from 'naive-ui'
import type { MenuOption } from 'naive-ui'
import SonarLogo from '@/components/SonarLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { useBreakpoint } from '@/composables/useBreakpoint'
import { confirmDanger, toastInfo } from '@/utils/feedback'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const { isTablet } = useBreakpoint()
const drawer = ref(false)

const icons: Record<string, string> = {
  dashboard: '◎',
  rules: '▣',
  tasks: '▶',
  results: '☰',
  proxies: '◈',
}

function item(label: string, key: string): MenuOption {
  return {
    label,
    key,
    icon: () => h('span', { class: 'nav-glyph' }, icons[key] ?? '·'),
  }
}

const menuOptions = computed<MenuOption[]>(() => [
  item('集群大屏', 'dashboard'),
  item('规则工坊', 'rules'),
  item('任务队列', 'tasks'),
  item('全局结果', 'results'),
  item('代理池', 'proxies'),
])

const activeKey = computed(() => {
  const name = String(route.name || '')
  if (name.startsWith('rule')) return 'rules'
  if (name.startsWith('task')) return 'tasks'
  if (name === 'results') return 'results'
  if (name === 'proxies') return 'proxies'
  return 'dashboard'
})

function go(key: string) {
  drawer.value = false
  void router.push({ name: key })
}

async function onLogout() {
  const ok = await confirmDanger({
    title: '退出登录',
    content: '确认离开作战室？未保存的规则编辑将丢失。',
    positiveText: '退出',
  })
  if (!ok) return
  auth.logout()
  toastInfo('已退出')
  void router.push({ name: 'login' })
}
</script>

<template>
  <div class="shell">
    <header v-if="isTablet" class="topbar">
      <n-button quaternary class="sonar-btn" @click="drawer = true">菜单</n-button>
      <SonarLogo compact />
      <n-button quaternary class="sonar-btn" @click="onLogout">退出</n-button>
    </header>

    <n-drawer v-model:show="drawer" placement="left" :width="240" :trap-focus="false">
      <n-drawer-content title="" :native-scrollbar="false" body-content-style="padding: 16px; background:#0C1A24">
        <SonarLogo />
        <n-menu
          class="mt-6"
          :value="activeKey"
          :options="menuOptions"
          @update:value="go"
        />
      </n-drawer-content>
    </n-drawer>

    <n-layout has-sider class="shell-layout">
      <n-layout-sider
        v-if="!isTablet"
        :width="240"
        :collapsed-width="0"
        bordered
        content-style="background: var(--bg-1); display:flex; flex-direction:column; height:100%;"
      >
        <div class="px-5 pt-6 pb-4">
          <SonarLogo />
        </div>
        <n-menu class="nav-menu" :value="activeKey" :options="menuOptions" @update:value="go" />
        <div class="mt-auto px-4 pb-5">
          <div class="mb-3 font-mono text-[11px] text-muted">{{ auth.username || 'admin' }}</div>
          <n-button class="sonar-btn w-full" secondary @click="onLogout">退出登录</n-button>
        </div>
      </n-layout-sider>
      <n-layout class="main-pane">
        <div class="main-inner">
          <router-view />
        </div>
      </n-layout>
    </n-layout>
  </div>
</template>

<style scoped>
.shell,
.shell-layout {
  height: 100%;
  min-height: 100vh;
  background: transparent;
}
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 12px;
  background: var(--bg-1);
  border-bottom: 1px solid var(--line);
}
.main-pane {
  background: transparent;
}
.main-inner {
  width: 100%;
  min-height: 100%;
  padding: 22px 24px 32px;
}
@media (max-width: 480px) {
  .main-inner {
    padding: 16px 12px 24px;
  }
}
:deep(.nav-glyph) {
  font-size: 13px;
  color: #3ee0c5;
}
:deep(.n-menu) {
  background: transparent;
}
:deep(.n-layout-sider) {
  background: var(--bg-1);
}
</style>
