<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { alignBuildRows, damageGain, weaponPromotions, type BuildPrepared, type BuildSkill, type CharacterBuild, type BuildConditions } from './build'
import BuildConditionsEditor from './BuildConditionsEditor.vue'
import type { BuildInvoke } from './build'
const props = defineProps<{ characterId: string; characterName: string; accountRef: string; roleRef: string; invoke: BuildInvoke }>()
const prepared = ref<BuildPrepared>(), result = ref<CharacterBuild>(), busy = ref(false), error = ref('')
const enemyLevel = ref(103), changeWeapon = ref(false), weaponID = ref(''), weaponLevel = ref(90), promote = ref(6), refinement = ref(1), critical = ref(false)
const equipmentSource = ref(''), changeEquipment = ref(false)
const conditions=ref<BuildConditions|null>(null)
let disposed = false, revision = 0
onUnmounted(() => { disposed = true })
const choice = () => ({ account_ref: props.accountRef, role_ref: props.roleRef, character_id: props.characterId })
const promotions = computed(() => weaponPromotions(weaponLevel.value))
watch(promotions, values => { if (!values.includes(promote.value)) promote.value = values.at(-1) ?? 0 }, { flush: 'sync' })
watch([enemyLevel, changeWeapon, weaponID, weaponLevel, promote, refinement, changeEquipment, equipmentSource], () => { result.value = undefined; revision++ }, { flush: 'sync' })
watch(conditions,()=>{result.value=undefined;revision++},{deep:true,flush:'sync'})
async function load() {
  if (busy.value) return
  busy.value = true; error.value = ''; result.value = undefined
  try {
    const value = await props.invoke<BuildPrepared>('build.prepare', choice())
    if (disposed) return
    conditions.value = null; prepared.value = value; enemyLevel.value = value.enemy_level; weaponID.value = value.weapon.id
    weaponLevel.value = value.weapon.level || 60
    promote.value = value.weapon.id ? value.weapon.promote ?? 5 : 5; refinement.value = value.weapon.refinement; changeWeapon.value = false; changeEquipment.value = false; equipmentSource.value = ''
    result.value = value.build
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '自动计算未完成，请刷新面板后重试。' }
  finally { if (!disposed) busy.value = false }
}
async function calculate() {
  if (busy.value) return
  busy.value = true; error.value = ''; result.value = undefined
  const started = revision
  try {
    if (!Number.isInteger(enemyLevel.value) || enemyLevel.value < 1 || enemyLevel.value > 200) throw new Error('敌人等级应为 1–200 的整数。')
    const candidate = weaponID.value ? { id: weaponID.value, level: weaponLevel.value, promote: promote.value, refinement: refinement.value } : { id: '', level: 0, promote: 0, refinement: 1 }
    if (changeWeapon.value && weaponID.value && (!Number.isInteger(weaponLevel.value) || !promotions.value.includes(promote.value) || !Number.isInteger(refinement.value) || refinement.value < 1 || refinement.value > 5)) throw new Error('请检查武器等级、突破阶段和精炼。')
    let sourceID = equipmentSource.value.trim()
    if (changeEquipment.value) {
      if (!sourceID) throw new Error('请填写装备来源角色。')
      if (!/^\d+$/.test(sourceID)) {
        const found = await props.invoke<{ entries: { id: string; name: string }[] }>('catalog.search', { query: sourceID, kind: 'character' })
        const exact = found.entries.filter(e => e.name === sourceID)
        if (exact.length === 1) sourceID = exact[0]!.id
        else if (found.entries.length === 1) sourceID = found.entries[0]!.id
        else throw new Error('装备来源角色未唯一匹配，请填写准确名称或 ID。')
      }
    }
    const output = await props.invoke<{ build: CharacterBuild }>('build.compare', { ...choice(), enemy_level: enemyLevel.value, ...(changeWeapon.value ? { candidate_weapon: candidate } : {}), ...(changeEquipment.value ? { equipment_from_character_id: sourceID } : {}),...(conditions.value?{conditions:conditions.value}:{}) })
    if (!disposed && revision === started) result.value = output.build
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '换装计算未完成，请稍后重试。' }
  finally { if (!disposed) busy.value = false }
}
const rows = computed(() => result.value ? alignBuildRows(result.value) : [])
const format = (v: number, digits = 1) => new Intl.NumberFormat('zh-CN', { maximumFractionDigits: digits }).format(v)
function display(row?: BuildSkill) { if (!row) return '—'; if (row.kind === 'text') return row.text || '—'; const value = critical.value ? row.critical : row.expected; return value == null ? '—' : format(value) }
function gain(row: ReturnType<typeof alignBuildRows>[number]) { const key = critical.value ? 'critical' : 'expected'; const value = damageGain(row.before?.[key], row.after?.[key]); return value == null ? '—' : `${value >= 0 ? '+' : ''}${format(value, 2)}%` }
</script>

<template>
  <section class="build-calculator separated" aria-labelledby="build-heading">
    <div class="section-heading"><div><h2 id="build-heading">{{ characterName }} · 角色自动计算</h2><p class="hint">按参考情境计算技能，可试换音擎或另一角色的整套装备。</p></div><button type="button" :disabled="busy" @click="load">{{ prepared ? '重新读取并计算' : '读取并自动计算' }}</button></div>
    <p v-if="error" role="alert" class="danger-text">{{ error }}</p><p v-if="busy" role="status" class="hint">正在读取面板并计算…</p>
    <form v-if="prepared" @submit.prevent="calculate"><fieldset :disabled="busy"><legend class="sr-only">参考伤害与换装条件</legend>
      <div class="query-form"><label>敌人等级<input v-model.number="enemyLevel" type="number" min="1" max="200" step="1" required></label><label class="check"><input v-model="changeWeapon" type="checkbox">试换音擎</label><label class="check"><input v-model="changeEquipment" type="checkbox">调入整套装备</label></div>
      <div v-if="changeWeapon" class="query-form"><label class="wide">候选装备<select v-model="weaponID"><option value="">未装备</option><option v-for="weapon in prepared.weapons" :key="weapon.id" :value="weapon.id">{{ weapon.name }}</option></select></label><template v-if="weaponID"><label>装备等级<input v-model.number="weaponLevel" type="number" min="1" max="60" step="1" required></label><label>突破阶段<select v-model.number="promote"><option v-for="value in promotions" :key="value" :value="value">阶段 {{ value }}</option></select></label><label>星级<select v-model.number="refinement"><option v-for="value in 5" :key="value" :value="value">{{ value }}</option></select></label></template></div>
      <div v-if="changeEquipment" class="query-form"><label class="wide">装备来源角色<input v-model="equipmentSource" placeholder="同一游戏账号下的角色名称或 ID" required></label><p class="hint wide">仅调入已查询到的整套装备，用于模拟；角色、技能和音擎仍按当前选择计算。</p></div>
      <BuildConditionsEditor v-model="conditions" :character-id="characterId" :disabled="busy" :invoke="invoke" />
      <div class="actions"><button type="submit">{{ conditions ? '计算自定义候选' : changeWeapon || changeEquipment ? '计算换装收益' : '按当前条件重算' }}</button><span class="hint">每次重新读取当前角色，试算不改变游戏装备。</span></div>
    </fieldset></form>
    <section v-if="result" class="build-results" aria-live="polite">
      <div class="section-heading"><div><h3>参考情境与计算结果</h3><p class="hint">当前：{{ result.baseline.weapon.name || '未装备' }}<template v-if="result.candidate"> → 候选：{{ result.candidate.weapon.name || '未装备' }}</template></p></div><label class="check"><input v-model="critical" type="checkbox">查看暴击值</label></div>
      <p v-if="result.equipment_from" class="hint">候选整套装备来自：{{ result.equipment_from.name }}。</p>
      <p class="hint">半血、满层、击杀、队伍等条件沿用各条参考规则。展开情境查看增益及适用技能；它们不代表角色实际处于该状态。按弱点抗性 −20%、基础防御系数 50 和未失衡状态计算。</p>
      <div class="table-scroll"><table><thead><tr><th scope="col">参考情境</th><th scope="col">{{ critical ? '当前暴击值' : '当前期望 / 效果' }}</th><template v-if="result.candidate"><th scope="col">换装后</th><th scope="col">变化</th></template></tr></thead><tbody><tr v-for="row in rows" :key="row.id"><th scope="row"><details><summary>{{ row.title }}</summary><div class="build-buffs"><p v-if="row.before">当前参考增益</p><ul v-if="row.before"><li v-for="(buff, index) in row.before.buffs" :key="index">{{ buff }}</li></ul><p v-if="row.after">候选参考增益</p><ul v-if="row.after"><li v-for="(buff, index) in row.after.buffs" :key="index">{{ buff }}</li></ul></div></details></th><td>{{ display(row.before) }}</td><template v-if="result.candidate"><td>{{ display(row.after) }}</td><td>{{ gain(row) }}</td></template></tr></tbody></table></div>
      <p class="hint">“—”表示该情境未适用、不支持暴击或无法计算变化比例。公式版本：{{ result.version }}。</p>
    </section>
  </section>
</template>

<style scoped>
.build-calculator fieldset{border:0;margin:0;padding:0;min-width:0}.build-calculator .section-heading{align-items:start;flex-wrap:wrap}.build-calculator .query-form{margin:16px 0}.build-results{margin-top:24px;display:grid;gap:16px}.build-results .section-heading{margin:0}.table-scroll{overflow-x:auto}.build-results table{width:100%;border-collapse:collapse}.build-results th,.build-results td{padding:12px 8px;border-bottom:1px solid var(--raylea-color-border);text-align:left;vertical-align:top}.build-results tbody th{font-weight:500;min-width:140px}.build-results td{white-space:nowrap;font-variant-numeric:tabular-nums}.build-buffs{font-size:12px;font-weight:400;color:var(--raylea-color-muted);max-width:56ch}.build-buffs p{margin-top:12px}.build-buffs ul{padding-left:18px}.build-buffs li+li{margin-top:6px}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}
@media(max-width:560px){.build-calculator .section-heading>button{width:100%}.build-results th,.build-results td{padding:10px 5px;font-size:12px}.build-results tbody th{min-width:115px}.build-results .table-scroll{max-width:100%}}
</style>
