<script setup lang="ts">
import { ref, watch } from 'vue'
interface Pool { estimated_start?: boolean; version: string; half: string; from: string; to: string; kind: string; characters5: string[]; characters4: string[]; weapons5: string[]; weapons4: string[] }
const props = defineProps<{ invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const version = ref(''), date = ref(''), pools = ref<Pool[]>([]), poolTotal = ref(0), sourceVersion = ref(''), busy = ref(false), error = ref('')
const bannerName=ref(''),bannerKind=ref('all'),poolOffset=ref(0),poolNext=ref<number|null>(null),appearances=ref<{name:string;kind:string;rarity:number;count:number;last_end:string;days_since_end:number|null}[]>([])
watch([version, date, bannerName, bannerKind], () => { pools.value = []; poolTotal.value = 0;poolOffset.value=0;poolNext.value=null;appearances.value=[] })
async function run(fn: () => Promise<void>) { if (busy.value) return; busy.value = true; error.value = ''; try { await fn() } catch (cause) { error.value = cause instanceof Error ? cause.message : '资料读取失败。' } finally { busy.value = false } }
async function loadPools(next=0) { const result = await props.invoke<{ pools: Pool[]; total: number; version: string;next_offset?:number|null;appearances:typeof appearances.value }>('banners.query', { version: version.value, date: date.value,query:bannerName.value,kind:bannerKind.value,offset:next }); poolOffset.value=next;poolNext.value=result.next_offset??null;appearances.value=result.appearances??[];pools.value = result.pools; poolTotal.value = result.total; sourceVersion.value = result.version }
void run(() => loadPools())
</script>
<template>
  <section class="resource-library">
    <h2>卡池历史</h2><p class="hint">资料来自固定参考快照 {{ sourceVersion }}，可离线查询，无需 CK。卡池和日期未作在线校准，不代表当前官方活动；日期沿用国服 UTC+8。</p>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p>
    <section><h3>版本卡池</h3><form @submit.prevent="run(()=>loadPools())"><fieldset :disabled="busy"><legend class="sr-only">卡池筛选</legend><label>游戏版本<input v-model="version" maxlength="32" placeholder="如 1.0；留空不限"></label><label>角色或装备名称<input v-model="bannerName" maxlength="128" type="search"></label><label>卡池类别<select v-model="bannerKind"><option value="all">全部</option><option value="character">角色</option><option value="weapon">装备</option></select></label><label>卡池日期<input v-model="date" type="date"></label><button type="submit">查询卡池</button></fieldset></form>
      <p class="hint">匹配 {{ poolTotal }} 期，当前页 {{ pools.length }} 期。</p>
      <details v-if="appearances.length"><summary>出现与复刻记录统计</summary><p class="hint">按收录记录统计次数；距离最近收录结束的天数，不证明此后没有复刻。</p><ul class="material-list"><li v-for="item in appearances" :key="item.kind+item.name"><strong>{{item.name}}</strong> · 收录 {{item.count}} 次 · 最近结束 {{item.last_end}}<span v-if="item.days_since_end!==null"> · 距今 {{item.days_since_end}} 天</span></li></ul></details>
      <ul class="pool-list"><li v-for="(item, index) in pools" :key="`${item.version}:${item.from}:${index}`"><h4>{{ item.version }} · {{ item.half }}</h4><p class="hint">{{ item.from || '未注明开始时间' }} — {{ item.to || '未注明结束时间' }}</p><p v-if="item.estimated_start" class="hint">开始时间按固定参考规则估算。</p><p v-if="item.characters5.length">S 级角色：{{ item.characters5.join('、') }}</p><p v-if="item.characters4.length">A 级角色：{{ item.characters4.join('、') }}</p><p v-if="item.weapons5.length">S 级装备：{{ item.weapons5.join('、') }}</p><p v-if="item.weapons4.length">A 级装备：{{ item.weapons4.join('、') }}</p></li></ul>
      <div class="actions"><button :disabled="busy||poolOffset===0" @click="run(()=>loadPools(Math.max(0,poolOffset-50)))">上一页卡池</button><button :disabled="busy||poolNext===null" @click="run(()=>loadPools(poolNext!))">下一页卡池</button></div>
    </section>
  </section>
</template>
<style scoped>
.resource-library{display:grid;gap:22px}.resource-library>section{display:grid;gap:16px}.resource-library fieldset{display:flex;gap:12px;align-items:end;flex-wrap:wrap;border:0;padding:0;min-width:0}.resource-library label{display:grid;gap:8px;flex:1 1 180px;min-width:0}.material-list,.pool-list{list-style:none;padding:0;display:grid;gap:12px}.material-list li,.pool-list li{padding:14px 0;border-top:1px solid var(--raylea-color-border,#ddd);overflow-wrap:anywhere}.material-list p,.pool-list p{margin-top:8px}.sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%)}
</style>
