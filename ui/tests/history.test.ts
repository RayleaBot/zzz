import { describe, expect, it } from 'vitest'
import { intervalText, rarityName } from '../src/history'
describe('local gacha interval boundaries', () => {
  it('distinguishes unknown ranks from missing earlier history', () => {
    expect(intervalText({ pulls: 10, lower_bound: true, uncertain: false })).toBe('至少 10 抽')
    expect(intervalText({ pulls: 10, lower_bound: true, uncertain: true })).toBe('稀有度缺失，间隔不确定')
    expect(intervalText({ pulls: 0, lower_bound: false, uncertain: false })).toBe('0 抽')
    expect(intervalText()).toBe('—')
  })
  it('uses game-specific rank meaning', () => {
    expect(rarityName('4')).toBe('S 级')
    expect(rarityName('2')).toBe('B 级')
  })
})
