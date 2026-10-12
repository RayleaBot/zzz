import { PluginUIError } from '@rayleabot/plugin-ui'

export interface HistoricalRecord {
  kind: string
  ref: string
  owner: { source_protocol: string; bot_id: string; actor_id?: string }
  label: string
  count: number
  revision: string
}

export async function historicalRecords(items: HistoricalRecord[], read: (path: string) => Promise<unknown>) {
  const missing = new Map<string, boolean>()
  const result: (HistoricalRecord & { state: 'missing' | 'removed' })[] = []
  for (const item of items) {
    const { source_protocol: protocol, bot_id: id } = item.owner
    if (!protocol || !id) {
      result.push({ ...item, state: 'missing' })
      continue
    }
    const key = JSON.stringify([protocol, id])
    if (!missing.has(key)) {
      try {
        await read(`/api/bot-accounts/${encodeURIComponent(protocol)}/${encodeURIComponent(id)}`)
        missing.set(key, false)
      } catch (error) {
        if (!(error instanceof PluginUIError) || error.status !== 404) throw error
        missing.set(key, true)
      }
    }
    if (missing.get(key)) result.push({ ...item, state: 'removed' })
  }
  return result
}
