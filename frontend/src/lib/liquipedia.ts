const ALLOWED_HOST = 'liquipedia.net'

export function validateLiquipediaURL(raw: string): { ok: true; url: string } | { ok: false; error: string } {
  const trimmed = raw.trim()
  if (!trimmed) {
    return { ok: false, error: 'URL is required' }
  }

  let parsed: URL
  try {
    parsed = new URL(trimmed)
  } catch {
    return { ok: false, error: 'Invalid URL' }
  }

  if (parsed.protocol !== 'https:') {
    return { ok: false, error: 'URL must use https' }
  }

  const host = parsed.hostname.toLowerCase()
  if (host !== ALLOWED_HOST && host !== `www.${ALLOWED_HOST}`) {
    return { ok: false, error: 'Only liquipedia.net links are supported' }
  }

  parsed.hash = ''
  parsed.hostname = ALLOWED_HOST
  parsed.protocol = 'https:'
  if (!parsed.pathname) {
    parsed.pathname = '/'
  }

  return { ok: true, url: parsed.toString() }
}
