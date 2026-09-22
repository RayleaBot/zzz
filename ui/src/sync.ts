export interface SyncInfo {
  ref: string; state: 'running' | 'completed' | 'canceled'; sequence: number
  pool: string; pages: number; fetched: number
  result?: { uid: string; region: string; added: number; total: number }
}
export type SyncInvoke = <T>(action: string, payload: Record<string, unknown>) => Promise<T>

function wait(signal: AbortSignal): Promise<void> {
  return new Promise(resolve => {
    const finish = () => { clearTimeout(timer); signal.removeEventListener('abort', finish); resolve() }
    const timer = setTimeout(finish, 1000)
    signal.addEventListener('abort', finish, { once: true })
    if (signal.aborted) finish()
  })
}

// Each page is a fresh management request. On transient failure the caller
// retains the last acknowledged sequence and can safely resume the same task.
export async function driveSync(invoke: SyncInvoke, initial: SyncInfo, signal: AbortSignal, progress: (value: SyncInfo) => void, pause = wait): Promise<SyncInfo> {
  let current = initial
  while (current.state === 'running') {
    if (signal.aborted) {
      await invoke('gacha.sync.cancel', { ref: current.ref })
      current = { ...current, state: 'canceled' }; progress(current); break
    }
    current = (await invoke<{ sync: SyncInfo }>('gacha.sync.step', { ref: current.ref, sequence: current.sequence })).sync
    progress(current)
    if (current.state === 'running') await pause(signal)
  }
  return current
}
