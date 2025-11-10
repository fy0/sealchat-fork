import { promises as fs } from 'node:fs'
import path from 'node:path'

const distDir = path.resolve(process.cwd(), 'dist-export-viewer')
const embedDir = path.resolve(process.cwd(), '../service/embed')

async function ensureDir(target) {
  await fs.mkdir(target, { recursive: true })
}

async function copyAsset(source, target) {
  await fs.copyFile(source, target)
  console.log(`[viewer] synced ${path.basename(source)} -> ${target}`)
}

async function main() {
  await ensureDir(embedDir)
  const cssSource = path.join(distDir, 'export_viewer.css')
  const jsSource = path.join(distDir, 'export_viewer.js')

  await copyAsset(cssSource, path.join(embedDir, 'export_viewer.css'))
  await copyAsset(jsSource, path.join(embedDir, 'export_viewer.js'))
}

main().catch((err) => {
  console.error('[viewer] 同步失败:', err)
  process.exit(1)
})
