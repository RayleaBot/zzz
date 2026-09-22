<script setup lang="ts">
import {computed,ref,watch} from 'vue'
interface Kind {id:string;label:string;metrics:{key:string;label:string;lower:boolean}[]}
interface Task {ref:string;challenge_kind:string;metric:string;threshold:number;hour:number;minute:number;weekday:number;role:{uid:string;nickname:string};owner:{actor_id:string};enabled:boolean;last_code:string;next_check_ms:number;expires_at_ms:number}
const props=defineProps<{choices:{key:string;label:string;role:{ref:string};account:{ref:string}}[];invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
const kinds=ref<Kind[]>([]),items=ref<Task[]>([]),selected=ref(''),kind=ref(''),metric=ref(''),threshold=ref(1),hour=ref(20),minute=ref(0),weekday=ref(0),days=ref(30),confirmed=ref(false),busy=ref(false),error=ref(''),notice=ref(''),checking=ref<{known:boolean;met:boolean;value:number;state?:string}|null>(null)
const chosenKind=computed(()=>kinds.value.find(v=>v.id===kind.value)),chosenMetric=computed(()=>chosenKind.value?.metrics.find(v=>v.key===metric.value))
watch(kind,()=>{metric.value=chosenKind.value?.metrics[0]?.key??''})
watch([selected,kind,metric,threshold,hour,minute,weekday,days],()=>{confirmed.value=false;checking.value=null})
async function run(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await fn()}catch(e){error.value=e instanceof Error?e.message:'挑战提醒操作未完成。'}finally{busy.value=false}}
async function load(){items.value=(await props.invoke<{items:Task[]}>('challenge.reminder.list')).items}
function payload(){const c=props.choices.find(v=>v.key===selected.value);if(!c)throw new Error('请选择已授权角色。');return {account_ref:c.account.ref,role_ref:c.role.ref,kind:kind.value,metric:metric.value,threshold:Number(threshold.value),hour:Number(hour.value),minute:Number(minute.value),weekday:Number(weekday.value),days:Number(days.value),confirm:confirmed.value}}
async function create(){if(!confirmed.value)return;await run(async()=>{await props.invoke('challenge.reminder.create',payload());confirmed.value=false;await load();notice.value='挑战提醒已开启。'})}
async function check(){await run(async()=>{checking.value=await props.invoke('challenge.reminder.check',payload())})}
async function remove(ref:string){await run(async()=>{await props.invoke('challenge.reminder.remove',{ref});await load();notice.value='挑战提醒已关闭。'})}
const weekdays=['每日','每周一','每周二','每周三','每周四','每周五','每周六','每周日']
const labels:Record<string,string>={notified:'已发送提醒',target_met:'已达到目标',metric_unknown:'官方未提供所选指标',check_attempted:'已尝试检查',notification_failed:'发送结果未确认',expired:'委托到期','plugin.upstream_device_required':'需要官方设备验证','plugin.upstream_challenge_required':'需要官方验证','plugin.account_delegation_denied':'委托失效','plugin.account_caller_denied':'插件授权已撤销','plugin.game_challenge_unavailable':'官方成绩暂不可用'}
void run(async()=>{kinds.value=(await props.invoke<{kinds:Kind[]}>('challenge.schema')).kinds;kind.value=kinds.value[0]?.id??'';await load()})
</script>
<template>
 <section class="challenge-reminders"><h2>挑战完成提醒</h2><p class="hint">在指定的北京时间检查本期官方成绩，未达目标时私聊账号所属用户。默认关闭；每次授权最多 90 天。每个角色每种玩法可设置一个目标，调整前先关闭旧任务。</p>
 <p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="notice" role="status">{{notice}}</p>
 <form @submit.prevent="create"><fieldset :disabled="busy"><legend>完成目标和检查时间</legend><label>挑战提醒账号<select v-model="selected" required><option value="" disabled>选择角色及接收者</option><option v-for="c in choices" :key="c.key" :value="c.key">{{c.label}}</option></select></label>
 <div class="fields"><label>提醒玩法<select v-model="kind"><option v-for="k in kinds" :key="k.id" :value="k.id">{{k.label}}</option></select></label><label>完成指标<select v-model="metric"><option v-for="m in chosenKind?.metrics" :key="m.key" :value="m.key">{{m.label}} · {{m.key}}</option></select></label><label>目标{{chosenMetric?.lower?'不超过':'至少达到'}}<input v-model="threshold" type="number" min="0" max="100000000000000" step="any" required></label></div>
 <div class="fields"><label>重复周期<select v-model="weekday"><option v-for="(label,i) in weekdays" :key="i" :value="i">{{label}}</option></select></label><label>时<input v-model="hour" type="number" min="0" max="23" required></label><label>分<input v-model="minute" type="number" min="0" max="59" required></label><label>委托天数<input v-model="days" type="number" min="1" max="90" required></label></div>
 <label class="check"><input v-model="confirmed" type="checkbox">允许定时读取所选玩法并私聊通知账号所属用户</label><div class="actions"><button :disabled="!selected||!confirmed" class="primary">开启挑战提醒</button><button type="button" :disabled="!selected" @click="check">立即检查挑战目标</button></div>
 </fieldset></form>
 <p v-if="checking" role="status">{{checking.state==='not_started'?'本期尚无完成记录':!checking.known?'官方未提供此指标':`${chosenMetric?.label}：${checking.value} · ${checking.met?'已达到目标':'尚未达到目标'}`}}</p>
 <div class="section-heading"><h3>已保存的挑战提醒</h3><button :disabled="busy" @click="run(load)">刷新挑战提醒</button></div><p v-if="!items.length" class="hint">没有已开启的挑战提醒。</p>
 <ul><li v-for="t in items" :key="t.ref"><strong>{{t.role.nickname}} · {{t.role.uid}} · {{kinds.find(k=>k.id===t.challenge_kind)?.label}}</strong><p>{{t.enabled?'已开启':'已暂停'}} · {{weekdays[t.weekday]}} {{String(t.hour).padStart(2,'0')}}:{{String(t.minute).padStart(2,'0')}} · {{t.metric}} 目标 {{t.threshold}} · 私聊 {{t.owner.actor_id}}</p><p class="hint">{{labels[t.last_code]||t.last_code||'等待首次检查'}} · 下次 {{new Date(t.next_check_ms).toLocaleString()}} · 到期 {{new Date(t.expires_at_ms).toLocaleString()}}</p><button :disabled="busy" @click="remove(t.ref)">关闭此挑战提醒</button></li></ul>
 </section>
</template>
<style scoped>.challenge-reminders{display:grid;gap:18px;border-top:1px solid var(--raylea-color-border);padding-top:24px}.challenge-reminders fieldset{display:grid;gap:16px;border:1px solid var(--raylea-color-border);padding:18px;border-radius:12px;min-width:0}.challenge-reminders label:not(.check){display:grid;gap:8px;min-width:0}.challenge-reminders select{min-width:0;width:100%}.fields{display:flex;flex-wrap:wrap;gap:14px}.fields label{flex:1 1 130px}.challenge-reminders ul{list-style:none;padding:0;display:grid;gap:16px}.challenge-reminders li{border-top:1px solid var(--raylea-color-border);padding-top:16px;overflow-wrap:anywhere}.challenge-reminders li p{margin:10px 0}</style>
