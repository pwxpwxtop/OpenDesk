import { createRouter, createWebHashHistory } from 'vue-router'
import ServersView from '../views/ServersView.vue'
import DesktopSessionView from '../views/DesktopSessionView.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'servers', component: ServersView },
    { path: '/desktop/:sessionId', name: 'desktop', component: DesktopSessionView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

export default router
