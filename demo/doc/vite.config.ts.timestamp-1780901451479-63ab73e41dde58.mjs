// vite.config.ts
import { defineConfig } from "file:///mnt/d/project/desktop/demo/node_modules/.pnpm/vite@5.4.21/node_modules/vite/dist/node/index.js";
import vue from "file:///mnt/d/project/desktop/demo/node_modules/.pnpm/@vitejs+plugin-vue@5.2.4_vite@5.4.21_vue@3.5.35/node_modules/@vitejs/plugin-vue/dist/index.mjs";
var vite_config_default = defineConfig({
  plugins: [vue({
    template: {
      compilerOptions: {
        // 保留模板中的空白字符（代码块换行依赖此设置）
        whitespace: "preserve"
      }
    }
  })],
  base: "/",
  server: {
    port: 5173
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    target: "es2015",
    cssTarget: "safari12"
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvbW50L2QvcHJvamVjdC9kZXNrdG9wL2RlbW9cIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9tbnQvZC9wcm9qZWN0L2Rlc2t0b3AvZGVtby92aXRlLmNvbmZpZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vbW50L2QvcHJvamVjdC9kZXNrdG9wL2RlbW8vdml0ZS5jb25maWcudHNcIjtpbXBvcnQgeyBkZWZpbmVDb25maWcgfSBmcm9tICd2aXRlJ1xuaW1wb3J0IHZ1ZSBmcm9tICdAdml0ZWpzL3BsdWdpbi12dWUnXG5cbmV4cG9ydCBkZWZhdWx0IGRlZmluZUNvbmZpZyh7XG4gIHBsdWdpbnM6IFt2dWUoe1xuICAgIHRlbXBsYXRlOiB7XG4gICAgICBjb21waWxlck9wdGlvbnM6IHtcbiAgICAgICAgLy8gXHU0RkREXHU3NTU5XHU2QTIxXHU2NzdGXHU0RTJEXHU3Njg0XHU3QTdBXHU3NjdEXHU1QjU3XHU3QjI2XHVGRjA4XHU0RUUzXHU3ODAxXHU1NzU3XHU2MzYyXHU4ODRDXHU0RjlEXHU4RDU2XHU2QjY0XHU4QkJFXHU3RjZFXHVGRjA5XG4gICAgICAgIHdoaXRlc3BhY2U6ICdwcmVzZXJ2ZScsXG4gICAgICB9LFxuICAgIH0sXG4gIH0pXSxcbiAgYmFzZTogJy8nLFxuICBzZXJ2ZXI6IHtcbiAgICBwb3J0OiA1MTczXG4gIH0sXG4gIGJ1aWxkOiB7XG4gICAgb3V0RGlyOiAnZGlzdCcsXG4gICAgYXNzZXRzRGlyOiAnYXNzZXRzJyxcbiAgICB0YXJnZXQ6ICdlczIwMTUnLFxuICAgIGNzc1RhcmdldDogJ3NhZmFyaTEyJyxcbiAgfVxufSlcbiJdLAogICJtYXBwaW5ncyI6ICI7QUFBbVEsU0FBUyxvQkFBb0I7QUFDaFMsT0FBTyxTQUFTO0FBRWhCLElBQU8sc0JBQVEsYUFBYTtBQUFBLEVBQzFCLFNBQVMsQ0FBQyxJQUFJO0FBQUEsSUFDWixVQUFVO0FBQUEsTUFDUixpQkFBaUI7QUFBQTtBQUFBLFFBRWYsWUFBWTtBQUFBLE1BQ2Q7QUFBQSxJQUNGO0FBQUEsRUFDRixDQUFDLENBQUM7QUFBQSxFQUNGLE1BQU07QUFBQSxFQUNOLFFBQVE7QUFBQSxJQUNOLE1BQU07QUFBQSxFQUNSO0FBQUEsRUFDQSxPQUFPO0FBQUEsSUFDTCxRQUFRO0FBQUEsSUFDUixXQUFXO0FBQUEsSUFDWCxRQUFRO0FBQUEsSUFDUixXQUFXO0FBQUEsRUFDYjtBQUNGLENBQUM7IiwKICAibmFtZXMiOiBbXQp9Cg==
