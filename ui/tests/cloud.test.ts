import { afterEach, expect, it, vi } from 'vitest'
import { cloudTask } from '../src/cloud'
afterEach(() => vi.useRealTimers())
function deferred<T>() { let resolve!: (value:T)=>void; const promise=new Promise<T>(r=>{resolve=r}); return {promise,resolve} }
it('releases tasks that arrive after page disposal',async()=>{
 const pending=deferred<any>(),calls:string[]=[]
 const task=cloudTask(async(action,payload)=>{calls.push(action+':'+(payload?.ref??''));return action==='cloud.start'?pending.promise:{} as any})
 const request=task.start({consent:true});task.dispose();pending.resolve({job:{ref:'late',state:'running'}});await request
 expect(task.job.value).toBeNull();expect(calls).toContain('cloud.cancel:late')
})
it('ignores a late poll after cancel and preserves the canceled result',async()=>{
 vi.useFakeTimers();const pending=deferred<any>()
 const task=cloudTask(async(action)=>action==='cloud.poll'?pending.promise:{job:{ref:'j',state:action==='cloud.cancel'?'canceled':'running'}} as any)
 await task.start({consent:true});const polling=task.poll();await task.cancel();pending.resolve({job:{ref:'j',state:'completed',view:{title:'stale'}}});await polling
 expect(task.job.value?.state).toBe('canceled');expect(task.job.value?.view).toBeUndefined();task.dispose()
})
it('allows a failed poll to be retried without creating another upstream request',async()=>{
 vi.useFakeTimers();let reads=0,starts=0
 const task=cloudTask(async(action)=>{if(action==='cloud.start')starts++;if(action==='cloud.poll'&&reads++===0)throw Error('temporary');return {job:{ref:'j',state:action==='cloud.poll'?'completed':'running'}} as any})
 await task.start({consent:true});await task.poll();expect(task.error.value).toBe('temporary');await task.poll()
 expect(task.job.value?.state).toBe('completed');expect(starts).toBe(1);expect(task.error.value).toBe('');task.dispose()
})
