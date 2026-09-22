import { describe, expect, it } from 'vitest'
import { driveSync, type SyncInfo, type SyncInvoke } from '../src/sync'

const initial: SyncInfo = { ref: 'synthetic-task', state: 'running', sequence: 0, pool: '100', pages: 0, fetched: 0 }
describe('official sync', () => {
  it('resumes from the last acknowledged page after failure', async () => {
    let saved = initial, calls = 0
    const invoke: SyncInvoke = async <T>(_action: string, params: Record<string, unknown>) => {
      calls++
      if (calls === 2) throw new Error('temporary')
      return { sync: { ...initial, sequence: Number(params.sequence) + 1, state: calls === 3 ? 'completed' : 'running' } } as T
    }
    const progress = (value: SyncInfo) => { saved = value }
    await expect(driveSync(invoke, initial, new AbortController().signal, progress, async () => {})).rejects.toThrow('temporary')
    expect(saved.sequence).toBe(1)
    const result = await driveSync(invoke, saved, new AbortController().signal, progress, async () => {})
    expect(result.state).toBe('completed'); expect(result.sequence).toBe(2)
  })
  it('cancels before another page and respects a completed in-flight commit', async () => {
    const canceled = new AbortController(); canceled.abort()
    const actions: string[] = []
    const invoke: SyncInvoke = async <T>(action: string) => { actions.push(action); return {} as T }
    expect((await driveSync(invoke, initial, canceled.signal, () => {})).state).toBe('canceled')
    expect(actions).toEqual(['gacha.sync.cancel'])
    const during = new AbortController()
    const committing: SyncInvoke = async <T>() => { during.abort(); return { sync: { ...initial, state: 'completed' } } as T }
    expect((await driveSync(committing, initial, during.signal, () => {})).state).toBe('completed')
  })
})
