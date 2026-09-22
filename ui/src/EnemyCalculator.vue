<script setup lang="ts">
import {computed,ref,watch} from 'vue'
import BusinessView,{type View} from './BusinessView.vue'
interface Enemy{name:string;aliases:string[]}
interface Modifier{id:string;group:string;names:string[];level:number}
const props=defineProps<{invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const enemies=ref<Enemy[]>([]),modifiers=ref<Modifier[]>([]),search=ref(''),name=ref(''),stat=ref('HP'),level=ref(90),choices=ref<Record<string,string>>({}),version=ref(''),busy=ref(false),error=ref(''),view=ref<View|null>(null)
const matches=computed(()=>enemies.value.filter(e=>!search.value||[e.name,...e.aliases].some(n=>n.includes(search.value)))),groups=computed(()=>[...new Set(modifiers.value.filter(m=>m.group.endsWith(stat.value)).map(m=>m.group))])
watch(stat,()=>{choices.value={};view.value=null})
async function run(work:()=>Promise<void>){busy.value=true;error.value='';try{await work()}catch(e){error.value=e instanceof Error?e.message:'原魔资料操作失败。'}finally{busy.value=false}}
void run(async()=>{const r=await props.invoke<{enemies:Enemy[];modifiers:Modifier[];version:string}>('enemies.schema');enemies.value=r.enemies;modifiers.value=r.modifiers;version.value=r.version;name.value=r.enemies[0]?.name??''})
</script>
<template><section><h2>原魔属性计算</h2><p class="hint">{{version}} · 固定资料，非当前版本在线校准。</p><form class="query-form" @submit.prevent="run(async()=>{view=(await invoke<{view:View}>('enemies.query',{name,stat,level,modifiers:Object.values(choices).filter(Boolean)})).view})"><label>筛选名称或别名<input v-model="search" :disabled="busy"></label><label>原魔<select v-model="name" :disabled="busy"><option v-for="e in matches" :key="e.name" :value="e.name">{{e.name}}</option></select></label><label>属性<select v-model="stat" :disabled="busy"><option value="HP">生命值</option><option value="ATK">攻击力</option></select></label><label>等级（0采用因子预设）<input v-model.number="level" type="number" min="0" max="200" :disabled="busy"></label><label v-for="group in groups" :key="group">{{group.startsWith('Special')?'联机人数':group.startsWith('Spiral')?'深境螺旋':'场景与秘境'}}<select v-model="choices[group]" :disabled="busy"><option value="">不附加</option><option v-for="m in modifiers.filter(m=>m.group===group)" :key="m.id" :value="m.id">{{m.names[0]}}{{m.level?' · '+m.level+'级':''}}</option></select></label><button :disabled="busy||!name">计算原魔属性</button></form><p v-if="error" role="alert" class="feedback danger">{{error}}</p><BusinessView v-if="view" :value="view"/></section></template>
