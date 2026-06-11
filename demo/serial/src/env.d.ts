/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface Window {
  __lhpanda__?: (method: string, params?: Record<string, any>) => Promise<any>
}
