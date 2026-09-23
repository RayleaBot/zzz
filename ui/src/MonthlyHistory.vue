<script setup lang="ts">
import {onUnmounted,ref} from 'vue'
import BusinessView,{type View} from './BusinessView.vue'
interface Item {month:string;year_estimated:boolean;saved_ms:number;amounts:Record<string,number>}
const props=defineProps<{accountRef:string;roleRef:string;invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const items=ref<Item[]>([]),totals=ref<Record<string,number>>({}),coverage=ref<Record<string,number>>({}),revision=ref(0),month=ref(''),busy=ref(false),error=ref(''),notice=ref(''),removeMonth=ref(''),view=ref<View|null>(null),stopping=ref(false)
let disposed=false
function selection(){return {account_ref:props.accountRef,role_ref:props.roleRef}}
async function load(){const r=await props.invoke<{items:Item[];totals:Record<string,number>;coverage:Record<string,number>;revision:number}>('monthly.list',selection());if(!disposed){items.value=r.items;totals.value=r.totals;coverage.value=r.coverage;revision.value=r.revision}}
async function run(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await fn()}catch(e){if(!disposed)error.value=e instanceof Error?e.message:'月报操作未完成。'}finally{busy.value=false}}
async function fetchMonths(all=false){stopping.value=false;await run(async()=>{let saved=0;try{const r=await props.invoke<{month:string;available_months:string[]}>('monthly.fetch',{...selection(),month:all?'':month.value});saved++;if(all){const current=r.month.replace('-','');for(const next of [...new Set(r.available_months)].slice(0,12)){if(stopping.value||disposed)break;if(next===current)continue;await props.invoke('monthly.fetch',{...selection(),month:next});saved++}}}finally{if(!disposed){notice.value=`本次已保存 ${saved} 个月${stopping.value?'，已停止后续读取':''}。`;await load()}}})}
async function open(item:Item){await run(async()=>{view.value=null;const r=await props.invoke<{view:View}>('monthly.get',{...selection(),month:item.month});if(!disposed){view.value=r.view}})}
async function remove(){await run(async()=>{await props.invoke('monthly.remove',{...selection(),month:removeMonth.value,revision:revision.value});removeMonth.value='';view.value=null;await load()})}
void run(load)
onUnmounted(()=>{disposed=true;stopping.value=true})
</script>
<template>
 <section class="monthly-history"><h3>保存月报与累计收入</h3><p class="hint">只统计已保存月份，未保存或官方已失效的月份不补算。</p>
 <p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="notice" role="status">{{notice}}</p>
 <form class="actions" @submit.prevent="fetchMonths()"><label>保存月份<input v-model="month" :disabled="busy" placeholder="YYYYMM；留空为默认月份"></label><button :disabled="busy">读取并保存此月</button><button type="button" :disabled="busy" @click="fetchMonths(true)">保存官方可用月份</button><button v-if="busy" type="button" @click="stopping=true">停止后续月份</button><button type="button" :disabled="busy" @click="run(load)">刷新已保存月报</button></form>
 <dl class="totals"><div v-for="(value,key) in totals" :key="key"><dt>{{key}} · 覆盖 {{coverage[key]}} 个月</dt><dd>{{value.toLocaleString()}}</dd></div></dl>
 <ul><li v-for="item in items" :key="item.month"><strong>{{item.month}}<span v-if="item.year_estimated">（年份推定）</span></strong><p>{{Object.entries(item.amounts).map(([k,v])=>`${k} ${v.toLocaleString()}`).join(' · ')||'未提供可累计的收入'}}</p><p class="hint">保存于 {{new Date(item.saved_ms).toLocaleString()}}</p><div class="actions"><button :disabled="busy" @click="open(item)">查看 {{item.month}} 月报</button><button :disabled="busy" @click="removeMonth=item.month">移除 {{item.month}} 月报</button></div></li></ul>
 <p v-if="!items.length&&!busy" class="hint">尚未保存月报。</p><section v-if="removeMonth" class="feedback"><p>移除 {{removeMonth}} 的本地月报？官方不再提供的月份将无法重新获取。</p><div class="actions"><button :disabled="busy" @click="removeMonth=''">保留月报</button><button :disabled="busy" class="destructive" @click="remove">确认移除此月</button></div></section>
 <BusinessView v-if="view" :value="view"/>
 </section>
</template>
<style scoped>.monthly-history{display:grid;gap:18px;border-top:1px solid var(--raylea-color-border);padding-top:20px}.monthly-history label{display:grid;gap:8px;flex:1 1 220px;min-width:0}.monthly-history form{align-items:end}.monthly-history ul{list-style:none;margin:0;padding:0;display:grid;gap:16px}.monthly-history li{padding:12px 0;border-top:1px solid var(--raylea-color-border)}.monthly-history p{margin:10px 0}.totals{display:flex;flex-wrap:wrap;gap:20px}.totals dd{margin:8px 0 0;font-size:1.3rem;font-weight:600}</style>
