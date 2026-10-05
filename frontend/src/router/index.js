import { createRouter, createWebHistory } from 'vue-router'
import ProjectList from '../pages/ProjectList.vue'
import ProjectDetail from '../pages/ProjectDetail.vue'
import Settings from '../pages/Settings.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'projects', component: ProjectList },
    { path: '/projects/:id', name: 'project-detail', component: ProjectDetail },
    { path: '/settings', name: 'settings', component: Settings },
  ],
})

export default router
