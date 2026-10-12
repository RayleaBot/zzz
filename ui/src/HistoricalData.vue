<script setup lang="ts">
import { computed, ref } from 'vue'
import { PluginUIClient, type BotAccount } from '@rayleabot/plugin-ui'
import { historicalRecords, type HistoricalRecord } from './history-accounts'
import { botLabel } from './bots'

const props = defineProps<{ client: PluginUIClient; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const rows = ref<Awaited<ReturnType<typeof historicalRecords>>>([])
const bots = ref<BotAccount[]>([]), cursor = ref<string | undefined>(), query = ref(''), selected = ref('')
const total = ref(0), busy = ref(false), loaded = ref(false), error = ref(''), notice = ref('')
const candidate = ref<HistoricalRecord | null>(null), detail = ref('')
const count = computed(() => rows.value.reduce((sum, row) => sum + row.count, 0))
const labels: Record<string, string> = { groups: '群设置与排行', reminders: '提醒与任务', daily: '每日抽卡同步', subscriptions: '群推送', interactions: '图片互动记录', preferences: '个人提醒设置', panels: '面板历史', monthly: '月报档案', background: '后台抽卡同步' }

async function run(work: () => Promise<void>) {
  if (busy.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try { await work() } catch (cause) { error.value = cause instanceof Error ? cause.message : '历史数据操作未完成。' }
  finally { busy.value = false }
}
async function loadBots(more = false) {
  const params = new URLSearchParams({ source_protocol: 'onebot11', limit: '100', query: query.value })
  if (more && cursor.value) params.set('cursor', cursor.value)
  const result = await props.client.apiRequest<{ items: BotAccount[]; next_cursor?: string }>('GET', `/api/bot-accounts?${params}`)
  bots.value = more ? [...bots.value, ...result.items] : result.items
  cursor.value = result.next_cursor
  if (!more) selected.value = ''
}
async function load() {
  loaded.value = false; candidate.value = null; detail.value = ''
  const result = await props.invoke<{ items: HistoricalRecord[]; total: number }>('history.list')
  const records = await historicalRecords(result.items, path => props.client.apiRequest('GET', path))
  rows.value = records; total.value = result.total
  await loadBots()
  loaded.value = true
}
function open(event: Event) {
  if ((event.target as HTMLDetailsElement).open && !loaded.value) void run(load)
}
async function inspect(row: HistoricalRecord) {
  await run(async () => { candidate.value = row; selected.value = ''; detail.value = ''; const result = await props.invoke<{ record: unknown }>('history.get', { kind: row.kind, ref: row.ref }); detail.value = JSON.stringify(result.record, null, 2) })
}
async function change(assign: boolean) {
  const row = candidate.value
  if (!row) return
  await run(async () => {
    const payload = { kind: row.kind, ref: row.ref, revision: row.revision, confirm: true }
    if (assign) {
      const bot = bots.value.find(item => item.bot_id === selected.value)
      if (!bot) throw new Error('请选择目录中的 OneBot11 账号。')
      // The host validates the directory entry and supplies ManagementBot.
      const client = new PluginUIClient({ pluginId: props.client.pluginId, pageId: 'groups', bot: { source_protocol: bot.source_protocol, bot_id: bot.bot_id } })
      await client.invokeAction('history.assign', payload)
    } else await props.invoke('history.remove', payload)
    await load()
    notice.value = assign ? '已补齐机器人账号。' : '历史记录已删除。'
  })
}
</script>

<template>
  <details class="settings history-data" @toggle="open">
    <summary>历史数据与归属</summary>
    <p class="hint">查看缺少账号或账号已不在目录中的记录。仅可为缺少账号的 OneBot11 记录补齐归属；QQ 官方记录和已有账号的记录不能改挂。</p>
    <div class="section-heading"><h3>待处理记录</h3><button :disabled="busy" @click="run(load)">刷新历史数据</button></div>
    <p v-if="error" class="feedback danger" role="alert">{{ error }}</p>
    <p v-if="notice" class="feedback" role="status">{{ notice }}</p>
    <template v-if="loaded">
      <p class="hint">待处理 {{ count }} 条 · 已检查 {{ total }} 条归属记录</p>
      <p v-if="!rows.length" class="empty">没有待处理的历史记录。</p>
      <ul class="archive-list"><li v-for="row in rows" :key="`${row.kind}:${row.ref}`"><div><strong>{{ labels[row.kind] }} · {{ row.label }}</strong><p class="hint">{{ row.state === 'missing' ? '待选择机器人账号' : '账号不在目录中' }} · {{ row.owner.source_protocol || '协议未记录' }} {{ row.owner.bot_id }} · {{ row.count }} 条</p></div><button :disabled="busy" @click="inspect(row)">查看与处理</button></li></ul>
      <section v-if="candidate" class="separated">
        <h3>{{ labels[candidate.kind] }} · {{ candidate.label }}</h3>
        <pre>{{ detail }}</pre>
        <template v-if="candidate.owner.source_protocol === 'onebot11' && !candidate.owner.bot_id">
          <form class="inline-form" @submit.prevent="run(() => loadBots())"><label>搜索机器人账号<input v-model="query" :disabled="busy"></label><button :disabled="busy">搜索</button></form>
          <label>归入机器人账号<select v-model="selected" :disabled="busy"><option value="">选择 OneBot11 账号</option><option v-for="bot in bots" :key="bot.bot_id" :value="bot.bot_id">{{ botLabel(bot) }} · {{ { online: '在线', offline: '离线', no_connection: '无连接' }[bot.state] }}</option></select></label>
          <div class="actions"><button v-if="cursor" :disabled="busy" @click="run(() => loadBots(true))">更多账号</button><button :disabled="busy || !selected || !detail" @click="change(true)">确认补齐归属</button></div>
        </template>
        <p class="hint">删除后无法恢复。定时任务将停止，本地保留的委托等待到期。</p>
        <div class="actions"><button :disabled="busy" @click="candidate = null">保留记录</button><button class="destructive" :disabled="busy || !detail" @click="change(false)">确认删除此记录</button></div>
      </section>
    </template>
  </details>
</template>

<style scoped>
.history-data section{display:grid;gap:16px}.history-data pre{max-height:360px;overflow:auto;white-space:pre-wrap;overflow-wrap:anywhere;padding:16px;border:1px solid var(--raylea-color-border);border-radius:8px}.history-data .archive-list li{overflow-wrap:anywhere}
</style>
