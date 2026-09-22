<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
interface Snapshot { ref: string; saved_at_ms: number; version: string; name: string; level: number; weapon: string }
const props = defineProps<{ choices: { key: string; label: string; role: { ref: string }; account: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), character = ref(''), items = ref<Snapshot[]>([]), before = ref(''), after = ref(''), view = ref<View | null>(null), removal = ref(''), busy = ref(false), error = ref(''), notice = ref('')
let resolvedID = ''
watch([selected, character], () => { items.value = []; before.value = ''; after.value = ''; resolvedID = ''; view.value = null; removal.value = '' })
watch([before, after], () => { view.value = null })
function payload() { const choice = props.choices.find(value => value.key === selected.value); if (!choice || !resolvedID) throw new Error('请选择账号与角色并读取历史。'); return { account_ref: choice.account.ref, role_ref: choice.role.ref, character_id: resolvedID } }
async function resolve() { let id = character.value.trim(); if (!/^\d+$/.test(id)) { const result = await props.invoke<{ entries: { id: string }[] }>('catalog.search', { query: id, kind: 'character' }); if (result.entries.length !== 1) throw new Error('角色名未唯一匹配，请使用角色 ID。'); id = result.entries[0]!.id }; resolvedID = id }
async function run(fn: () => Promise<void>) { if (busy.value) return; busy.value = true; error.value = ''; notice.value = ''; try { await fn() } catch (cause) { error.value = cause instanceof Error ? cause.message : '历史操作失败。' } finally { busy.value = false } }
async function load() { items.value = (await props.invoke<{ items: Snapshot[] }>('panel.history.list', payload())).items; if (!items.value.some(item => item.ref === before.value)) before.value = items.value[1]?.ref ?? ''; if (!items.value.some(item => item.ref === after.value)) after.value = items.value[0]?.ref ?? '' }
async function list() { await run(async () => { await resolve(); await load(); view.value = null }) }
async function save() { await run(async () => { await resolve(); await props.invoke('panel.history.save', payload()); await load(); notice.value = '已重新读取官方面板并保存快照。' }) }
async function get(ref: string) { await run(async () => { view.value = (await props.invoke<{ view: View }>('panel.history.get', { ...payload(), ref })).view }) }
async function compare() { await run(async () => { view.value = (await props.invoke<{ view: View }>('panel.history.compare', { ...payload(), before_ref: before.value, after_ref: after.value })).view }) }
async function remove(ref: string) { await run(async () => { await props.invoke('panel.history.remove', { ...payload(), ref }); removal.value = ''; view.value = null; await load(); notice.value = '历史快照已移除。' }) }
async function download(ref: string) { await run(async () => { const result = await props.invoke<{ export: unknown }>('panel.history.export', { ...payload(), ref }); const url = URL.createObjectURL(new Blob([JSON.stringify(result.export, null, 2)], { type: 'application/json' })); const a = document.createElement('a'); a.href = url; a.download = `panel-${resolvedID}-${ref}.json`; a.click(); setTimeout(() => URL.revokeObjectURL(url), 1000) }) }
function date(time: number) { return new Date(time).toLocaleString() }
</script>
<template>
  <section class="panel-history">
    <h2>本地面板历史</h2><p class="hint">主动保存官方面板，比较不同时间的属性、装备与技能。每个角色最多保留 20 份；历史读取仍需当前账号授权，不会写回游戏。</p>
    <form @submit.prevent="list"><fieldset :disabled="busy"><legend class="sr-only">选择面板</legend><label>历史所属账号<select v-model="selected" required><option value="" disabled>选择账号</option><option v-for="choice in choices" :key="choice.key" :value="choice.key">{{ choice.label }}</option></select></label><label>历史角色名称或 ID<input v-model="character" required placeholder="角色全名或 ID"></label><div class="actions"><button type="submit" :disabled="!selected || !character.trim()">读取历史</button><button class="primary" type="button" :disabled="!selected || !character.trim()" @click="save">读取官方面板并保存</button></div></fieldset></form>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p><p v-if="notice" role="status" class="feedback">{{ notice }}</p>
    <p v-if="!items.length" class="empty">当前没有已读取的历史快照。</p>
    <form v-if="items.length >= 2" @submit.prevent="compare"><fieldset :disabled="busy"><legend>选择两份快照</legend><label>对比前<select v-model="before"><option v-for="item in items" :key="item.ref" :value="item.ref">{{ date(item.saved_at_ms) }} · {{ item.weapon }}</option></select></label><label>对比后<select v-model="after"><option v-for="item in items" :key="item.ref" :value="item.ref">{{ date(item.saved_at_ms) }} · {{ item.weapon }}</option></select></label><button :disabled="!before || !after || before === after" type="submit">对比面板</button></fieldset></form>
    <ul class="snapshot-list"><li v-for="item in items" :key="item.ref"><strong>{{ date(item.saved_at_ms) }} · {{ item.name }}</strong><p class="hint">等级 {{ item.level }} · {{ item.weapon }} · {{ item.version }}</p><div class="actions"><button :disabled="busy" @click="get(item.ref)">查看快照</button><button :disabled="busy" @click="download(item.ref)">导出快照</button><button :disabled="busy" @click="removal = item.ref">移除快照</button></div><div v-if="removal === item.ref" class="feedback"><p>确认移除此份本地快照？</p><div class="actions"><button :disabled="busy" @click="removal = ''">保留</button><button class="destructive" :disabled="busy" @click="remove(item.ref)">确认移除快照</button></div></div></li></ul>
    <BusinessView v-if="view" :value="view" />
  </section>
</template>
<style scoped>
.panel-history{display:grid;gap:20px}.panel-history fieldset{display:grid;gap:16px;border:0;padding:0;margin:0;min-width:0}.panel-history legend{margin-bottom:14px}.panel-history label{display:grid;gap:8px;min-width:0}.panel-history select{min-width:0;width:100%}.snapshot-list{list-style:none;padding:0;display:grid;gap:18px}.snapshot-list li{display:grid;gap:12px;padding:16px 0;border-top:1px solid var(--raylea-color-border,#ddd);overflow-wrap:anywhere}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}
</style>
