import { ref } from 'vue'
import type { View } from './BusinessView.vue'
export interface CloudJob {
  ref: string
  state: 'running' | 'completed' | 'failed' | 'canceled'
  result?:Record<string,unknown>
  view?: View
  message?: string
}
type Invoke = <T>(action: string, payload?: Record<string, unknown>) => Promise<T>

// Each controller owns one task. Late responses cannot replace a newer selection.
export function cloudTask(invoke: Invoke,namespace='cloud') {
  const job = ref<CloudJob | null>(null), busy = ref(false), error = ref('')
  let epoch = 0, disposed = false, timer: ReturnType<typeof setTimeout> | undefined, polling = false
  const release = (ref: string) => invoke(namespace+'.cancel', { ref }).catch(() => {})
  function clear() {
    epoch++; clearTimeout(timer); busy.value = false; error.value = ''
    if (job.value) void release(job.value.ref)
    job.value = null
  }
  async function poll() {
    if (disposed || polling || job.value?.state !== 'running') return
    clearTimeout(timer); polling = true; error.value = ''
    const ref = job.value.ref, current = epoch
    try {
      const result = await invoke<{ job: CloudJob }>(namespace+'.poll', { ref })
      if (disposed || current !== epoch || job.value?.ref !== ref) return
      job.value = result.job
    } catch (cause) {
      if (!disposed && current === epoch) error.value = cause instanceof Error ? cause.message : '云任务读取失败，请刷新进度。'
    } finally {
      polling = false
      if (!disposed && job.value?.state === 'running' && !error.value) timer = setTimeout(() => void poll(), 1000)
    }
  }
  async function start(payload: Record<string, unknown>,action=namespace+'.start') {
    if (disposed || busy.value || job.value?.state === 'running') return
    clear(); busy.value = true; const current = epoch
    try {
      const result = await invoke<{ job: CloudJob }>(action, payload)
      if (disposed || current !== epoch) { void release(result.job.ref); return }
      job.value = result.job
      if (result.job.state === 'running') timer = setTimeout(() => void poll(), 400)
    } catch (cause) {
      if (!disposed && current === epoch) error.value = cause instanceof Error ? cause.message : '云查询未能开始。'
    } finally { if (current === epoch) busy.value = false }
  }
  async function cancel() {
    if (!job.value || busy.value) return
    clearTimeout(timer); const ref = job.value.ref, current = ++epoch; busy.value = true; error.value = ''
    try {
      const result = await invoke<{ job: CloudJob }>(namespace+'.cancel', { ref })
      if (!disposed && current === epoch) job.value = result.job
    } catch (cause) {
      if (!disposed && current === epoch) error.value = cause instanceof Error ? cause.message : '取消请求未确认，请重试。'
    } finally { if (current === epoch) busy.value = false }
  }
  function dispose() { disposed = true; clear() }
  return { job, busy, error, start, poll, cancel, clear, dispose }
}
