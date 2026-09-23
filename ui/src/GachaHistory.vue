<script setup lang="ts">
import { ref, watch } from 'vue'
import { intervalText, rarityName, poolNames, type History } from './history'
const props = defineProps<{ uid: string; region: string; disabled: boolean; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const pool = ref(''), rank = ref(''), query = ref(''), from = ref(''), to = ref('')
const data = ref<History | null>(null), busy = ref(false), error = ref(''), offset = ref(0)
const versions=ref<{items:{version:string;half:string;total:number;top:number;unknown:number;estimated:number}[];total:number;unclassified:number;version:string;timezone:string}|null>(null)
async function loadVersions(){if(busy.value||props.disabled)return;busy.value=true;error.value='';try{versions.value=await props.invoke('gacha.versions',{uid:props.uid,region:props.region})}catch(e){error.value=e instanceof Error?e.message:'版本统计读取失败。'}finally{busy.value=false}}
let generation = 0
watch([pool, rank, query, from, to], () => { data.value = null; error.value = ''; generation++ })
async function load(next = 0) {
  if (busy.value || props.disabled) return
  busy.value = true; error.value = ''
  const current = ++generation, revision = next ? data.value?.revision : undefined
  try {
    const result = await props.invoke<{ history: History }>('gacha.history', { uid: props.uid, region: props.region, pool: pool.value, rank: rank.value, query: query.value, from: from.value, to: to.value, offset: next, limit: 50, ...(revision ? { revision } : {}) })
    if (current === generation) { data.value = result.history; offset.value = next }
  } catch (cause) { if (current === generation) { data.value = null; error.value = cause instanceof Error ? cause.message : '记录读取失败，请重新查询。' } }
  finally { busy.value = false }
}
void load()
</script>
<template>
  <section class="gacha-history" aria-labelledby="history-heading">
    <h2 id="history-heading">抽卡明细 · {{ uid }}</h2>
    <p class="hint">仅统计本地保存的记录。间隔按完整档案计算，筛选不会重新起算；历史仍可能存在缺口。</p>
    <details><summary>按版本与半期汇总整个档案</summary><p class="hint">独立于下方筛选。按资料中的国服卡池区间换算档案时区；估算起点、版本冲突和未覆盖记录明确列出。</p><button :disabled="busy||disabled" type="button" @click="loadVersions">读取版本统计</button><template v-if="versions"><p class="hint">共 {{versions.total}} 抽，未分类 {{versions.unclassified}} 抽 · {{versions.timezone}} → UTC+8 · {{versions.version}}</p><div class="history-table"><table><thead><tr><th>版本</th><th>半期</th><th>抽数</th><th>已知最高稀有度</th><th>稀有度未知</th><th>涉及估算起点</th></tr></thead><tbody><tr v-for="v in versions.items" :key="v.version+v.half"><td>{{v.version}}</td><td>{{v.half}}</td><td>{{v.total}}</td><td>{{v.top}}</td><td>{{v.unknown}}</td><td>{{v.estimated}}</td></tr></tbody></table></div></template></details>
    <form @submit.prevent="load(0)"><fieldset :disabled="busy || disabled"><legend class="sr-only">筛选记录</legend>
      <label>明细卡池<select v-model="pool"><option value="">全部卡池</option><option v-for="(name, id) in poolNames" :key="id" :value="id">{{ name }}</option></select></label>
      <label>稀有度<select v-model="rank"><option value="">全部稀有度</option><option v-for="value in ['4','3','2']" :key="value" :value="value">{{ rarityName(value) }}</option><option value="unknown">未知稀有度</option></select></label>
      <label>物品名或 ID<input v-model="query" type="search" maxlength="128"></label><label>开始日期<input v-model="from" type="date"></label><label>结束日期<input v-model="to" type="date"></label>
      <button class="primary" type="submit">查询明细</button>
    </fieldset></form>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p>
    <p v-if="busy" role="status">正在读取记录…</p>
    <template v-if="data">
      <dl class="history-metrics"><div><dt>符合筛选</dt><dd>{{ data.total }} 抽</dd></div><div><dt>最高稀有度占比</dt><dd>{{ data.analytics.top_rate === null ? '无法确定' : data.analytics.top_rate.toFixed(2) + '%' }}</dd></div><div><dt>完整间隔均值</dt><dd>{{ data.analytics.average_interval === null ? '无完整间隔' : data.analytics.average_interval.toFixed(2) + ' 抽' }}</dd></div><div><dt>完整间隔样本</dt><dd>{{ data.analytics.complete_intervals }}</dd></div></dl>
      <p class="hint">占比是筛选结果的记录占比，不代表游戏抽卡概率；首段或稀有度不完整的间隔不计入均值。</p>
      <details><summary>按月份统计</summary><p class="hint">沿用档案区服本地时间。</p><div class="history-table"><table><thead><tr><th>月份</th><th>抽数</th><th>最高稀有度</th><th>未知</th></tr></thead><tbody><tr v-for="item in data.analytics.months" :key="item.month"><td>{{ item.month }}</td><td>{{ item.total }}</td><td>{{ item.top }}</td><td>{{ item.unknown }}</td></tr></tbody></table></div></details>
      <details><summary>物品分布 · {{ data.analytics.item_kinds }} 种</summary><p v-if="data.analytics.item_kinds > 100" class="hint">显示数量最多的 100 种，可按物品名查询其余记录。</p><ul class="history-items"><li v-for="item in data.analytics.items" :key="`${item.id}:${item.rank}`"><span>{{ item.name || item.id }} · {{ rarityName(item.rank) }}</span><strong>{{ item.count }}</strong></li></ul></details>
      <div v-if="data.records.length" class="history-table"><table><thead><tr><th>时间</th><th>物品</th><th>稀有度</th><th>卡池</th><th>最高稀有度间隔</th></tr></thead><tbody><tr v-for="item in data.records" :key="`${item.gacha_type}:${item.id}`"><td>{{ item.time }}</td><td>{{ item.name || item.item_id }}</td><td>{{ rarityName(item.rank_type ?? '') }}</td><td>{{ poolNames[item.gacha_type] || item.gacha_type }}</td><td>{{ intervalText(item.interval) }}</td></tr></tbody></table></div>
      <p v-else class="empty">没有符合筛选的记录。</p>
      <div class="actions"><button :disabled="busy || disabled || offset === 0" @click="load(Math.max(0, offset - 50))">上一页明细</button><span>第 {{ Math.floor(offset / 50) + 1 }} 页</span><button :disabled="busy || disabled || data.next_offset === null" @click="load(data.next_offset!)">下一页明细</button><button :disabled="busy || disabled" @click="load(0)">重新读取第一页</button></div>
    </template>
  </section>
</template>
<style scoped>
.gacha-history{display:grid;gap:18px}.gacha-history fieldset{display:flex;flex-wrap:wrap;align-items:end;gap:12px;padding:0;border:0;min-width:0}.gacha-history label{display:grid;gap:6px;flex:1 1 150px;min-width:0}.history-metrics{display:flex;flex-wrap:wrap;gap:24px}.history-metrics dd{margin:6px 0 0;font-weight:600}.history-metrics dt{font-size:13px}.history-table{overflow:auto;max-width:100%}.history-table table{border-collapse:collapse;width:100%;font-size:13px}.history-table td,.history-table th{padding:10px;text-align:left;border-bottom:1px solid var(--raylea-color-border,#ccc);white-space:nowrap}.history-items{display:grid;gap:10px;list-style:none;padding:0}.history-items li{display:flex;justify-content:space-between;gap:16px}.gacha-history details>div,.gacha-history details>p{margin-top:14px}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}
</style>
