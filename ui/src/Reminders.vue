<script setup lang="ts">
import { ref, watch } from 'vue'
interface Task { ref: string; role: { uid: string; nickname: string }; owner: { actor_id: string; source_adapter: string }; threshold: number; enabled: boolean; armed: boolean; expires_at_ms: number; last_checked_ms: number; last_attempt_ms: number; next_check_ms: number; last_code: string }
const props = defineProps<{ choices: { key: string; label: string; role: { ref: string }; account: { ref: string; owner: { actor_id: string } } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const items = ref<Task[]>([]), selected = ref(''), threshold = ref(80), days = ref(30), confirmed = ref(false), busy = ref(false), error = ref(''), notice = ref('')
watch(selected, () => { confirmed.value = false })
const labels: Record<string, string> = { checked: '已检查', notified: '已发送提醒', notification_attempted: '已尝试发送', notification_failed: '发送未确认', expired: '委托到期', 'plugin.account_caller_denied': '游戏授权已撤销', 'plugin.account_delegation_denied': '委托失效', 'plugin.account_not_found': '账号已移除', 'plugin.account_role_denied': '角色授权失效', 'plugin.upstream_auth_invalid': '账号需要重新登录', 'plugin.upstream_device_required': '需要官方设备验证，已退避', 'plugin.upstream_challenge_required': '需要官方验证，已退避', 'plugin.vault_locked': '账号库锁定', 'plugin.game_note_invalid': '体力数据暂不兼容' }
function date(value: number) { return value ? new Date(value).toLocaleString() : '尚无记录' }
async function load() { items.value = (await props.invoke<{ items: Task[] }>('reminder.list')).items }
async function run(fn: () => Promise<void>) { if (busy.value) return; busy.value = true; error.value = ''; notice.value = ''; try { await fn() } catch (cause) { error.value = cause instanceof Error ? cause.message : '提醒操作未完成。' } finally { busy.value = false } }
async function create() { const choice = props.choices.find(value => value.key === selected.value); if (!choice || !confirmed.value) return; await run(async () => { await props.invoke('reminder.create', { account_ref: choice.account.ref, role_ref: choice.role.ref, threshold: Number(threshold.value), days: Number(days.value) }); await load(); confirmed.value = false; notice.value = '提醒已开启，下一次检查将在十分钟内执行。' }) }
async function remove(ref: string) { await run(async () => { const result = await props.invoke<{ delegation_revoked: boolean }>('reminder.remove', { ref }); await load(); notice.value = result.delegation_revoked ? '提醒与账号委托已关闭。' : '本地提醒已关闭；账号服务暂未确认撤销，原委托到期后失效，此任务不会再使用它。' }) }
void run(load)
</script>
<template>
  <section class="reminders">
    <h2>体力提醒</h2><p class="hint">每十分钟读取一次已授权角色的体力，达到阈值后私聊账号所属用户。持续高于阈值只通知一次，低于阈值后重新待发。默认关闭，开启时创建有期限的只读委托。</p>
    <p v-if="error" class="feedback danger" role="alert">{{ error }}</p><p v-if="notice" class="feedback" role="status">{{ notice }}</p>
    <form @submit.prevent="create"><fieldset :disabled="busy"><legend>新提醒</legend>
      <label>提醒账号与接收者<select v-model="selected" required><option value="" disabled>选择角色</option><option v-for="choice in choices" :key="choice.key" :value="choice.key">{{ choice.label }}</option></select></label>
      <div class="reminder-fields"><label>体力阈值（%）<input v-model="threshold" type="number" min="1" max="100" step="1" required></label><label>授权天数<input v-model="days" type="number" min="1" max="90" step="1" required></label></div>
      <label class="check"><input v-model="confirmed" type="checkbox">允许定时读取所选角色体力并私聊通知账号所属用户</label><button class="primary" :disabled="!selected || !confirmed" type="submit">开启提醒</button>
    </fieldset></form>
    <div class="section-heading"><h3>已保存的提醒</h3><button :disabled="busy" @click="run(load)">刷新提醒</button></div>
    <p v-if="!items.length" class="empty">尚未开启提醒。</p>
    <ul v-else class="reminder-list"><li v-for="item in items" :key="item.ref"><div class="section-heading"><strong>{{ item.role.nickname }} · {{ item.role.uid }}</strong><button :disabled="busy" @click="remove(item.ref)">关闭并移除提醒</button></div>
      <p>{{ item.enabled ? (item.armed ? '等待达到阈值' : '等待体力低于阈值') : '已暂停' }} · {{ item.threshold }}% · 私聊 {{ item.owner.actor_id }}（{{ item.owner.source_adapter }}）</p>
      <p class="hint">到期：{{ date(item.expires_at_ms) }}<br>最近检查：{{ date(item.last_checked_ms) }} · {{ labels[item.last_code] || item.last_code || '等待首次检查' }}<br>下一次可检查：{{ date(item.next_check_ms) }}</p>
    </li></ul>
    <p class="hint">发送结果不确定时不自动补发，以免重复通知。设备验证失败会暂停查询至少一天；撤销授权、凭据失效或到期后暂停，处理后可移除并重新开启。</p>
  </section>
</template>
<style scoped>
.reminders{display:grid;gap:20px}.reminders fieldset{display:grid;gap:16px;border:1px solid var(--raylea-color-border,#ddd);border-radius:12px;padding:18px;min-width:0}.reminders label:not(.check){display:grid;gap:8px;min-width:0}.reminders select{width:100%;min-width:0}.reminder-fields{display:flex;gap:16px;flex-wrap:wrap}.reminder-fields label{flex:1 1 150px}.reminder-list{list-style:none;padding:0;display:grid;gap:16px}.reminder-list>li{border-top:1px solid var(--raylea-color-border,#ddd);padding-top:16px}.reminder-list p{margin-top:10px;overflow-wrap:anywhere}
</style>
