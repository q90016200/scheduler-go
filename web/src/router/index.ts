import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/LoginView.vue'), meta: { public: true } },
  { path: '/', redirect: '/tasks' },
  { path: '/tasks', name: 'tasks', component: () => import('../views/TaskListView.vue') },
  {
    path: '/tasks/:id/executions',
    name: 'task-executions',
    component: () => import('../views/TaskExecutionHistoryView.vue'),
    props: true,
  },
  { path: '/tasks/:id', name: 'task-detail', component: () => import('../views/TaskDetailView.vue'), props: true },
  { path: '/executions', name: 'all-executions', component: () => import('../views/AllExecutionsView.vue') },
  {
    path: '/executions/:id',
    name: 'execution-detail',
    component: () => import('../views/ExecutionLogDetailView.vue'),
    props: true,
  },
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

export default router
