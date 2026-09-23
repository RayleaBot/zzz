const zzzPropertyKeys={HP:'生命值',ATK:'攻击力',DEF:'防御力',Impact:'冲击力',CRITRate:'暴击率',CRITDMG:'暴击伤害',AnomalyMastery:'异常掌控',AnomalyProficiency:'异常精通',PenRatio:'穿透率',EnergyRegen:'能量自动回复',SheerForce:'贯穿力'};
const zzzPercentKeys=new Set(['CRITRate','CRITDMG','PenRatio']);
const immutableZZZWeapons=_.cloneDeep(calcFnc.weapon),immutableZZZSets=_.cloneDeep(calcFnc.set);
function freshZZZRules(record){calcFnc.weapon=_.cloneDeep(immutableZZZWeapons);calcFnc.set=_.cloneDeep(immutableZZZSets);const rules=characterRule.default?(Array.isArray(characterRule.default)?characterRule.default:[characterRule.default]):[{...characterRule}];calcFnc.character[record.id]=_.cloneDeep(rules)}
function zzzGear(input){
 const sets=Object.entries(referenceMaps.SuitData),counts={};for(const piece of input)counts[piece.set_name]=(counts[piece.set_name]||0)+1;
 const stat=s=>({property_id:Number(s.id||s.key),property_name:property.idToName(s.id||s.key)||'',base:String(s.value)+(s.percent?'%':'')});
 return input.map(piece=>{
  const set=sets.find(([id,data])=>id===piece.set_id||data.name===piece.set_name);if(!set)throw Error('equipment.set_missing');
  if(counts[piece.set_name]>=2&&!calcFnc.set[piece.set_name])throw Error('equipment.rule_missing');
  const properties=piece.sub.map(stat);
  return {equipment_type:piece.slot,main_properties:[stat(piece.main)],properties,equip_suit:{suit_id:Number(set[0]),name:piece.set_name,own:counts[piece.set_name]},get_property:id=>Number(properties.find(p=>p.property_id===id)?.base||0)};
 });
}
function zzzWeapon(input){
 if(!input.id)return null;
 const record=referenceMaps.WeaponId2Data[input.id];if(!record||!calcFnc.weapon[record.Name])throw Error('weapon.rule_missing');
 return EnkaFormat.Weapon.main({Id:Number(input.id),Level:input.level,UpgradeLevel:input.refinement,BreakLevel:input.promote});
}
function zzzPropertyValue(property,part){const value=String(property?.[part]??'');const n=Number(value.replace('%',''));if(!value||!Number.isFinite(n))throw Error('attribute.missing');return value.endsWith('%')?n/100:n}
function zzzModel(info,gear,weapon,promote,core){return EnkaFormat.Property.main(info,gear,weapon,{PromotionLevel:promote,CoreSkillEnhancement:core})}
function zzzPromotions(level,weapon){const starts=weapon?[1,10,20,30,40,50]:[1,10,20,30,40,50];return starts.flatMap((n,i)=>level>=n&&level<=[10,20,30,40,50,60][i]?[i+(weapon?0:1)]:[])}
function inferZZZPromotions(info,gear,input,core){
 const promotions=input.promote==null?zzzPromotions(info.level,false):[input.promote];
 const weapons=!input.weapon.id?[0]:input.weapon.promote==null?zzzPromotions(input.weapon.level,true):[input.weapon.promote];let best;
 for(const promote of promotions)for(const weaponPromote of weapons){
  const selected={...input.weapon,promote:weaponPromote},weapon=zzzWeapon(selected),properties=zzzModel(info,gear,weapon,promote,core);let score=0;
  for(const key of ['HP','ATK','DEF']){const observed=input.attributes[key+'Base'];if(!Number.isFinite(observed))throw Error('attribute.base_missing');const value=zzzPropertyValue(properties.find(p=>p.property_name===zzzPropertyKeys[key]),'base');score+=Math.abs(value-observed)/Math.max(1,observed)}
  if(input.weapon.id&&input.weapon_main?.length){for(const stat of input.weapon_main){const actual=weapon.main_properties.find(p=>p.property_id===Number(stat.id));if(!actual)throw Error('weapon.property');score+=Math.abs(Number(actual.base)-stat.value)/Math.max(1,stat.value)}}
  if(!best||score<best.score)best={score,promote,selected,weapon,properties};
 }
 if(!best||!Number.isFinite(best.score)||best.score>0.03)throw Error('attributes.reference_mismatch');return best;
}
function zzzAvatar(info,gear,weapon,skills,properties){const avatar=new AvatarProperties();Object.assign(avatar,info,{equip:gear,weapon,skills,properties});return avatar}
function describeZZZBuff(buff){
 const percents=new Set(['倍率','增伤','易伤','无视抗性','无视防御','穿透率','失衡易伤','暴击率','暴击伤害','异常增伤','异常暴击率','异常暴击伤害','贯穿增伤']);
 const percentage=percents.has(buff.type),value=Number((buff.value*(percentage?100:1)).toFixed(2)),unit=percentage?'%':buff.type==='异常持续时间'?'秒':'';
 const names={Physical:'物理',Fire:'火',Ice:'冰',Electric:'电',Ether:'以太'};
 const elements=(Array.isArray(buff.element)?buff.element:[buff.element]).filter(Boolean).map(e=>names[e]||e).join('、');
 return `${buff.name}：${buff.type} ${value}${unit}${elements?' · '+elements:''}`;
}
function zzzScenario(info,input,gear,selected,promote,core,original,conditions){
 freshZZZRules({id:info.id});
 const weapon=zzzWeapon(selected),properties=zzzModel(info,gear,weapon,promote,core);
 for(const [key,name]of Object.entries(zzzPropertyKeys)){
  const next=properties.find(p=>p.property_name===name),before=original.find(p=>p.property_name===name);if(!next||!before)continue;
  for(const [part,suffix]of [['base','Base'],['final','']]){const observed=input.attributes[key+suffix];if(!Number.isFinite(observed))continue;const value=observed+zzzPropertyValue(next,part)-zzzPropertyValue(before,part);next[part]=zzzPercentKeys.has(key)?String(value*100)+'%':String(value)}
 }
 const skills=Object.entries(input.talents).map(([skill_type,level])=>({skill_type:Number(skill_type),level}));
 const avatar=zzzAvatar(info,gear,weapon,skills,properties),calc=avatarPipeline.avatar_calc(avatar);if(!calc)throw Error('character.rule_missing');
 calc.defEnemy('level',input.enemy_level);
 if(conditions){
  const kept=calc.buffM.buffs.filter(b=>b.name==='驱动盘5号位'||b.source==='套装'&&b.check===2||!(b.source==='音擎'?conditions.disable_weapon:b.source==='套装'?conditions.disable_equipment:conditions.disable_character));
  calc.buffM.buffs.splice(0,calc.buffM.buffs.length,...kept);
  if(conditions.enemy_resistance!=null)calc.defEnemy('resistance',conditions.enemy_resistance/100);
  const types={atkPct:'攻击力',atkPlus:'攻击力',hpPct:'生命值',hpPlus:'生命值',defPct:'防御力',defPlus:'防御力',cpct:'暴击率',cdmg:'暴击伤害',dmg:'增伤',enemyDmg:'易伤',ignore:'无视防御',resistance:'无视抗性',proficiency:'异常精通',anomaly:'异常增伤',sheer:'贯穿增伤',impact:'冲击力',stun:'失衡易伤'};
  const baseKeys={atkPct:'ATK',hpPct:'HP',defPct:'DEF'},flat=new Set(['atkPlus','hpPlus','defPlus','proficiency','impact']);
  for(const source of [{name:'用户填写',bonuses:conditions.bonuses},...(conditions.team||[])])for(const [key,value]of Object.entries(source.bonuses||{})){
   if(!value)continue;const base=baseKeys[key];
   calc.buffM.new({source:'技能',name:`自定义·${source.name}：${key} +${value}`,type:types[key],value:base?()=>calc.base_properties[base]*value/100:flat.has(key)?()=>value:value/100});
  }
 }
 const rows=calc.calc();if(!rows.length)throw Error('rule.empty');
 const results=rows.map(row=>{
  const values=row.result;if(!Number.isFinite(values.expectDMG)||!Number.isFinite(values.critDMG))throw Error('rule.nonfinite');
  const index=calc.skills.findIndex(s=>s.type===row.skill.type&&s.name===row.skill.name);
  return {id:String(index)+':'+row.skill.type,title:row.skill.name,expected:values.expectDMG,critical:row.skill.isAnomalyDMG&&values.critDMG===0?null:values.critDMG,kind:'damage',buffs:row.usefulBuffs.map(describeZZZBuff)};
 });
 if(weapon&&supplementalZZZWeapons.has(String(selected.id))&&!conditions?.disable_weapon){
  const description=referenceMaps.WeaponId2Data[selected.id].Talents[String(selected.refinement)].Desc.replace(/<[^>]*>/g,'');
  results.push({id:'weapon-effect',title:'音擎被动 · 完整效果与参考条件',expected:null,critical:null,text:description,kind:'text',buffs:[zzzPassiveNotes[selected.id]]});
  if(String(selected.id)==='13112'&&avatar.avatar_profession===5){
   const multiplier=[...referenceMaps.WeaponId2Data[selected.id].Talents[String(selected.refinement)].Desc.matchAll(/<color=#2BAD00>(.*?)<\/color>/g)].map(m=>parseFloat(m[1]))[1]/100;
   const defense=calc.get('防御力',calc.initial_properties.DEF),base=defense*multiplier;
   results.push({id:'weapon-cylinder-base',title:'比格气缸附加伤害基础值',expected:null,critical:null,text:`防御 ${defense.toFixed(2)} × ${multiplier.toFixed(2)} = ${base.toFixed(2)}；必定暴击，尚未计暴伤、敌人减伤和元素条件。`,kind:'text',buffs:['固定描述未给出独立附加伤害属性，仅列可确定的基础值']});
  }
 }
 return {weapon:{...selected,name:weapon?.name||''},attributes:avatar.initial_properties,results};
}
function runBuild(record,_weapons,input){
 charData[record.id]=record.data.calculation;
 freshZZZRules(record);
 const partner=record.data.partner,info={id:Number(record.id),name_mi18n:record.name,level:input.level,rank:input.rank,element_type:Number(partner.ElementType),sub_element_type:input.sub_element??EnkaFormat.parseInfo({Id:Number(record.id)}).sub_element_type,avatar_profession:Number(partner.WeaponType)};
 if(!elementType2element(info.element_type))throw Error('element.unsupported');
 const core=input.talents['5']-1;if(!Number.isInteger(core)||core<0||core>6)throw Error('talent.core');
 const gear=zzzGear(input.equipment),original=inferZZZPromotions(info,gear,input,core);
 const baseline=zzzScenario(info,input,gear,original.selected,original.promote,core,original.properties);let candidate=null;
 if(input.candidate_weapon||input.candidate_equipment||input.conditions){const next=input.candidate_weapon||original.selected;if(next.id&&!zzzPromotions(next.level,true).includes(next.promote))throw Error('weapon.promote');const nextGear=input.candidate_equipment?zzzGear(input.candidate_equipment):gear;candidate=zzzScenario(info,input,nextGear,next,original.promote,core,original.properties,input.conditions)}
 return {source:'simulation',version:'zzz-fb66219cec-reference-v1',character_id:record.id,character:record.name,enemy_level:input.enemy_level,baseline,candidate};
}
// ZZZ-Plugin's 伤害 (apps/damage.ts): the saved panel becomes a ZZZAvatarInfo,
// avatar_calc registers the agent's rule with the W-Engine and drive disc
// effects upstream has, and one skill is compared against swapped sub and main
// stats. input.skill is the number written after 伤害, or '' for none; without
// one the skill is the one the comparison picks (the main skill, else the
// highest expected damage). The plugin's own W-Engine supplements stay out, as
// upstream gives those W-Engines no effect.
function runDamage(record,_weapons,input){
 if(typeof scoreRule!=='undefined')scoreFnc[record.id]=scoreRule.default;
 if(typeof characterRule!=='undefined'){charData[record.id]=record.data.calculation;freshZZZRules(record)}
 for(const id of supplementalZZZWeapons)delete calcFnc.weapon[referenceMaps.WeaponId2Data[id].Name];
 const avatar=new ZZZAvatarInfo(input.avatar),calc=avatarPipeline.avatar_calc(avatar),damages=calc?.calc();
 if(!calc||!damages?.length)return {damages:[]};
 // Upstream only clamps a number above the count, so the one right after the
 // last skill leaves no damage to draw; it is clamped to the last here.
 let index=input.skill?Math.min(Math.max(Number(input.skill)-1,0),damages.length-1):null;
 const sub=calc.calc_sub_differences(index===null?undefined:damages[index].skill);
 if(index===null){const chosen=sub[0]?.[0].damage.skill;index=((chosen&&damages.findIndex(({skill})=>skill.name===chosen.name&&skill.type===chosen.type)+1)||damages.length)-1}
 const damage=damages[index],main=calc.calc_main_differences(damage.skill);
 // Each table's columns are the stats added and its rows the stats removed.
 const table=rows=>({columns:(rows[0]||[]).map(d=>({name:d.add.shortName,value:d.add.valueBase})),rows:rows.map(row=>({name:row[0].del.shortName,value:row[0].del.valueBase,differences:row.map(d=>d.difference)}))});
 return {
  damages:damages.map(d=>({name:d.skill.name,critical:d.result.critDMG,expected:d.result.expectDMG})),
  skill:index,anomaly:!!damage.skill.isAnomalyDMG,sheer:!!damage.skill.isSheerDMG,areas:damage.areas,
  buffs:damage.usefulBuffs.map(b=>({name:b.name,source:b.source,type:b.type,value:b.value})),
  sub:table(sub),main:table(main),weights:avatar.scoreWeight,
 };
}
// Showcase agents become official avatar entries through ZZZ-Plugin's
// Enka2Mys, which skips an agent its data does not know with a warning.
function runShowcase(_record,_weapons,input){
 logger.warn=()=>{};
 return {avatar_list:EnkaFormat.Enka2Mys(input.avatars)};
}
// Drive disc scoring follows ZZZ-Plugin: Score.getFinalWeight picks the rule,
// Score.main rates each disc, and the Equip and avatar getters grade them.
function runScore(record,_weapons,input){
 if(typeof scoreRule!=='undefined')scoreFnc[record.id]=scoreRule.default;
 const partner=record.data.partner,avatar=new AvatarProperties();
 Object.assign(avatar,{id:Number(record.id),rank:input.rank,element_type:Number(partner.ElementType),avatar_profession:Number(partner.WeaponType),properties:input.properties});
 const [title,weight]=ZZZScore.getFinalWeight(avatar);
 const pieces=input.equipment.map(disc=>{
  const properties=disc.sub.map(s=>({property_id:Number(s.id),count:getEquipPropertyEnhanceCount(s.id,s.value)}));
  const equip={equipment_type:disc.slot,level:disc.level,rarity:disc.rarity,main_properties:[{property_id:Number(disc.main)}],properties};
  const score=ZZZScore.main(equip,weight);
  if(!Number.isFinite(score))throw Error('score.nonfinite');
  return {slot:disc.slot,score,grade:new EquipGrade(score).comment,props:properties.map(p=>({id:p.property_id,count:p.count,weight:weight[p.property_id]||0}))};
 });
 Object.assign(avatar,{equip:pieces,scoreWeight:weight});
 // Substat totals as ZZZ-Plugin's propertyStats: rolls include the initial one.
 const stats={};
 for(const piece of pieces)for(const p of piece.props){const stat=stats[p.id]??={id:p.id,name:property.idToShortName2(p.id),weight:p.weight,value:'0',count:0};stat.count+=p.count+1}
 const statList=Object.values(stats);
 for(const stat of statList)if(baseValueData[stat.id]){stat.value=(baseValueData[stat.id]*stat.count).toFixed(1);if([11102,12102,13102,20103,21103].includes(stat.id))stat.value+='%'}
 return {title,pieces,total:avatar.equip_score,grade:avatar.equip_comment,weights:weight,stats:_.orderBy(statList,['count','weight'],['desc','desc'])};
}
