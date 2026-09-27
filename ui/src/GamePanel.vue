<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { usePluginHost } from '@rayleabot/plugin-ui'
import BusinessView, { type View } from './BusinessView.vue'
import PanelDetails, { type Panel } from './PanelDetails.vue'
import BuildCalculator from './BuildCalculator.vue'
import GachaHistory from './GachaHistory.vue'
import Reminders from './Reminders.vue'
import ChallengeReminders from './ChallengeReminders.vue'
import PanelHistory from './PanelHistory.vue'
import ResourceLibrary from './ResourceLibrary.vue'
import GroupSettings from './GroupSettings.vue'
import GameSignin from './GameSignin.vue'
import { exportArchiveFile, parseImport, type Archive, type GachaRecord } from './uigf'
import { driveSync, type SyncInfo } from './sync'
import GachaTasks from './GachaTasks.vue'
import MonthlyHistory from './MonthlyHistory.vue'
import MonthlyCollection from './MonthlyCollection.vue'
import CommunityPanel from './CommunityPanel.vue'
import CodesPanel from './CodesPanel.vue'
import PublicContent from './PublicContent.vue'
import HelpPanel from './HelpPanel.vue'
import ArtworkPanel from './ArtworkPanel.vue'
import GuidesPanel from './GuidesPanel.vue'

interface Operation { name: string; label: string; command: string; input: string }
interface Game { id: string; name: string; prefix: string; region: string; operations: Operation[] }
interface Role { ref: string; uid: string; nickname: string; region: string; level: number }
interface Account { cloud_configured?:string[]; ref: string; account_label: string; roles: Role[]; status: string; owner: { actor_id: string } }
interface CatalogEntry { id: string; name: string; kind: string; rarity: number; element?: string; weapon?: string }
interface SavedArchive { uid: string; region: string; timezone: number; lang: string; records: number; revision: number }
interface Transfer { ref: string; uid: string; region: string; timezone: number; lang: string; total: number }

const host = usePluginHost()
const page = host.client.pageId || 'overview'
const game = ref<Game | null>(null)
const bots = ref<{source_protocol:string;source_adapter:string;id:string}[]>([])
const catalogVersion = ref('')
const busy = ref(false), error = ref(''), notice = ref('')
const accountIssue = ref('')
const accounts = ref<Account[]>([])
const nextAccountPage = ref<number | null>(null)
const selection = ref(''), operationName = ref(''), period = ref(1), character = ref(''), queryMonth = ref('')
const publicUID = ref('')
const publicProfileUID=ref(''), publicProfileRegion=ref('')
const view = ref<View | null>(null)
const panels = ref<Panel[]>([]), scoreCharacter = ref('')
const search = ref(''), kind = ref(''), entries = ref<CatalogEntry[]>([])
const archives = ref<SavedArchive[]>([])
const historyArchive = ref<SavedArchive | null>(null), historyKey = ref(0)
const syncInfo = ref<SyncInfo | null>(null), fullSync = ref(false)
let syncController: AbortController | undefined
const imported = ref<Archive[]>([]), importIndex = ref(0), importRegion = ref('')
const importFile = ref<HTMLInputElement | null>(null)
const removeCandidate = ref<SavedArchive | null>(null), removeDialog = ref<HTMLDialogElement | null>(null)
const provider = ref('raylea.mihoyo-accounts'), imageReplies = ref(true), aliases = ref<Record<string, string>>({})
const aliasConflicts=ref<{name:string;target:string;builtin:string[]}[]>([])
const aliasName = ref(''), aliasTarget = ref('')
let disposed = false
const selectedOperation = computed(() => game.value?.operations.find(item => item.name === operationName.value))
const detailPanel = computed(() => panels.value.find(panel => panel.id === scoreCharacter.value))
const selectedRole = computed(() => roleOptions.value.find(item => item.key === selection.value))
watch(selectedOperation, () => { period.value = 1; queryMonth.value = ''; panels.value = [];view.value=null })
watch(selection, () => { panels.value = []; view.value = null })
watch([period,queryMonth],()=>{view.value=null})
const roleOptions = computed(() => accounts.value.flatMap(account => account.roles.map(role => ({ key: JSON.stringify([account.ref, role.ref]), label: `${role.nickname} · ${role.uid} · 用户 ${account.owner.actor_id}`, role, account }))))
const regions = [{ id: 'prod_gf_cn', name: '国服' }, { id: 'prod_gf_us', name: '美服（绝区零）' }, { id: 'prod_gf_eu', name: '欧服（绝区零）' }, { id: 'prod_gf_jp', name: '亚服（绝区零）' }, { id: 'prod_gf_sg', name: '港澳台服（绝区零）' }]
const selectedImport = computed(() => imported.value[importIndex.value])

async function invoke<T>(action: string, payload: Record<string, unknown> = {}): Promise<T> { return await host.client.invokeAction(action, payload) as unknown as T }
async function run(work: () => Promise<void>, success = '') {
  if (busy.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try { await work(); if (success) notice.value = success } catch (cause) { error.value = cause instanceof Error ? cause.message : '操作未完成，请重试。' }
  finally { busy.value = false }
}
async function loadAccounts(page = 0) {
  try {
    const result = await invoke<{ accounts: { items: Account[]; next_page: number | null } }>('accounts.list', { page })
    accounts.value = page === 0 ? result.accounts.items : [...accounts.value, ...result.accounts.items]
    nextAccountPage.value = result.accounts.next_page
    if (!roleOptions.value.some(item => item.key === selection.value)) selection.value = roleOptions.value[0]?.key ?? ''
    accountIssue.value = ''
  } catch (cause) { accountIssue.value = cause instanceof Error ? cause.message : '账号服务暂不可用。' }
}
async function loadCatalog() { const result = await invoke<{ entries: CatalogEntry[] }>('catalog.search', { query: search.value, kind: kind.value }); entries.value = result.entries }
async function loadArchives() { archives.value = (await invoke<{ items: SavedArchive[] }>('gacha.list')).items }
async function load() {
  const result = await invoke<{ game: Game; bots?:{source_protocol:string;source_adapter:string;id:string}[]; catalog_version: string; settings: { account_provider: string; image_replies: boolean; custom_aliases: Record<string, string> } }>('status')
  bots.value = result.bots ?? []
  game.value = result.game; catalogVersion.value = result.catalog_version
  operationName.value ||= game.value.operations[0]?.name ?? ''
  importRegion.value ||= game.value.region
  publicProfileRegion.value ||= game.value.region
  provider.value = result.settings.account_provider; imageReplies.value = result.settings.image_replies; aliases.value = result.settings.custom_aliases || {}
  if (page === 'help' || page === 'media' || page === 'content' || page === 'resources' || page === 'groups' || page === 'guides') return
  if (page === 'catalog') await loadCatalog()
  else if (page === 'gacha') await Promise.all([loadArchives(), loadAccounts()])
  else await loadAccounts()
}
void host.ready.then(() => run(load)).catch(cause => { error.value = cause instanceof Error ? cause.message : '页面连接失败。' })

function chosen() { const current = roleOptions.value.find(item => item.key === selection.value); if (!current) throw new Error('请先选择已授权的角色。'); return { account_ref: current.account.ref, role_ref: current.role.ref } }
async function queryPublicProfile(){await run(async()=>{panels.value=[];view.value=(await invoke<{view:View}>('public.query',{uid:publicProfileUID.value.trim(),region:publicProfileRegion.value})).view})}
async function query() {
  await run(async () => {
    view.value=null
    const input: Record<string, unknown> = {}
    if (selectedOperation.value?.input === 'period') input.schedule_type = period.value
    if (selectedOperation.value?.input === 'year_month' && queryMonth.value.trim()) input.month = queryMonth.value.trim()
    if (selectedOperation.value?.input === 'agents') {
      let id = character.value.trim()
      if (!/^\d+$/.test(id)) {
        const found = await invoke<{ entries: CatalogEntry[] }>('catalog.search', { query: id, kind: 'character' })
        const exact = found.entries.filter(entry => entry.name === id || entry.id === id)
        if (exact.length === 1) id = exact[0]!.id
        else if (found.entries.length === 1) id = found.entries[0]!.id
        else throw new Error('角色名未唯一匹配，请在图鉴中确认角色 ID。')
      }
      input.id_list = [id]
    }
    const result = await invoke<{ view: View; panels?: Panel[] }>('query', { ...chosen(), operation: operationName.value, input })
    view.value = result.view; panels.value = result.panels ?? []; scoreCharacter.value = panels.value[0]?.id ?? ''
  })
}
async function scorePanel() {
  await run(async () => {
    const result = await invoke<{ view: View; panels: Panel[] }>('panel.score', { ...chosen(), character_id: scoreCharacter.value })
    view.value = result.view; panels.value = result.panels
  })
}
async function setDefault() { await run(async () => { await invoke('accounts.select', chosen()) }, '默认角色已保存到账号服务') }
async function showcase() { await run(async () => { panels.value=[];view.value = (await invoke<{ view: View }>('showcase', { uid: publicUID.value.trim() })).view }) }
async function showEntry(entry: CatalogEntry) { await run(async () => { view.value = (await invoke<{ view: View }>('catalog.get', { id: entry.id })).view; aliasTarget.value = entry.id }) }
async function saveSettings(nextAliases:Record<string,string>=aliases.value) { await run(async () => { const checked=await invoke<{aliases:Record<string,string>;conflicts:typeof aliasConflicts.value}>('aliases.validate',{aliases:nextAliases});await host.client.saveSettings({ account_provider: provider.value.trim(), image_replies: imageReplies.value, custom_aliases: checked.aliases });aliases.value=checked.aliases;aliasConflicts.value=checked.conflicts }, '设置已保存') }
async function addAlias() { if (!aliasName.value.trim() || !aliasTarget.value) return; const name = aliasName.value.trim().toLowerCase(); if (Object.hasOwn(aliases.value, name)) { error.value = '此别名已存在，请先移除再重新添加。'; return }; await saveSettings({ ...aliases.value, [name]: aliasTarget.value }); if(!error.value)aliasName.value = '' }
async function removeAlias(name: string) { const copy = { ...aliases.value }; delete copy[name]; await saveSettings(copy) }

async function synchronize(resume = false) {
  await run(async () => {
    if (!resume || !syncInfo.value) syncInfo.value = (await invoke<{ sync: SyncInfo }>('gacha.sync.start', { ...chosen(), full: fullSync.value })).sync
    syncController = new AbortController()
    if (disposed) syncController.abort()
    const completed = await driveSync(invoke, syncInfo.value!, syncController.signal, value => { syncInfo.value = value })
    if (completed.state === 'completed' && completed.result) {
      notice.value = `同步完成，新增 ${completed.result.added} 条，档案共 ${completed.result.total} 条。`
      await loadArchives()
    } else notice.value = '同步已取消，原有档案保持不变。'
  })
  syncController = undefined
}
async function cancelSync() {
  if (syncController) { syncController.abort(); return }
  if (syncInfo.value?.state === 'running') await run(async () => {
    await invoke('gacha.sync.cancel', { ref: syncInfo.value!.ref })
    syncInfo.value = { ...syncInfo.value!, state: 'canceled' }
  }, '同步已取消，原有档案保持不变。')
}

async function readImport(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file || !game.value) return
  await run(async () => {
    if (file.size > 64 * 1024 * 1024) throw new Error('文件超过 64 MiB，请拆分后导入。')
    const parsed = parseImport(await file.text())
    if (!parsed.length) throw new Error('文件中没有本游戏的抽卡档案。')
    imported.value = parsed; importIndex.value = 0
  })
  if (importFile.value) importFile.value.value = ''
}
async function importArchive() {
  if (!selectedImport.value) return
  await run(async () => {
    const archive = selectedImport.value!
    const { transfer } = await invoke<{ transfer: Transfer }>('gacha.import.start', { uid: archive.uid, region: importRegion.value, timezone: archive.timezone, lang: archive.lang })
    try {
      for (let offset = 0; offset < archive.list.length; offset += 300) {
        if (disposed) throw new Error('导入已中止。')
        await invoke('gacha.import.append', { ref: transfer.ref, offset, records: archive.list.slice(offset, offset + 300) })
        notice.value = `正在导入 ${Math.min(offset + 300, archive.list.length)} / ${archive.list.length} 条记录…`
      }
      const result = await invoke<{ added: number; total: number }>('gacha.import.finish', { ref: transfer.ref })
      notice.value = `已导入 ${result.added} 条新记录，档案共 ${result.total} 条。`
      imported.value = []; await loadArchives()
    } catch (cause) { await invoke('gacha.import.cancel', { ref: transfer.ref }).catch(() => undefined); throw cause }
  })
}
async function exportArchive(archive: SavedArchive) {
  await run(async () => {
    const { transfer } = await invoke<{ transfer: Transfer }>('gacha.export.start', { uid: archive.uid, region: archive.region })
    const records: GachaRecord[] = []
    try {
      let more = true
      while (more) {
        if (disposed) throw new Error('导出已中止。')
        const chunk = await invoke<{ records: GachaRecord[]; more: boolean }>('gacha.export.read', { ref: transfer.ref, offset: records.length })
        if (chunk.more && !chunk.records.length) throw new Error('导出没有继续返回记录，请重试。')
        records.push(...chunk.records); more = chunk.more
      }
      const output = exportArchiveFile({ uid: transfer.uid, timezone: transfer.timezone, lang: transfer.lang, list: records })
      const url = URL.createObjectURL(new Blob([JSON.stringify(output.content, null, 2)], { type: 'application/json' }))
      const link = document.createElement('a'); link.href = url; link.download = `${game.value!.id}-${archive.uid}-${output.format}.json`; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000)
      notice.value = output.format === 'UIGF' ? 'UIGF 导出文件已生成。' : '完整档案已导出，包含新频段记录，可在此页面重新导入。'
    } finally { await invoke('gacha.export.close', { ref: transfer.ref }).catch(() => undefined) }
  })
}
async function showSummary(archive: SavedArchive) { await run(async () => { view.value = (await invoke<{ view: View }>('gacha.summary', { uid: archive.uid, region: archive.region })).view }); historyArchive.value = archive; historyKey.value++ }
async function removeArchive() {
  if (!removeCandidate.value) return
  await run(async () => { await invoke('gacha.remove', { uid: removeCandidate.value!.uid, region: removeCandidate.value!.region }); await loadArchives(); view.value = null; historyArchive.value = null; removeDialog.value?.close(); removeCandidate.value = null }, '抽卡档案已移除')
}
onUnmounted(() => { disposed = true; syncController?.abort() })
</script>

<template>
  <main :aria-busy="busy || host.loading.value">
    <div v-if="error" class="feedback danger" role="alert">{{ error }}</div>
    <p v-if="notice" class="feedback" role="status">{{ notice }}</p>
    <p v-if="!game" class="empty">正在读取插件资料… <button :disabled="busy" @click="run(load)">重新连接</button></p>
    <template v-else>
      <template v-if="page === 'overview'">
        <section aria-labelledby="query-heading"><div class="section-heading"><h2 id="query-heading">游戏查询</h2><button :disabled="busy" @click="run(() => loadAccounts())">刷新账号</button></div>
          <p v-if="accountIssue" class="feedback attention">{{ accountIssue }} 可在米游社账号管理页扫码或调整授权。</p>
          <p v-else-if="!roleOptions.length" class="empty">没有已授权角色。请先向机器人发送“扫码登录”，并授权{{ game.name }}。</p>
          <form v-else class="query-form" @submit.prevent="query">
            <label class="wide">游戏角色<select v-model="selection" :disabled="busy"><option v-for="item in roleOptions" :key="item.key" :value="item.key">{{ item.label }}</option></select></label>
            <label>查询内容<select v-model="operationName" :disabled="busy"><option v-for="item in game.operations" :key="item.name" :value="item.name">{{ item.label }}</option></select></label>
            <label v-if="selectedOperation?.input === 'period'">期数<select v-model="period" :disabled="busy"><option :value="1">本期</option><option :value="2">上期</option></select></label>
            <label v-if="selectedOperation?.input === 'year_month'">月份<input v-model="queryMonth" :disabled="busy" inputmode="numeric" placeholder="YYYYMM，留空查询默认月份"></label>
            <label v-if="selectedOperation?.input === 'agents'">角色名称或 ID<input v-model="character" :disabled="busy" required placeholder="输入角色全名或 ID"></label>
            <div class="actions wide"><button class="primary" type="submit" :disabled="busy">{{ busy ? '正在查询…' : '查询' }}</button><button type="button" :disabled="busy" @click="setDefault">设为此用户的默认角色</button><button v-if="nextAccountPage !== null" type="button" :disabled="busy" @click="run(() => loadAccounts(nextAccountPage!))">更多账号</button></div>
          </form>
        </section>
        <section class="separated" aria-labelledby="public-heading"><h2 id="public-heading">公开展柜</h2><p class="muted">读取游戏内角色展柜并保存为该 UID 的面板，无需绑定账号。</p><form class="inline-form" @submit.prevent="showcase"><label><span class="sr-only">公开 UID</span><input v-model="publicUID" required inputmode="numeric" pattern="[0-9]{6,12}" placeholder="输入游戏 UID" :disabled="busy"></label><button type="submit" :disabled="busy">读取展柜</button></form></section>
        <details class="settings"><summary>通过公共查询池查看基础资料</summary><p class="hint">仅使用账号所有者主动开启的公共查询用途。没有可用账号或额度时会明确提示，不会借用其他私人账号。</p><form class="inline-form" @submit.prevent="queryPublicProfile"><label>公开资料 UID<input v-model="publicProfileUID" inputmode="numeric" pattern="[0-9]{6,12}" required :disabled="busy"></label><label>公开资料区服<select v-model="publicProfileRegion" :disabled="busy"><option v-for="region in regions.filter(r=>r.id.endsWith('_cn'))" :key="region.id" :value="region.id">{{region.name}}</option></select></label><button type="submit" :disabled="busy">查询公开基础资料</button></form></details>
        <section v-if="panels.length" class="separated" aria-labelledby="score-heading"><h2 id="score-heading">装备评分</h2><p class="hint">按角色专属权重比较装备词条，权重随当前属性、影画与配装自动选择；评分不代表实际伤害。</p><form class="query-form" @submit.prevent="scorePanel"><label>评分角色<select v-model="scoreCharacter" :disabled="busy"><option v-for="panel in panels" :key="panel.id" :value="panel.id">{{ panel.name }}</option></select></label><div class="actions wide"><button :disabled="busy || !scoreCharacter" type="submit">计算装备评分</button></div></form></section>
        <BusinessView v-if="view" :value="view" />
        <MonthlyHistory v-if="operationName.endsWith('.monthly') && selectedRole" :key="selection" :account-ref="selectedRole.account.ref" :role-ref="selectedRole.role.ref" :invoke="invoke" />
        <PanelDetails v-if="detailPanel" :panel="detailPanel" />
        <BuildCalculator v-if="detailPanel && selectedRole" :key="`build:${selection}:${detailPanel.id}`" :character-id="detailPanel.id" :character-name="detailPanel.name" :account-ref="selectedRole.account.ref" :role-ref="selectedRole.role.ref" :invoke="invoke" />
        <details class="settings"><summary>查询设置</summary><form @submit.prevent="saveSettings()"><label>账号服务插件 ID<input v-model="provider" required :disabled="busy"></label><label class="check"><input v-model="imageReplies" type="checkbox" :disabled="busy">聊天查询优先发送图片</label><button :disabled="busy" type="submit">保存设置</button></form></details>
      </template>

      <template v-else-if="page === 'content'"><PublicContent :prefix="game.prefix" :invoke="invoke"/></template>
      <template v-else-if="page === 'help'"><HelpPanel :invoke="invoke"/></template>
      <template v-else-if="page === 'media'"><ArtworkPanel :prefix="game.prefix" :invoke="invoke"/></template>
      <template v-else-if="page === 'guides'"><GuidesPanel :invoke="invoke"/></template>
      <template v-else-if="page === 'assets'"><CodesPanel :roles="roleOptions" :invoke="invoke"/><button v-if="nextAccountPage!==null" :disabled="busy" @click="run(()=>loadAccounts(nextAccountPage!))">更多账号</button></template>
      <template v-else-if="page === 'community'"><CommunityPanel :accounts="accounts" :invoke="invoke"/><button v-if="nextAccountPage!==null" :disabled="busy" @click="run(()=>loadAccounts(nextAccountPage!))">更多社区账号</button></template>
      <template v-else-if="page === 'signin'">
        <p v-if="accountIssue" role="alert" class="feedback attention">{{accountIssue}}</p>
        <GameSignin :choices="roleOptions" :invoke="invoke" />
        <button v-if="nextAccountPage !== null" :disabled="busy" @click="run(() => loadAccounts(nextAccountPage!))">更多账号</button>
      </template>
      <template v-else-if="page === 'groups'">
        <GroupSettings :bots="bots" :invoke="invoke" />
      </template>
      <template v-else-if="page === 'resources'">
        <ResourceLibrary :invoke="invoke" />
      </template>
      <template v-else-if="page === 'history'">
        <p v-if="accountIssue" role="alert" class="feedback attention">{{ accountIssue }}</p>
        <PanelHistory :choices="roleOptions" :invoke="invoke" />
        <button v-if="nextAccountPage !== null" :disabled="busy" @click="run(() => loadAccounts(nextAccountPage!))">更多账号</button>
      </template>
      <template v-else-if="page === 'reminders'">
        <p v-if="accountIssue" role="alert" class="feedback attention">{{ accountIssue }}</p>
        <Reminders :choices="roleOptions" :invoke="invoke" />
        <ChallengeReminders :choices="roleOptions" :invoke="invoke" />
        <MonthlyCollection :choices="roleOptions" :invoke="invoke" />
        <button v-if="nextAccountPage !== null" :disabled="busy" @click="run(() => loadAccounts(nextAccountPage!))">更多账号</button>
      </template>
      <template v-else-if="page === 'catalog'">
        <div class="section-heading"><div><h2>资料图鉴</h2><p class="muted">本地资料 · {{ catalogVersion }}</p></div></div>
        <form class="catalog-search" @submit.prevent="run(loadCatalog)"><label><span class="sr-only">搜索角色、装备或 ID</span><input v-model="search" placeholder="搜索角色、装备或 ID" :disabled="busy"></label><label><span class="sr-only">资料类型</span><select v-model="kind" :disabled="busy"><option value="">全部类型</option><option value="character">角色</option><option value="weapon">装备</option><option value="buddy">邦布</option></select></label><button type="submit" :disabled="busy">搜索</button></form>
        <div class="catalog-layout"><section><ul class="catalog-list"><li v-for="entry in entries" :key="entry.id"><button type="button" :disabled="busy" @click="showEntry(entry)" :aria-label="`查看${entry.name}`"><strong>{{ entry.name }}</strong><span>{{ entry.element || entry.weapon || entry.id }}</span></button></li></ul><p v-if="!entries.length" class="empty">没有匹配资料，请尝试全名或 ID。</p><p v-if="entries.length === 50" class="hint">显示前 50 项，请使用关键词缩小范围。</p></section><BusinessView v-if="view" :value="view" /><p v-else class="empty">选择一项查看资料。</p></div>
        <details class="settings"><summary>自定义别名</summary><p v-if="aliasConflicts.length" class="feedback attention">以下别名覆盖了内置名称：{{aliasConflicts.map(v=>v.name).join('、')}}。</p><p class="hint">先选择资料，再为它添加常用称呼；自定义别名优先于内置名称。</p><form class="inline-form" @submit.prevent="addAlias"><label><span class="sr-only">别名</span><input v-model="aliasName" placeholder="输入别名" required :disabled="busy"></label><button :disabled="busy || !aliasTarget" type="submit">添加别名</button></form><ul class="alias-list"><li v-for="(id, name) in aliases" :key="name"><span>{{ name }} → {{ id }}</span><button type="button" :disabled="busy" :aria-label="`移除别名${name}`" @click="removeAlias(String(name))">移除</button></li></ul></details>
      </template>

      <template v-else-if="page === 'gacha'">
        <div class="section-heading"><div><h2>抽卡档案</h2><p class="muted">导入、分析与导出自己的抽卡记录。</p></div><button :disabled="busy" @click="run(loadArchives)">刷新列表</button></div>
        <section aria-labelledby="sync-heading" class="separated"><div class="section-heading"><h3 id="sync-heading">官方同步</h3><button :disabled="busy" @click="run(() => loadAccounts())">刷新账号</button></div>
          <p class="hint">复用已绑定账号的授权。只能获取官方仍保留的历史；读完全部频段后合并保存。</p>
          <p v-if="accountIssue" class="feedback attention">{{ accountIssue }}</p>
          <p v-else-if="!roleOptions.length" class="empty">请先扫码登录并授权此游戏，也可以在下方导入已有记录。</p>
          <form v-else class="query-form" @submit.prevent="synchronize()">
            <label class="wide">同步角色<select v-model="selection" :disabled="busy || syncInfo?.state === 'running'"><option v-for="item in roleOptions" :key="item.key" :value="item.key">{{ item.label }}</option></select></label>
            <label class="check wide"><input v-model="fullSync" type="checkbox" :disabled="busy || syncInfo?.state === 'running'">全量读取官方历史（默认只同步新增记录）</label>
            <div class="actions wide"><button v-if="syncInfo?.state !== 'running'" class="primary" type="submit" :disabled="busy">同步记录</button><button v-else-if="!busy" class="primary" type="button" @click="synchronize(true)">继续同步</button><button v-if="syncInfo?.state === 'running'" type="button" @click="cancelSync">取消同步</button><button v-if="nextAccountPage !== null" type="button" :disabled="busy" @click="run(() => loadAccounts(nextAccountPage!))">更多账号</button></div>
          </form>
          <p v-if="syncInfo" class="hint" role="status">{{ syncInfo.state === 'running' ? (busy ? '正在同步' : '同步已暂停，可继续或取消') : (syncInfo.state === 'completed' ? '已完成' : '已取消') }} · 已读取 {{ syncInfo.pages }} 页，{{ syncInfo.fetched }} 条记录</p>
          <p class="hint">上方即时同步需要保持页面打开；需要离开页面时，可使用下方后台同步。</p>
        </section>
        <GachaTasks :choices="roleOptions" :invoke="invoke" />
        <details class="import-panel"><summary>导入记录</summary><p class="hint">支持 <a href="https://uigf.org/en/standards/uigf.html" target="_blank" rel="noreferrer">UIGF</a> v4 和本插件导出的完整档案。不接收抽卡链接。</p><div class="import-fields"><label>选择 JSON 文件<input ref="importFile" type="file" accept=".json,application/json" :disabled="busy" @change="readImport"></label></div>
          <form v-if="imported.length" class="import-fields" @submit.prevent="importArchive"><label>文件中的账号<select v-model="importIndex" :disabled="busy"><option v-for="(archive, index) in imported" :key="`${archive.uid}:${index}`" :value="index">{{ archive.uid }} · {{ archive.list.length }} 条 · UTC{{ archive.timezone >= 0 ? '+' : '' }}{{ archive.timezone }}</option></select></label><label>游戏区服<select v-model="importRegion" :disabled="busy"><option v-for="region in regions" :key="region.id" :value="region.id">{{ region.name }}</option></select></label><button class="primary" type="submit" :disabled="busy">{{ busy ? '正在导入…' : '导入所选账号' }}</button></form>
        </details>
        <p v-if="!archives.length" class="empty">还没有抽卡档案。同步或导入后可查看统计并随时导出。</p>
        <ul v-else class="archive-list"><li v-for="archive in archives" :key="`${archive.uid}:${archive.region}`"><div><strong class="mono">{{ archive.uid }}</strong><p class="muted">{{ regions.find(region => region.id === archive.region)?.name || archive.region }} · {{ archive.records }} 条记录</p></div><div class="actions"><button :disabled="busy" @click="showSummary(archive)">查看统计</button><button :disabled="busy" @click="exportArchive(archive)">导出</button><button class="danger-text" :disabled="busy" @click="removeCandidate = archive; removeDialog?.showModal()">移除</button></div></li></ul>
        <BusinessView v-if="view" :value="view" />
        <GachaHistory v-if="historyArchive" :key="historyKey" :uid="historyArchive.uid" :region="historyArchive.region" :invoke="invoke" :disabled="busy" />
      </template>
    </template>
    <footer class="hint"><a href="./source.zip" download>本版本对应源码</a></footer>
    <dialog ref="removeDialog" aria-labelledby="remove-title" @cancel="busy && $event.preventDefault()"><h2 id="remove-title">移除抽卡档案 {{ removeCandidate?.uid }}</h2><p>将删除本插件保存的此区服记录，并取消对应的后台同步。建议先导出保留副本。</p><p v-if="error" role="alert" class="danger-text">{{ error }}</p><div class="actions"><button autofocus :disabled="busy" @click="removeDialog?.close()">取消</button><button class="destructive" :disabled="busy" @click="removeArchive">移除档案</button></div></dialog>
  </main>
</template>
