<script setup lang="ts">
import {computed,ref,watch} from 'vue'
import BusinessView,{type View} from './BusinessView.vue'
import AccountTasks from './AccountTasks.vue'
interface Account {ref:string;account_label:string;owner:{actor_id:string};cloud_configured?:string[]}
interface Mission {mission_id?:number;happened_times?:number;mission_cnt?:number;is_get_award?:boolean}
const props=defineProps<{accounts:Account[];invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const selected=ref(''),busy=ref(false),error=ref(''),view=ref<View|null>(null),missions=ref<Mission[]>([]),posts=ref<{post_id:string;title:string}[]>([]),post=ref(''),step=ref('read'),confirmed=ref(false),cloudConfirmed=ref(false)
const account=computed(()=>props.accounts.find(a=>a.ref===selected.value))
watch(selected,()=>{view.value=null;missions.value=[];posts.value=[];post.value='';confirmed.value=false;cloudConfirmed.value=false})
watch([step,post],()=>{confirmed.value=false})
async function run(action:string,payload:Record<string,unknown>={}){if(busy.value||!selected.value)return;busy.value=true;error.value='';try{const r=await props.invoke<{view:View;result:{data:{states?:Mission[];posts?:typeof posts.value}}}>(action,{account_ref:selected.value,...payload});view.value=r.view;if(action==='community.status')missions.value=r.result.data.states??[];if(action==='community.posts'){posts.value=r.result.data.posts??[];post.value=posts.value[0]?.post_id??''}}catch(e){error.value=e instanceof Error?e.message:'操作未完成。'}finally{busy.value=false;confirmed.value=false;cloudConfirmed.value=false}}
const stepNames:Record<string,string>={read:'浏览并计入任务',like:'点赞',unlike:'取消点赞',share:'分享任务'}
</script>
<template>
 <section class="community-panel"><h2>米游社任务与云游戏</h2><p class="hint">米游币任务按米游社账号统计，三游戏共用同一登录。这里可查询进度并明确执行单次操作；云游戏的独立授权在“米游社账号”管理页加密配置。</p>
 <label>社区账号<select v-model="selected" :disabled="busy"><option value="" disabled>选择账号</option><option v-for="a in accounts" :key="a.ref" :value="a.ref">{{a.account_label}} · 用户 {{a.owner.actor_id}}</option></select></label><p v-if="error" role="alert" class="feedback danger">{{error}}</p>
 <template v-if="account"><div class="actions"><button :disabled="busy" @click="run('community.status')">查询米游币与任务</button><button :disabled="busy" @click="run('community.posts')">读取本游戏讨论区帖子</button><button :disabled="busy" @click="run('community.run',{step:'sign',confirm:true})">为此账号社区签到</button></div>
 <div v-if="missions.length" class="table-scroll"><table><thead><tr><th>任务编号</th><th>已发生次数</th><th>要求次数</th><th>奖励状态</th></tr></thead><tbody><tr v-for="(m,i) in missions" :key="i"><td>{{m.mission_id??'未提供'}}</td><td>{{m.happened_times??'未提供'}}</td><td>{{m.mission_cnt??'未提供'}}</td><td>{{m.is_get_award===true?'已领取':m.is_get_award===false?'未领取':'未提供'}}</td></tr></tbody></table></div>
 <form v-if="posts.length" @submit.prevent="confirmed&&run('community.run',{step,post_id:post,confirm:true})"><fieldset :disabled="busy"><legend>所选帖子操作</legend><label>讨论区帖子<select v-model="post"><option v-for="p in posts" :key="p.post_id" :value="p.post_id">{{p.title}} · {{p.post_id}}</option></select></label><label>社区操作<select v-model="step"><option v-for="(name,id) in stepNames" :key="id" :value="id">{{name}}</option></select></label><label class="check"><input v-model="confirmed" type="checkbox">确认以所选账号对该帖子执行{{stepNames[step]}}</label><button :disabled="!confirmed||!post">执行所选社区操作</button></fieldset></form>
 <section><h3>云绝区零</h3><p class="hint">授权{{account.cloud_configured?.includes('zzz')?'已配置':'未配置'}}。钱包查询可能同时领取免费时长，因此需要明确确认；只读状态按钮显示上次保存的快照。</p><button :disabled="busy" @click="run('cloudgame.status')">查看最近云游戏状态</button><form @submit.prevent="cloudConfirmed&&run('cloudgame.sign',{confirm:true})"><fieldset :disabled="busy"><legend>查询并领取免费时长</legend><label class="check"><input v-model="cloudConfirmed" type="checkbox">确认查询此云账号钱包并领取可能发放的每日时长</label><button :disabled="!cloudConfirmed||!account.cloud_configured?.includes('zzz')">查询云钱包并签到</button></fieldset></form></section>
 <BusinessView v-if="view" :value="view"/>
 </template><AccountTasks :accounts="accounts" :invoke="invoke"/></section>
</template>
<style scoped>.community-panel{display:grid;gap:20px}.community-panel label:not(.check){display:grid;gap:8px;min-width:0}.community-panel select{min-width:0;width:100%}.community-panel fieldset{display:grid;gap:14px;border:1px solid var(--raylea-color-border);padding:16px;border-radius:12px;min-width:0}.community-panel>section{display:grid;gap:14px}.table-scroll{overflow-x:auto}table{width:100%;border-collapse:collapse}th,td{text-align:left;padding:10px;white-space:nowrap;border-bottom:1px solid var(--raylea-color-border)}.community-panel ul{list-style:none;padding:0;display:grid;gap:10px}.community-panel li{display:flex;flex-wrap:wrap;gap:10px;justify-content:space-between}</style>
