<script setup lang="ts">
import { computed, ref, watch } from 'vue'
type Kind = 'character' | 'weapon' | 'standard'
interface Selection { kind: Kind; banner_id: string; featured: string; fate_target: string; fate_limit: number }
interface Banner { id: string; version: string; half: string; from: string; characters5: string[]; weapons5: string[] }
interface Draw { name: string; rarity: number; kind: Kind; featured: boolean; guaranteed: boolean; fate: boolean; interval: number; index: number; time_ms: number }
interface Status { model: string; active: Kind; selections: Record<Kind,Selection>; pools: Record<Kind,{five:number;four:number;up_five:boolean;up_four:boolean;fate:number}>; deck:{banners:Banner[]}; total:number;daily_used:number;daily_limit:number;daily_remaining:number;history:Draw[];owned:Record<string,number>;batch:{draws:Draw[];selection:Selection}|null }
const props=defineProps<{game:string;invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const status=ref<Status|null>(null),draft=ref<Selection>({kind:'character',banner_id:'',featured:'',fate_target:'',fate_limit:2}),quota=ref(100),busy=ref(false),error=ref(''),notice=ref(''),last=ref<Draw[]>([]),confirmReset=ref(false),pending=ref<{count:number;request_ref:string}|null>(null)
const names:Record<Kind,string>={character:'角色',weapon:props.game==='genshin'?'武器':'光锥',standard:'常驻'}
const banner=computed(()=>status.value?.deck.banners.find(item=>item.id===draft.value.banner_id))
const dirty=computed(()=>{const current=status.value?.selections[draft.value.kind];return !current||status.value!.active!==draft.value.kind||current.banner_id!==draft.value.banner_id||current.featured!==draft.value.featured||(props.game==='genshin'&&draft.value.kind==='weapon'&&(current.fate_target!==draft.value.fate_target||current.fate_limit!==draft.value.fate_limit))})
const inventory=computed(()=>Object.entries(status.value?.owned??{}).sort((a,b)=>b[1]-a[1]))
watch(()=>draft.value.kind,kind=>{if(status.value)draft.value={...status.value.selections[kind]};last.value=[]})
watch([()=>draft.value.kind,()=>draft.value.banner_id,()=>draft.value.featured,()=>draft.value.fate_target,()=>draft.value.fate_limit],()=>{last.value=[]})
function chooseBanner(){if(!banner.value)return;draft.value.featured=draft.value.kind==='character'?banner.value.characters5[0]!:banner.value.weapons5[0]!;draft.value.fate_target=props.game==='genshin'&&draft.value.kind==='weapon'?banner.value.weapons5[0]!:'';last.value=[]}
function accept(value:Status){status.value=value;draft.value={...value.selections[value.active]};quota.value=value.daily_limit;if(value.batch)last.value=value.batch.draws}
async function run(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await fn()}catch(cause){error.value=cause instanceof Error?cause.message:'模拟操作未完成。'}finally{busy.value=false}}
async function load(){accept(await props.invoke<Status>('simulation.status'))}
async function select(){await run(async()=>{accept(await props.invoke<Status>('simulation.select',{...draft.value}));notice.value='模拟卡池已保存。'})}
async function draw(count:number){await run(async()=>{if(!pending.value)last.value=[];pending.value??={count,request_ref:globalThis.crypto?.randomUUID?.()??`${Date.now()}-${Math.random()}`};accept(await props.invoke<Status>('simulation.draw',{...pending.value}));pending.value=null})}
async function endRetry(){await run(async()=>{await load();pending.value=null;notice.value='已读取实际保存的模拟统计。本次尝试已结束，已完成的模拟抽数保留。'})}
async function reset(){if(!confirmReset.value)return;await run(async()=>{accept(await props.invoke<Status>('simulation.reset',{confirm:true}));last.value=[];confirmReset.value=false;notice.value='模拟记录与保底已重置，当天已用次数保留。'})}
async function configure(){await run(async()=>{accept(await props.invoke<Status>('simulation.configure',{daily_limit:Number(quota.value)}));notice.value='每日模拟额度已保存。'})}
void run(load)
</script>
<template>
 <section class="gacha-simulation">
  <h2>模拟抽卡</h2><p class="hint">固定参考娱乐模型，不代表官方概率、保底规则或当前卡池。不消耗真实货币，模拟记录不会进入 UIGF。管理页使用独立档案，与聊天模拟分开。</p>
  <p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="notice" role="status" class="feedback">{{notice}}</p>
  <template v-if="status">
   <p class="hint">模型：{{status.model}} · 今日剩余 {{status.daily_remaining}} / {{status.daily_limit}} 抽 · 国服时间 04:00 重置</p>
   <form @submit.prevent="select"><fieldset :disabled="busy||!!pending"><legend>选择参考卡池</legend><label>模拟种类<select v-model="draft.kind"><option v-for="(name,kind) in names" :key="kind" :value="kind">{{name}}</option></select></label>
    <label v-if="draft.kind!=='standard'">参考期次<select v-model="draft.banner_id" @change="chooseBanner"><option v-for="item in status.deck.banners" :key="item.id" :value="item.id">{{item.version}} · {{item.half}} · {{item.from.slice(0,10)}}</option></select></label>
    <label v-if="draft.kind==='character'||draft.kind==='weapon'&&game==='starrail'">模拟目标<select v-model="draft.featured"><option v-for="name in (draft.kind==='character'?banner?.characters5:banner?.weapons5)??[]" :key="name" :value="name">{{name}}</option></select></label>
    <template v-if="draft.kind==='weapon'&&game==='genshin'"><label>模拟武器定轨<select v-model="draft.fate_target"><option value="">取消定轨</option><option v-for="name in banner?.weapons5??[]" :key="name" :value="name">{{name}}</option></select></label><label>模拟命定值上限<select v-model.number="draft.fate_limit"><option :value="1">1 点模型</option><option :value="2">2 点参考模型</option></select></label><p class="hint">更换模拟武器卡池、定轨目标或模型会清空命定值；角色和武器的保底分别保存。</p></template>
    <button type="submit" :disabled="!dirty">保存模拟卡池</button>
   </fieldset></form>
   <p v-if="dirty" class="hint">先保存卡池选择，再进行模拟。</p>
   <div class="actions"><button class="primary" :disabled="busy||dirty||!!pending||status.daily_remaining<1" @click="draw(1)">模拟单抽</button><button class="primary" :disabled="busy||dirty||!!pending||status.daily_remaining<10" @click="draw(10)">模拟十连</button><button :disabled="busy||!!pending" @click="run(load)">刷新模拟统计</button></div>
   <div v-if="pending" class="feedback"><p>上次请求结果尚未确认，可用同一请求重试以避免重复计数。</p><div class="actions"><button :disabled="busy" @click="draw(pending!.count)">重试同一模拟请求</button><button :disabled="busy" @click="endRetry">刷新统计并结束本次尝试</button></div></div>
   <ul v-if="last.length" class="draw-results"><li v-for="(item,index) in last" :key="index"><strong>{{item.name}}</strong><span>{{item.rarity}} 星{{item.featured?' · 目标池':''}}{{item.fate?' · 定轨':''}}{{item.guaranteed?' · 保证目标池':''}}</span></li></ul>
   <dl class="simulation-counters"><div v-for="(name,kind) in names" :key="kind"><dt>{{name}}</dt><dd>连续 {{status.pools[kind].five}} 抽未出五星<br>连续 {{status.pools[kind].four}} 抽未出四星<br>{{status.pools[kind].up_five?'下次五星保证来自目标池':'下次五星按参考模型判定'}}<template v-if="kind==='weapon'&&game==='genshin'"><br>命定值 {{status.pools.weapon.fate}}</template></dd></div></dl>
   <details><summary>模拟获得物品 · {{inventory.length}} 种</summary><ul class="inventory"><li v-for="[name,count] in inventory" :key="name"><span>{{name}}</span><strong>{{count}}</strong></li></ul></details>
   <details><summary>最近模拟历史 · {{status.history.length}} 抽</summary><ul class="inventory"><li v-for="(item,index) in [...status.history].reverse()" :key="index"><span>{{item.name}} · {{item.rarity}} 星 · {{names[item.kind]}}</span><span>{{new Date(item.time_ms).toLocaleString()}}</span></li></ul></details>
   <details><summary>额度与重置</summary><form @submit.prevent="configure"><label>每日模拟额度<input v-model.number="quota" type="number" min="10" max="1000" step="1" :disabled="busy||!!pending" required></label><button type="submit" :disabled="busy||!!pending">保存模拟额度</button></form><p class="hint">额度适用于本游戏每个模拟档案。重置历史不会恢复当天已用次数。</p><label class="check"><input v-model="confirmReset" type="checkbox" :disabled="busy||!!pending">清空当前模拟档案的历史、物品和保底</label><button type="button" class="destructive" :disabled="busy||!!pending||!confirmReset" @click="reset">确认重置模拟</button></details>
  </template>
 </section>
</template>
<style scoped>
.gacha-simulation{display:grid;gap:20px}.gacha-simulation fieldset{display:grid;gap:16px;padding:0;border:0;min-width:0}.gacha-simulation legend{margin-bottom:14px}.gacha-simulation label:not(.check){display:grid;gap:8px;min-width:0}.gacha-simulation select{width:100%;min-width:0}.draw-results{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:12px;list-style:none;padding:0}.draw-results li{display:grid;gap:8px;border:1px solid var(--raylea-color-border,#ddd);border-radius:10px;padding:14px}.draw-results span{font-size:12px}.simulation-counters{display:flex;flex-wrap:wrap;gap:24px}.simulation-counters dd{margin:8px 0 0;font-size:13px;line-height:1.8}.inventory{list-style:none;padding:0;display:grid;gap:12px;margin-top:16px}.inventory li{display:flex;justify-content:space-between;gap:16px;flex-wrap:wrap}.gacha-simulation details>form{display:grid;gap:12px;margin-top:16px}.gacha-simulation details>p,.gacha-simulation details>button{margin-top:16px}
</style>
