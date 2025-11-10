import { resolve } from 'node:path'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  base: './',
  build: {
    emptyOutDir: true,
    outDir: 'dist-export-viewer',
    sourcemap: false,
    lib: {
      entry: resolve(__dirname, 'src/export/viewer.ts'),
      name: 'SealChatExportViewer',
      fileName: () => 'export_viewer.js',
      formats: ['iife'],
    },
    rollupOptions: {
      output: {
        assetFileNames: () => 'export_viewer.[ext]',
      },
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        api: 'modern-compiler',
      },
    },
  },
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
})
