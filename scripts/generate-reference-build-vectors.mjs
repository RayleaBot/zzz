// Compares the packaged calculator with ZZZ-Plugin's own damage pipeline on
// synthetic agents and records the results as regression vectors, with one
// official-shaped panel per agent for the build tests. No account,
// application or external API is used.
//
// Usage: node scripts/generate-reference-build-vectors.mjs
// Writes internal/assets/testdata/calc-vectors.json and
// internal/app/testdata/build-panels.json.
import fs from 'node:fs/promises'
import path from 'node:path'
import vm from 'node:vm'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const calc = path.join(root, 'internal/assets/calc')
const metadata = JSON.parse(await fs.readFile(path.join(calc, 'catalog.json'), 'utf8'))

const scripts = new Map()
async function context(record) {
  const context = vm.createContext({ record })
  const files = [path.join(root, 'internal/reference/vendor/lodash.js'), ...['data.js', 'bootstrap.js', 'common.js', 'buffs.js', record.script, 'runner.js'].map(file => path.join(calc, file))]
  for (const file of files) {
    if (!scripts.has(file)) scripts.set(file, new vm.Script(await fs.readFile(file, 'utf8'), { filename: file }))
    scripts.get(file).runInContext(context, { timeout: 1000 })
  }
  return context
}

// Upstream errors and warnings fail the case instead of passing silently.
const strictLogger = `
  logger.error = (...items) => { throw items.find(v => v && v.stack) || Error('reference.error') }
  logger.warn = (...items) => { throw Error(items.map(v => typeof v === 'string' ? v : JSON.stringify(v)).join(' ')) }`

// syntheticInput builds an agent at the given level and Mindscape with a
// 啄木鸟电音 set and 激素朋克, the chosen W-Engine, and the highest skill
// levels of that level.
const syntheticInput = `(() => {
  const core = level === 60 ? 7 : 1
  const promote = level === 60 ? 6 : 2
  const weaponPromote = level === 60 ? 5 : 1
  const chosen = weapon ? { id: weapon.id, level, promote: weaponPromote, refinement: rank === 6 ? 5 : 1 } : { id: '', level: 0, promote: 0, refinement: 1 }
  const info = EnkaFormat.parseInfo({ Id: Number(record.id), Level: level, TalentLevel: rank })
  const stat = (id, value, percent = false) => ({ id: String(id), key: String(id), value, percent })
  const elem = { 200: 31503, 201: 31603, 202: 31703, 203: 31803, 205: 31903 }[info.element_type]
  const mains = [stat(11103, 2200), stat(12103, 316), stat(13103, 184), stat(20103, 24, true), stat(elem, 30, true), stat(12102, 30, true)]
  const equipment = mains.map((main, i) => ({ slot: i + 1, set_name: i < 4 ? '啄木鸟电音' : '激素朋克', main, sub: [stat(23203, 9), stat(20103, 2.4, true), stat(21103, 4.8, true), stat(31203, 9)] }))
  const gear = zzzGear(equipment)
  const props = zzzModel(info, gear, zzzWeapon(chosen), promote, core - 1)
  const attributes = {}
  for (const [key, name] of Object.entries(zzzPropertyKeys)) {
    const value = props.find(p => p.property_name === name)
    if (value) {
      attributes[key] = zzzPropertyValue(value, 'final')
      attributes[key + 'Base'] = zzzPropertyValue(value, 'base')
    }
  }
  const talents = Object.fromEntries([0, 1, 2, 3, 5, 6].map(type => [type, type === 5 ? core : (level === 60 ? 12 : 4) + (rank === 6 ? 4 : 0)]))
  return { level, promote, rank, weapon: chosen, equipment, talents, trees: [], attributes, sub_element: info.sub_element_type, enemy_level: 70 }
})()`

// The upstream pipeline run directly, apart from the wrapper's anchoring and
// promotion inference, gives the numbers to compare with.
const upstreamResults = `(() => {
  const info = EnkaFormat.parseInfo({ Id: Number(record.id), Level: input.level, TalentLevel: input.rank })
  const gear = zzzGear(input.equipment)
  const weapon = zzzWeapon(input.weapon)
  const props = zzzModel(info, gear, weapon, input.promote, input.talents['5'] - 1)
  const avatar = zzzAvatar(info, gear, weapon, Object.entries(input.talents).map(([type, level]) => ({ skill_type: Number(type), level })), props)
  freshZZZRules(record)
  const calc = avatarPipeline.avatar_calc(avatar)
  calc.defEnemy('level', input.enemy_level)
  return calc.calc().map(r => ({ title: r.skill.name, expected: r.result.expectDMG, critical: r.skill.isAnomalyDMG && r.result.critDMG === 0 ? null : r.result.critDMG }))
})()`

// officialPanel is the agent as the official API would return it.
const officialPanel = `(() => {
  const info = EnkaFormat.parseInfo({ Id: Number(record.id), Level: input.level, TalentLevel: input.rank })
  const gear = zzzGear(input.equipment)
  const weapon = zzzWeapon(input.weapon)
  return {
    ...info,
    weapon: weapon ? { ...weapon, promote_level: input.weapon.promote } : null,
    equip: gear.map((g, i) => ({ ...g, id: i + 1, level: 15, name: g.equip_suit.name + '测试盘', rarity: 'S' })),
    properties: zzzModel(info, gear, weapon, input.promote, input.talents['5'] - 1),
    skills: Object.entries(input.talents).map(([type, level]) => ({ skill_type: Number(type), level, items: [] })),
    promote_level: input.promote,
  }
})()`

const vectors = []
const failures = []
const panels = {}
// Agents without an upstream calc.js are in the catalog for scoring only.
for (const record of metadata.characters.filter(record => record.script)) {
  for (const rank of [0, 6]) {
    for (const level of [20, 60]) {
      const c = await context(record)
      c.rank = rank
      c.level = level
      // The first W-Engine of the specialty whose effect ZZZ-Plugin models,
      // not one of the passive notes in calc-supplements/weapons.js.
      const supplemental = vm.runInContext('supplementalZZZWeapons', c)
      c.weapon = metadata.weapons.find(w => w.type === record.weapon_type && !supplemental.has(w.id))
      try {
        vm.runInContext(strictLogger, c)
        const input = vm.runInContext(syntheticInput, c, { timeout: 1000 })
        c.input = input
        const result = vm.runInContext('runBuild(record, [], input)', c, { timeout: 1500 })
        const expected = vm.runInContext(upstreamResults, c, { timeout: 1500 })
        if (result.baseline.results.length !== expected.length) throw Error('result count')
        expected.forEach((b, i) => {
          for (const key of ['expected', 'critical']) {
            const actual = result.baseline.results[i][key]
            if (actual == null && b[key] == null) continue
            if (!Number.isFinite(actual) || !Number.isFinite(b[key]) || Math.abs(actual - b[key]) > 1e-9 * Math.max(1, Math.abs(b[key]))) {
              throw Error(`result difference ${i} ${key}`)
            }
          }
        })
        vectors.push({ key: record.key, input, results: expected })
        if (rank === 0 && level === 60) panels[record.id] = vm.runInContext(officialPanel, c, { timeout: 1000 })
      } catch (error) {
        failures.push({ key: record.key, rank, level, reason: error.message, stack: error.stack?.split('\n').slice(0, 5) })
      }
    }
  }
}
await fs.writeFile(path.join(root, 'internal/assets/testdata/calc-vectors.json'), JSON.stringify(vectors) + '\n')
await fs.writeFile(path.join(root, 'internal/app/testdata/build-panels.json'), JSON.stringify(panels) + '\n')
console.log(JSON.stringify({ characters: new Set(vectors.map(v => v.key)).size, vectors: vectors.length, failures: failures.length, sample: failures.slice(0, 6) }))
