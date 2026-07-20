import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/', redirect: '/tasks' },
  { path: '/tasks', name: 'tasks', component: () => import('../views/TaskListView.vue') },
  { path: '/tasks/new', name: 'task-new', component: () => import('../views/TaskFormView.vue') },
  { path: '/tasks/:id/edit', name: 'task-edit', component: () => import('../views/TaskFormView.vue'), props: true },
  {
    path: '/tasks/:id/executions',
    name: 'task-executions',
    component: () => import('../views/ExecutionHistoryView.vue'),
    props: true,
  },
  { path: '/executions', name: 'executions', component: () => import('../views/RecentRunsView.vue') },
  { path: '/groups', name: 'groups', component: () => import('../views/TaskGroupListView.vue') },
  { path: '/groups/new', name: 'group-new', component: () => import('../views/TaskGroupFormView.vue') },
  {
    path: '/groups/:id/edit',
    name: 'group-edit',
    component: () => import('../views/TaskGroupFormView.vue'),
    props: true,
  },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('../views/NotFoundView.vue'), meta: { public: true } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (auth.user === null && !to.meta.public) {
    await auth.fetchMe()
  }
  if (!to.meta.public && !auth.isLoggedIn) {
    return '/login'
  }
  if (to.path === '/login' && auth.isLoggedIn) {
    return '/tasks'
  }
})

export default router
