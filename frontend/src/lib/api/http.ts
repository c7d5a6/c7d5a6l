/** Shared API error extraction from failed responses. */
export async function readApiError(res: Response, fallback?: string): Promise<string> {
  try {
    const data = (await res.json()) as { error?: string }
    if (data.error) return data.error
  } catch {
    /* ignore non-JSON bodies */
  }
  return fallback ?? `request failed (${res.status})`
}
