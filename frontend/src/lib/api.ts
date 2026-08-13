/** Production API origin is set at deploy (`VITE_API_BASE`). Empty in local/dev (Vite proxy). */
export function apiUrl(path: string): string {
  const base = String(import.meta.env.VITE_API_BASE ?? '').replace(/\/$/, '')
  return `${base}${path}`
}
