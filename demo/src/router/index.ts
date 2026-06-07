import { createRouter, createWebHashHistory } from 'vue-router'
import Home from '../views/Home.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: Home },
    {
      path: '/preferences',
      name: 'preferences',
      component: () => import('../views/Preferences.vue'),
    },
    {
      path: '/docs',
      component: () => import('../views/Documentation.vue'),
      children: [
        { path: '', name: 'docs', component: () => import('../views/docs/Overview.vue') },
        { path: 'credentials', name: 'docs-credentials', component: () => import('../views/docs/CredentialsDoc.vue') },
        { path: 'proxy', name: 'docs-proxy', component: () => import('../views/docs/ProxyDoc.vue') },
        { path: 'window', name: 'docs-window', component: () => import('../views/docs/WindowDoc.vue') },
        { path: 'build', name: 'docs-build', component: () => import('../views/docs/BuildDoc.vue') },
        { path: 'bridge', name: 'docs-bridge', component: () => import('../views/docs/BridgeDoc.vue') },
      ],
    },
  ],
})

export default router
