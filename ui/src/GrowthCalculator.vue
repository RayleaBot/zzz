<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
interface Skill { id: number; name: string; current_level: number; target_level: number; max_level: number }
interface Plan { character_id: string; name: string; current_level: number; target_level: number; max_level: number; weapon_id: number; weapon_name: string; current_weapon_level: number; target_weapon_level: number; max_weapon_level: number; skills: Skill[] }
const props = defineProps<{ choices: { key: string; label: string; role: { ref: string }; account: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), character = ref(''), busy = ref(false), error = ref(''), plan = ref<Plan | null>(null), view = ref<View | null>(null)
watch([selected, character], () => { plan.value = null; view.value = null })
watch(plan, () => { view.value = null }, { deep: true })
function choice() { const item = props.choices.find(v => v.key === selected.value); if (!item) throw new Error('请选择已授权账号。'); return { account_ref: item.account.ref, role_ref: item.role.ref } }
async function run(fn: () => Promise<void>) { if (busy.value) return; busy.value = true; error.value = ''; try { await fn() } catch (cause) { error.value = cause instanceof Error ? cause.message : '养成查询未完成。' } finally { busy.value = false } }
async function prepare() { await run(async () => { plan.value = null; view.value = null; let id = character.value.trim(); if (!/^\d+$/.test(id)) { const result = await props.invoke<{ entries: { id: string }[] }>('catalog.search', { query: id, kind: 'character' }); if (result.entries.length !== 1) throw new Error('角色名未唯一匹配，请填写角色 ID。'); id = result.entries[0]!.id }; plan.value = (await props.invoke<{ plan: Plan }>('growth.prepare', { ...choice(), character_id: id })).plan }) }
async function compute() { if (!plan.value) return; await run(async () => { view.value = (await props.invoke<{ view: View }>('growth.compute', { ...choice(), character_id: plan.value!.character_id, plan: plan.value })).view }) }
function maximize() { if (!plan.value) return; plan.value.target_level = plan.value.max_level; if (plan.value.weapon_id) plan.value.target_weapon_level = plan.value.max_weapon_level; for (const skill of plan.value.skills) skill.target_level = skill.max_level }
</script>
<template>
  <section class="growth-calculator">
    <h2>养成材料计算</h2><p class="hint">读取角色当前等级和技能，再向官方计算器查询所选目标的材料需求。仅作计算，不升级角色或消耗材料。</p>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p>
    <form @submit.prevent="prepare"><fieldset :disabled="busy"><legend class="sr-only">选择养成角色</legend><label>养成账号<select v-model="selected" required><option value="" disabled>选择已授权角色账号</option><option v-for="item in choices" :key="item.key" :value="item.key">{{ item.label }}</option></select></label><label>养成角色名称或 ID<input v-model="character" required></label><button class="primary" type="submit" :disabled="!selected || !character.trim()">读取当前养成等级</button></fieldset></form>
    <form v-if="plan" @submit.prevent="compute"><fieldset :disabled="busy"><legend>{{ plan.name }} · 目标等级</legend>
      <label>角色等级（当前 {{ plan.current_level }}）<input v-model.number="plan.target_level" type="number" :min="plan.current_level" :max="plan.max_level" step="1" required></label>
      <label v-if="plan.weapon_id">{{ plan.weapon_name }}等级（当前 {{ plan.current_weapon_level }}）<input v-model.number="plan.target_weapon_level" type="number" :min="plan.current_weapon_level" :max="plan.max_weapon_level" step="1" required></label>
      <div class="growth-skills"><label v-for="skill in plan.skills" :key="skill.id">{{ skill.name }}（当前 {{ skill.current_level }}）<input v-model.number="skill.target_level" type="number" :min="skill.current_level" :max="skill.max_level" step="1" required></label></div>
      <div class="actions"><button type="button" @click="maximize">目标全部设为上限</button><button class="primary" type="submit">计算所需材料</button></div>
    </fieldset></form>
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
<style scoped>
.growth-calculator{display:grid;gap:22px}.growth-calculator fieldset{display:grid;gap:16px;border:0;padding:0;min-width:0}.growth-calculator legend{margin-bottom:16px;font-weight:600}.growth-calculator label{display:grid;gap:8px;min-width:0}.growth-calculator select{min-width:0;width:100%}.growth-skills{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:16px}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}
</style>
