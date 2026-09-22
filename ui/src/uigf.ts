export type GameID = 'genshin' | 'starrail' | 'zzz'
export interface GachaRecord { id: string; gacha_type: string; item_id: string; time: string; uigf_gacha_type?: string; gacha_id?: string; name?: string; item_type?: string; rank_type?: string; count?: string }
export interface Archive { uid: string; timezone: number; lang: string; list: GachaRecord[] }
export const fields: Record<GameID, string> = { genshin: 'hk4e', starrail: 'hkrpg', zzz: 'nap' }
const recordKeys = ['id', 'gacha_type', 'item_id', 'time', 'uigf_gacha_type', 'gacha_id', 'name', 'item_type', 'rank_type', 'count'] as const
function object(value: unknown): Record<string, unknown> { if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('记录文件结构不正确。'); return value as Record<string, unknown> }
function uid(value: unknown): string { if (typeof value === 'number' && !Number.isSafeInteger(value)) throw new Error('UID 超出安全数值范围，请使用字符串。'); const result = String(value ?? ''); if (!/^\d{1,19}$/.test(result)) throw new Error('缺少有效 UID。'); return result }
function record(value: unknown, game: GameID): GachaRecord {
  const raw = object(value)
  const result: Record<string, string> = {}
  for (const key of recordKeys) if (raw[key] !== undefined) { if (typeof raw[key] !== 'string') throw new Error(`记录字段 ${key} 必须是字符串。`); result[key] = raw[key] }
  if (!/^\d{1,19}$/.test(result.id ?? '') || !/^\d+$/.test(result.item_id ?? '') || !result.gacha_type || !/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(result.time ?? '')) throw new Error('记录缺少 ID、物品、卡池或时间。')
  if (game === 'genshin' && !result.uigf_gacha_type) result.uigf_gacha_type = result.gacha_type === '400' ? '301' : result.gacha_type
  if (game === 'starrail' && !result.gacha_id) throw new Error('星铁记录缺少卡池 ID。')
  return result as unknown as GachaRecord
}

// Only standardized business fields are copied. Unknown metadata, auth links
// and credential-shaped fields from an imported file never enter host IPC.
export function parseImport(text: string, game: GameID, legacyTimezone: number): Archive[] {
  if (new TextEncoder().encode(text).length > 64 * 1024 * 1024) throw new Error('文件超过 64 MiB，请拆分后导入。')
  const root = object(JSON.parse(text)), info = object(root.info)
  let sources: unknown[]
  const version = String(info.version ?? '')
  if (info.format === 'raylea-gacha' && info.version === 1) {
    if (root.game !== game) return []
    sources = [root.archive]
  } else if (['v4.0', 'v4.1', 'v4.2'].includes(version)) {
    sources = root[fields[game]] as unknown[] ?? []
    if (!Array.isArray(sources)) throw new Error('游戏记录必须是账号数组。')
  } else {
    const legacyVersion = String(info.uigf_version ?? info.srgf_version ?? '')
    if (!(game === 'genshin' && /^v?[23]\./.test(legacyVersion)) && !(game === 'starrail' && /^v?1\./.test(legacyVersion))) throw new Error('不支持此记录版本。请选择 UIGF v4、原神旧版 UIGF 或星铁 SRGF。')
    sources = [{ uid: info.uid, timezone: info.region_time_zone ?? legacyTimezone, lang: info.lang, list: root.list }]
  }
  if (sources.length > 128) throw new Error('单个文件账号过多，请拆分后导入。')
  return sources.map(value => {
    const source = object(value)
    if (!Array.isArray(source.list) || source.list.length > 200000) throw new Error('单个账号需要有效记录列表，且不能超过 20 万条。')
    const timezone = source.timezone
    if (typeof timezone !== 'number' || !Number.isInteger(timezone) || timezone < -12 || timezone > 14) throw new Error('缺少有效的游戏区服时区。')
    return { uid: uid(source.uid), timezone, lang: typeof source.lang === 'string' ? source.lang : 'zh-cn', list: source.list.map(value => record(value, game)) }
  })
}

export function exportUIGF(game: GameID, archive: Archive): Record<string, unknown> {
  if (game === 'zzz' && archive.list.some(item => ['102', '103'].includes(item.gacha_type))) throw new Error('此档案含 UIGF 尚未收录的频段，请使用完整档案导出。')
  return { info: { export_timestamp: Math.floor(Date.now() / 1000), export_app: `RayleaBot ${game}`, export_app_version: '0.1.0', version: 'v4.1' }, [fields[game]]: [{ uid: archive.uid, timezone: archive.timezone, lang: archive.lang, list: archive.list.map(value => record(value, game)) }] }
}

export function exportArchiveFile(game: GameID, archive: Archive): { format: string; content: Record<string, unknown> } {
  if (game === 'zzz' && archive.list.some(item => ['102', '103'].includes(item.gacha_type))) {
    return { format: 'RayleaGacha', content: { info: { format: 'raylea-gacha', version: 1 }, game, archive: { uid: archive.uid, timezone: archive.timezone, lang: archive.lang, list: archive.list.map(item => record(item, game)) } } }
  }
  return { format: 'UIGF', content: exportUIGF(game, archive) }
}
