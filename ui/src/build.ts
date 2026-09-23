export type BuildInvoke = <T>(action: string, payload?: Record<string, unknown>) => Promise<T>
export interface BuildWeapon { id: string; name?: string; level: number; promote: number | null; refinement: number }
export interface BuildConditions { disable_character:boolean; disable_weapon:boolean; disable_equipment:boolean; enemy_resistance?:number; bonuses:Record<string,number>; team:{character_id:string;name:string;bonuses:Record<string,number>}[] }
export interface BuildSkill { id: string; title: string; expected: number | null; critical: number | null; text?: string; buffs: string[]; kind: string }
export interface BuildScenario { weapon: BuildWeapon; attributes: Record<string, number>; results: BuildSkill[] }
export interface CharacterBuild { source: 'simulation'; version: string; character: string; enemy_level: number; baseline: BuildScenario; candidate: BuildScenario | null; equipment_from?: { character_id: string; name: string } }
export interface BuildPrepared { character: string; weapon: BuildWeapon; weapons: { id: string; name: string }[]; talents: Record<string, number>; enemy_level: number; version: string; build: CharacterBuild }
export function alignBuildRows(result: CharacterBuild) {
  const before = new Map(result.baseline.results.map(row => [row.id, row])), after = new Map(result.candidate?.results.map(row => [row.id, row]) ?? [])
  return [...new Set([...before.keys(), ...after.keys()])].map(id => ({ id, title: before.get(id)?.title ?? after.get(id)!.title, before: before.get(id), after: after.get(id) }))
}
export function damageGain(before: number | null | undefined, after: number | null | undefined): number | null {
  if (before == null || after == null || !Number.isFinite(before) || !Number.isFinite(after) || before === 0) return null
  return (after - before) / before * 100
}
export function weaponPromotions(level: number) {
  const steps = [1, 10, 20, 30, 40, 50, 60]
  return steps.slice(0, -1).flatMap((start, index) => level >= start && level <= steps[index + 1]! ? [index] : [])
}
