const getMapData=name=>{if(!Object.hasOwn(referenceMaps,name))throw Error('metadata.missing');return referenceMaps[name]};
const settings={getConfig:()=>({damage_debug_log:false})};
const logger={debug(){},info(){},green:value=>value,warn(){throw Error('reference.warning')},error(){throw Error('reference.failure')}};
const console={log(){}};
const charData={};
