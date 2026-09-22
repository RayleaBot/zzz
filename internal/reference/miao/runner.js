function characterObject(record){
 const data=record.data;
 return {id:Number(record.id),name:record.name,game:record.game,detail:data,elem:record.element,weapon:data.weapon,weaponTypeName:{sword:'单手剑',claymore:'双手剑',polearm:'长柄武器',bow:'弓',catalyst:'法器'}[data.weapon]||data.weapon,sp:data.sp,
  isElem:elem=>Format.sameElem(elem,record.element),getCalcRule:()=>characterRule,
  getLvAttr(level,promote){const item=data.attr[promote];if(!item)throw Error('character.promote');return Object.fromEntries(Object.entries(item.attrs).map(([key,value])=>[key,value*1+(item.grow?.[key]||0)*(level-1)]))}
 };
}
function artifactObject(gear){
 const adapt=stat=>({...stat,key:currentGame==='sr'?({physical:'phy',lightning:'elec'}[stat.key]||stat.key):stat.key});
 const counts={},items=gear.map(g=>({main:adapt(g.main),attrs:g.sub.map(adapt)}));
 for(const piece of gear){if(piece.set_name)counts[piece.set_name]=(counts[piece.set_name]||0)+1}
 const sets={};for(const [name,count]of Object.entries(counts)){if(count>=4)sets[name]=4;else if(count>=2)sets[name]=2}
 const result={...sets};
 Object.defineProperties(result,{
  forEach:{value:fn=>items.forEach(fn)},
  eachArtisSet:{value:fn=>{for(const [name,n]of Object.entries(sets)){fn({name},2);if(n>=4)fn({name},4)}}}
 });
 return result;
}
function levels(profile){
 const out={talentLevel:profile.talent};
 for(const key of (currentGame==='gs'?'a,e,q':'a,a2,e,e1,e2,q,q2,q3,t,t2,xe,xe2,me,me2,mt,mt1,mt2').split(',')){
  const base=key.replace(/[123]$/,'');const level=profile.talent[base];out[key]={};
  if(currentGame==='gs'){
   for(const [name,values]of Object.entries(profile.char.detail.talentData?.[key]||{})){if(!Number.isInteger(level)||level<1||values[level-1]===undefined)throw Error('talent.'+key);out[key][name]=values[level-1]}
  }else{
   for(const table of Object.values(profile.char.detail.talent?.[key]?.tables||{})){if(!Number.isInteger(level)||level<1)throw Error('talent.'+key);out[key][table.name]=table.isSame?table.values[0]:table.values[level-1]}
  }
 }
 return out;
}
function combatBuffs(profile,conditions){
 let list=conditions?.disable_character?[]:lodash.cloneDeep(characterRule.buffs||[]);
 const weapon=Weapon.get(profile.weapon.id);
 // Miao's weapon and artifact adapters append only combat effects. Permanent
 // effects are already part of the static panel and must not be counted twice.
 if(weapon&&!conditions?.disable_weapon)list.push(...weapon.getWeaponAffixBuffs(profile.weapon.affix,false));
 if(!conditions?.disable_equipment)profile.artis.eachArtisSet((set,count)=>{for(const buff of ArtifactSet.getArtisSetBuff(set.name,count))if(buff&&!buff.isStatic)list.push({...lodash.cloneDeep(buff),title:set.name+count+'：'+buff.title})});
 if(conditions){
  for(const source of [{name:'用户填写',bonuses:conditions.bonuses},...(conditions.team||[])])for(const [key,value]of Object.entries(source.bonuses||{})){
   if(!value)continue;const target={enemyDmg:'enemydmg',resistance:'kx'}[key]||key;
   list.push({title:`自定义·${source.name}：${key} +${value}`,sort:1,data:{[target]:value}});
  }
  if(conditions.enemy_resistance!=null)list.push({title:`自定义敌人抗性 ${conditions.enemy_resistance}%`,sort:-1,data:{kx:(currentGame==='gs'?10:0)-conditions.enemy_resistance}});
 }
 return list.map((buff,index)=>{
  if(typeof buff==='string'){
   const names={vaporize:'蒸发',melt:'融化',swirl:'扩散',aggravate:'超激化',spread:'蔓激化'},key=buff;
   buff={title:`元素精通：${names[key]||key}伤害提高[_${key}]%`+(['aggravate','spread'].includes(key)?`，伤害值提升[_${key}num]`:''),mastery:key,sort:9};
  }
  return {...buff,sort:buff.sort??1,_index:index};
 }).filter(Boolean).sort((a,b)=>a.sort-b.sort||a._index-b._index);
}
function legalPromotions(level,weapon){
 const steps=currentGame==='gs'?(weapon?[1,20,40,50,60,70,80,90]:[1,20,40,50,60,70,80,90,100]):[1,20,30,40,50,60,70,80];
 return steps.slice(0,-1).flatMap((left,index)=>level>=left&&level<=steps[index+1]?[index]:[]);
}
function inferPromotions(profile,observed){
 const cp=profile.promote==null?legalPromotions(profile.level,false):[profile.promote];
 const wp=!profile.weapon.id?[0]:(profile.weapon.promote==null?legalPromotions(profile.weapon.level,true):[profile.weapon.promote]);
 let best;
 for(const promote of cp)for(const weaponPromote of wp){
  try{
   const test={...profile,promote,weapon:{...profile.weapon,promote:weaponPromote}};
   const attrs=new Attr(test).calc();let score=0,count=0;
   for(const key of ['hp','atk','def'])if(Number.isFinite(observed[key+'Base'])){score+=Math.abs(attrs[key+'Base']-observed[key+'Base'])/Math.max(1,observed[key+'Base']);count++}
   if(!count)throw Error('attributes.base_missing');
   if(!best||score<best.score)best={score,promote,weaponPromote,attrs};
  }catch(e){ /* Another legitimate promotion may fit the same boundary level. */ }
 }
 if(!best||!Number.isFinite(best.score)||best.score>0.03)throw Error('attributes.reference_mismatch');
 profile.promote=best.promote;profile.weapon.promote=best.weaponPromote;
 return best.attrs;
}
function calculateProfile(profile,observed,staticOriginal,enemyLevel,conditions){
 const staticNow=new Attr(profile).calc();
 const anchored={...staticNow,staticAttr:staticNow.staticAttr};
 for(const [key,value]of Object.entries(observed))if(Number.isFinite(value)&&Number.isFinite(staticOriginal[key])&&Number.isFinite(staticNow[key]))anchored[key]=value+staticNow[key]-staticOriginal[key];
 const talent=levels(profile),trees={};
 for(const tid of profile.trees){const match=/1?\d{4}(\d{3})/.exec(tid);if(match)trees[match[1]]=true}
 const meta={characterName:profile.char.name,level:profile.level,cons:profile.cons,talent,trees,weapon:profile.weapon};
 const originalAttr=DmgAttr.getAttr({attr:anchored,weapon:profile.weapon,char:profile.char,game:currentGame});
 const buffs=combatBuffs(profile,conditions);
 const defParams=typeof characterRule.defParams==='function'?characterRule.defParams(meta):characterRule.defParams||{};
 // miao's single-damage mode (group ranks) takes the first detail computed
 // among those named by defDmgKey, else the one at defDmgIdx, else the first.
 const defKey=typeof characterRule.defDmgKey==='function'?characterRule.defDmgKey(meta):characterRule.defDmgKey||'';
 const defIdx=typeof characterRule.defDmgIdx==='function'?characterRule.defDmgIdx(meta):characterRule.defDmgIdx||-1;
 const results=[];
 for(const [index,configured]of (characterRule.details||[]).entries()){
  const isDefault=defKey?configured.dmgKey===defKey:index===(defIdx>-1?defIdx:0);
  let detail=configured;
  if(typeof detail==='function'){
   const {attr}=DmgAttr.calcAttr({originalAttr,buffs,artis:profile.artis,meta,game:currentGame});
   detail=detail({...DmgAttr.getDs(attr,meta),talent,attr,profile});
  }
  if(!detail||detail.isStatic||detail.cons&&meta.cons<detail.cons)continue;
  const params=lodash.merge({},defParams,typeof detail.params==='function'?detail.params(meta):detail.params||{});
  const {attr,msg}=DmgAttr.calcAttr({originalAttr,buffs,artis:profile.artis,meta,params,talent:detail.talent||'',game:currentGame});
  if(conditions&&[conditions.bonuses,...(conditions.team||[]).map(t=>t.bonuses)].some(b=>b&&(b.ignore||b.enemyDef))){attr.enemy.def=Math.max(0,Math.min(100,attr.enemy.def));attr.enemy.ignore=Math.max(0,Math.min(100,attr.enemy.ignore))}
  const ds=lodash.merge({talent},DmgAttr.getDs(attr,meta,params));ds.artis=profile.artis;
  if(detail.check&&!detail.check(ds))continue;
  if(!detail.dmg)continue;
  const fn=DmgCalc.getDmgFn({ds,attr,level:profile.level,enemyLv:enemyLevel,game:currentGame});
  const calculated=detail.dmg(ds,fn);
  const text=calculated?.type==='text';
  if(text&&/(NaN|Infinity|undefined)/.test(String(calculated.avg)))throw Error('rule.nonfinite_text.'+index);
  if(!calculated||(!text&&!Number.isFinite(calculated.avg))||calculated.dmg!=null&&!Number.isFinite(calculated.dmg))throw Error('rule.nonfinite.'+index);
  const title=typeof detail.title==='function'?detail.title(ds):detail.title;
  results.push({id:String(index),title:String(title),expected:text?null:calculated.avg,text:text?String(calculated.avg):'',critical:calculated.dmg??null,buffs:msg,kind:calculated.type||'damage',default:isDefault&&!results.some(r=>r.default)});
 }
 if(!results.length)throw Error('rule.empty');
 const attributes={};for(const key of ['hp','atk','def','hpBase','atkBase','defBase','speed','cpct','cdmg','mastery','recharge','dmg','phy','stance'])if(Number.isFinite(anchored[key]))attributes[key]=anchored[key];
 return {weapon:profile.weapon,promote:profile.promote,attributes,results};
}
// 面板换装 follows miao's ProfileChange: the changed panel's properties are
// calculated from the character, weapon, artifacts and traces with Attr, as
// Avatar.calcAttr does.
function runChange(record,weapons,input){
 currentGame=record.game;weaponCatalog=weapons;
 if(typeof characterRule==='undefined')globalThis.characterRule={};
 const char=characterObject(record),weapon=Weapon.get(input.weapon.id);
 if(input.weapon.id&&!weapon)throw Error('weapon.missing');
 const promote=input.promote??Attr.calcPromote(input.level,currentGame);
 const weaponPromote=input.weapon.promote??(weapon?Attr.calcPromote(input.weapon.level,currentGame):0);
 const profile={game:currentGame,char,id:char.id,elem:char.elem,level:input.level,promote,cons:input.rank,talent:input.talents,trees:input.trees,weapon:{...input.weapon,promote:weaponPromote,name:weapon?.name||'',affix:input.weapon.refinement},artis:artifactObject(input.equipment)};
 const attrs=new Attr(profile).calc();
 const attributes={};for(const [key,value]of Object.entries(attrs))if(Number.isFinite(value))attributes[key]=value;
 return {promote,weapon_promote:weaponPromote,attributes,weapon_attrs:weapon?weapon.calcAttr(profile.weapon.level,weaponPromote):null};
}
function runBuild(record,weapons,input){
 currentGame=record.game;weaponCatalog=weapons;
 const char=characterObject(record),weapon=Weapon.get(input.weapon.id);
 if(input.weapon.id&&!weapon)throw Error('weapon.missing');
 const profile={game:currentGame,char,id:char.id,elem:char.elem,level:input.level,promote:input.promote,cons:input.rank,talent:input.talents,trees:input.trees,weapon:{...input.weapon,name:weapon?.name||'',affix:input.weapon.refinement},artis:artifactObject(input.equipment)};
 const original=inferPromotions(profile,input.attributes);
 if(input.resolve_identity===true)return {promote:profile.promote,weapon_promote:profile.weapon.promote};
 const baseline=calculateProfile(profile,input.attributes,original,input.enemy_level);
 let candidate=null;
 if(input.candidate_weapon||input.candidate_equipment||input.conditions){
  const proposed=input.candidate_weapon||profile.weapon;
  const next=Weapon.get(proposed.id);if(proposed.id&&(!next||next.type!==char.weapon))throw Error('weapon.type');
  const candidateProfile={...profile,weapon:{...proposed,name:next?.name||'',affix:proposed.refinement},artis:input.candidate_equipment?artifactObject(input.candidate_equipment):profile.artis};
  if(next&&!legalPromotions(candidateProfile.weapon.level,true).includes(candidateProfile.weapon.promote))throw Error('weapon.promote');
  candidate=calculateProfile(candidateProfile,input.attributes,original,input.enemy_level,input.conditions);
 }
 return {source:'simulation',version:'miao-7f6f1c84-reference-v1',character_id:record.id,character:record.name,enemy_level:input.enemy_level,baseline,candidate};
}
// Artis adapter for ArtisMarkCfg: set abbreviations and main-stat checks follow
// miao Artis.is, isAttr and ArtisSet.getSetData.
// scoringKey maps panel attribute keys to miao's; Star Rail names two
// elements differently.
function scoringKey(k){return currentGame==='sr'?({physical:'phy',lightning:'elec'}[k]||k):k}
function scoringArtifacts(gear){
 const key=scoringKey;
 const pieces={},counts={};
 for(const g of gear){
  pieces[g.slot]={set:g.set_name,main:{key:key(g.main.key),value:g.main.value},attrs:g.sub.map(s=>({key:key(s.key),value:s.value}))};
  counts[g.set_name]=(counts[g.set_name]||0)+1;
 }
 const names=[],abbrs=[],full=[];
 for(const [name,count]of Object.entries(counts)){
  if(count<2)continue;const n=count>=4?4:2;
  names.push(name);abbrs.push((artiMeta.setAbbr[name]||name)+n);full.push(name+n);
 }
 const sets=[...abbrs,...full];
 return {names,pieces,
  is(check,pos=''){
   if(!pos)return String(check).split(',').some(s=>sets.includes(s));
   const attrs=String(check).split(','),dmgIdx=currentGame==='gs'?'4':'5';
   return String(pos).split(',').every(p=>{const main=pieces[p]?.main.key||'';return attrs.includes(main)||p===dmgIdx&&attrs.includes('dmg')&&Format.isElem(main)});
  }
 };
}
function runScore(record,weapons,input){
 currentGame=record.game;weaponCatalog=weapons;
 const data=record.data,weapon=input.weapon.id?Weapon.get(input.weapon.id):false;
 const char={name:record.name,abbr:data.abbr||record.name,game:currentGame,isGs:currentGame==='gs',baseAttr:data.baseAttr,getArtisCfg:()=>typeof scoreRule==='undefined'?false:scoreRule.default};
 const artis=scoringArtifacts(input.equipment);
 const profile={game:currentGame,char,id:Number(record.id),elem:record.element,attr:input.attributes,cons:input.rank,artis,weapon:{name:weapon?.name||input.weapon.name||'',affix:input.weapon.refinement,bonusKey:weapon?.detail?.attr?.bonusKey}};
 const cfg=ArtisMarkCfg.getCfg(profile);
 // Substat detail as upstream's getMarkDetail shows it: rolls are the
 // official upgrade count plus the initial roll, efficiency is the value in
 // maximum single rolls.
 const attrMap=Meta.getMeta(currentGame,'arti').attrMap;
 const rolled=s=>{const k=scoringKey(s.key),max=attrMap[k]?.value;return {key:k,value:s.value,upNum:(s.times||0)+1,eff:max?s.value/max:0}};
 const pieces=[],all={};let total=0;
 for(const g of input.equipment){
  const arti=artis.pieces[g.slot];
  const mark=ArtisMark.getMark({charCfg:cfg,idx:g.slot,arti,elem:profile.elem,game:currentGame,id:profile.id});
  if(!Number.isFinite(mark))throw Error('score.nonfinite');
  const attrs=g.sub.map(rolled);
  for(const a of attrs){const t=all[a.key]||(all[a.key]={key:a.key,value:0,upNum:0,eff:0});t.value+=a.value;t.upNum+=a.upNum;t.eff+=a.eff}
  total+=mark;pieces.push({slot:g.slot,score:mark,grade:ArtisMark.getMarkClass(mark)||'MAX',mark:Format.comma(mark,1),main:ArtisMark.formatArti(arti.main,cfg.attrs,true,currentGame),attrs:ArtisMark.formatArtiAttrs(attrs,cfg.attrs,currentGame)});
 }
 const allAttrs=ArtisMark.formatArti(lodash.sortBy(Object.values(all),['eff']).reverse(),false,false,currentGame);
 return {title:cfg.classTitle,weights:lodash.mapValues(cfg.attrs,a=>a.weight),pieces,total,mark:Format.comma(total,1),grade:ArtisMark.getMarkClass(total/(currentGame==='gs'?5:6))||'MAX',all_attrs:Array.isArray(allAttrs)?allAttrs:[],titles:ArtisMark.getKeyTitleMap(currentGame)};
}
