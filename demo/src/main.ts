// Polyfill: Promise.allSettled (ES2020)，旧版 WebView 可能不支持
if (!Promise.allSettled) {
  Promise.allSettled = function <T>(promises: Iterable<T | PromiseLike<T>>) {
    return Promise.all(
      Array.from(promises).map((p) =>
        Promise.resolve(p).then(
          (value) => ({ status: 'fulfilled' as const, value }),
          (reason) => ({ status: 'rejected' as const, reason })
        )
      )
    )
  }
}

import 'resize-observer-polyfill'
import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
const app = createApp(App)
app.use(ElementPlus)
app.use(router)
app.mount('#app')
