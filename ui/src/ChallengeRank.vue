<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
interface Scope {source_protocol:string;source_adapter:string;bot_id:string;group_id:string}
interface Metric {key:string;label:string;lower:boolean}
interface Kind {id:string;label:string;metrics:Metric[]}
interface Entry {actor_id:string;nickname:string;uid:string;metrics:Record<string,number>;updated_ms:number}
interface Partition {partition:string;items:Entry[];total:number;next_offset:number|null}
const props=defineProps<{scope:Scope;invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const kinds=ref<Kind[]>([]),kind=ref(''),season=ref(''),dimension=ref(''),offset=ref(0),groups=ref<Partition[]>([]),revision=ref(0),busy=ref(false),error=ref(''),confirm=ref(false),loaded=ref(false)
const selected=computed(()=>kinds.value.find(v=>v.id===kind.value)),hasNext=computed(()=>groups.value.some(v=>v.next_offset!==null))
let generation=0,disposed=false
watch([()=>props.scope,kind,season,dimension],()=>{generation++;groups.value=[];offset.value=0;confirm.value=false;loaded.value=false})
async function load(start=0){if(busy.value)return;const current=++generation;busy.value=true;error.value='';confirm.value=false;try{const r=await props.invoke<{groups:Partition[];revision:number}>('challenge.list',{scope:props.scope,kind:kind.value,season:season.value,dimension:dimension.value,offset:start});if(!disposed&&current===generation){groups.value=r.groups;revision.value=r.revision;offset.value=start;loaded.value=true}}catch(e){if(current===generation)error.value=e instanceof Error?e.message:'群榜读取失败。'}finally{busy.value=false}}
async function clear(){if(busy.value||!confirm.value)return;busy.value=true;error.value='';try{await props.invoke('challenge.clear',{scope:props.scope,kind:kind.value,season:season.value,revision:revision.value,confirmed:true});groups.value=[];loaded.value=false;confirm.value=false}catch(e){error.value=e instanceof Error?e.message:'清空未完成。'}finally{busy.value=false}}
void props.invoke<{kinds:Kind[]}>('challenge.schema').then(r=>{if(!disposed){kinds.value=r.kinds;kind.value=r.kinds[0]?.id??''}}).catch(e=>{error.value=e instanceof Error?e.message:'玩法列表读取失败。'})
onUnmounted(()=>{disposed=true;generation++})
</script>
<template>
 <section class="challenge-rank"><h3>群 {{scope.group_id}} 的挑战榜</h3><p class="hint">仅收录本人在此群主动提交的官方成绩，按区服及官方期次分开。重新读取会按当前规则排列已保存成绩。完整提交方式见本游戏帮助。</p>
 <form @submit.prevent="load()"><fieldset :disabled="busy"><legend class="sr-only">挑战榜筛选</legend><label>挑战玩法<select v-model="kind"><option v-for="v in kinds" :key="v.id" :value="v.id">{{v.label}}</option></select></label><label>官方期次<input v-model="season" maxlength="128" placeholder="留空显示已收录期次"></label><label>排序指标<select v-model="dimension"><option value="">综合顺序</option><option v-for="m in selected?.metrics" :key="m.key" :value="m.key">{{m.label}}（{{m.lower?'越低越前':'越高越前'}}）</option></select></label><button :disabled="!kind">读取或重排群榜</button></fieldset></form>
 <p v-if="error" role="alert" class="danger-text">{{error}}</p><p v-if="loaded&&!groups.length" class="hint">当前条件没有主动提交的成绩。</p>
 <section v-for="g in groups" :key="g.partition"><h4>{{g.partition}} · {{g.total}} 人</h4><div class="rank-scroll"><table><thead><tr><th>名次</th><th>用户 / UID</th><th v-for="m in selected?.metrics" :key="m.key">{{m.label}}</th><th>更新时间</th></tr></thead><tbody><tr v-for="(r,i) in g.items" :key="r.actor_id"><td>{{offset+i+1}}</td><td>{{r.nickname}}<br>{{r.uid}}</td><td v-for="m in selected?.metrics" :key="m.key">{{r.metrics[m.key]??'未提供'}}</td><td>{{new Date(r.updated_ms).toLocaleString()}}</td></tr></tbody></table></div></section>
 <div class="actions"><button :disabled="busy||offset===0" @click="load(Math.max(0,offset-20))">上一页成绩</button><button :disabled="busy||!hasNext" @click="load(offset+20)">下一页成绩</button><button :disabled="busy||!loaded||!groups.length" @click="confirm=true">清空所选玩法及期次</button></div>
 <section v-if="confirm" class="feedback"><p>将删除此群的 {{selected?.label}} {{season||'全部期次'}} 成绩，需用户重新提交才能恢复。</p><div class="actions"><button :disabled="busy" @click="confirm=false">保留成绩</button><button :disabled="busy" class="destructive" @click="clear">确认清空挑战成绩</button></div></section>
 </section>
</template>
<style scoped>.challenge-rank{display:grid;gap:16px}.challenge-rank fieldset{display:flex;flex-wrap:wrap;gap:12px;border:0;padding:0;min-width:0;align-items:end}.challenge-rank label{display:grid;gap:6px;flex:1 1 160px;min-width:0}.rank-scroll{overflow-x:auto;max-width:100%}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:10px;border-bottom:1px solid var(--raylea-color-border);white-space:nowrap}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}</style>
