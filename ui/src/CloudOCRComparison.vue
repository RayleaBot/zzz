<script setup lang="ts">
import {computed,ref,watch} from 'vue'
import BusinessView,{type View} from './BusinessView.vue'
interface Account {ref:string;owner:{actor_id:string};roles:{ref:string;uid:string;nickname:string}[]}
const props=defineProps<{jobRef:string;slot:number;needsIdentity:boolean;invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const accounts=ref<Account[]>([]),selected=ref(''),character=ref(''),inheritIdentity=ref(false),view=ref<View|null>(null),busy=ref(false),error=ref(''),nextPage=ref<number|null>(null)
const choices=computed(()=>accounts.value.flatMap(a=>a.roles.map(r=>({key:JSON.stringify([a.ref,r.ref]),label:`${r.nickname} · ${r.uid} · 用户 ${a.owner.actor_id}`,account:a.ref,role:r.ref}))))
watch([selected,character,inheritIdentity],()=>{view.value=null})
async function run(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';try{await fn()}catch(e){error.value=e instanceof Error?e.message:'OCR 对比未完成。'}finally{busy.value=false}}
async function load(page=0){await run(async()=>{const r=await props.invoke<{accounts:{items:Account[];next_page:number|null}}>('accounts.list',{page});accounts.value=page===0?r.accounts.items:[...accounts.value,...r.accounts.items];nextPage.value=r.accounts.next_page})}
async function compare(){const c=choices.value.find(v=>v.key===selected.value);if(!c)return;await run(async()=>{view.value=null;view.value=(await props.invoke<{view:View}>('cloud.ocr.compare',{ref:props.jobRef,account_ref:c.account,role_ref:c.role,character_id:character.value.trim(),inherit_identity:inheritIdentity.value})).view})}
</script>
<template>
 <section class="ocr-comparison"><h3>在本人角色上试算识别装备</h3><p class="hint">将新装备放入部位 {{slot}}，与角色当前官方面板比较。截图旧装备不自动认定为此角色当前装备，试算不修改游戏或已保存面板。</p><p v-if="error" role="alert" class="feedback danger">{{error}}</p>
 <div class="actions"><button :disabled="busy" @click="load()">读取可用账号以作对比</button><button v-if="nextPage!==null" :disabled="busy" @click="load(nextPage!)">更多对比账号</button></div>
 <form v-if="choices.length" @submit.prevent="compare"><fieldset :disabled="busy"><legend class="sr-only">OCR 试算角色</legend><label>OCR 对比账号<select v-model="selected" required><option value="" disabled>选择角色账号</option><option v-for="c in choices" :key="c.key" :value="c.key">{{c.label}}</option></select></label><label>OCR 对比角色名称或 ID<input v-model="character" required></label><label v-if="needsIdentity" class="check"><input v-model="inheritIdentity" type="checkbox">截图未提供套装或品质时，明确沿用所选角色此部位当前装备的信息</label><button :disabled="!selected||!character||needsIdentity&&!inheritIdentity" class="primary">比较当前面板与 OCR 装备</button></fieldset></form>
 <BusinessView v-if="view" :value="view" />
 </section>
</template>
<style scoped>.ocr-comparison{display:grid;gap:16px;border-top:1px solid var(--raylea-color-border);padding-top:20px}.ocr-comparison fieldset{display:grid;gap:14px;min-width:0;border:0;padding:0}.ocr-comparison label:not(.check){display:grid;gap:8px;min-width:0}.ocr-comparison select{min-width:0;width:100%}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}</style>
