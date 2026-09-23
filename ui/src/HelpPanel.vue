<script setup lang="ts">
import{ref}from'vue'
const props=defineProps<{invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T>}>()
interface Command{id:string;name:string;description:string;usage:string;permission:string;trigger:{names?:string[]|null}}
interface Info{plugin:{name:string;version:string;license:string;min_core_version:string};commands:Command[];catalog_version:string;resource_version:string}
const query=ref(''),info=ref<Info|null>(null),busy=ref(false),error=ref('')
async function load(){busy.value=true;error.value='';try{info.value=await props.invoke('help.query',{query:query.value.trim()})}catch(e){error.value=e instanceof Error?e.message:'帮助读取失败。'}finally{busy.value=false}}
void load()
</script>
<template><section><h2>帮助与版本</h2><p v-if="info">{{info.plugin.name}} {{info.plugin.version}} · {{info.plugin.license}} · 最低宿主 {{info.plugin.min_core_version}}</p><p class="hint" v-if="info">图鉴：{{info.catalog_version}}<br>材料与卡池：{{info.resource_version}}</p><p>命令示例中的 # 替换为机器人配置的前缀。插件更新在宿主的插件安装/更新入口进行；本地固定资料随插件版本更新，官方公开内容可在对应页面重新查询。</p><form class="inline-form" @submit.prevent="load"><label>筛选命令<input v-model="query" :disabled="busy" placeholder="功能名或关键词"></label><button :disabled="busy">查找帮助</button></form><p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="info&&!info.commands.length" class="empty">没有匹配指令。</p><dl v-if="info" class="help-commands"><div v-for="c in info.commands" :key="c.id"><dt>{{c.usage||c.name}}</dt><dd>{{c.description}}<span v-if="(c.trigger.names?.length??0)>1"> · 入口：{{c.trigger.names!.join('、')}}</span></dd></div></dl></section></template>
<style scoped>.help-commands{display:grid;gap:14px}.help-commands>div{display:grid;gap:6px;padding-block:12px;border-bottom:1px solid var(--raylea-color-border)}.help-commands dt{font-weight:600}.help-commands dd{margin:0;overflow-wrap:anywhere}label{display:grid;gap:8px}</style>
