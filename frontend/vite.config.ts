import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'
import tailwindcss from '@tailwindcss/vite'

const rootDir = path.dirname(fileURLToPath(import.meta.url))
const certDir = path.resolve(rootDir, 'certs')
const certFile = path.join(certDir, 'c7d5a6l.lo.pem')
const keyFile = path.join(certDir, 'c7d5a6l.lo-key.pem')
const httpsReady = fs.existsSync(certFile) && fs.existsSync(keyFile)
const useLocalHttps = process.env.C7D5A6L_HTTPS === '1' || process.env.C7D5A6L_HTTPS === 'true'

if (useLocalHttps && !httpsReady) {
  throw new Error(
    'C7D5A6L_HTTPS=1 but certs missing under frontend/certs/. Run: npm run setup:https',
  )
}

function pagesBase(): string {
  const raw = process.env.C7D5A6L_PAGES_BASE?.trim()
  if (!raw) return '/'
  return raw.endsWith('/') ? raw : `${raw}/`
}

const appVersion = (
  JSON.parse(fs.readFileSync(path.join(rootDir, 'package.json'), 'utf8')) as { version: string }
).version

export default defineConfig({
  base: pagesBase(),
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(appVersion),
  },
  plugins: [solid(), tailwindcss()],
  server: useLocalHttps
    ? {
        host: 'c7d5a6l.lo',
        // Telegram Login iframe CSP only allows default HTTPS port (443).
        port: 443,
        strictPort: true,
        https: {
          cert: fs.readFileSync(certFile),
          key: fs.readFileSync(keyFile),
        },
        proxy: {
          '/api': 'http://localhost:18765',
          '/health': 'http://localhost:18765',
        },
      }
    : {
        port: 3000,
        proxy: {
          '/api': 'http://localhost:18765',
          '/health': 'http://localhost:18765',
        },
      },
})
