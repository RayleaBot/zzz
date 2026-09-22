<script setup lang="ts">
import { ref } from 'vue'
import ChallengeRank from './ChallengeRank.vue'
interface Scope { source_protocol: string; source_adapter: string; bot_id: string; group_id: string }
interface Group { scope: Scope; revision: number; config: { enabled?: boolean; image_replies?: boolean; aliases?: Record<string,string> }; rank_entries: number; challenge_entries?: number }
const props=defineProps<{ bots:{source_protocol:string;source_adapter:string;id:string}[];invoke:<T>(action:string,payload?:Record<string,unknown>)=>Promise<T> }>()
const rankScope=ref<Scope|null>(null),board=ref(true)
const groups=ref<Group[]>([]),page=ref(0),next=ref<number|null>(null),busy=ref(false),error=ref(''),notice=ref(''),editing=ref<Group|null>(null),bot=ref(''),groupID=ref(''),enabled=ref('default'),images=ref('default'),aliases=ref<{name:string;id:string}[]>([]),clearing=ref<Group|null>(null)
function edit(item:Group){editing.value=item;enabled.value=item.config.enabled===undefined?'default':String(item.config.enabled);images.value=item.config.image_replies===undefined?'default':String(item.config.image_replies);aliases.value=Object.entries(item.config.aliases??{}).map(([name,id])=>({name,id}));clearing.value=null}
async function load(){const result=await props.invoke<{items:Group[];next_page:number|null;challenge_board:boolean}>('groups.list',{page:page.value});groups.value=result.items;next.value=result.next_page;board.value=result.challenge_board}
async function run(fn:()=>Promise<void>){if(busy.value)return;busy.value=true;error.value='';notice.value='';try{await fn()}catch(cause){error.value=cause instanceof Error?cause.message:'群设置操作未完成。'}finally{busy.value=false}}
function create(){const chosen=props.bots.find(b=>JSON.stringify([b.source_protocol,b.source_adapter,b.id])===bot.value);if(!chosen||!groupID.value.trim())return;edit({scope:{source_protocol:chosen.source_protocol,source_adapter:chosen.source_adapter,bot_id:chosen.id,group_id:groupID.value.trim()},revision:0,config:{},rank_entries:0})}
async function save(){if(!editing.value)return;await run(async()=>{const map:Record<string,string>={};for(const row of aliases.value){const key=row.name.trim().toLowerCase();if(!key||!row.id.trim()||Object.hasOwn(map,key))throw new Error('别名与资料 ID 不可为空，别名不能重复。');map[key]=row.id.trim()};const config={aliases:map,...(enabled.value==='default'?{}:{enabled:enabled.value==='true'}),...(images.value==='default'?{}:{image_replies:images.value==='true'})};await props.invoke('groups.set',{scope:editing.value!.scope,revision:editing.value!.revision,config});editing.value=null;await load();notice.value='群内游戏设置已保存。'})}
async function clear(){if(!clearing.value)return;await run(async()=>{await props.invoke('groups.clear',{scope:clearing.value!.scope,revision:clearing.value!.revision});clearing.value=null;editing.value=null;await load();notice.value='已恢复默认设置并清空此群的本地参评记录。'})}
async function turn(value:number){page.value=value;await run(load)}
void run(load)
</script>
<template>
 <section class="group-settings">
  <h2>群内设置与本地排行</h2><p class="hint">群内设置优先于本游戏全局设置，按机器人和群分别保存。排行只收录用户主动提交的装备评分与挑战成绩；云端排名不在此管理。</p>
  <p v-if="error" role="alert" class="feedback danger">{{error}}</p><p v-if="notice" role="status" class="feedback">{{notice}}</p>
  <form @submit.prevent="create"><fieldset :disabled="busy"><legend>添加群设置</legend><label>群所属机器人<select v-model="bot" required><option value="" disabled>选择机器人</option><option v-for="b in bots" :key="JSON.stringify(b)" :value="JSON.stringify([b.source_protocol,b.source_adapter,b.id])">{{b.source_adapter}} · {{b.id}}（{{b.source_protocol}}）</option></select></label><label>群 ID<input v-model="groupID" maxlength="256" required></label><button type="submit">编辑此群</button></fieldset></form>
  <form v-if="editing" @submit.prevent="save"><fieldset :disabled="busy"><legend>群 {{editing.scope.group_id}} · {{editing.scope.source_adapter}}</legend><label>本群游戏功能<select v-model="enabled"><option value="default">默认开启</option><option value="true">开启</option><option value="false">关闭</option></select></label><label>本群回复方式<select v-model="images"><option value="default">跟随全局</option><option value="true">图片优先</option><option value="false">文本</option></select></label>
   <div v-for="(row,i) in aliases" :key="i" class="alias-row"><label>群别名<input v-model="row.name" maxlength="64" required></label><label>资料 ID<input v-model="row.id" required></label><button type="button" @click="aliases.splice(i,1)">移除此别名</button></div>
   <div class="actions"><button type="button" :disabled="aliases.length>=64" @click="aliases.push({name:'',id:''})">添加群别名</button><button class="primary" type="submit">保存群设置</button><button type="button" @click="editing=null">取消编辑</button></div>
  </fieldset></form>
  <div class="section-heading"><h3>已保存的群</h3><button :disabled="busy" @click="run(load)">刷新群设置</button></div><p v-if="!groups.length" class="empty">当前页没有群设置。</p>
  <ul class="group-list"><li v-for="item in groups" :key="JSON.stringify(item.scope)"><strong>{{item.scope.group_id}} · {{item.scope.source_adapter}} · {{item.scope.bot_id}}</strong><p class="hint">{{item.config.enabled===false?'游戏功能关闭':'游戏功能开启'}} · {{item.rank_entries}} 条面板排名 · <template v-if="board">{{item.challenge_entries??0}} 条挑战成绩 · </template>{{Object.keys(item.config.aliases??{}).length}} 个群别名</p><div class="actions"><button :disabled="busy" @click="edit(item)">编辑群设置</button><button v-if="board" :disabled="busy" @click="rankScope=item.scope">查看挑战榜</button><button :disabled="busy" @click="clearing=item">恢复默认并清空排行</button></div></li></ul>
  <ChallengeRank v-if="rankScope" :key="JSON.stringify(rankScope)" :scope="rankScope" :invoke="invoke" />
  <section v-if="clearing" class="feedback"><h3>清空群 {{clearing.scope.group_id}} 的本游戏记录</h3><p>将恢复本群默认设置并删除全部主动提交的本地评分和挑战成绩。</p><div class="actions"><button :disabled="busy" @click="clearing=null">保留群记录</button><button class="destructive" :disabled="busy" @click="clear">确认清空群记录</button></div></section>
  <div class="actions"><button :disabled="busy||page===0" @click="turn(page-1)">上一页群</button><span>第 {{page+1}} 页</span><button :disabled="busy||next===null" @click="turn(next!)">下一页群</button></div>
 </section>
</template>
<style scoped>
.group-settings{display:grid;gap:22px}.group-settings fieldset{display:grid;gap:16px;min-width:0;border:1px solid var(--raylea-color-border,#ddd);padding:18px;border-radius:12px}.group-settings label{display:grid;gap:8px;min-width:0}.group-settings select{width:100%;min-width:0}.group-list{display:grid;gap:18px;list-style:none;padding:0}.group-list li{display:grid;gap:12px;border-top:1px solid var(--raylea-color-border,#ddd);padding-top:16px;overflow-wrap:anywhere}.alias-row{display:flex;gap:12px;align-items:end;flex-wrap:wrap}.alias-row label{flex:1 1 130px}
</style>
