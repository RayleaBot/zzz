<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'

interface Source { id: string; name: string; state: 'missing' | 'downloading' | 'ready' | 'bundled' | 'on_demand'; commit?: string; files: number; bytes: number; downloaded_ms?: number; received?: number; error?: string }
const props = defineProps<{ invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T>; prefix: string }>()
const sources = ref<Source[]>([]), busy = ref(false), error = ref(''), removeID = ref('')
const downloading = computed(() => sources.value.some(source => source.state === 'downloading'))
let timer: ReturnType<typeof setTimeout> | undefined, disposed = false

function megabytes(bytes: number) { return (bytes / 1e6).toFixed(1) + ' MB' }
function describe(source: Source) {
  if (source.state === 'downloading') return `下载中，已接收 ${megabytes(source.received ?? 0)}`
  if (source.state === 'missing') return '未下载'
  if (source.state === 'on_demand') return `出图时按需下载 · 已缓存 ${source.files} 个文件（${megabytes(source.bytes)}）`
  const parts = [(source.state === 'bundled' ? '随插件安装包附带 ' : '') + `${source.files} 个文件`, megabytes(source.bytes)]
  if (source.commit) parts.push('版本 ' + source.commit.slice(0, 7))
  if (source.downloaded_ms) parts.push(new Date(source.downloaded_ms).toLocaleString())
  return parts.join(' · ')
}
function schedule() {
  clearTimeout(timer)
  if (!disposed && downloading.value) timer = setTimeout(() => void act('artwork.status'), 2000)
}
async function act(action: string, id = '') {
  busy.value = action !== 'artwork.status'
  error.value = ''
  try {
    sources.value = (await props.invoke<{ sources: Source[] }>(action, id ? { id } : {})).sources
    removeID.value = ''
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : '素材操作未完成。'
  } finally {
    busy.value = false
    schedule()
  }
}
void act('artwork.status')
onUnmounted(() => { disposed = true; clearTimeout(timer) })
</script>

<template>
  <section class="artwork">
    <div class="section-heading"><h2>上游图片素材</h2></div>
    <p>图片面板与图鉴使用这些素材。插件包已随附模板用到的上游图片；“更新”从上游公开仓库下载新版本到插件数据目录，下载的文件优先使用，删除后恢复使用随附的文件。图鉴库需下载后使用，缺少素材时图片使用简化样式。也可以由超级管理员发送“{{ prefix }}素材更新”。</p>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p>
    <ul class="artwork-list">
      <li v-for="source in sources" :key="source.id">
        <div>
          <strong>{{ source.name }}</strong>
          <span role="status">{{ describe(source) }}</span>
          <span v-if="source.error" class="feedback danger">上次下载失败：{{ source.error }}</span>
        </div>
        <div class="actions">
          <button v-if="source.state !== 'on_demand'" :disabled="busy || source.state === 'downloading'" @click="act('artwork.download', source.id)">{{ source.state === 'ready' || source.state === 'bundled' ? '更新' : '下载' }}</button>
          <button v-if="source.state === 'ready' || (source.state === 'on_demand' && source.files > 0)" :disabled="busy" @click="removeID = source.id">删除</button>
        </div>
        <div v-if="removeID === source.id" class="actions">
          <span>确认删除已下载的{{ source.name }}？</span>
          <button :disabled="busy" @click="act('artwork.delete', source.id)">确认删除</button>
          <button :disabled="busy" @click="removeID = ''">取消</button>
        </div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.artwork { display: grid; gap: 16px }
.artwork-list { padding: 0; list-style: none; display: grid; gap: 16px }
.artwork-list li { padding-block: 12px; border-bottom: 1px solid var(--raylea-color-border); display: flex; gap: 12px; flex-wrap: wrap; justify-content: space-between }
.artwork-list li > div:first-child { display: grid; gap: 6px; min-width: 0; overflow-wrap: anywhere }
</style>
