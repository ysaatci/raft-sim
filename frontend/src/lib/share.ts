import type { Config, RecordedAction } from '@/client/types'

/**
 * What a shared link reproduces. Runs are deterministic, so a scenario ID,
 * or a config plus the recorded actions, rebuilds the exact same run.
 */
export type Link =
  { scenario: string; time: number } | { config: Config; actions: RecordedAction[]; time: number }

/** Encodes a link as a URL fragment (without the leading '#'). */
export function encodeLink(link: Link): string {
  const p = new URLSearchParams()
  if ('scenario' in link) p.set('scenario', link.scenario)
  else p.set('run', toBase64Url(JSON.stringify({ c: link.config, a: link.actions })))
  p.set('t', String(Math.round(link.time)))
  return p.toString()
}

/** Decodes a URL fragment; null if it is not a valid link. */
export function decodeLink(fragment: string): Link | null {
  const p = new URLSearchParams(fragment.replace(/^#/, ''))
  const time = Number(p.get('t') ?? 0)
  if (!Number.isFinite(time) || time < 0) return null
  const scenario = p.get('scenario')
  if (scenario) return { scenario, time }
  const run = p.get('run')
  if (!run) return null
  try {
    const { c, a } = JSON.parse(fromBase64Url(run))
    if (typeof c !== 'object' || !Array.isArray(a)) return null
    return { config: c, actions: a, time }
  } catch {
    return null
  }
}

function toBase64Url(s: string): string {
  const bytes = new TextEncoder().encode(s)
  return btoa(String.fromCharCode(...bytes))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
}

function fromBase64Url(s: string): string {
  const bin = atob(s.replace(/-/g, '+').replace(/_/g, '/'))
  return new TextDecoder().decode(Uint8Array.from(bin, (ch) => ch.charCodeAt(0)))
}
