const getMapData=name=>{if(!Object.hasOwn(referenceMaps,name))throw Error('metadata.missing');return referenceMaps[name]};
const settings={getConfig:()=>({damage_debug_log:false})};
// ZZZ-Plugin's calculation logs the errors it recovers from, such as a skill or
// buff that fails, and carries on; the log is not kept here.
const logger={debug(){},info(){},green:value=>value,blue:value=>value,magenta:value=>value,red:value=>value,yellow:value=>value,warn(){},error(){}};
const console={log(){}};
const charData={};
