<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
import { cloudTask } from './cloud'
const props=defineProps<{invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>;roles:{key:string;label:string;role:{ref:string;region:string};account:{ref:string}}[]}>()
const task=cloudTask(props.invoke,'content'), selection=ref(''), code=ref(''), confirmed=ref(false), busy=ref(false), error=ref(''), view=ref<View|null>(null)
const rows=computed(()=>(task.job.value?.result?.items??[]) as {code:string;reward:string;expires_at?:string;expiry_estimated:boolean}[])
const role=computed(()=>props.roles.find(v=>v.key===selection.value))
const overseas=computed(()=>!!role.value&&!['cn_gf01','cn_qd01','prod_gf_cn','prod_qd_cn'].includes(role.value.role.region))
async function redeem(){if(!role.value||!confirmed.value||busy.value)return;busy.value=true;error.value='';view.value=null;try{view.value=(await props.invoke<{view:View}>('redeem.run',{account_ref:role.value.account.ref,role_ref:role.value.role.ref,code:code.value.trim(),confirm:true})).view;confirmed.value=false}catch(e){error.value=e instanceof Error?e.message:'兑换未完成。'}finally{busy.value=false}}
onUnmounted(task.dispose)
</script>
<template><section><div class="section-heading"><h2>官方前瞻兑换码</h2><button :disabled="task.busy.value||task.job.value?.state==='running'" @click="task.start({kind:'codes'})">查询公开兑换码</button></div>
<p>读取本游戏官方入口；是否可兑换以官方结果为准。国服请在游戏内兑换。</p>
<p v-if="task.error.value||task.job.value?.message" class="feedback danger" role="alert">{{task.error.value||task.job.value?.message}}</p>
<p v-if="task.job.value?.state==='running'">正在查询… <button @click="task.cancel">取消</button><button @click="task.poll">刷新进度</button></p>
<template v-if="task.job.value?.state==='completed'"><h3>{{task.job.value.result?.title}}</h3><p v-if="!rows.length" class="empty">当前官方入口没有可读取的兑换码。</p><ul><li v-for="row in rows" :key="row.code"><strong>{{row.code}}</strong> · {{row.reward}} <span v-if="row.expires_at">· {{row.expiry_estimated?'参考估计期限':'有效期'}}：{{row.expires_at}}</span> <button @click="code=row.code;confirmed=false">填入兑换</button></li></ul></template>
</section><section><h2>海外角色兑换</h2><form class="query-form" @submit.prevent="redeem"><label class="wide">游戏角色<select v-model="selection" :disabled="busy" @change="confirmed=false;view=null"><option value="">选择已授权角色</option><option v-for="r in roles" :key="r.key" :value="r.key">{{r.label}}</option></select></label><label>兑换码<input v-model="code" maxlength="64" :disabled="busy" @input="confirmed=false;view=null"></label><label class="check wide"><input v-model="confirmed" type="checkbox" :disabled="busy||!overseas">确认使用此代码为所选角色兑换</label><p v-if="role&&!overseas" class="wide">当前参考仅支持海外网页兑换，请为此国服角色在游戏内兑换。</p><button :disabled="busy||!overseas||!confirmed||!code.trim()">提交兑换</button></form><p v-if="error" class="feedback danger" role="alert">{{error}}</p><BusinessView v-if="view" :value="view"/></section></template>
