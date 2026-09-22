// Bundles the pinned ZZZ-Plugin calculation modules. Only numerical modules
// are transpiled; nothing here runs ZZZ-Plugin or Yunzai startup or loaders.
//
// Usage: node scripts/bundle-reference-calculation.mjs <参考项目/2026-09-15>
//
// Writes internal/assets/calc: data.js (upstream id maps and the
// agent aliases), common.js (Calculator, BuffManager and the drive disc Score),
// buffs.js (weapon and drive disc effects), catalog.json,
// characters/<id>-<name>.js from calc.js and scores/<id>-<name>.js from
// score.js. Agents without calc.js enter the catalog for scoring only.
// bootstrap.js and runner.js in the same directory are maintained by hand.
import fs from 'node:fs/promises'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { calcScriptName, transpile } from './calc-bundle.mjs'

const references = process.argv[2] && path.resolve(process.argv[2])
if (!references) throw new Error('Usage: bundle-reference-calculation.mjs <references>')
const plugin = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const typescript = createRequire(path.join(plugin, 'ui/package.json'))('typescript')
const root = path.join(references, 'ZZZ-Plugin-dev')
const calc = path.join(plugin, 'internal/assets/calc')
const read = file => fs.readFile(path.join(root, file), 'utf8')
const allowedImports = ['#interface', './BuffManager.ts', './Calculator.ts', '../avatar.js', './BuffManager.js', '../../utils/file.js', '../../lib/convert.js', '../../lib/settings.js', './avatar.js', 'lodash', '../equip.ts', '../../lib/score.js', '../damage/BuffManager.js', '../../lib/convert/property.js', '../damage/avatar.js', './convert/property.js', '../utils/file.js', './convert.js']

function compile(source, file) {
  // Burnice's six-shadow inner burn calls calc_skill directly, bypassing the
  // anomaly detection in new; see DATA_SOURCES.md.
  if (file === 'src/model/damage/character/柏妮思/calc.js') {
    const before = "type: '灼烧',"
    if (source.split(before).length !== 2) throw new Error('Burnice compatibility patch no longer applies')
    source = source.replace(before, "type: '灼烧',\n        isAnomalyDMG: true,")
  }
  return transpile(typescript, file, source, allowedImports)
}
const module = async file => compile(await read(file), file)

for (const directory of ['characters', 'scores']) {
  await fs.rm(path.join(calc, directory), { recursive: true, force: true })
  await fs.mkdir(path.join(calc, directory), { recursive: true })
}
const maps = {}
for (const name of ['PartnerId2Data', 'WeaponId2Data', 'SuitData', 'AnomalyData', 'Property2Name', 'ElementData', 'EquipScore', 'EquipMainStats', 'EquipBaseValue']) {
  maps[name] = JSON.parse(await read('resources/map/' + name + '.json'))
}

// Upstream calculators are keyed by Chinese directory names; resolve them to
// agent ids through the plugin catalog, whose aliases import-aliases.py takes
// from the upstream alias list.
const pluginCatalog = JSON.parse(await fs.readFile(path.join(plugin, 'internal/assets/catalog.json'), 'utf8'))
const names = {}
for (const entry of pluginCatalog.entries.filter(e => e.kind === 'character')) {
  for (const name of [entry.name, ...(entry.aliases || [])]) names[name] = entry.id
}
// Score.ts resolves EquipScore.json names through char.aliasToId.
await fs.writeFile(path.join(calc, 'data.js'), 'const referenceMaps=' + JSON.stringify(maps) + ';\nconst characterAliases=' + JSON.stringify(names) + ';\n')

let common = `const property=${await module('src/lib/convert/property.ts')};\n`
common += `const element=${await module('src/lib/convert/element.ts')};\n`
common += `const buffRuntime=${await module('src/model/damage/BuffManager.ts')};\nconst {BuffManager,runtime,elementType2element,anomalyEnum,elementEnum}=buffRuntime;\n`
common += `const {Calculator}=${await module('src/model/damage/Calculator.ts')};\nconst EnkaFormat=${await module('src/model/Enka/formater.ts')};\n`
const avatarModel = typescript.createSourceFile('avatar-model.ts', await read('src/model/avatar.ts'), typescript.ScriptTarget.Latest, true)
const avatarClass = avatarModel.statements.find(n => typescript.isClassDeclaration(n) && n.name?.text === 'ZZZAvatarInfo')
const members = avatarClass.members.filter(n => ['getProperty', 'basic_properties', 'base_properties', 'initial_properties', 'equip_score', 'equip_comment'].includes(n.name?.getText(avatarModel))).map(n => n.getText(avatarModel)).join('\n')
common += `const {AvatarProperties}=${compile('export class AvatarProperties {\n' + members + '\n}', 'selected-properties.ts')};\n`
const avatarSource = await read('src/model/damage/avatar.ts')
const avatarAst = typescript.createSourceFile('avatar.ts', avatarSource, typescript.ScriptTarget.Latest, true)
const picked = avatarAst.statements.filter(s => typescript.isFunctionDeclaration(s) && ['avatar_calc', 'weapon_buff', 'set_buff', 'debug'].includes(s.name?.text)).map(s => s.getText(avatarAst)).join('\n')
common += `const avatarPipeline=${compile('const weakMapCalc=new WeakMap();\n' + picked, 'selected-avatar.ts')};\n`
// Drive disc scoring: Score.ts with lib/score.ts, and the grade getters of the
// Equip and ZZZAvatarInfo models.
common += `const {rarityEnum,professionEnum}=runtime;\nconst {idToName,nameToId}=property;\nconst scoreFnc={};\n`
common += `const char={aliasToId:name=>characterAliases[name]??null,idToData:id=>referenceMaps.PartnerId2Data[id]};\n`
common += `const {baseValueData,formatScoreWeight,getEquipPropertyEnhanceCount}=${await module('src/lib/score.ts')};\n`
common += `const ZZZScore=${await module('src/model/score/Score.ts')}.default;\n`
const equipModel = typescript.createSourceFile('equip.ts', await read('src/model/equip.ts'), typescript.ScriptTarget.Latest, true)
const equipClass = equipModel.statements.find(n => typescript.isClassDeclaration(n) && n.name?.text === 'Equip')
const comment = equipClass.members.find(n => n.name?.getText(equipModel) === 'comment').getText(equipModel)
common += `const {EquipGrade}=${compile('export class EquipGrade {\nconstructor(score){this.score=score}\n' + comment + '\n}', 'selected-equip.ts')};\n`
await fs.writeFile(path.join(calc, 'common.js'), common)

// Unreleased agents carry the placeholder full name '...'.
const agentName = partner => (partner.full_name && partner.full_name !== '...' ? partner.full_name : partner.name)
const catalog = { version: 'zzz-fb66219cec-reference-v1', characters: [], weapons: [], sets: [], missing_weapons: [] }
const elements = { 200: 'physical', 201: 'fire', 202: 'ice', 203: 'lightning', 205: 'ether', 300: 'lumiflux' }
for (const name of (await fs.readdir(path.join(root, 'src/model/damage/character'))).sort()) {
  if (name === '模板') continue
  const directory = 'src/model/damage/character/' + name
  const optional = async file => {
    try {
      return await read(directory + '/' + file)
    } catch (error) {
      if (error.code === 'ENOENT') return undefined
      throw error
    }
  }
  const code = await optional('calc.js')
  const data = await optional('data.json')
  const score = await optional('score.js')
  if (!score && !(code && data)) continue
  const id = names[name]
  if (!id || !maps.PartnerId2Data[id]) throw new Error('Unmapped character ' + name)
  const partner = maps.PartnerId2Data[id]
  const key = 'zzz_' + id
  const displayName = agentName(partner)
  const record = { key, game: 'zzz', id, name: displayName, element: elements[partner.ElementType] || String(partner.ElementType), weapon_type: String(partner.WeaponType), script: '', data: { partner } }
  if (code && data) {
    record.script = calcScriptName(key, displayName)
    record.data.calculation = JSON.parse(data)
    await fs.writeFile(path.join(calc, record.script), `const characterRule=${compile(code, directory + '/calc.js')};\n`)
  }
  if (score) {
    record.score_script = calcScriptName(key, displayName).replace(/^characters\//, 'scores/')
    await fs.writeFile(path.join(calc, record.score_script), `const scoreRule=${compile(score, directory + '/score.js')};\n`)
  }
  catalog.characters.push(record)
}
// Every other agent is scored with its EquipScore.json entry or the default
// rules for its specialty.
for (const [id, partner] of Object.entries(maps.PartnerId2Data)) {
  // The protagonists 哲 and 铃 have no specialty and are not agents.
  if (catalog.characters.some(c => c.id === id) || !partner.WeaponType) continue
  const displayName = agentName(partner)
  catalog.characters.push({ key: 'zzz_' + id, game: 'zzz', id, name: displayName, element: elements[partner.ElementType] || String(partner.ElementType), weapon_type: String(partner.WeaponType), script: '', data: { partner } })
}

let buffs = 'const calcFnc={character:{},weapon:{},set:{}};\n'
const weaponNames = new Set()
for (const kind of ['weapon', 'set']) {
  for (const file of (await fs.readdir(path.join(root, 'src/model/damage', kind))).sort()) {
    if (!file.endsWith('.js') || file.includes('_user') || file === '模板.js') continue
    const name = file.slice(0, -3)
    buffs += `calcFnc.${kind}[${JSON.stringify(name)}]=${await module('src/model/damage/' + kind + '/' + file)};\n`
    if (kind === 'weapon') weaponNames.add(name)
    else catalog.sets.push(name)
  }
}
// Authored effects for W-Engines whose pinned descriptions are complete.
const supplements = await fs.readFile(path.join(plugin, 'scripts/calc-supplements/weapons.js'), 'utf8')
buffs += '\n' + supplements + '\n'
for (const match of supplements.matchAll(/(?:add\(|for\(const id of \[)([^\n]+)/g)) {
  for (const id of match[1].matchAll(/'(\d{5})'/g)) {
    if (maps.WeaponId2Data[id[1]]) weaponNames.add(maps.WeaponId2Data[id[1]].Name)
  }
}
await fs.writeFile(path.join(calc, 'buffs.js'), buffs)
for (const [id, data] of Object.entries(maps.WeaponId2Data)) {
  if (!weaponNames.has(data.Name)) {
    catalog.missing_weapons.push({ id, name: data.Name })
    continue
  }
  catalog.weapons.push({ game: 'zzz', id, name: data.Name, type: String(data.Profession), data })
}
await fs.writeFile(path.join(calc, 'catalog.json'), JSON.stringify(catalog) + '\n')
console.log(JSON.stringify({ characters: catalog.characters.length, weapons: catalog.weapons.length, sets: catalog.sets.length, missing_weapons: catalog.missing_weapons.length }))
