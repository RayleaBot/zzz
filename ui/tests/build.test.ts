import { describe, expect, it } from 'vitest'
import { alignBuildRows, damageGain, weaponPromotions, type CharacterBuild } from '../src/build'
describe('character build comparison', () => {
  it('aligns scenario identities and preserves conditional additions', () => {
    const row = (id: string) => ({ id, title: id, expected: 100, critical: null, buffs: [], kind: 'damage' })
    const weapon = { id: '', level: 0, promote: 0, refinement: 1 }
    const result: CharacterBuild = { source: 'simulation', version: 'fixture', character: 'fixture', enemy_level: 90, baseline: { weapon, attributes: {}, results: [row('1'), row('3')] }, candidate: { weapon, attributes: {}, results: [row('2'), row('3')] } }
    expect(alignBuildRows(result).map(r => [r.id, r.before?.id, r.after?.id])).toEqual([['1', '1', undefined], ['3', '3', '3'], ['2', undefined, '2']])
  })
  it('does not turn an absent or zero baseline into invented growth', () => {
    expect(damageGain(0, 120)).toBeNull(); expect(damageGain(null, 10)).toBeNull(); expect(damageGain(100, undefined)).toBeNull()
    expect(damageGain(100, 120)).toBe(20); expect(damageGain(100, 0)).toBe(-100)
  })
  it('allows both legitimate breakthrough stages at a level boundary', () => {
    expect(weaponPromotions('genshin', 80)).toEqual([5, 6]); expect(weaponPromotions('genshin', 90)).toEqual([6])
    expect(weaponPromotions('starrail', 80)).toEqual([6]); expect(weaponPromotions('starrail', 81)).toEqual([])
  })
})
