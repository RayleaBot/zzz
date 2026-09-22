<script setup lang="ts">
import { ref, watch } from 'vue'
import BusinessView, { type View } from './BusinessView.vue'
const props=defineProps<{choices:{key:string;label:string;role:{ref:string};account:{ref:string}}[];invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const selected=ref(''),busy=ref(false),error=ref(''),view=ref<View|null>(null)
interface Task {ref:string;role:{uid:string;nickname:string};owner:{actor_id:string};enabled:boolean;hour?:number;expires_at_ms:number;next_check_ms:number;last_code:string;notify?:boolean}
const tasks=ref<Task[]>([]),hour=ref(8),days=ref(30),notify=ref(false),notice=ref('')
const taskLabels:Record<string,string>={signed:'签到成功',already_signed:'今日已签到',sign_attempted:'已尝试，结果待确认',expired:'委托已到期','plugin.upstream_device_required':'需要官方设备验证','plugin.upstream_challenge_required':'需要官方验证','plugin.upstream_first_sign_required':'需要在官方应用首次签到','plugin.account_delegation_denied':'委托失效','plugin.account_caller_denied':'游戏授权已撤销','plugin.vault_locked':'账号库锁定，稍后检查'}
function date(ms:number){return ms?new Date(ms).toLocaleString('zh-CN',{timeZone:'Asia/Shanghai'}):'等待检查'}
watch(selected,()=>{view.value=null;error.value=''})
async function run(action:string){if(busy.value)return;const choice=props.choices.find(item=>item.key===selected.value);if(!choice)return;busy.value=true;error.value='';view.value=null;try{const result=await props.invoke<{view:View}>(action,{account_ref:choice.account.ref,role_ref:choice.role.ref,...(action==='signin.run'?{confirm:true}:{})});view.value=result.view}catch(cause){error.value=cause instanceof Error?cause.message:'签到操作未完成。'}finally{busy.value=false}}
async function loadTasks(){tasks.value=(await props.invoke<{items:Task[]}>('signin.task.list')).items}
async function changeTask(action:string,ref?:string){if(busy.value)return;const choice=props.choices.find(item=>item.key===selected.value);if(action==='signin.task.create'&&!choice)return;busy.value=true;error.value='';notice.value='';try{const result=await props.invoke<{delegation_revoked?:boolean}>(action,ref?{ref}:{account_ref:choice!.account.ref,role_ref:choice!.role.ref,hour:Number(hour.value),days:Number(days.value),notify:notify.value,confirm:true});await loadTasks();notice.value=action==='signin.task.create'?'每日自动签到已开启。':result.delegation_revoked?'自动签到与委托已停止。':'本地自动签到已停止；账号服务暂未确认撤销，此任务不会再执行。'}catch(cause){error.value=cause instanceof Error?cause.message:'自动签到设置未完成。'}finally{busy.value=false}}
void loadTasks().catch(cause=>{error.value=cause instanceof Error?cause.message:'任务读取失败。'})
</script>
<template>
 <section class="game-signin"><h2>游戏签到</h2><p class="hint">通过米游社官方游戏签到领取当日奖励，可单次执行或在下方显式开启每日任务。已签到不会重复提交；首次签到或验证要求需在官方应用处理。</p>
  <label>签到账号与角色<select v-model="selected" :disabled="busy"><option value="" disabled>选择已授权角色</option><option v-for="item in choices" :key="item.key" :value="item.key">{{item.label}}</option></select></label>
  <div class="actions"><button :disabled="busy||!selected" @click="run('signin.status')">查询签到状态</button><button :disabled="busy||!selected" @click="run('signin.rewards')">查看当月奖励</button><button class="primary" :disabled="busy||!selected" @click="run('signin.run')">为所选角色签到</button></div>
  <p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="notice" role="status" class="feedback">{{notice}}</p><BusinessView v-if="view" :value="view"/><p class="hint">验证完成后，可先查询签到状态，再执行当日签到。</p>
  <details><summary>每日自动签到</summary><p class="hint">显式开启后，在期限内每天北京时间所选时刻后的首次检查执行，每十分钟检查一次。默认不发送结果通知。未确认的提交不在当天自动重试，可手动查询状态。</p>
   <form @submit.prevent="changeTask('signin.task.create')"><fieldset :disabled="busy||!selected"><legend>为上方所选角色开启</legend><label>北京时间（小时）<input v-model.number="hour" type="number" min="0" max="23" step="1" required></label><label>授权天数<input v-model.number="days" type="number" min="1" max="90" step="1" required></label><label class="check"><input v-model="notify" type="checkbox">私聊账号所属用户，通知每日结果</label><button type="submit" class="primary">开启每日自动签到</button></fieldset></form>
  </details>
  <section class="sign-tasks"><h3>自动签到任务</h3><p v-if="!tasks.length" class="hint">尚未开启自动签到。</p><ul><li v-for="task in tasks" :key="task.ref"><strong>{{task.role.nickname}} · {{task.role.uid}}</strong><p>{{task.enabled?'已开启':'已暂停'}} · 每日北京时间 {{task.hour??0}}:00 · {{task.notify?'私聊通知 '+task.owner.actor_id:'不发送结果通知'}}</p><p class="hint">{{taskLabels[task.last_code]||task.last_code||'等待首次检查'}}<br>下次可检查：{{date(task.next_check_ms)}}<br>到期：{{date(task.expires_at_ms)}}（北京时间）</p><button :disabled="busy" @click="changeTask('signin.task.remove',task.ref)">停止此自动签到</button></li></ul></section>
 </section>
</template>
<style scoped>.game-signin{display:grid;gap:20px}.game-signin label:not(.check){display:grid;gap:8px;min-width:0}.game-signin select{width:100%;min-width:0}.game-signin details>p,.game-signin details>form{margin-top:16px}.game-signin fieldset{display:grid;gap:16px;min-width:0;padding:16px;border:1px solid var(--raylea-color-border,#ddd);border-radius:12px}.sign-tasks ul{list-style:none;padding:0;display:grid;gap:16px}.sign-tasks li{display:grid;gap:12px;border-top:1px solid var(--raylea-color-border,#ddd);padding-top:16px}</style>
