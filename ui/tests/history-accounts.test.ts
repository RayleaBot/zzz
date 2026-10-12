import { describe, expect, it, vi } from 'vitest'
import { PluginUIError } from '@rayleabot/plugin-ui'
import { historicalRecords, type HistoricalRecord } from '../src/history-accounts'

const record = (protocol: string, id: string, ref = id): HistoricalRecord => ({ kind: 'groups', ref, owner: { source_protocol: protocol, bot_id: id }, label: '群', count: 1, revision: 'v1' })

describe('historical account directory', () => {
  it('checks exact accounts, keeps offline accounts, and counts only missing or removed owners', async () => {
    const read = vi.fn(async (path: string) => {
      if (path.endsWith('/removed')) throw new PluginUIError('platform.resource_not_found', 'gone', 404)
      return { state: 'no_connection' }
    })
    const rows = await historicalRecords([record('onebot11', ''), record('onebot11', 'removed'), record('onebot11', 'removed', 'second'), record('qqofficial', 'opaque:/id'), record('onebot11', 'not-in-first-page')], read)
    expect(rows.map(row => row.state)).toEqual(['missing', 'removed', 'removed'])
    expect(read).toHaveBeenCalledTimes(3)
    expect(read).toHaveBeenCalledWith('/api/bot-accounts/qqofficial/opaque%3A%2Fid')
    expect(read).toHaveBeenCalledWith('/api/bot-accounts/onebot11/not-in-first-page')
  })
  it('does not treat directory failures as missing accounts', async () => {
    for (const error of [new PluginUIError('platform.failed', 'unavailable', 503), new Error('offline')]) {
      await expect(historicalRecords([record('onebot11', 'bot')], async () => { throw error })).rejects.toBe(error)
    }
  })
})
