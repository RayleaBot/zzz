import { describe, expect, it } from 'vitest'
import { exportArchiveFile, exportUIGF, parseImport, type GameID } from '../src/uigf'

const sample = { id: '1844674407370955101', item_id: '10001', name: 'Synthetic item', time: '2026-09-01 08:00:00', rank_type: '5', count: '1' }

describe('UIGF records', () => {
	 it('round-trips new ZZZ pools without mislabeling them as UIGF or merging pity groups', () => {
		const archive = { uid: '100000001', timezone: 0, lang: 'zh-cn', list: [{ ...sample, gacha_type: '102' }] }
		expect(() => exportUIGF('zzz', archive)).toThrow('UIGF')
		const file = exportArchiveFile('zzz', archive)
		expect(file.format).toBe('RayleaGacha')
		expect(parseImport(JSON.stringify(file.content), 'zzz', 8)).toEqual([archive])
		expect(parseImport(JSON.stringify(file.content), 'genshin', 8)).toEqual([])
	 })
  it.each([{ game: 'genshin', key: 'hk4e', pool: '400' }, { game: 'starrail', key: 'hkrpg', pool: '21' }, { game: 'zzz', key: 'nap', pool: '2' }] as const)('round-trips $game without changing large IDs or local time', ({ game, key, pool }) => {
    const input = { info: { version: 'v4.1', authkey: 'synthetic-must-not-transfer' }, [key]: [{ uid: 100000001, timezone: 8, lang: 'zh-cn', list: [{ ...sample, gacha_type: pool, gacha_id: '9001', cookie_token: 'synthetic-must-not-transfer' }] }] }
    const [archive] = parseImport(JSON.stringify(input), game, -5)
    expect(archive!.timezone).toBe(8)
    expect(archive!.list[0]!.id).toBe(sample.id)
    expect(archive!.list[0]!.time).toBe(sample.time)
    const output = exportUIGF(game, archive!)
    expect(JSON.stringify(output)).not.toContain('synthetic-must-not-transfer')
    expect(parseImport(JSON.stringify(output), game, 0)).toEqual([archive])
  })

  it('keeps other games out of the selected archive', () => {
    expect(parseImport(JSON.stringify({ info: { version: 'v4.0' }, hkrpg: [] }), 'genshin', 8)).toEqual([])
  })

  it('uses an explicit legacy timezone and rejects lossy numeric record IDs', () => {
    const legacy = { info: { uigf_version: 'v3.0', uid: '100000001' }, list: [{ ...sample, gacha_type: '301' }] }
    expect(parseImport(JSON.stringify(legacy), 'genshin', -5)[0]!.timezone).toBe(-5)
    legacy.list[0]!.id = 1844674407370955101 as unknown as string
    expect(() => parseImport(JSON.stringify(legacy), 'genshin', 8)).toThrow('字符串')
  })
})
