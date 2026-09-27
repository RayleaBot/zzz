<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
interface Task {
  ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string }
  full: boolean; notify: boolean; state: string; last_code: string; started_ms: number; finished_ms?: number
  progress: { pages: number; fetched: number; result?: { added: number; total: number } }
}
const props = defineProps<{ choices: { key: string; label: string; account: { ref: string }; role: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), full = ref(false), notify = ref(false)
const tasks = ref<Task[]>([]), busy = ref(false), loading = ref(false), error = ref(''), loadError = ref(''), notice = ref('')
let disposed = false, revision = 0
const states: Record<string, string> = { running: '同步中', completed: '已完成', failed: '未完成', canceled: '已取消' }
const reasons: Record<string, string> = {
  sync_running: '正在读取官方记录', sync_completed: '同步完成', sync_canceled: '已取消，原档案保持不变', archive_removed: '档案已移除，同步已取消', sync_timeout: '后台期限内未读完，原档案保持不变',
  'sync_completed.notification_failed': '记录已保存，结果通知未能发送', 'plugin.game_sync_conflict': '档案发生变化，请确认后重新同步', 'plugin.game_sync_invalid': '官方分页或记录异常，原档案保留',
  'plugin.account_caller_denied': '本游戏的账号授权已撤销', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权已变更',
  'plugin.upstream_auth_invalid': '账号登录已失效', 'plugin.upstream_device_required': '请先在官方应用完成设备验证', 'plugin.upstream_challenge_required': '请先在官方应用完成验证',
  'plugin.vault_locked': '账号库已锁定，稍后重试', 'plugin.service_unavailable': '账号插件不可用，稍后重试',
}
function date(ms: number) { return new Date(ms).toLocaleString('zh-CN', { timeZone: 'Asia/Shanghai' }) }
async function load() {
  const request = ++revision; loading.value = true
  try {
    const result = await props.invoke<{ items: Task[] }>('gacha.task.list')
    if (!disposed && request === revision) { tasks.value = result.items; loadError.value = '' }
  } catch (cause) { if (!disposed && request === revision) loadError.value = cause instanceof Error ? cause.message : '后台同步读取失败。' }
  finally { if (!disposed && request === revision) loading.value = false }
}
async function change(action: 'gacha.task.start' | 'gacha.task.cancel', ref?: string) {
  if (busy.value) return
  const choice = props.choices.find(c => c.key === selected.value)
  if (action === 'gacha.task.start' && !choice) return
  busy.value = true; revision++; loading.value = false; error.value = ''; notice.value = ''
  try {
    await props.invoke(action, ref ? { ref } : { account_ref: choice!.account.ref, role_ref: choice!.role.ref, full: full.value, notify: notify.value, confirm: true })
    if (disposed) return
    notice.value = action === 'gacha.task.start' ? '后台同步已开始，关闭页面仍会继续。' : '后台同步已取消，原档案保持不变。'
    await load()
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '后台同步操作未完成。' }
  finally { if (!disposed) busy.value = false }
}
void load()
const timer = setInterval(() => { if (!busy.value && !loading.value) void load() }, 5000)
// Closing this view only stops polling; the sync runs on in the plugin.
onUnmounted(() => { disposed = true; revision++; clearInterval(timer) })
</script>
<template>
  <section class="gacha-tasks separated">
    <div class="section-heading"><h3>后台同步</h3><button :disabled="busy || loading" @click="load">刷新进度</button></div>
    <p class="hint">开始后关闭页面仍会继续，两页之间至少间隔一秒；全部卡池读取成功后合并。插件重启会中止未完成的同步，已有档案保留。删除档案会取消对应的同步。</p>
    <details><summary>开始后台同步</summary>
      <form @submit.prevent="change('gacha.task.start')"><fieldset :disabled="busy"><legend class="sr-only">后台同步设置</legend>
        <label>后台同步角色<select v-model="selected" required><option value="" disabled>选择已授权角色</option><option v-for="c in choices" :key="c.key" :value="c.key">{{ c.label }}</option></select></label>
        <label class="check"><input v-model="full" type="checkbox">全量读取官方仍保留的历史（默认只读新增记录）</label>
        <label class="check"><input v-model="notify" type="checkbox">完成后私聊账号所属用户，通知结果</label>
        <button class="primary" type="submit" :disabled="!selected">开始后台同步</button>
      </fieldset></form>
    </details>
    <p v-if="error || loadError" role="alert" class="feedback danger">{{ error || loadError }}</p><p v-if="notice" role="status" class="feedback">{{ notice }}</p>
    <p v-if="!tasks.length" class="hint">尚无后台同步。</p>
    <ul class="task-list"><li v-for="task in tasks" :key="task.ref">
      <div class="task-heading"><strong>{{ task.role.nickname }} · {{ task.role.uid }}</strong><span>{{ states[task.state] || '状态待确认' }}</span></div>
      <p>{{ task.full ? '全量' : '增量' }} · {{ task.notify ? `通知 ${task.owner.actor_id}` : '不发送结果通知' }}</p>
      <p>已读取 {{ task.progress.pages || 0 }} 页、{{ task.progress.fetched || 0 }} 条。<span v-if="task.progress.result">新增 {{ task.progress.result.added }} 条，共 {{ task.progress.result.total }} 条。</span></p>
      <p class="hint">{{ reasons[task.last_code] || '本次同步未完成，可查看插件诊断后重试。' }}<br>开始于 {{ date(task.started_ms) }}<template v-if="task.finished_ms">，结束于 {{ date(task.finished_ms) }}</template>（北京时间）</p>
      <div v-if="task.state === 'running'" class="actions"><button :disabled="busy" @click="change('gacha.task.cancel', task.ref)">取消此同步</button></div>
    </li></ul>
  </section>
</template>
<style scoped>
.gacha-tasks{display:grid;gap:20px}.gacha-tasks .section-heading{margin-bottom:0}.gacha-tasks fieldset{display:grid;gap:16px;padding:16px 0 0;border:0;min-width:0}.gacha-tasks label:not(.check){display:grid;gap:8px;min-width:0}.task-list{list-style:none;padding:0;margin:0}.task-list li{display:grid;gap:12px;padding:20px 0;border-bottom:1px solid var(--raylea-color-border)}.task-heading{display:flex;justify-content:space-between;flex-wrap:wrap;gap:8px}.task-heading span{font-weight:600}
</style>
