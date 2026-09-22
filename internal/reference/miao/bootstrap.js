// Only numerical/formatting dependencies used by the pinned miao modules.
const lodash = _;
let Format;
const Data={eachStr:(text,fn)=>String(text).split(',').forEach(fn)};
const Cfg={get:(_key,defaultValue)=>defaultValue};
class Base {
 constructor(){this.game='gs';return new Proxy(this,{get(target,key,receiver){if(key in target)return Reflect.get(target,key,receiver);if(target._get)return target._get.call(receiver,key);return target.meta?.[key]}})}
 get isGs(){return this.game==='gs'}
 get isSr(){return this.game==='sr'}
}
// Upstream getMeta(game, type, key) returns one entry when key is given.
const Meta={getMeta:(_game,kind,key)=>{const meta=kind==='weapon'?{weaponBuffs}:artiMeta;return key?meta[key]:meta}};
const console={log(){}};
const logger={info(){},warn(){},error(){},debug(){}};
let currentGame,weaponCatalog;
const ArtifactSet={
 getArtisSetBuff(name,count){let data=artifactBuffs[name]?.[count]||artifactBuffs[name+count];if(!data)return [];return Array.isArray(data)?data:[data]}
};
function loadWeaponDefinitions(definitions,game){
 if(typeof definitions!=='function')return definitions;
 if(game==='gs'){
  const step=(start,increase=0)=>Array.from({length:6},(_,i)=>start+(increase||start/4)*i);
  return definitions(step,(key,start,increase)=>({title:key+'提高[key]',isStatic:true,refine:{[key]:step(start,increase)}}));
 }
 return definitions((idx,key)=>({isStatic:true,idx,key}),(title,key,idx)=>{
  if(typeof key==='object')return tables=>({title,data:Object.fromEntries(Object.entries(key).map(([k,n])=>[k,tables[n]]))});
  return {title,idx,key};
 });
}
class ReferenceWeapon {
 constructor(record){this.name=record.name;this.type=record.type;this.id=record.id;this.detail=record.data;this.game=currentGame}
 calcAttr(level,promote){
  const attr=this.detail.attr;
  if(this.game==='sr'){
   const values=attr[promote];if(!values)throw Error('weapon.promote');
   return Object.fromEntries(Object.entries(values.attrs).map(([k,v])=>[k,v*1+(this.detail.growAttr?.[k]||0)*(level-1)]));
  }
  const steps=[1,20,40,50,60,70,80,90];let left,right;
  for(let i=0;i<steps.length-1;i++)if(i===promote&&level>=steps[i]&&level<=steps[i+1]){left=steps[i];right=steps[i+1];break}
  if(!right)throw Error('weapon.promote');
  const a=attr.atk[left+'+']??attr.atk[left],b=attr.atk[right];
  const x=attr.bonusData[left+'+']??attr.bonusData[left],y=attr.bonusData[right],n=Math.ceil((right-left)/5);
  return {atkBase:a+(b-a)*(level-left)/(right-left),attr:{key:attr.bonusKey,value:x+(n-Math.ceil((right-level)/5))*(y-x)/n}};
 }
 getWeaponAffixBuffs(affix,isStatic=true){
  let definitions=lodash.cloneDeep(weaponBuffs[this.id]||weaponBuffs[this.name]||[]);
  if(!Array.isArray(definitions))definitions=[definitions];
  const tables=Object.fromEntries(Object.entries(this.detail.skill?.tables||{}).map(([k,v])=>[k,v[affix-1]]));
  const result=[];
  for(let data of definitions){
   if(typeof data==='function')data=data(tables);
   if(!data||!!data.isStatic!==!!isStatic)continue;
   if(isStatic){
    const values={};if(data.idx&&data.key)values[data.key]=tables[data.idx];
    for(const [k,v]of Object.entries(data.refine||{}))values[k]=v[affix-1]*(data.buffCount||1);
    if(Object.keys(values).length)result.push({isStatic:true,data:values});
   }else{
    data.title=/：/.test(data.title)?data.title:this.name+'：'+(data.title||'额外效果');data.data=data.data||{};
    if(data.idx&&data.key)data.data[data.key]=tables[data.idx];
    else for(const [k,v]of Object.entries(data.refine||{}))data.data[k]=({refine})=>v[refine]*(data.buffCount||1);
    result.push(data);
   }
  }
  return result;
 }
}
const Weapon={get(name){const record=weaponCatalog.find(w=>w.name===name||w.id===String(name));return record?new ReferenceWeapon(record):false}};
