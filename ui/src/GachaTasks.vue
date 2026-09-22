<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
interface Task {
  ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }
  kind: 'once' | 'daily'; hour: number; full: boolean; notify: boolean
  state: string; last_code: string; next_check_ms: number; expires_at_ms: number; restarts: number
  progress: { pages: number; fetched: number; result?: { added: number; total: number } }
}
const props = defineProps<{ choices: { key: string; label: string; account: { ref: string }; role: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), kind = ref('once'), days = ref(7), hour = ref(8), full = ref(false), notify = ref(false)
const tasks = ref<Task[]>([]), busy = ref(false), loading = ref(false), error = ref(''), loadError = ref(''), notice = ref('')
let disposed = false, revision = 0
const states: Record<string, string> = { creating: '创建中', waiting: '等待调度', running: '同步中', completed: '已完成', paused: '已暂停', expired: '已到期' }
const reasons: Record<string, string> = {
  queued: '已排队', sync_running: '正在读取官方记录', sync_completed: '本轮同步完成', expired: '授权已到期，请移除后重新创建', archive_removed: '档案已移除，自动同步已暂停',
  'sync_completed.notification_failed': '记录已保存，结果通知未能发送', 'plugin.game_sync_conflict': '档案发生变化，请确认后重新运行', 'plugin.game_sync_invalid': '官方分页或记录异常，原档案保留',
  'plugin.account_delegation_denied': '委托已撤销或失效，请重新创建', 'plugin.account_caller_denied': '本游戏的账号授权已撤销', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权已变更',
  'plugin.upstream_auth_invalid': '账号登录已失效', 'plugin.upstream_device_required': '请先在官方应用完成设备验证', 'plugin.upstream_challenge_required': '请先在官方应用完成验证',
  'plugin.vault_locked': '账号库已锁定，稍后重试', 'plugin.service_unavailable': '账号插件不可用，稍后重试', 'plugin.game_sync_busy': '其他同步正在占用名额，稍后重试', 'plugin.account_delegation_throttled': '分页请求过于频繁，稍后重试',
}
function date(ms: number) { return ms ? new Date(ms).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) : '等待下一次调度' }
watch(kind, value => { days.value = value === 'daily' ? 30 : 7 })
async function load() {
  const request = ++revision; loading.value = true
  try {
    const result = await props.invoke<{ items: Task[] }>('gacha.task.list')
    if (!disposed && request === revision) { tasks.value = result.items; loadError.value = '' }
  } catch (cause) { if (!disposed && request === revision) loadError.value = cause instanceof Error ? cause.message : '后台任务读取失败。' }
  finally { if (!disposed && request === revision) loading.value = false }
}
async function change(action: string, ref?: string) {
  if (busy.value) return
  const choice = props.choices.find(c => c.key === selected.value)
  if (action === 'gacha.task.create' && !choice) return
  busy.value = true; revision++; loading.value = false; error.value = ''; notice.value = ''
  try {
    const result = await props.invoke<{ delegation_revoked?: boolean }>(action, ref ? { ref, confirm: true } : {
      account_ref: choice!.account.ref, role_ref: choice!.role.ref, kind: kind.value,
      days: Number(days.value), hour: Number(hour.value), full: full.value, notify: notify.value, confirm: true,
    })
    if (disposed) return
    notice.value = action === 'gacha.task.create' ? '后台同步已开启，关闭页面仍会继续。' : action === 'gacha.task.run' ? '任务已排队重新运行。' : result.delegation_revoked ? '后台任务与委托已停止。' : '后台任务已停止；账号服务暂未确认撤销，原委托等待到期。'
    await load()
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '任务操作未完成。' }
  finally { if (!disposed) busy.value = false }
}
void load()
const timer = setInterval(() => { if (!busy.value && !loading.value) void load() }, 5000)
// Closing this view only stops polling. The user chose a persistent host task.
onUnmounted(() => { disposed = true; revision++; clearInterval(timer) })
</script>
<template>
  <section class="gacha-tasks separated">
    <div class="section-heading"><h3>后台与每日同步</h3><button :disabled="busy || loading" @click="load">刷新任务</button></div>
    <p class="hint">关闭页面后仍会继续，每分钟最多读取三页；全部卡池读取成功后合并。重启会重新读取未完成的一轮，已有档案保留。删除档案会暂停对应任务。</p>
    <details><summary>创建后台同步任务</summary>
      <form @submit.prevent="change('gacha.task.create')"><fieldset :disabled="busy"><legend class="sr-only">后台同步设置</legend>
        <label>后台同步角色<select v-model="selected" required><option value="" disabled>选择已授权角色</option><option v-for="c in choices" :key="c.key" :value="c.key">{{ c.label }}</option></select></label>
        <div class="task-fields"><label>同步安排<select v-model="kind"><option value="once">单次后台同步</option><option value="daily">每日同步</option></select></label><label>授权天数<input v-model.number="days" type="number" min="1" max="90" step="1" required></label><label v-if="kind === 'daily'">北京时间（小时）<input v-model.number="hour" type="number" min="0" max="23" step="1" required></label></div>
        <label class="check"><input v-model="full" type="checkbox">每轮全量读取官方仍保留的历史（默认只读新增记录）</label>
        <label class="check"><input v-model="notify" type="checkbox">每轮完成后私聊账号所属用户，通知结果</label>
        <button class="primary" type="submit" :disabled="!selected">{{ kind === 'daily' ? '开启每日同步' : '开始单次后台同步' }}</button>
      </fieldset></form>
    </details>
    <p v-if="error || loadError" role="alert" class="feedback danger">{{ error || loadError }}</p><p v-if="notice" role="status" class="feedback">{{ notice }}</p>
    <p v-if="!tasks.length" class="hint">尚无后台同步任务。单次完成后可重新运行，或停止后调整设置。</p>
    <ul class="task-list"><li v-for="task in tasks" :key="task.ref">
      <div class="task-heading"><strong>{{ task.role.nickname }} · {{ task.role.uid }}</strong><span>{{ states[task.state] || '状态待确认' }}</span></div>
      <p>{{ task.kind === 'daily' ? `每天北京时间 ${task.hour}:00` : '单次后台同步' }} · {{ task.full ? '全量' : '增量' }} · {{ task.notify ? `通知 ${task.owner.actor_id}` : '不发送结果通知' }}</p>
      <p>本轮已读取 {{ task.progress.pages || 0 }} 页、{{ task.progress.fetched || 0 }} 条。<span v-if="task.progress.result">最近完成：新增 {{ task.progress.result.added }} 条，共 {{ task.progress.result.total }} 条。</span></p>
      <p class="hint">{{ reasons[task.last_code] || (task.last_code ? '本次请求未完成，可查看插件诊断后重试。' : '等待首次运行') }}<span v-if="task.restarts"> · 本轮重新读取 {{ task.restarts }} 次</span><br>授权到期：{{ date(task.expires_at_ms) }}（北京时间）<template v-if="['waiting', 'running'].includes(task.state)"><br>下次可检查：{{ date(task.next_check_ms) }}</template></p>
      <div class="actions"><button :disabled="busy || ['creating', 'running', 'expired'].includes(task.state) || task.expires_at_ms <= Date.now()" @click="change('gacha.task.run', task.ref)">重新运行此同步</button><button :disabled="busy" @click="change('gacha.task.remove', task.ref)">停止此同步任务</button></div>
    </li></ul>
  </section>
</template>
<style scoped>
.gacha-tasks{display:grid;gap:20px}.gacha-tasks .section-heading{margin-bottom:0}.gacha-tasks fieldset{display:grid;gap:16px;padding:16px 0 0;border:0;min-width:0}.gacha-tasks label:not(.check){display:grid;gap:8px;min-width:0}.task-fields{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:16px}.task-list{list-style:none;padding:0;margin:0}.task-list li{display:grid;gap:12px;padding:20px 0;border-bottom:1px solid var(--raylea-color-border)}.task-heading{display:flex;justify-content:space-between;flex-wrap:wrap;gap:8px}.task-heading span{font-weight:600}@media(max-width:720px){.task-fields{grid-template-columns:1fr}}
</style>
