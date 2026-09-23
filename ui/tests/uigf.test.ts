import { describe, expect, it } from 'vitest'
import { exportArchiveFile, exportUIGF, parseImport } from '../src/uigf'

const sample = { id: '1844674407370955101', item_id: '10001', name: 'Synthetic item', time: '2026-09-01 08:00:00', rank_type: '5', count: '1' }

describe('UIGF records', () => {
	 it('round-trips new ZZZ pools without mislabeling them as UIGF or merging pity groups', () => {
		const archive = { uid: '100000001', timezone: 0, lang: 'zh-cn', list: [{ ...sample, gacha_type: '102' }] }
		expect(() => exportUIGF(archive)).toThrow('UIGF')
		const file = exportArchiveFile(archive)
		expect(file.format).toBe('RayleaGacha')
		expect(parseImport(JSON.stringify(file.content))).toEqual([archive])
		expect(parseImport(JSON.stringify({ ...file.content, game: 'other' }))).toEqual([])
	 })
  it('round-trips without changing large IDs or local time', () => {
    const input = { info: { version: 'v4.1', authkey: 'synthetic-must-not-transfer' }, nap: [{ uid: 100000001, timezone: 8, lang: 'zh-cn', list: [{ ...sample, gacha_type: '2', gacha_id: '9001', cookie_token: 'synthetic-must-not-transfer' }] }] }
    const [archive] = parseImport(JSON.stringify(input))
    expect(archive!.timezone).toBe(8)
    expect(archive!.list[0]!.id).toBe(sample.id)
    expect(archive!.list[0]!.time).toBe(sample.time)
    const output = exportUIGF(archive!)
    expect(JSON.stringify(output)).not.toContain('synthetic-must-not-transfer')
    expect(parseImport(JSON.stringify(output))).toEqual([archive])
  })

  it('keeps other games out of the selected archive', () => {
    expect(parseImport(JSON.stringify({ info: { version: 'v4.0' }, hkrpg: [] }))).toEqual([])
  })

  it('rejects legacy formats and lossy numeric record IDs', () => {
    expect(() => parseImport(JSON.stringify({ info: { uigf_version: 'v3.0', uid: '100000001' }, list: [{ ...sample, gacha_type: '2' }] }))).toThrow('不支持此记录版本')
    const input = { info: { version: 'v4.1' }, nap: [{ uid: '100000001', timezone: 8, list: [{ ...sample, gacha_type: '2', id: 1844674407370955101 as unknown as string }] }] }
    expect(() => parseImport(JSON.stringify(input))).toThrow('字符串')
  })
})
