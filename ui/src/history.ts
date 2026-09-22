import type { GachaRecord, GameID } from './uigf'
export interface Interval { pulls: number; lower_bound: boolean; uncertain: boolean }
export interface History {
  records: (GachaRecord & { interval?: Interval })[]; total: number; next_offset: number | null; revision: string
  analytics: { ranks: Record<string, number>; top_rate: number | null; complete_intervals: number; average_interval: number | null; from: string; to: string; months: { month: string; total: number; top: number; unknown: number }[]; items: { id: string; name: string; rank: string; count: number }[]; item_kinds: number }
}
export const poolNames: Record<GameID, Record<string, string>> = {
  genshin: { '100': '新手祈愿', '200': '常驻祈愿', '301': '角色活动祈愿（共享）', '302': '武器活动祈愿', '500': '集录祈愿' },
  starrail: { '1': '常驻跃迁', '2': '始发跃迁', '11': '角色活动跃迁', '12': '光锥活动跃迁', '21': '联动角色跃迁', '22': '联动光锥跃迁' },
  zzz: { '1': '常驻频道', '2': '独家频道', '3': '音擎频道', '5': '邦布频道', '102': '独家重映', '103': '音擎回响' },
}
export function intervalText(value?: Interval): string { return !value ? '—' : value.uncertain ? '稀有度缺失，间隔不确定' : `${value.lower_bound ? '至少 ' : ''}${value.pulls} 抽` }
export function rarityName(game: GameID, rank: string): string { return rank === 'unknown' || !rank ? '未知稀有度' : game === 'zzz' ? ({ '4': 'S 级', '3': 'A 级', '2': 'B 级' }[rank] ?? rank) : `${rank} 星` }
