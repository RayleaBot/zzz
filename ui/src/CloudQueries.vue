<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import BusinessView from './BusinessView.vue'
import CloudOCRComparison from './CloudOCRComparison.vue'
import CloudExchange from './CloudExchange.vue'
import { cloudTask } from './cloud'
const props = defineProps<{ game:string; prefix: string; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const main = cloudTask(props.invoke), detail = cloudTask(props.invoke)
const { job, busy, error } = main
interface FilterField {key:string;label:string;kind:string;options?:string[]}
const filterError=ref('')
const authenticated=ref(false),filterFields=ref<FilterField[]>([]),advanced=ref<{field:string;op:string;value:string|number}[]>([])
function fieldSchema(key:string){return filterFields.value.find(v=>v.key===key)}
async function loadFilterSchema(){filterError.value='';try{filterFields.value=(await props.invoke<{fields:FilterField[]}>('cloud.filter_schema')).fields??[]}catch(e){filterError.value=e instanceof Error?e.message:'高级筛选资料未能读取。'}}
void loadFilterSchema()
watch(advanced,()=>{main.clear();detail.clear()},{deep:true})
watch(authenticated,value=>{if(!value)advanced.value=[];main.clear();detail.clear()})
const mode = ref('usage'), uid = ref(''), character = ref(''), consent = ref(false)
const imageURL=ref(''),ocrSlot=ref(1),forge=ref(true)
const uidList=ref(''),characterIDs=ref(''),rankQuery=ref('dmg'),refreshRank=ref(false),cloudVersion=ref('')
const sort = ref('dmg_avg'), limit = ref(20), op = ref(''), cons = ref(0), op2 = ref(''), cons2 = ref(6), selected = ref('')
const operators = [{value:'=',label:'等于'},{value:'>=',label:'大于等于'},{value:'<=',label:'小于等于'},{value:'>',label:'大于'},{value:'<',label:'小于'},{value:'!=',label:'不等于'}]
const pending = computed(() => busy.value || job.value?.state === 'running')
const detailPending = computed(() => detail.busy.value || detail.job.value?.state === 'running')
const panelJob = computed(() => mode.value === 'custom' ? detail.job.value : job.value)
const panels = computed(() => panelJob.value?.panels ?? [])
const panel = computed(() => panels.value.find(p => p.id === selected.value))
watch([mode, uid, character, sort, limit, op, cons, op2, cons2,uidList,characterIDs,rankQuery,refreshRank,cloudVersion,imageURL,ocrSlot,forge], () => { main.clear(); detail.clear(); selected.value = '' })
watch(mode,value=>{if(value==='akasha_stygian')authenticated.value=false;if(value==='group_rank'&&rankQuery.value==='all')rankQuery.value='dmg'})
watch(panels, list => { selected.value = list[0]?.id ?? '' })
watch(consent, accepted => { if (!accepted) { main.clear(); detail.clear() } })
async function start() {
  if (!consent.value) return
  detail.clear()
  const filters = []
  if (op.value) filters.push({ op: op.value, value: cons.value })
  if (op2.value) filters.push({ op: op2.value, value: cons2.value })
  await main.start({ mode: mode.value, uid: uid.value.trim(), character_id: character.value.trim(), consent: true, sort: sort.value, limit: limit.value, filters, image_url:imageURL.value.trim(),slot:Number(ocrSlot.value),forge:forge.value,uids:uidList.value.split(/[\s,，]+/).filter(Boolean),character_ids:characterIDs.value.split(/[\s,，]+/).filter(Boolean),query:rankQuery.value,refresh:refreshRank.value,version:cloudVersion.value.trim(),authenticated:authenticated.value,advanced_filters:advanced.value.map(f=>({...f,value:fieldSchema(f.field)?.kind==='number'||fieldSchema(f.field)?.kind==='choice'?Number(f.value):f.value})) })
}
async function readPanel(index: number) {
  if (!consent.value || !job.value || detailPending.value) return
  await detail.start({ mode: 'rank_panel', ref: job.value.ref, index, consent: true,authenticated:authenticated.value })
}
onUnmounted(() => { main.dispose(); detail.dispose() })
</script>
<template>
  <section class="cloud-queries">
    <h2>云排名与面板</h2>
    <p class="hint">由第三方 ark.ivny.cn 提供，可使用匿名额度或本游戏已配置的授权额度。排名按服务收录与算法计算，云面板是服务保存的历史资料。</p>
    <p class="hint">授权查询使用账号插件管理页“ark 授权令牌”中配置的令牌，由账号插件代发请求，本插件不保存令牌。</p>
    <details><summary>验证跨机器人 UID 关联</summary><p class="hint">在机器人本人私聊发送“{{ prefix }}云验证 获取 UID 确认”，随后发送“{{ prefix }}云验证 进度”取回签名验证码。将验证码设为游戏签名，生效后发送“{{ prefix }}云验证 提交 UID 确认”。完成云验证后，可私聊发送“{{prefix}}云面板 获取 UID 确认”读取长期云面板，再用“{{prefix}}云面板 进度/保存”查看或保存。提交、状态与长期面板请求会把你的 QQ 与 UID 发送给 ark；仅支持 OneBot11，本页不能代填他人 QQ。</p></details>
    <form @submit.prevent="start">
      <fieldset :disabled="pending || detailPending"><legend class="sr-only">云查询参数</legend>
        <label class="check"><input v-model="authenticated" type="checkbox" :disabled="mode==='akasha_stygian'">使用已配置的 ark 授权令牌</label>
        <label>云查询内容<select v-model="mode"><option value="usage">查询额度</option><option value="rank">玩家角色排名</option><option value="distribution">角色总体分布</option><option value="custom">自定义角色排名</option><option value="panel">接收云端面板</option><option value="ocr">装备截图识别与比较</option><option value="self_rank">同 UID 多角色排名</option><option value="group_rank">同角色多 UID 排名</option><option v-if="prefix==='原神'" value="stygian">幽境排名 · ark</option><option v-if="prefix==='原神'" value="akasha_stygian">历史幽境排名 · Akasha</option></select></label>
        <label v-if="['rank','panel','self_rank'].includes(mode)">发送给云服务的公开 UID<input v-model="uid" inputmode="numeric" pattern="[0-9]{6,12}" required></label>
        <p v-if="mode === 'panel'" class="hint">请先在原机器人向 ark 上传面板，再于十分钟有效期内接收。这里只查看角色资料，不覆盖本地数据。</p>
        <label v-if="['rank', 'distribution', 'custom','group_rank'].includes(mode)">角色名称或 ID<input v-model="character" required></label>
        <label v-if="['group_rank','stygian','akasha_stygian'].includes(mode)">公开 UID 列表<textarea v-model="uidList" rows="3" maxlength="700" placeholder="逗号或空格分隔，最多 50 个" required></textarea></label>
        <template v-if="mode==='ocr'"><label>装备图片 HTTPS 地址<input v-model="imageURL" type="url" maxlength="4096" placeholder="https://…" required></label><label>截图装备部位<select v-model="ocrSlot"><option v-for="slot in (prefix==='原神'?5:6)" :key="slot" :value="slot">部位 {{slot}}</option></select></label><label class="check"><input v-model="forge" type="checkbox">截图包含重塑 / 重投前后两件装备</label><p class="hint">图片链接将发送给 ark OCR；请使用不含账号凭据的游戏装备截图。识别后可自行选择角色作模拟对比。</p></template>
        <label v-if="mode==='self_rank'">角色 ID 列表<textarea v-model="characterIDs" rows="3" maxlength="3000" placeholder="逗号或空格分隔，最多 250 个" required></textarea></label>
        <label v-if="mode==='akasha_stygian'">幽境游戏版本<input v-model="cloudVersion" pattern="[0-9]{1,2}[._][0-9]{1,2}" placeholder="例如 6.0" required></label>
        <p v-if="mode==='akasha_stygian'" class="hint">UID 列表与版本将发送至 akasha.cv，不发送 ark 令牌。与 ark 分开展示服务结果。</p>
        <label v-if="['rank','group_rank'].includes(mode)">排名类别<select v-model="rankQuery"><option value="dmg">伤害</option><option value="mark">装备评分</option><option v-if="mode==='rank'" value="all">两类分别展示</option></select></label>
        <label v-if="mode==='rank'" class="check"><input v-model="refreshRank" type="checkbox">请求 ark 刷新此角色的收录排名</label>
        <template v-if="mode === 'custom'">
          <div class="query-grid"><label>排名依据<select v-model="sort"><option value="dmg_avg">伤害降序</option><option value="mark_score">装备评分降序</option></select></label><label>显示数量<input v-model.number="limit" type="number" min="1" max="50" required></label></div>
          <p class="hint">以下命座 / 星魂条件同时生效。启用授权令牌后可添加高级筛选。</p>
          <div class="query-grid"><label>命座 / 星魂条件<select v-model="op" aria-label="命座 / 星魂条件"><option value="">不限</option><option v-for="o in operators" :key="o.value" :value="o.value">{{ o.label }}</option></select></label><label v-if="op">命座 / 星魂数<input v-model.number="cons" type="number" min="0" max="6" required></label></div>
          <div class="query-grid"><label>第二项命座 / 星魂条件<select v-model="op2" aria-label="第二项命座 / 星魂条件"><option value="">不限</option><option v-for="o in operators" :key="o.value" :value="o.value">{{ o.label }}</option></select></label><label v-if="op2">第二项命座 / 星魂数<input v-model.number="cons2" type="number" min="0" max="6" required></label></div>
          <p v-if="authenticated&&filterError" role="alert">{{filterError}} <button type="button" @click="loadFilterSchema">重试读取筛选资料</button></p>
          <fieldset v-if="authenticated" class="advanced-filters"><legend>高级筛选</legend><div v-for="(f,i) in advanced" :key="i" class="query-grid"><label>第 {{i+1}} 项筛选字段<select v-model="f.field" @change="f.value=fieldSchema(f.field)?.kind==='choice'?1:''"><option v-for="field in filterFields" :key="field.key" :value="field.key">{{field.label}}</option></select></label><label>第 {{i+1}} 项比较方式<select v-model="f.op"><option v-for="o in operators" :key="o.value" :value="o.value">{{o.label}}</option></select></label><label>第 {{i+1}} 项筛选值<select v-if="fieldSchema(f.field)?.kind==='choice'" v-model="f.value"><option v-for="(name,n) in fieldSchema(f.field)?.options" :key="n" :value="n+1">{{name}}</option></select><input v-else v-model="f.value" :type="fieldSchema(f.field)?.kind==='number'?'number':'text'" min="0" max="100000000000000" step="any" maxlength="64" required></label><button type="button" @click="advanced.splice(i,1)">移除第 {{i+1}} 项</button></div><button type="button" :disabled="advanced.length>=16||!filterFields.length" @click="advanced.push({field:filterFields[0]!.key,op:'>=',value:0})">添加高级筛选</button></fieldset>
        </template>
        <label class="check"><input v-model="consent" type="checkbox">同意将所填 UID、角色、筛选条件及选择的图片链接发送给所选第三方服务（ark.ivny.cn 或 akasha.cv），并按选择读取公开面板；不发送 CK 或 QQ</label>
        <button class="primary" type="submit" :disabled="!consent">开始云查询</button>
      </fieldset>
    </form>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p>
    <div v-if="job?.state === 'running'" role="status" class="feedback">云服务正在查询，最长等待 25 秒。<div class="actions"><button :disabled="busy" @click="main.poll">刷新云进度</button><button :disabled="busy" @click="main.cancel">取消云查询</button></div></div>
    <p v-if="job?.state === 'failed'" role="alert" class="feedback danger">{{ job.message }}</p><p v-if="job?.state === 'canceled'" role="status">云查询已取消。</p>
    <BusinessView v-if="job?.state === 'completed' && job.view" :value="job.view" />
    <CloudOCRComparison v-if="job?.state==='completed' && job.ocr" :key="job.ref" :job-ref="job.ref" :slot="job.ocr.slot" :needs-identity="job.ocr.needs_identity" :invoke="invoke" />
    <ol v-if="job?.ranking?.rows.length" class="cloud-ranking">
      <li v-for="row in job.ranking.rows" :key="row.index">
        <div class="rank-heading"><strong>{{ row.index + 1 }}. UID {{ row.uid || '未提供' }}</strong><span>{{ sort === 'mark_score' ? '装备评分' : '伤害评分' }} {{ (sort === 'mark_score' ? row.score : row.damage) || '未提供' }}</span></div>
        <p class="muted">等级 {{ row.level || '未提供' }} · 命座 / 星魂 {{ row.cons || '未提供' }} · {{ row.weapon || '未提供装备' }}</p>
        <details><summary>查看榜单配装详情</summary><BusinessView :value="row.view" /></details>
        <button :disabled="!row.can_read_panel || !consent || detailPending" @click="readPanel(row.index)">读取第 {{ row.index + 1 }} 名云面板</button>
      </li>
    </ol>
    <section v-if="detail.job.value || detail.error.value" class="cloud-detail">
      <h3>榜单公开面板</h3>
      <p v-if="detail.error.value" role="alert" class="feedback danger">{{ detail.error.value }}</p>
      <div v-if="detail.job.value?.state === 'running'" role="status">正在读取所选面板。<div class="actions"><button :disabled="detail.busy.value" @click="detail.poll">刷新面板进度</button><button :disabled="detail.busy.value" @click="detail.cancel">取消面板读取</button></div></div>
      <p v-if="detail.job.value?.state === 'failed'" role="alert">{{ detail.job.value.message }}</p>
      <p v-if="detail.job.value?.state === 'canceled'" role="status">面板读取已取消。</p>
      <BusinessView v-if="detail.job.value?.state === 'completed' && detail.job.value.view" :value="detail.job.value.view" />
    </section>
    <template v-if="panels.length"><label>云面板角色<select v-model="selected"><option v-for="p in panels" :key="p.id" :value="p.id">{{ p.name }} · {{ p.id }}</option></select></label><BusinessView v-if="panel" :value="panel.view" /></template>
    <p v-else-if="panelJob?.state === 'completed' && panelJob.panels" role="status">服务尚未保存此 UID 的角色面板。</p>
    <CloudExchange :game="game" :authenticated="authenticated" :invoke="invoke" />
  </section>
</template>
<style scoped>
.cloud-queries{display:grid;gap:20px}.cloud-queries fieldset{display:grid;gap:16px;border:0;padding:0;min-width:0}.cloud-queries label:not(.check){display:grid;gap:8px;min-width:0}.cloud-queries select{width:100%;min-width:0}.query-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.cloud-ranking{list-style:none;padding:0;margin:0}.cloud-ranking>li{display:grid;gap:12px;padding:20px 0;border-bottom:1px solid var(--raylea-color-border)}.rank-heading{display:flex;flex-wrap:wrap;gap:8px 24px;justify-content:space-between}.rank-heading span{font-variant-numeric:tabular-nums}.cloud-ranking button{justify-self:start}.cloud-ranking summary{cursor:pointer}.cloud-detail{display:grid;gap:16px}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}@media(max-width:480px){.query-grid{grid-template-columns:1fr}}
</style>
