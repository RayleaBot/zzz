// Project-authored passives derived from the pinned WeaponId2Data descriptions.
// Only fixed metadata is read. Full proc timing is reported, not treated as DPS.
const supplementalZZZWeapons = new Set();
const zzzPassiveNotes = {};
function installZZZWeaponSupplements() {
 const get=(id,index,percent=true,stack=1)=>Array.from({length:5},(_,r)=>{
  const text=referenceMaps.WeaponId2Data[id].Talents[String(r+1)].Desc;
  const matches=[...text.matchAll(/<color=#2BAD00>(.*?)<\/color>/g)].map(m=>Number.parseFloat(m[1]));
  if(!Number.isFinite(matches[index]))throw Error('weapon.supplement_metadata');
  return matches[index]*(percent?.01:1)*stack;
 });
 const add=(id,buffs=[],condition='按已触发且未过期的条件计算；逐条效果见下方说明')=>{
  const data=referenceMaps.WeaponId2Data[id];if(!data)throw Error('weapon.supplement_missing');
  calcFnc.weapon[data.Name]={buffs};supplementalZZZWeapons.add(String(id));zzzPassiveNotes[id]=condition;
 };
 const b=(id,type,index,options={},percent=true,stack=1)=>({name:referenceMaps.WeaponId2Data[id].Name+'：补充规则 '+type,type,value:get(id,index,percent,stack),...options});
 // These change energy, Daze, survivability or buildup, not a single hit's damage.
 for(const id of ['12003','12007','12008','12012','12014','13002','13006','13011','13016','13127','14003'])add(id,[],'被动只影响能量、失衡、减伤或异常积蓄；单次伤害不加成，完整效果明确保留');
 add('12016',[b('12016','增伤',0,{range:['A']})],'强化特殊技后十秒内的普通攻击');
 add('13017',[b('13017','防御力',0),b('13017','防御力',1)],'防御力常驻提升与强化特殊技后的四十秒提升同时生效');
 add('13018',[b('13018','增伤',1)],'攻击处于属性异常状态的敌人；乱流回复能量单独说明');
 add('13020',[b('13020','增伤',1)],'支援突击后三十秒；失衡值加成不计为伤害');
 add('13021',[{name:'血髓秘匣：超过100%的未截断暴击率转增伤',type:'增伤',value:({calc})=>{
  const rank=calc.avatar.weapon.star-1,rate=calc.get('暴击率',calc.initial_properties.CRITRate);
  return Math.min(Math.max(0,rate-1)*100*get('13021',1)[rank],get('13021',2)[rank]);
 }}],'按局内未截断暴击率计算超额转换，保留最大增伤上限');
 add('13112',[{name:'比格气缸：受击后下一击必暴',type:'暴击率',value:1}], '受击后下一次命中；附加防御伤害单独列基础值，不猜测描述未给出的元素');
 add('14105',[b('14105','贯穿增伤',0,{element:'Ice'},true,3),b('14105','暴击率',1)],'生命降低增益三层，当前生命不高于50%');
 add('14130',[b('14130','暴击率',0),b('14130','无视防御',1,{},true,2)],'火属性追加攻击触发的无视防御叠满两层');
 add('14134',[b('14134','攻击力',1,{teamTarget:true,stackable:false}),b('14134','生命值',2,{teamTarget:true,stackable:false}),b('14134','暴击伤害',3,{teamTarget:true,stackable:false})],'已开启或延长以太帷幕；能量回复单独说明');
 add('14145',[b('14145','增伤',1,{teamTarget:true,stackable:false}),b('14145','生命值',2,{teamTarget:true,stackable:false})],'以太帷幕触发后四十五秒；能量回复单独说明');
 add('14148',[b('14148','暴击伤害',2,{teamTarget:true,stackable:false,check:({avatar})=>avatar.element_type===200})],'物理强化特殊技效果三层，暴伤团队增益生效；失衡与能量单独说明');
 add('14150',[b('14150','异常精通',0,{},false),b('14150','增伤',1,{check:({avatar})=>avatar.element_type===205}),b('14150','异常增伤',2,{check:({avatar})=>avatar.element_type===205})],'以太装备者处于前场且触发增益，目标处于异常状态');
 add('14151',[b('14151','异常精通',0,{},false),b('14151','增伤',1,{teamTarget:true,stackable:false,check:({avatar})=>avatar.element_type===205},true,2),b('14151','异常精通',2,{check:({avatar})=>avatar.element_type===205},false)],'以太普通/强化特殊技触发两层增益');
 add('14152',[b('14152','暴击率',0),b('14152','无视防御',1,{element:'Electric'})],'消耗能量或进入战斗后，无视防御增益仍有效');
 add('14154',[b('14154','增伤',0,{element:'Ice',check:({avatar})=>avatar.element_type===202},true,2),b('14154','异常增伤',1,{range:['异放'],check:({avatar})=>avatar.element_type===202})],'冰属性装备者的增益两层；异放只作用于明确标为异放的伤害，不套用到普通异常');
 add('14155',[{name:'日冕遗蜕：固定资料的20%暴击率',type:'暴击率',value:.2},b('14155','无视抗性',0,{element:'Ether',check:({avatar})=>avatar.name_mi18n==='佩洛伊斯'})],'佩洛伊斯的日蚀状态；暴击率按描述固定20%');
 add('14156',[b('14156','异常精通',0,{},false),b('14156','异常增伤',1,{range:['乱流','风化'],check:({avatar})=>avatar.element_type===204},true,2),b('14156','异常精通',2,{teamTarget:true,stackable:false,check:({avatar})=>avatar.element_type===204},false)],'风属性强化特殊技增益两层；仅对应乱流/风化条目，不替换已有异常类型');
 add('14157',[b('14157','冲击力',0,{},false),b('14157','无视抗性',1,{element:'Fire'}),b('14157','增伤',3,{teamTarget:true,stackable:false,check:({avatar})=>avatar.element_type===201},true,2)],'火属性强化特殊技增益两层；后台回复能量单独说明');
 add('14158',[b('14158','异常精通',0,{},false),b('14158','异常增伤',1),b('14158','增伤',2,{teamTarget:true,stackable:false})],'已触发异化后的三十秒增益');
 add('14159',[b('14159','暴击伤害',0,{},true,2),b('14159','无视抗性',1,{element:'Ice'})],'普通与强化特殊技重击分别提供一层兵锋，共两层并触发彻骨');
 add('14161',[b('14161','暴击率',0),b('14161','增伤',1,{element:'Electric'}),b('14161','增伤',2,{element:'Electric',range:['锐化']})],'强化特殊技或毁伤触发；额外加成只作用明确标为锐化的电伤');
 add('14162',[b('14162','暴击率',0),b('14162','无视抗性',1,{element:'Wind'})],'资料未提供正式名称；风属性强化特殊技触发。对其他队员的增伤不加给装备者自己');
}
installZZZWeaponSupplements();
