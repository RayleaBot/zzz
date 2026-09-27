<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
const props = defineProps<{ choices: { key: string; label: string; account: { ref: string }; role: { ref: string } }[]; invoke: <T>(action: string, payload?: Record<string, unknown>) => Promise<T> }>()
const selected = ref(''), full = ref(false), notify = ref(false)
const busy = ref(false), error = ref(''), notice = ref('')
let disposed = false
// The action answers once the sync has started; the sync itself goes on in
// the plugin after this page closes.
async function start() {
  const choice = props.choices.find(c => c.key === selected.value)
  if (busy.value || !choice) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await props.invoke('gacha.sync.background', { account_ref: choice.account.ref, role_ref: choice.role.ref, full: full.value, notify: notify.value, confirm: true })
    if (!disposed) notice.value = notify.value ? '后台同步已开始，关闭页面仍会继续；结束后私聊通知账号所属用户。' : '后台同步已开始，关闭页面仍会继续；读完全部卡池后合并到档案。'
  } catch (cause) { if (!disposed) error.value = cause instanceof Error ? cause.message : '后台同步未能开始。' }
  finally { if (!disposed) busy.value = false }
}
onUnmounted(() => { disposed = true })
</script>
<template>
  <section class="gacha-background separated">
    <div class="section-heading"><h3>后台同步</h3></div>
    <p class="hint">关闭页面后仍会继续，两页之间至少间隔一秒；全部卡池读取成功后才合并，失败时原档案保留。插件重启会中断未完成的同步。</p>
    <form @submit.prevent="start"><fieldset :disabled="busy"><legend class="sr-only">后台同步设置</legend>
      <label>后台同步角色<select v-model="selected" required><option value="" disabled>选择已授权角色</option><option v-for="c in choices" :key="c.key" :value="c.key">{{ c.label }}</option></select></label>
      <label class="check"><input v-model="full" type="checkbox">全量读取官方仍保留的历史（默认只读新增记录）</label>
      <label class="check"><input v-model="notify" type="checkbox">结束后私聊账号所属用户，通知结果</label>
      <button class="primary" type="submit" :disabled="!selected">开始后台同步</button>
    </fieldset></form>
    <p v-if="error" role="alert" class="feedback danger">{{ error }}</p><p v-if="notice" role="status" class="feedback">{{ notice }}</p>
  </section>
</template>
<style scoped>
.gacha-background{display:grid;gap:20px}.gacha-background .section-heading{margin-bottom:0}.gacha-background fieldset{display:grid;gap:16px;justify-items:start;padding:0;border:0;min-width:0;max-width:580px}.gacha-background label:not(.check){display:grid;gap:8px;width:100%;min-width:0}
</style>
