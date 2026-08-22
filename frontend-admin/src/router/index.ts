import { createRouter, createWebHistory } from 'vue-router'
import { TOKEN_KEY } from '@/constants'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        { path: '', redirect: '/dashboard' },
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
        },
        { path: 'rules', name: 'rules', component: () => import('@/views/RulesView.vue') },
        {
          path: 'rules/new',
          name: 'rule-new',
          component: () => import('@/views/RuleEditorView.vue'),
        },
        {
          path: 'rules/:id',
          name: 'rule-edit',
          component: () => import('@/views/RuleEditorView.vue'),
        },
        { path: 'tasks', name: 'tasks', component: () => import('@/views/TasksView.vue') },
        {
          path: 'tasks/:id',
          name: 'task-detail',
          component: () => import('@/views/TaskDetailView.vue'),
        },
        { path: 'results', name: 'results', component: () => import('@/views/ResultsView.vue') },
        { path: 'proxies', name: 'proxies', component: () => import('@/views/ProxiesView.vue') },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (!to.meta.public && !token) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && token) {
    return { name: 'dashboard' }
  }
  return true
})

export default router
