const property=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.nameZHToNameEN = exports.nameToId = exports.nameToShortName3 = exports.idToShortName3 = exports.idToShortName2 = exports.idToName = exports.idToSignName = void 0;
exports.idToClassName = idToClassName;
const propertyData = getMapData('Property2Name');
/** 属性css命名 */
const prop_id = {
    111: 'hpmax',
    121: 'attack',
    131: 'def',
    122: 'breakstun',
    201: 'crit',
    211: 'critdam',
    314: 'elementabnormalpower',
    312: 'elementmystery',
    231: 'penratio',
    232: 'penvalue',
    305: 'sprecover',
    310: 'spgetratio',
    115: 'spmax',
    315: 'physdmg',
    316: 'fire',
    317: 'ice',
    318: 'thunder',
    319: 'dungeonbuffether',
    323: 'wind',
    25: 'sharpnessaccumulate',
    28: 'laceration',
};
/** 职业css命名 */
const pro_id = {
    1: 'attack',
    2: 'stun',
    3: 'anomaly',
    4: 'support',
    5: 'defense',
    6: 'rupture',
    7: 'armorer',
};
/**
 * 获取属性css类名
 * @param _id 属性id
 */
function idToClassName(_id) {
    const propId = +_id.toString().slice(0, 3);
    const propIcon = prop_id[propId];
    if (!propIcon)
        return null;
    return propIcon;
}
/**
 * 获取属性标识
 * @param id 属性id
 */
const idToSignName = (id) => {
    const result = propertyData[id];
    if (!result)
        return null;
    return result[0];
};
exports.idToSignName = idToSignName;
/**
 * 获取属性全称
 * @param id 属性id
 */
const idToName = (id) => {
    const result = propertyData[id];
    if (!result)
        return null;
    return result[1];
};
exports.idToName = idToName;
/**
 * 获取属性2字简称
 * @param id 属性id
 */
const idToShortName2 = (id) => {
    const result = propertyData[id];
    if (!result)
        return '';
    return result[2];
};
exports.idToShortName2 = idToShortName2;
/**
 * 获取属性2~3字简称
 * @param id 属性id
 */
const idToShortName3 = (id) => {
    const result = propertyData[id];
    if (!result)
        return '';
    return result[3];
};
exports.idToShortName3 = idToShortName3;
/**
 * 获取属性2~3字简称
 */
const nameToShortName3 = (propName) => {
    for (const id in propertyData) {
        if (propertyData[id][1] === propName)
            return propertyData[id][3];
    }
    ;
    return propName;
};
exports.nameToShortName3 = nameToShortName3;
/**
 * 属性名转id
 * @param propName 属性名
 */
const nameToId = (propName) => {
    for (const id in propertyData) {
        if (propertyData[id][1] === propName ||
            propertyData[id][1].replace('属性', '') === propName)
            return Number(id);
    }
    ;
    return 0;
};
exports.nameToId = nameToId;
/**
 * 中文属性名转英文属性名
 * @param propNameZH 属性名
 */
const nameZHToNameEN = (propNameZH) => {
    var _a;
    for (const id in propertyData) {
        if (((_a = propertyData[id]) === null || _a === void 0 ? void 0 : _a[1]) === propNameZH)
            return propertyData[id][0];
    }
    ;
    return '';
};
exports.nameZHToNameEN = nameZHToNameEN;

return exports;})();
const element=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.idToData = idToData;
exports.idToName = idToName;
exports.idToPropertyId = idToPropertyId;
const ElementData = getMapData('ElementData');
/**
 * 元素ID转元素数据
 * @param id element_type
 * @param sub_id sub_element_type
 */
function idToData(id, sub_id = 0) {
    id = Number(id);
    sub_id = Number(sub_id);
    return ElementData.find(i => i.element_type === id && i.sub_element_type === sub_id) || null;
}
/**
 * 获取元素名（en_sub）
 */
function idToName(id, sub_id = 0) {
    const data = idToData(id, sub_id);
    if (!data)
        return '';
    return data.en_sub;
}
/**
 * ID转属性ID
 */
function idToPropertyId(id) {
    const data = idToData(id);
    if (!data)
        return null;
    return data.property_id;
}

return exports;})();
const buffRuntime=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BuffManager = exports.runtime = exports.elementType2element = exports.professionEnum = exports.buffTypeEnum = exports.anomalyEnum = exports.elementEnum = exports.rarityEnum = void 0;
var rarityEnum;
(function (rarityEnum) {
    rarityEnum[rarityEnum["S"] = 0] = "S";
    rarityEnum[rarityEnum["A"] = 1] = "A";
    rarityEnum[rarityEnum["B"] = 2] = "B";
})(rarityEnum || (exports.rarityEnum = rarityEnum = {}));
var elementEnum;
(function (elementEnum) {
    elementEnum[elementEnum["Physical"] = 200] = "Physical";
    elementEnum[elementEnum["Fire"] = 201] = "Fire";
    elementEnum[elementEnum["Ice"] = 202] = "Ice";
    elementEnum[elementEnum["Electric"] = 203] = "Electric";
    elementEnum[elementEnum["Ether"] = 205] = "Ether";
})(elementEnum || (exports.elementEnum = elementEnum = {}));
var anomalyEnum;
(function (anomalyEnum) {
    // 伤害异常
    anomalyEnum[anomalyEnum["\u5F3A\u51FB"] = 0] = "\u5F3A\u51FB";
    anomalyEnum[anomalyEnum["\u707C\u70E7"] = 1] = "\u707C\u70E7";
    anomalyEnum[anomalyEnum["\u788E\u51B0"] = 2] = "\u788E\u51B0";
    anomalyEnum[anomalyEnum["\u611F\u7535"] = 3] = "\u611F\u7535";
    anomalyEnum[anomalyEnum["\u4FB5\u8680"] = 4] = "\u4FB5\u8680";
    anomalyEnum[anomalyEnum["\u7D0A\u4E71"] = 5] = "\u7D0A\u4E71";
    // 状态异常（异常持续时间buff应作用于对应的状态异常）
    anomalyEnum[anomalyEnum["\u754F\u7F29"] = 6] = "\u754F\u7F29";
    anomalyEnum[anomalyEnum["\u971C\u5BD2"] = 7] = "\u971C\u5BD2";
})(anomalyEnum || (exports.anomalyEnum = anomalyEnum = {}));
var buffTypeEnum;
(function (buffTypeEnum) {
    // 通用乘区
    buffTypeEnum[buffTypeEnum["\u653B\u51FB\u529B"] = 0] = "\u653B\u51FB\u529B";
    buffTypeEnum[buffTypeEnum["\u500D\u7387"] = 1] = "\u500D\u7387";
    buffTypeEnum[buffTypeEnum["\u589E\u4F24"] = 2] = "\u589E\u4F24";
    buffTypeEnum[buffTypeEnum["\u6613\u4F24"] = 3] = "\u6613\u4F24";
    buffTypeEnum[buffTypeEnum["\u65E0\u89C6\u6297\u6027"] = 4] = "\u65E0\u89C6\u6297\u6027";
    buffTypeEnum[buffTypeEnum["\u65E0\u89C6\u9632\u5FA1"] = 5] = "\u65E0\u89C6\u9632\u5FA1";
    buffTypeEnum[buffTypeEnum["\u7A7F\u900F\u503C"] = 6] = "\u7A7F\u900F\u503C";
    buffTypeEnum[buffTypeEnum["\u7A7F\u900F\u7387"] = 7] = "\u7A7F\u900F\u7387";
    buffTypeEnum[buffTypeEnum["\u5931\u8861\u6613\u4F24"] = 8] = "\u5931\u8861\u6613\u4F24";
    // 直伤乘区
    buffTypeEnum[buffTypeEnum["\u66B4\u51FB\u7387"] = 9] = "\u66B4\u51FB\u7387";
    buffTypeEnum[buffTypeEnum["\u66B4\u51FB\u4F24\u5BB3"] = 10] = "\u66B4\u51FB\u4F24\u5BB3";
    // 异常乘区
    buffTypeEnum[buffTypeEnum["\u5F02\u5E38\u7CBE\u901A"] = 11] = "\u5F02\u5E38\u7CBE\u901A";
    buffTypeEnum[buffTypeEnum["\u5F02\u5E38\u589E\u4F24"] = 12] = "\u5F02\u5E38\u589E\u4F24";
    buffTypeEnum[buffTypeEnum["\u5F02\u5E38\u66B4\u51FB\u7387"] = 13] = "\u5F02\u5E38\u66B4\u51FB\u7387";
    buffTypeEnum[buffTypeEnum["\u5F02\u5E38\u66B4\u51FB\u4F24\u5BB3"] = 14] = "\u5F02\u5E38\u66B4\u51FB\u4F24\u5BB3";
    buffTypeEnum[buffTypeEnum["\u5F02\u5E38\u6301\u7EED\u65F6\u95F4"] = 15] = "\u5F02\u5E38\u6301\u7EED\u65F6\u95F4";
    // 贯穿乘区
    buffTypeEnum[buffTypeEnum["\u8D2F\u7A7F\u529B"] = 16] = "\u8D2F\u7A7F\u529B";
    buffTypeEnum[buffTypeEnum["\u8D2F\u7A7F\u589E\u4F24"] = 17] = "\u8D2F\u7A7F\u589E\u4F24";
    // 其他属性，一般不直接影响伤害，但可能用于buff是否生效判断/转模
    buffTypeEnum[buffTypeEnum["\u751F\u547D\u503C"] = 18] = "\u751F\u547D\u503C";
    buffTypeEnum[buffTypeEnum["\u9632\u5FA1\u529B"] = 19] = "\u9632\u5FA1\u529B";
    buffTypeEnum[buffTypeEnum["\u51B2\u51FB\u529B"] = 20] = "\u51B2\u51FB\u529B";
    buffTypeEnum[buffTypeEnum["\u5F02\u5E38\u638C\u63A7"] = 21] = "\u5F02\u5E38\u638C\u63A7";
})(buffTypeEnum || (exports.buffTypeEnum = buffTypeEnum = {}));
var professionEnum;
(function (professionEnum) {
    professionEnum[professionEnum["\u5F3A\u653B"] = 1] = "\u5F3A\u653B";
    professionEnum[professionEnum["\u51FB\u7834"] = 2] = "\u51FB\u7834";
    professionEnum[professionEnum["\u5F02\u5E38"] = 3] = "\u5F02\u5E38";
    professionEnum[professionEnum["\u652F\u63F4"] = 4] = "\u652F\u63F4";
    professionEnum[professionEnum["\u9632\u62A4"] = 5] = "\u9632\u62A4";
    professionEnum[professionEnum["\u547D\u7834"] = 6] = "\u547D\u7834";
    professionEnum[professionEnum["\u950B\u5FA1"] = 7] = "\u950B\u5FA1";
})(professionEnum || (exports.professionEnum = professionEnum = {}));
/** ID 2 EN */
const elementType2element = (elementType) => elementEnum[elementType];
exports.elementType2element = elementType2element;
exports.runtime = { elementType2element: exports.elementType2element, rarityEnum, elementEnum, anomalyEnum, buffTypeEnum, professionEnum };
let depth = 0, weakMapCheck = new WeakMap();
/**
 * Buff管理器
 * 用于管理角色局内Buff
 */
class BuffManager {
    constructor(avatar) {
        this.buffs = [];
        /** 套装计数 */
        this.setCount = {};
        this.defaultBuff = {};
        this.avatar = avatar;
    }
    new(buff) {
        if (Array.isArray(buff)) {
            buff.forEach(b => this.new(b));
            return this.buffs;
        }
        // 简化参数
        if (!buff.name && (buff.source || this.defaultBuff.source) === '套装' && this.defaultBuff.name && typeof buff.check === 'number')
            buff.name = this.defaultBuff.name + buff.check;
        const oriBuff = buff;
        // @ts-expect-error
        buff = { status: true, ...this.defaultBuff, ...buff };
        if (buff.range && !Array.isArray(buff.range))
            buff.range = oriBuff.range = [buff.range];
        if (!buff.source) {
            if (buff.name.includes('核心') || buff.name.includes('天赋'))
                buff.source = oriBuff.source = '核心被动';
            else if (buff.name.includes('额外能力'))
                buff.source = oriBuff.source = '额外能力';
            else if (/^\d影/.test(buff.name))
                buff.source = oriBuff.source = '影画';
            else if (buff.name.includes('技'))
                buff.source = oriBuff.source = '技能';
        }
        for (const key of ['name', 'value', 'source']) {
            if (!buff[key])
                return logger.warn(`无效buff：缺少${key}字段`, buff);
        }
        if (buffTypeEnum[buffTypeEnum[buff.type]] !== buff.type)
            return logger.warn(`无效buff：非法type字段`, buff);
        // 音擎buff职业检查
        if (buff.source === '音擎') {
            const professionCheck = (avatar) => {
                var _a;
                const weapon_profession = (_a = avatar.weapon) === null || _a === void 0 ? void 0 : _a.profession;
                if (!weapon_profession)
                    return true;
                return avatar.avatar_profession === weapon_profession;
            };
            const oriCheck = typeof buff.check === 'function' && buff.check;
            buff.check = ({ avatar, buffM, calc, runtime }) => professionCheck(avatar) && (!oriCheck || oriCheck({ avatar, buffM, calc, runtime }));
            // 影画buff影画数检查
        }
        else if (buff.source === '影画' && !buff.check) {
            buff.check = oriBuff.check = +buff.name[0];
        }
        this.buffs.push(buff);
        return this.buffs;
    }
    _filter(buffs, param, valueOcalc) {
        depth++;
        try {
            if (typeof param === 'string') {
                buffs = buffs.filter(buff => buff[param] === valueOcalc);
            }
            else if (typeof param === 'object') {
                buffs = buffs.filter(buff => {
                    if (buff.status === false)
                        return false;
                    const judge = (() => {
                        var _a;
                        // 未传入calc时不判断range、include、exclude
                        if (typeof valueOcalc !== 'object' || Array.isArray(valueOcalc))
                            return true;
                        // buff指定排除该技能
                        if (buff.exclude && buff.exclude.includes(valueOcalc.skill.type))
                            return false;
                        // 11 10 01
                        if (buff.range || buff.include) {
                            // 11 01 存在include且满足时则直接返回true
                            if (buff.include && buff.include.includes(valueOcalc.skill.type))
                                return true;
                            // 01 没有range则代表只有include，直接返回false
                            if (!buff.range)
                                return false;
                            // 11 10 直接返回range的结果即可
                            const buffRange = buff.range;
                            const skillRange = (_a = param.range) === null || _a === void 0 ? void 0 : _a.filter(r => typeof r === 'string');
                            if (!(skillRange === null || skillRange === void 0 ? void 0 : skillRange.length))
                                return true; // 对任意类型生效
                            // buff作用范围向后覆盖生效
                            // 存在重定向时，range与type全匹配时生效，redirect向后覆盖生效
                            else if (param.redirect) {
                                if (skillRange.some(ST => buffRange.some(BT => BT === ST)))
                                    return true;
                                const redirect = Array.isArray(param.redirect) ? param.redirect : [param.redirect];
                                if (buffRange.some(BT => redirect.some(RT => RT.startsWith(BT))))
                                    return true;
                                return false;
                            }
                            // 不存在重定向时，range向后覆盖生效
                            return skillRange.some(ST => buffRange.some(BT => ST.startsWith(BT)));
                        }
                        // 00
                        return true;
                    })();
                    if (!judge)
                        return false;
                    for (const key in param) {
                        if (key === 'redirect' || key === 'range')
                            continue;
                        if (key === 'element') {
                            if (!buff.element || !param.element)
                                continue; // 对任意属性生效
                            if (Array.isArray(buff.element)) {
                                if (buff.element.includes(param.element))
                                    continue;
                                return false;
                            }
                        }
                        // @ts-expect-error
                        if (buff[key] !== param[key])
                            return false;
                    }
                    if (buff.check) {
                        if (typeof buff.check === 'number') {
                            if (buff.source === '套装' && (this.setCount[buff.name.replace(/\d$/, '')] < buff.check))
                                return false;
                            else if (buff.source === '影画' && (this.avatar.rank < buff.check))
                                return false;
                        }
                        else if (valueOcalc) {
                            if (weakMapCheck.has(buff)) {
                                // console.log(`depth：${depth} ${buff.name}：${weakMapCheck.get(buff)}`)
                                if (!weakMapCheck.get(buff))
                                    return false;
                            }
                            else {
                                weakMapCheck.set(buff, false);
                                if (!buff.check({
                                    avatar: this.avatar,
                                    buffM: this,
                                    calc: valueOcalc,
                                    runtime: exports.runtime
                                }))
                                    return false;
                                weakMapCheck.set(buff, true);
                            }
                        }
                        else {
                            logger.debug('未传入calc：' + buff.name);
                            return false;
                        }
                    }
                    if (buff.teamTarget) {
                        if (typeof buff.teamTarget === 'function') {
                            const result = buff.teamTarget({ teammates: [], avatar: this.avatar, buffM: this, calc: valueOcalc, runtime: exports.runtime });
                            if (Array.isArray(result))
                                return result.includes(this.avatar);
                            return result;
                        }
                        return buff.teamTarget;
                    }
                    return true;
                });
            }
            else {
                buffs = buffs.filter(param);
            }
        }
        catch (e) {
            logger.error(e);
        }
        if (--depth === 0) {
            // console.log('重置weakMapCheck')
            weakMapCheck = new WeakMap();
        }
        return buffs;
    }
    filter(param, valueOcalc) {
        // @ts-expect-error
        return this._filter(this.buffs, param, valueOcalc);
    }
    /** 遍历buff列表 */
    forEach(fnc) {
        return this.buffs.forEach(fnc);
    }
    /** 查找指定buff */
    find(type, value) {
        return this.buffs.find(buff => buff[type] === value);
    }
    operator(key, value, fnc) {
        const isMatch = typeof key === 'object' ?
            (targetBuff) => Object.entries(key).every(([k, v]) => targetBuff[k] === v) :
            (targetBuff) => targetBuff[key] === value;
        this.forEach(buff => isMatch(buff) && (fnc || value)(buff));
    }
    close(key, value) {
        if (typeof key === 'object')
            this.operator(key, buff => buff.status = false);
        else
            this.operator(key, value, buff => buff.status = false);
    }
    open(key, value) {
        if (typeof key === 'object')
            this.operator(key, buff => buff.status = true);
        else
            this.operator(key, value, buff => buff.status = true);
    }
    default(param, value) {
        if (typeof param === 'object') {
            this.defaultBuff = param;
        }
        else {
            if (value === undefined)
                delete this.defaultBuff[param];
            else
                this.defaultBuff[param] = value;
        }
    }
}
exports.BuffManager = BuffManager;

return exports;})();
const {BuffManager,runtime,elementType2element,anomalyEnum,elementEnum}=buffRuntime;
const {Calculator}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Calculator = void 0;
const subBaseValueData = {
    "生命值百分比": [0.03, '3.0%'],
    "生命值": [112, '112'],
    "攻击力百分比": [0.03, '3.0%'],
    "攻击力": [19, '19'],
    "防御力百分比": [0.048, '4.8%'],
    "防御力": [15, '15'],
    "暴击率": [0.024, '2.4%'],
    "暴击伤害": [0.048, '4.8%'],
    "穿透值": [9, '9'],
    "异常精通": [9, '9']
};
const mainBaseValueData = {
    "生命值百分比": [0.3, '30%'],
    "攻击力百分比": [0.3, '30%'],
    "防御力百分比": [0.48, '48%'],
    "暴击率": [0.24, '24%'],
    "暴击伤害": [0.48, '48%'],
    "异常精通": [92, '92'],
    "穿透率": [0.24, '24%'],
    "物理属性伤害加成": [0.3, '30%'],
    "火属性伤害加成": [0.3, '30%'],
    "冰属性伤害加成": [0.3, '30%'],
    "电属性伤害加成": [0.3, '30%'],
    "以太属性伤害加成": [0.3, '30%'],
    "异常掌控": [0.3, '30%'],
    "冲击力": [0.18, '18%'],
    "能量自动回复": [0.6, '60%']
};
const AnomalyData = getMapData('AnomalyData');
const ratioAble = new Set(['生命值', '防御力', '攻击力', '冲击力', '异常掌控']);
class Calculator {
    constructor(buffM) {
        this.skills = [];
        /** 对当前所计算的技能有用的buff、计算后的buff */
        this.usefulBuffResults = new Map();
        /** 技能伤害缓存 */
        this.cache = Object.create(null);
        /** 角色属性缓存 */
        this.props = {};
        this.defaultSkill = {};
        this.buffM = buffM;
        this.avatar = this.buffM.avatar;
        this.enemy = {
            level: this.avatar.level,
            basicDEF: 50,
            resistance: -0.2
        };
    }
    get base_properties() {
        return this.avatar.base_properties;
    }
    get initial_properties() {
        return this.avatar.initial_properties;
    }
    debug(...args) {
        if (!settings.getConfig('config').damage_debug_log)
            return;
        logger.debug(...args);
    }
    defEnemy(param, value) {
        if (typeof param === 'string' && value !== undefined) {
            this.enemy[param] = value;
        }
        else if (typeof param === 'object') {
            _.merge(this.enemy, param);
        }
    }
    new(skill) {
        var _a, _b;
        if (Array.isArray(skill)) {
            skill.forEach(s => this.new(s));
            return this.skills;
        }
        const oriSkill = skill;
        skill = { ...this.defaultSkill, ...skill };
        if (!skill.element)
            skill.element = oriSkill.element = elementType2element(this.avatar.element_type);
        for (const key of ['name', 'type']) {
            if (!skill[key])
                return logger.warn(`无效skill：缺少${key}字段`, skill);
        }
        if (skill.check && +skill.check) {
            const num = skill.check;
            skill.check = oriSkill.check = ({ avatar }) => avatar.rank >= num;
        }
        (_a = skill.isAnomalyDMG) !== null && _a !== void 0 ? _a : (skill.isAnomalyDMG = oriSkill.isAnomalyDMG = typeof anomalyEnum[skill.type.slice(0, 2)] === 'number');
        (_b = skill.isSheerDMG) !== null && _b !== void 0 ? _b : (skill.isSheerDMG = oriSkill.isSheerDMG = this.avatar.avatar_profession === runtime.professionEnum.命破 && elementType2element(this.avatar.element_type) === skill.element && !skill.isAnomalyDMG);
        this.skills.push(skill);
        return this.skills;
    }
    /** 查找指定已注册技能 */
    find_skill(key, value) {
        return this.skills.find(skill => skill[key] === value);
    }
    calc_skill(skill) {
        var _a, _b, _c, _d, _e, _f, _g, _h, _j;
        if (typeof skill === 'string') {
            const MySkill = this.find_skill('type', skill);
            if (!MySkill)
                return;
            return this.calc_skill(MySkill);
        }
        this.skill = skill;
        if (!skill.banCache && this.cache[skill.type])
            return this.cache[skill.type];
        if (skill.check && !skill.check({ avatar: this.avatar, buffM: this.buffM, calc: this, runtime }))
            return;
        this.debug(`${logger.green(skill.type)}${skill.name}伤害计算：`);
        if (skill.dmg) {
            const dmg = skill.dmg(this);
            if (!dmg.skill || dmg.skill.name !== skill.name) {
                dmg.skill = skill;
            }
            this.debug('自定义计算最终伤害：', dmg.result);
            this.usefulBuffResults.clear();
            return dmg;
        }
        const props = this.props = skill.props || {};
        // 缩小筛选范围
        const usefulBuffs = this.buffM.filter({
            element: skill.element,
            range: [skill.type],
            redirect: skill.redirect
        }, this);
        const areas = {};
        if (skill.before)
            skill.before({ avatar: this.avatar, calc: this, usefulBuffs, skill, props, areas, runtime });
        this.debug(`有效buff*${usefulBuffs.length}/${this.buffM.buffs.length}`);
        const { isAnomalyDMG = false, isSheerDMG = false } = skill;
        // 基础伤害区
        if (!areas.BasicArea) {
            let Multiplier = props.倍率;
            if (!Multiplier) {
                if (skill.multiplier) { // 显式指定
                    switch (typeof skill.multiplier) {
                        case 'number':
                            Multiplier = skill.multiplier;
                            break;
                        case 'string':
                            Multiplier = this.get_SkillMultiplier(skill.multiplier);
                            break;
                        case 'object':
                            Multiplier = skill.multiplier[this.get_SkillLevel(skill.type[0]) - 1];
                            break;
                        case 'function':
                            Multiplier = skill.multiplier({ avatar: this.avatar, buffM: this.buffM, calc: this, runtime });
                            break;
                        default:
                            Multiplier = this.get_SkillMultiplier(skill.type);
                            logger.warn('无效的技能倍率：', skill);
                    }
                }
                else if (isAnomalyDMG) {
                    Multiplier = (skill.type.startsWith('紊乱') ?
                        this.get_DiscoverMultiplier(skill) :
                        this.get_AnomalyMultiplier(skill, usefulBuffs, skill.name.includes('每') ? 1 : 0)) || 0;
                }
                else {
                    Multiplier = this.get_SkillMultiplier(skill.type);
                }
                const ExtraMultiplier = this.get_ExtraMultiplier(skill, usefulBuffs);
                Multiplier += ExtraMultiplier;
                if (!Multiplier) {
                    this.usefulBuffResults.clear();
                    return logger.warn('技能倍率缺失：', skill);
                }
                if (ExtraMultiplier)
                    this.debug(`最终倍率：${Multiplier}`);
            }
            props.倍率 = Multiplier;
            if (isSheerDMG) {
                areas.BasicArea = this.get_SheerForce(skill, usefulBuffs) * Multiplier;
            }
            else {
                areas.BasicArea = this.get_ATK(skill, usefulBuffs) * Multiplier;
            }
        }
        this.debug(`基础伤害区：${areas.BasicArea}`);
        // 暴击区
        let CRITRate = 0, CRITDMG = 0;
        if (!areas.CriticalArea) {
            if (isAnomalyDMG) {
                if (!skill.type.startsWith('紊乱')) { // 紊乱暂无异常暴击区
                    CRITRate = this.get_AnomalyCRITRate(skill, usefulBuffs);
                    CRITDMG = this.get_AnomalyCRITDMG(skill, usefulBuffs);
                }
            }
            else {
                CRITRate = this.get_CRITRate(skill, usefulBuffs);
                CRITDMG = this.get_CRITDMG(skill, usefulBuffs);
            }
            areas.CriticalArea = 1 + CRITRate * CRITDMG;
        }
        areas.CriticalArea !== 1 && this.debug(`暴击期望：${areas.CriticalArea}`);
        // 通用乘区
        (_a = areas.BoostArea) !== null && _a !== void 0 ? _a : (areas.BoostArea = this.get_BoostArea(skill, usefulBuffs));
        (_b = areas.VulnerabilityArea) !== null && _b !== void 0 ? _b : (areas.VulnerabilityArea = this.get_VulnerabilityArea(skill, usefulBuffs));
        (_c = areas.ResistanceArea) !== null && _c !== void 0 ? _c : (areas.ResistanceArea = this.get_ResistanceArea(skill, usefulBuffs));
        (_d = areas.DefenceArea) !== null && _d !== void 0 ? _d : (areas.DefenceArea = isSheerDMG ? 1 : this.get_DefenceArea(skill, usefulBuffs));
        (_e = areas.StunVulnerabilityArea) !== null && _e !== void 0 ? _e : (areas.StunVulnerabilityArea = this.get_StunVulnerabilityArea(skill, usefulBuffs));
        // 异常乘区
        if (isAnomalyDMG) {
            (_f = areas.AnomalyProficiencyArea) !== null && _f !== void 0 ? _f : (areas.AnomalyProficiencyArea = this.get_AnomalyProficiencyArea(skill, usefulBuffs));
            (_g = areas.AnomalyBoostArea) !== null && _g !== void 0 ? _g : (areas.AnomalyBoostArea = this.get_AnomalyBoostArea(skill, usefulBuffs));
            (_h = areas.LevelArea) !== null && _h !== void 0 ? _h : (areas.LevelArea = this.get_LevelArea());
        }
        // 贯穿乘区
        if (isSheerDMG) {
            (_j = areas.SheerBoostArea) !== null && _j !== void 0 ? _j : (areas.SheerBoostArea = this.get_SheerBoostArea(skill, usefulBuffs));
        }
        const { BasicArea, CriticalArea, BoostArea, VulnerabilityArea, ResistanceArea, DefenceArea, AnomalyProficiencyArea, LevelArea, AnomalyBoostArea, StunVulnerabilityArea, SheerBoostArea = 1 } = areas;
        const commonArea = BasicArea * BoostArea * VulnerabilityArea * ResistanceArea * DefenceArea * StunVulnerabilityArea;
        const result = isAnomalyDMG ?
            {
                critDMG: (CriticalArea !== 1) ? commonArea * (CRITDMG + 1) * AnomalyProficiencyArea * LevelArea * AnomalyBoostArea : 0,
                expectDMG: commonArea * CriticalArea * AnomalyProficiencyArea * LevelArea * AnomalyBoostArea
            } : {
            critDMG: commonArea * (CRITDMG + 1) * SheerBoostArea,
            expectDMG: commonArea * CriticalArea * SheerBoostArea
        };
        const damageHandler = {
            fnc: (fnc) => {
                damage.result.critDMG = fnc(damage.result.critDMG);
                damage.result.expectDMG = fnc(damage.result.expectDMG);
            },
            x: (n) => {
                this.debug('伤害系数：' + n);
                damage.fnc(v => v * n);
            },
            add: (d) => {
                if (typeof d === 'string')
                    d = this.calc_skill(d);
                if (!d)
                    return;
                this.debug('增加伤害：' + d.skill.name, d.result);
                damage.result.expectDMG += d.result.expectDMG;
                damage.result.critDMG += d.result.critDMG || d.result.expectDMG;
            },
            del: (d) => {
                if (typeof d === 'string')
                    d = this.calc_skill(d);
                if (!d)
                    return;
                this.debug('减少伤害：' + d.skill.name, d.result);
                damage.result.expectDMG -= d.result.expectDMG;
                damage.result.critDMG -= d.result.critDMG || d.result.expectDMG;
            }
        };
        const damage = new Proxy({
            skill,
            usefulBuffs: _.sortBy(Array.from(this.usefulBuffResults.values()), ['type', 'value']).reverse(),
            props,
            areas,
            result
        }, {
            get: (target, prop) => {
                if (prop in damageHandler) {
                    return damageHandler[prop];
                }
                return target[prop];
            }
        });
        if (skill.after) {
            skill.after({ avatar: this.avatar, calc: this, usefulBuffs, skill, damage, runtime });
        }
        this.debug('最终伤害：', result);
        if (!skill.banCache)
            this.cache[skill.type] = damage;
        this.usefulBuffResults.clear();
        // console.log(damage)
        return damage;
    }
    calc_showInPanel_buffs() {
        return this.buffM.buffs
            .filter(buff => buff.showInPanel)
            .map(buff => {
            try {
                const value = this.calc_final_value(buff);
                // 计算buff最大值
                let max = 0;
                // 已指定max，直接获取
                if (buff.max) {
                    if (buff.max === Infinity) {
                        max = 0;
                    }
                    else {
                        max = this.calc_final_value({
                            ...buff,
                            value: buff.max,
                            max: undefined
                        });
                    }
                }
                else {
                    // 未指定max，自动计算理论最大值
                    const { _base_properties, _initial_properties } = this.avatar;
                    // @ts-expect-error
                    this.avatar._base_properties = this.avatar._initial_properties = new Proxy({}, {
                        get: (target, prop) => {
                            // this.debug(`计算buff理论最大值，访问属性：${String(prop)}`)
                            return Number.MAX_SAFE_INTEGER;
                        }
                    });
                    try {
                        this.props = {};
                        max = this.calc_final_value(buff);
                    }
                    catch { }
                    this.avatar._base_properties = _base_properties;
                    this.avatar._initial_properties = _initial_properties;
                }
                if (max === Infinity || !max || max > 9999) {
                    max = 0;
                }
                return { ...buff, value, max };
            }
            catch (e) {
                logger.error('buff计算错误：', buff, e);
                return;
            }
        })
            .filter(v => v && v.value);
    }
    calc() {
        return this.skills.map(skill => {
            try {
                return this.calc_skill(skill);
            }
            catch (e) {
                logger.error('伤害计算错误：', e);
                return;
            }
        }).filter(v => { var _a, _b; return v && ((_a = v.result) === null || _a === void 0 ? void 0 : _a.expectDMG) && !((_b = v.skill) === null || _b === void 0 ? void 0 : _b.isHide); });
    }
    /**
     * 计算副词条伤害差异
     * @param types 需进行比较的词条数组
     */
    calc_sub_differences(skill, types) {
        // 未指定types时，筛选评分权重大于0的词条进行差异计算
        if (!types || !types.length) {
            types = Object.entries(this.avatar.scoreWeight)
                .reduce((acc, [id, weight]) => {
                if (weight > 0) {
                    const type = property.idToName(id);
                    if (type && subBaseValueData[type]) {
                        acc.push({ type, weight });
                    }
                }
                return acc;
            }, [])
                .sort((a, b) => b.weight - a.weight) // 按权重从大到小排序
                .slice(0, 6) // 默认最多6个
                .map(({ type }) => type);
        }
        const base = {};
        types.forEach(t => base[t] = t.includes('百分比') ? this.base_properties[property.nameZHToNameEN(t.replace('百分比', ''))] * subBaseValueData[t][0] : subBaseValueData[t][0]);
        this.debug(logger.red('副词条差异计算变化值：'), base);
        const buffs = types.map(t => ({
            name: t,
            shortName: property.nameToShortName3(t),
            type: t.replace('百分比', ''),
            value: base[t],
            valueBase: subBaseValueData[t][1]
        }));
        buffs.push({
            // @ts-expect-error
            name: '空白对照',
            shortName: '对照组',
            // @ts-expect-error
            type: '',
            value: 0,
            // @ts-expect-error
            valueBase: '0'
        });
        // @ts-expect-error
        return this.calc_differences(buffs, skill);
    }
    /**
     * 计算主词条伤害差异
     * @param types 需进行比较的词条数组
     */
    calc_main_differences(skill, types) {
        var _a;
        // 未指定types时，筛选评分权重大于0的词条进行差异计算
        if (!types || !types.length) {
            types = Object.entries(this.avatar.scoreWeight)
                .reduce((acc, [id, weight]) => {
                if (weight > 0) {
                    const type = property.idToName(id);
                    if (type && mainBaseValueData[type]) {
                        acc.push({ type, weight });
                    }
                }
                return acc;
            }, [])
                .sort((a, b) => b.weight - a.weight) // 按权重从大到小排序
                .slice(0, 8)
                .map(({ type }) => type);
        }
        const base = {};
        types.forEach(t => base[t] = (t.includes('百分比') || ['异常掌控', '冲击力', '能量自动回复'].includes(t)) ?
            this.base_properties[property.nameZHToNameEN(t.replace('百分比', ''))] * mainBaseValueData[t][0] :
            mainBaseValueData[t][0]);
        this.debug(logger.red('主词条差异计算变化值：'), base);
        const buffs = types.map(t => {
            const data = {
                name: t,
                shortName: property.nameToShortName3(t),
                type: (t.includes('属性伤害加成') ? '增伤' : t.replace('百分比', '')),
                value: base[t],
                element: (t.includes('属性伤害加成') ? property.nameZHToNameEN(t).replace('DMGBonus', '') : ''),
                valueBase: mainBaseValueData[t][1]
            };
            if (!data.element)
                delete data.element;
            return data;
        });
        buffs.push({
            // @ts-expect-error
            name: '空白对照',
            shortName: '对照组',
            // @ts-expect-error
            type: '',
            value: 0,
            // @ts-expect-error
            valueBase: '0'
        });
        const equips = ((_a = this.avatar.equip) === null || _a === void 0 ? void 0 : _a.reduce((acc, e) => {
            var _a;
            if (e.equipment_type < 4)
                return acc;
            const name = (_a = e.main_properties[0]) === null || _a === void 0 ? void 0 : _a.property_name;
            if (name)
                acc.push(name);
            return acc;
        }, ['空白对照'])) || [];
        // @ts-expect-error 只保留装备的主词条del
        const main_differences = this.calc_differences(buffs, skill);
        return main_differences.filter(v => {
            const name1 = v[0].del.name.replace('百分比', '');
            const name2 = name1.replace('属性', '');
            return equips.some(e => e === name1 || e === name2);
        });
    }
    calc_differences(buffs, skill) {
        var _a, _b;
        if (!skill) {
            skill = this.find_skill('isMain', true) // 主技能
                || ((_a = this.calc().sort((a, b) => b.result.expectDMG - a.result.expectDMG)[0]) === null || _a === void 0 ? void 0 : _a.skill); // 伤害最高技能
        }
        else if (typeof skill === 'string') {
            const MySkill = this.find_skill('type', skill);
            if (!MySkill)
                return [];
            return this.calc_differences(buffs, MySkill);
        }
        const oriDamage = this.calc_skill(skill);
        this.cache = Object.create(null);
        const result = [];
        for (const i_del in buffs) {
            result[i_del] = [];
            const buff_del = buffs[i_del];
            const { name: name_del = buff_del.type, value: value_del } = buff_del;
            this.debug(logger.blue(`差异计算：${name_del}`));
            this.buffM.buffs.push({
                ...buff_del,
                name: logger.green(`差异计算：${name_del}`),
                value: ({ calc }) => -calc.calc_value(value_del) // 转为负值
            });
            for (const i_add in buffs) {
                const buff_add = buffs[i_add];
                (_b = buff_add.name) !== null && _b !== void 0 ? _b : (buff_add.name = buff_add.type);
                const data = result[i_del][i_add] = {
                    add: buff_add,
                    del: buff_del,
                    damage: oriDamage,
                    difference: 0
                };
                const { name: name_add = buff_add.type } = buff_add;
                if (name_del === name_add)
                    continue;
                this.debug(logger.yellow(`差异计算：${name_del}->${name_add}`));
                this.buffM.buffs.push({
                    ...buff_add,
                    name: logger.green(`差异计算：${name_del}->${name_add}`)
                });
                const newDamage = this.calc_skill(skill);
                this.buffM.buffs.pop();
                this.cache = Object.create(null);
                data.damage = newDamage;
                data.difference = newDamage.result.expectDMG - oriDamage.result.expectDMG;
                this.debug(logger.magenta(`差异计算：${name_del}->${name_add} 伤害变化：${data.difference}`));
            }
            this.buffM.buffs.pop();
        }
        return result;
    }
    default(param, value) {
        if (typeof param === 'object') {
            this.defaultSkill = param;
        }
        else {
            if (value === undefined)
                delete this.defaultSkill[param];
            else
                this.defaultSkill[param] = value;
        }
    }
    /**
     * 获取技能等级
     * @param baseType 技能基类 'A', 'E', 'C', 'R', 'T', 'L'
     * @see [技能命名标准](https://github.com/ZZZure/ZZZ-Plugin/blob/dev/src/model/damage/README.md#技能类型命名标准)
     */
    get_SkillLevel(baseType) {
        var _a;
        const id = ['A', 'E', 'C', 'R', , 'T', 'L'].indexOf(baseType);
        if (id === -1)
            return 1;
        return Number(((_a = this.avatar.skills.find(({ skill_type }) => skill_type === id)) === null || _a === void 0 ? void 0 : _a.level) || 1);
    }
    /**
     * 获取技能倍率
     * @param type 参见技能命名标准
     * @see [技能命名标准](https://github.com/ZZZure/ZZZ-Plugin/blob/dev/src/model/damage/README.md#技能类型命名标准)
     */
    get_SkillMultiplier(type) {
        var _a;
        const SkillLevel = this.get_SkillLevel(type[0]);
        this.debug(`${type[0]}等级：${SkillLevel}`);
        const Multiplier = (_a = charData[this.avatar.id].skill[type]) === null || _a === void 0 ? void 0 : _a[SkillLevel - 1];
        this.debug(`技能倍率：${Multiplier}`);
        return Multiplier;
    }
    get_AnomalyData(anomaly) {
        if (!anomaly) {
            return AnomalyData.find(({ element_type, sub_element_type, multiplier }) => multiplier &&
                element_type === this.avatar.element_type &&
                sub_element_type === this.avatar.sub_element_type);
        }
        let a = AnomalyData.filter(({ element_type }) => element_type === this.avatar.element_type);
        if (anomaly === '紊乱')
            a = a.filter(({ discover }) => discover);
        else
            a = a.filter(({ name, multiplier }) => name === anomaly && multiplier);
        if (a.length === 1)
            return a[0];
        a = a.filter(({ sub_element_type }) => sub_element_type === this.avatar.sub_element_type);
        return a[0];
    }
    /** 获取属性异常倍率 */
    get_AnomalyMultiplier(skill, usefulBuffs, times = 0) {
        const anomalyData = this.get_AnomalyData(skill === null || skill === void 0 ? void 0 : skill.type.slice(0, 2));
        if (!anomalyData)
            return;
        // 未指定触发次数时，自动计算最大触发次数
        if (!times && anomalyData.duration && anomalyData.interval) {
            const AnomalyDuration = this.get_AnomalyDuration(skill, usefulBuffs, anomalyData.duration);
            times = Math.floor((AnomalyDuration * 10) / (anomalyData.interval * 10));
        }
        const Multiplier = anomalyData.multiplier * (times || 1);
        this.debug(`倍率：${Multiplier}`);
        return Multiplier;
    }
    /** 获取紊乱倍率 */
    get_DiscoverMultiplier(skill) {
        const anomalyData = this.get_AnomalyData(skill === null || skill === void 0 ? void 0 : skill.type.slice(0, 2));
        if (!anomalyData)
            return;
        const AnomalyDuration = this.get_AnomalyDuration({
            ...skill,
            name: anomalyData.name,
            type: anomalyData.name
        }, this.buffM.buffs, anomalyData.duration);
        const times = Math.floor((AnomalyDuration * 10) / (anomalyData.interval * 10));
        const discover = anomalyData.discover;
        const Multiplier = discover.fixed_multiplier + times * discover.multiplier;
        this.debug(`${anomalyData.name}紊乱倍率：${Multiplier}`);
        return Multiplier;
    }
    /** 计算buff增益值（可能为百分比） */
    calc_value(value, buff) {
        var _a, _b;
        switch (typeof value) {
            case 'number': return value;
            case 'function': {
                if (buff)
                    buff.status = false;
                const v = +value({ avatar: this.avatar, buffM: this.buffM, calc: this, runtime }) || 0;
                if (buff)
                    buff.status = true;
                return v;
            }
            case 'string': return ((_b = (_a = charData[this.avatar.id].buff) === null || _a === void 0 ? void 0 : _a[value]) === null || _b === void 0 ? void 0 : _b[this.get_SkillLevel(value[0]) - 1]) || 0;
            case 'object': {
                if (!Array.isArray(value) || !buff)
                    return 0;
                switch (buff.source) {
                    case '音擎': return this.avatar.weapon ? value[this.avatar.weapon.star - 1] || 0 : 0;
                    case '核心被动':
                    case '额外能力': return value[this.get_SkillLevel('T') - 1] || 0;
                }
            }
            default: return 0;
        }
    }
    /**
     * 计算buff增益最终值
     * - `buff.value`为数值/字符串/数组类型且计算结果绝对值<1时按 **`初始数值`** 百分比提高处理
     * @param buff 待计算buff
     * @param initial 指定初始数值，若不指定则根据角色初始属性和buff类型自动获取
     */
    calc_final_value(buff) {
        let initial;
        const _calc_final_value = (buff) => {
            const { value } = buff;
            const add = this.calc_value(value, buff);
            if (!add || !ratioAble.has(buff.type) || Math.abs(add) >= 1)
                return add;
            if (!(typeof value === 'number' || typeof value === 'string' || Array.isArray(value)))
                return add;
            if (!initial) {
                if (buff.percentBase === 'initial') {
                    initial = this.initial_properties[property.nameZHToNameEN(buff.type)] || 0;
                }
                else {
                    initial = this.base_properties[property.nameZHToNameEN(buff.type)] || 0;
                }
            }
            if (!initial)
                return add;
            return add * initial;
        };
        const value = _calc_final_value(buff);
        if (!value || !buff.max)
            return value;
        const max = _calc_final_value({ ...buff, value: buff.max });
        return Math.min(value, max);
    }
    /**
     * 获取局内属性原始值
     */
    get(type, initial, skill = this.skill, usefulBuffs = this.buffM.buffs) {
        var _a;
        var _b;
        const nonStackableBuffRecord = new Map();
        return (_a = (_b = this.props)[type]) !== null && _a !== void 0 ? _a : (_b[type] = this.buffM._filter(usefulBuffs, {
            element: skill === null || skill === void 0 ? void 0 : skill.element,
            range: [skill === null || skill === void 0 ? void 0 : skill.type],
            redirect: skill === null || skill === void 0 ? void 0 : skill.redirect,
            type
        }, this).reduce((previousValue, buff) => {
            // 计算最终增益值
            const add = this.calc_final_value(buff);
            // 检查不可叠加buff
            if (buff.stackable === false) {
                const recorded = nonStackableBuffRecord.get(buff.name);
                if (recorded) {
                    const recordedValue = this.usefulBuffResults.get(recorded).value;
                    if (Math.abs(recordedValue) >= Math.abs(add)) {
                        this.debug(`\tBuff：${buff.name}已存在，且数值相同/更高，不计入结果`);
                        return previousValue;
                    }
                    this.debug(`\tBuff：${buff.name}已存在，且数值更低，替换为更高数值`);
                    previousValue -= recordedValue;
                    this.usefulBuffResults.delete(recorded);
                }
                nonStackableBuffRecord.set(buff.name, buff);
            }
            if (!this.usefulBuffResults.has(buff))
                this.usefulBuffResults.set(buff, { ...buff, value: add });
            this.debug(`\tBuff：${buff.name}对${(buff.include ? (buff.range ? [buff.range, buff.include] : buff.include) : buff.range) || '全类型'}增加${add}${buff.element || ''}${type}`);
            return previousValue + add;
        }, initial));
    }
    /** 攻击力 */
    get_ATK(skill, usefulBuffs) {
        let ATK = this.get('攻击力', this.initial_properties.ATK, skill, usefulBuffs);
        ATK = this.min_max(0, 10000, ATK);
        this.debug(`攻击力：${ATK}`);
        return ATK;
    }
    /** 额外倍率 */
    get_ExtraMultiplier(skill, usefulBuffs) {
        const ExtraMultiplier = this.get('倍率', 0, skill, usefulBuffs);
        ExtraMultiplier && this.debug(`额外倍率：${ExtraMultiplier}`);
        return ExtraMultiplier;
    }
    /** 暴击率 */
    get_CRITRate(skill, usefulBuffs) {
        let CRITRate = this.get('暴击率', this.initial_properties.CRITRate, skill, usefulBuffs);
        CRITRate = this.min_max(0, 1, CRITRate);
        this.debug(`暴击率：${CRITRate}`);
        return CRITRate;
    }
    /** 暴击伤害 */
    get_CRITDMG(skill, usefulBuffs) {
        let CRITDMG = this.get('暴击伤害', this.initial_properties.CRITDMG, skill, usefulBuffs);
        CRITDMG = this.min_max(0, 5, CRITDMG);
        this.debug(`暴击伤害：${CRITDMG}`);
        return CRITDMG;
    }
    /** 增伤区 */
    get_BoostArea(skill, usefulBuffs) {
        let BoostArea = this.get('增伤', 1, skill, usefulBuffs);
        BoostArea = this.min_max(0, 6, BoostArea);
        this.debug(`增伤区：${BoostArea}`);
        return BoostArea;
    }
    /** (减)易伤区 */
    get_VulnerabilityArea(skill, usefulBuffs) {
        let VulnerabilityArea = this.get('易伤', 1, skill, usefulBuffs);
        VulnerabilityArea = this.min_max(0.2, 2, VulnerabilityArea);
        this.debug(`易伤区：${VulnerabilityArea}`);
        return VulnerabilityArea;
    }
    /** 失衡易伤区 */
    get_StunVulnerabilityArea(skill, usefulBuffs) {
        let StunVulnerabilityArea = this.get('失衡易伤', 1, skill, usefulBuffs);
        StunVulnerabilityArea = this.min_max(0.2, 5, StunVulnerabilityArea);
        StunVulnerabilityArea !== 1 && this.debug(`失衡易伤区：${StunVulnerabilityArea}`);
        return StunVulnerabilityArea;
    }
    /** 抗性区 */
    get_ResistanceArea(skill, usefulBuffs) {
        let ResistanceArea = this.get('无视抗性', 1 - this.enemy.resistance, skill, usefulBuffs);
        ResistanceArea = this.min_max(0, 2, ResistanceArea);
        this.debug(`抗性区：${ResistanceArea}`);
        return ResistanceArea;
    }
    /** 无视防御 */
    get_IgnoreDEF(skill, usefulBuffs) {
        const IgnoreDEF = this.get('无视防御', 0, skill, usefulBuffs);
        IgnoreDEF && this.debug(`无视防御：${IgnoreDEF}`);
        return IgnoreDEF;
    }
    /** 穿透值 */
    get_Pen(skill, usefulBuffs) {
        let Pen = this.get('穿透值', this.initial_properties.Pen, skill, usefulBuffs);
        Pen = Math.min(Pen, 1000);
        Pen && this.debug(`穿透值：${Pen}`);
        return Pen;
    }
    /** 穿透率 */
    get_PenRatio(skill, usefulBuffs) {
        let PenRatio = this.get('穿透率', this.initial_properties.PenRatio, skill, usefulBuffs);
        PenRatio = Math.min(PenRatio, 2);
        PenRatio && this.debug(`穿透率：${PenRatio}`);
        return PenRatio;
    }
    /** 防御区 */
    get_DefenceArea(skill, usefulBuffs) {
        const get_base = (level) => Math.floor(0.1551 * Math.min(60, level) ** 2 + 3.141 * Math.min(60, level) + 47.2039);
        /** 等级基数 */
        const base = get_base(this.avatar.level);
        /** 基础防御 */
        const DEF = this.enemy.basicDEF / 50 * get_base(this.enemy.level);
        const IgnoreDEF = this.get_IgnoreDEF(skill, usefulBuffs);
        const Pen = this.get_Pen(skill, usefulBuffs);
        const PenRatio = this.get_PenRatio(skill, usefulBuffs);
        /** 防御 */
        const defence = DEF * (1 - IgnoreDEF);
        /** 有效防御 */
        const effective_defence = Math.max(0, defence * (1 - PenRatio) - Pen);
        const DefenceArea = this.min_max(0, 1, base / (effective_defence + base));
        this.debug(`防御区：${DefenceArea}`);
        return DefenceArea;
    }
    /** 等级区 */
    get_LevelArea(level = this.avatar.level) {
        const LevelArea = +(1 + 1 / 59 * (level - 1)).toFixed(4);
        this.debug(`等级区：${LevelArea}`);
        return LevelArea;
    }
    /** 异常精通 */
    get_AnomalyProficiency(skill, usefulBuffs) {
        const AnomalyProficiency = this.get('异常精通', this.initial_properties.AnomalyProficiency, skill, usefulBuffs);
        this.debug(`异常精通：${AnomalyProficiency}`);
        return AnomalyProficiency;
    }
    /** 异常掌控 */
    get_AnomalyMastery(skill, usefulBuffs) {
        let AnomalyMastery = this.get('异常掌控', this.initial_properties.AnomalyMastery, skill, usefulBuffs);
        AnomalyMastery = this.min_max(0, 1000, AnomalyMastery);
        this.debug(`异常掌控：${AnomalyMastery}`);
        return AnomalyMastery;
    }
    /** 异常精通区 */
    get_AnomalyProficiencyArea(skill, usefulBuffs) {
        const AnomalyProficiency = this.get_AnomalyProficiency(skill, usefulBuffs);
        const AnomalyProficiencyArea = this.min_max(0, 10, AnomalyProficiency / 100);
        this.debug(`异常精通区：${AnomalyProficiencyArea}`);
        return AnomalyProficiencyArea;
    }
    /** 异常增伤区 */
    get_AnomalyBoostArea(skill, usefulBuffs) {
        let AnomalyBoostArea = this.get('异常增伤', 1, skill, usefulBuffs);
        AnomalyBoostArea = this.min_max(0, 3, AnomalyBoostArea);
        AnomalyBoostArea !== 1 && this.debug(`异常增伤区：${AnomalyBoostArea}`);
        return AnomalyBoostArea;
    }
    /** 异常暴击率 */
    get_AnomalyCRITRate(skill, usefulBuffs) {
        let AnomalyCRITRate = this.get('异常暴击率', 0, skill, usefulBuffs);
        AnomalyCRITRate = this.min_max(0, 1, AnomalyCRITRate);
        AnomalyCRITRate && this.debug(`异常暴击率：${AnomalyCRITRate}`);
        return AnomalyCRITRate;
    }
    /** 异常暴击伤害 */
    get_AnomalyCRITDMG(skill, usefulBuffs) {
        let AnomalyCRITDMG = this.get('异常暴击伤害', 0, skill, usefulBuffs);
        AnomalyCRITDMG = this.min_max(0, 5, AnomalyCRITDMG);
        AnomalyCRITDMG && this.debug(`异常暴击伤害：${AnomalyCRITDMG}`);
        return AnomalyCRITDMG;
    }
    /** 异常持续时间 */
    get_AnomalyDuration(skill, usefulBuffs, duration = 0) {
        const AnomalyDuration = +this.get('异常持续时间', duration, skill, usefulBuffs).toFixed(1);
        this.debug(`异常持续时间：${AnomalyDuration}`);
        return AnomalyDuration;
    }
    /** 生命值 */
    get_HP(skill, usefulBuffs) {
        let HP = this.get('生命值', this.initial_properties.HP, skill, usefulBuffs);
        HP = this.min_max(0, 100000, HP);
        this.debug(`生命值：${HP}`);
        return HP;
    }
    /** 防御力 */
    get_DEF(skill, usefulBuffs) {
        let DEF = this.get('防御力', this.initial_properties.DEF, skill, usefulBuffs);
        DEF = this.min_max(0, 1000, DEF);
        this.debug(`防御力：${DEF}`);
        return DEF;
    }
    /** 冲击力 */
    get_Impact(skill, usefulBuffs) {
        let Impact = this.get('冲击力', this.initial_properties.Impact, skill, usefulBuffs);
        Impact = this.min_max(0, 1000, Impact);
        this.debug(`冲击力：${Impact}`);
        return Impact;
    }
    /** 贯穿力 */
    get_SheerForce(skill, usefulBuffs) {
        // 默认取 攻击力*0.3
        let SheerForce = Math.trunc(this.get_ATK(skill, usefulBuffs) * 0.3);
        SheerForce = this.get('贯穿力', SheerForce, skill, usefulBuffs);
        SheerForce = this.min_max(0, 10000, SheerForce);
        this.debug(`贯穿力：${SheerForce}`);
        return SheerForce;
    }
    /** 贯穿增伤区 */
    get_SheerBoostArea(skill, usefulBuffs) {
        let SheerBoostArea = this.get('贯穿增伤', 1, skill, usefulBuffs);
        SheerBoostArea = this.min_max(0.2, 9, SheerBoostArea);
        SheerBoostArea !== 1 && this.debug(`贯穿增伤区：${SheerBoostArea}`);
        return SheerBoostArea;
    }
    min_max(min, max, value) {
        return Math.min(Math.max(value, min), max);
    }
}
exports.Calculator = Calculator;

return exports;})();
const EnkaFormat=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Skill = exports.Property = exports.Weapon = exports.Equip = void 0;
exports.parseInfo = parseInfo;
exports.Enka2Mys = Enka2Mys;
var Rarity;
(function (Rarity) {
    Rarity[Rarity["S"] = 4] = "S";
    Rarity[Rarity["A"] = 3] = "A";
    Rarity[Rarity["B"] = 2] = "B";
})(Rarity || (Rarity = {}));
const WeaponId2Data = getMapData('WeaponId2Data');
const PartnerId2Data = getMapData('PartnerId2Data');
const SuitData = getMapData('SuitData');
const id2zh = {
    111: '生命值',
    121: '攻击力',
    122: '冲击力',
    131: '防御力',
    201: '暴击率',
    211: '暴击伤害',
    231: '穿透率',
    232: '穿透值',
    305: '能量自动回复',
    312: '异常精通',
    314: '异常掌控',
    315: '物理伤害加成',
    316: '火属性伤害加成',
    317: '冰属性伤害加成',
    318: '电属性伤害加成',
    319: '以太伤害加成'
};
const zh2id = Object.fromEntries(Object.entries(id2zh).map(([key, value]) => [value, +key]));
const id2en = {
    111: 'HpMax',
    121: 'Attack',
    122: 'BreakStun',
    131: 'Defence',
    201: 'Crit',
    211: 'CritDamage',
    231: 'PenRate',
    232: 'PenDelta',
    305: 'SpRecover',
    312: 'ElementMystery',
    314: 'ElementAbnormalPower',
    315: 'PhysDmgBonus',
    316: 'FireDmgBonus',
    317: 'IceDmgBonus',
    318: 'ThunderDmgBonus',
    319: 'EtherDmgBonus'
};
const en2id = Object.fromEntries(Object.entries(id2en).map(([key, value]) => [value, +key]));
const percentPropId = [11102, 12102, 12202, 13102, 20103, 21103, 23103, 30502, 31402, 31503, 31603, 31703, 31803, 31903];
function get_base(propId, value) {
    if (percentPropId.includes(propId)) {
        const v = value / 100;
        return v.toFixed(v % 1 === 0 ? 0 : 1) + '%';
    }
    return Math.trunc(value).toString();
}
class Equip {
    constructor(enkaEquip) {
        this.enkaEquip = enkaEquip;
        this.Equipment = this.enkaEquip.Equipment;
        this.id = this.Equipment.Id;
        this.data = SuitData[`${this.id.toString().slice(0, 3)}00`];
        if (!this.data) {
            throw new Error(`驱动盘数据缺失: ${this.id}`);
        }
        this.info = this.init();
        this.equip = this.info;
    }
    static main(EquippedList) {
        var _a, _b;
        const equips = [];
        for (const equip of EquippedList) {
            if ((_a = equip.Equipment) === null || _a === void 0 ? void 0 : _a.Id) {
                const e = new Equip(equip);
                equips.push(e.main());
            }
        }
        // 统计own
        const cache = {};
        for (const equip of equips) {
            const suit_id = equip.equip_suit.suit_id;
            (_b = cache[suit_id]) !== null && _b !== void 0 ? _b : (cache[suit_id] = equips.reduce((acc, cur) => {
                if (cur.equip_suit.suit_id === suit_id) {
                    return acc + 1;
                }
                return acc;
            }, 0));
            equip.equip_suit.own = cache[suit_id];
        }
        return equips;
    }
    main() {
        this.equip.properties = this.properties(this.Equipment.RandomPropertyList, false);
        this.equip.main_properties = this.properties(this.Equipment.MainPropertyList, true);
        this.equip.equip_suit = this.equip_suit();
        return this.equip;
    }
    init() {
        return {
            id: this.id,
            level: this.Equipment.Level,
            name: `${this.data.name}[${this.enkaEquip.Slot}]`,
            icon: '',
            rarity: Rarity[+String(this.id)[3]] || 'S',
            equipment_type: this.enkaEquip.Slot,
            invalid_property_cnt: 0,
            all_hit: false
        };
    }
    properties(enkaProperties, isMain) {
        const properties = [];
        for (const p of enkaProperties) {
            const property = {};
            const propId = p.PropertyId;
            property.property_name = id2zh[propId.toString().slice(0, 3)];
            property.property_id = propId;
            const value = p.PropertyValue * (isMain ? (1 + this.info.level * ({
                S: 0.2,
                A: 0.25,
                B: 0.3
            })[this.info.rarity]) : p.PropertyLevel);
            property.base = get_base(propId, value);
            property.level = p.PropertyLevel;
            property.valid = false;
            property.system_id = 0;
            property.add = p.PropertyLevel - 1;
            properties.push(property);
        }
        return properties;
    }
    equip_suit() {
        return {
            suit_id: +`${this.id.toString().slice(0, 3)}00`,
            name: this.data.name,
            own: 0,
            desc1: this.data.desc1,
            desc2: this.data.desc2
        };
    }
}
exports.Equip = Equip;
class Weapon {
    constructor(enkaWeapon) {
        this.enkaWeapon = enkaWeapon;
        this.id = this.enkaWeapon.Id;
        this.data = WeaponId2Data[this.id];
        if (!this.data) {
            throw new Error(`音擎数据缺失: ${this.id}`);
        }
        this.info = this.init();
        this.weapon = this.info;
    }
    static main(enkaWeapon) {
        if (!enkaWeapon)
            return null;
        const weapon = new Weapon(enkaWeapon);
        return weapon.main();
    }
    main() {
        this.weapon.properties = this.properties(false);
        this.weapon.main_properties = this.properties(true);
        return this.weapon;
    }
    init() {
        return {
            id: this.id,
            level: this.enkaWeapon.Level,
            name: this.data.Name,
            star: this.enkaWeapon.UpgradeLevel,
            icon: '',
            rarity: this.data.Rarity,
            talent_title: this.data.Talents[1].Name,
            talent_content: this.data.Talents[1].Desc,
            profession: this.data.Profession
        };
    }
    properties(isMain) {
        const property = {};
        const p = this.data[isMain ? 'BaseProperty' : 'RandProperty'];
        const propId = p.Id;
        property.property_name = id2zh[propId.toString().slice(0, 3)];
        if (isMain && property.property_name === '攻击力') {
            property.property_name = '基础攻击力';
        }
        property.property_id = propId;
        // 计算属性值
        let value;
        const baseValue = p.Value;
        if (isMain) { // 基础属性
            value = baseValue * (1 +
                (this.data.Level[this.info.level].Rate +
                    this.data.Stars[this.enkaWeapon.BreakLevel].StarRate) / 10000);
        }
        else { // 高级属性
            value = baseValue * (1 + this.data.Stars[this.enkaWeapon.BreakLevel].RandRate / 10000);
        }
        property.base = get_base(propId, value);
        property.level = 0;
        property.valid = false;
        property.system_id = 0;
        property.add = 0;
        return [property];
    }
}
exports.Weapon = Weapon;
let isToFixed = true;
class Property {
    constructor(info, equips, weapon, enkaAvatar) {
        this.info = info;
        this.equips = equips;
        this.weapon = weapon;
        this.enkaAvatar = enkaAvatar;
        this.keepPercent = [201, 211, 231, 315, 316, 317, 318, 319];
        this.data = PartnerId2Data[this.info.id];
    }
    static main(info, equips, weapon, enkaAvatar) {
        return new Property(info, equips, weapon, enkaAvatar).main();
    }
    main() {
        const initial = this.initial();
        const properties = initial.map(prop => {
            const propId = prop.property_id;
            const isPercent = this.keepPercent.includes(propId);
            if (!isToFixed) {
                return {
                    property_name: prop.property_name,
                    property_id: prop.property_id,
                    base: isPercent ? `${prop.base}%` : `${prop.base}`,
                    add: isPercent ? `${prop.add}%` : `${prop.add}`,
                    final: isPercent ? `${prop.final}%` : `${prop.final}`,
                };
            }
            const isTofixed2 = propId === 305; // 能量自动回复
            return {
                property_name: prop.property_name,
                property_id: prop.property_id,
                base: isPercent ? `${prop.base.toFixed(1)}%` : `${isTofixed2 ? prop.base.toFixed(2) : prop.base}`,
                add: isPercent ? `${prop.add.toFixed(1)}%` : `${isTofixed2 ? prop.add.toFixed(2) : prop.add}`,
                final: isPercent ? `${prop.final.toFixed(1)}%` : `${isTofixed2 ? prop.final.toFixed(2) : prop.final}`,
            };
        });
        // 筛选属伤
        const elementType2PropId = {
            200: 315,
            201: 316,
            202: 317,
            203: 318,
            205: 319
        };
        const elementIds = Object.values(elementType2PropId);
        const propId = elementType2PropId[this.info.element_type];
        if (propId) {
            return properties.filter(prop => !elementIds.includes(prop.property_id) || prop.property_id === propId);
        }
        return properties;
    }
    /** 计算基础属性 */
    base() {
        // const base = new Proxy({} as Record<IdsString, number>, {
        //   get(target, prop: IdsString) {
        //     return target[prop] || 0
        //   },
        //   set(target, prop: IdsString, value: number) {
        //     console.log(`${prop}: ${String(target[prop]).padStart(10, ' ')} => ${String(value).padEnd(10, ' ')} ${value - target[prop]}`)
        //     target[prop] = value
        //     return true
        //   }
        // })
        const base = {};
        const ids = Object.keys(id2en).map(Number);
        ids.forEach((id) => {
            const prop = id2en[id];
            if (Object.prototype.hasOwnProperty.call(this.data, prop)) {
                // @ts-expect-error
                base[id] = +this.data[prop] || 0;
            }
            else {
                base[id] = 0;
            }
        });
        // 生命、攻击、防御随等级额外提升
        [111, 121, 131].forEach((id) => {
            const prop = id2en[id];
            // 等级提升
            if (this.info.level) {
                const growth = ({
                    111: this.data.HpGrowth,
                    121: this.data.AttackGrowth,
                    131: this.data.DefenceGrowth
                })[id] || 0;
                base[id] += (this.info.level - 1) * growth / 10000;
            }
            // 突破提升
            const PromotionLevel = this.enkaAvatar.PromotionLevel || 0;
            base[id] += this.data.Level[PromotionLevel][prop] || 0;
        });
        // 核心技额外提升
        const CoreSkillEnhancement = this.enkaAvatar.CoreSkillEnhancement || 0;
        if (CoreSkillEnhancement) {
            const extra = this.data.ExtraLevel[CoreSkillEnhancement].Extra;
            const extraIds = Object.keys(extra).map(Number);
            extraIds.forEach((id) => {
                // 攻击力百分比
                if ([12102].includes(id))
                    return;
                base[id.toString().slice(0, 3)] += extra[id].Value || 0;
            });
        }
        // 处理音擎基础属性
        if (this.weapon) {
            for (const property of this.weapon.main_properties) {
                base[property.property_id.toString().slice(0, 3)] += +property.base;
            }
        }
        // console.log('----------------base保留整数----------------')
        // 保留整数
        ids.forEach(id => base[id] = Math.trunc(base[id]));
        [201, 211, 231, 305].forEach(id => base[id] /= 100);
        return base;
    }
    en2id(data) {
        const idData = {};
        const ens = Object.keys(data);
        ens.forEach((en) => {
            const id = en2id[en];
            idData[id] = data[en] || 0;
        });
        return idData;
    }
    en2zh(data) {
        const idData = this.en2id(data);
        return this.id2zh(idData);
    }
    id2zh(idData) {
        const zhData = {};
        const ids = Object.keys(idData).map(Number);
        ids.forEach(id => {
            const zh = id2zh[id];
            if (zhData[zh]) {
                if (!idData[id])
                    return;
                zhData[zh] += idData[id];
            }
            else {
                zhData[zh] = idData[id] || 0;
            }
        });
        return zhData;
    }
    /** 计算初始属性 */
    initial(base = this.id2zh(this.base())) {
        var _a;
        const properties = Object.entries(base)
            .reduce((acc, [key, value]) => {
            const property_id = zh2id[key];
            acc[property_id] = {
                property_name: key,
                property_id,
                base: value,
                add: 0,
                final: 0
            };
            return acc;
        }, {});
        const all_properties = [];
        // 处理音擎高级属性
        if ((_a = this.weapon) === null || _a === void 0 ? void 0 : _a.properties.length) {
            all_properties.push(...this.weapon.properties);
        }
        // 处理驱动盘词条
        for (const equip of this.equips) {
            equip.properties.length && all_properties.push(...equip.properties);
            equip.main_properties.length && all_properties.push(...equip.main_properties);
        }
        // 处理驱动盘套装效果
        const suitIds = Array.from(new Set(this.equips.filter(equip => equip.equip_suit.own >= 2).map(v => v.equip_suit.suit_id)));
        for (const suitId of suitIds) {
            const suitData = SuitData[suitId];
            if (!suitData || !suitData.properties) {
                logger.warn(`驱动盘套装数据缺失: ${suitId}，跳过套装效果计算`);
                continue;
            }
            if (!suitData.properties.length)
                continue;
            all_properties.push(...suitData.properties);
        }
        // 核心技额外提升
        const CoreSkillEnhancement = this.enkaAvatar.CoreSkillEnhancement || 0;
        if (CoreSkillEnhancement) {
            const extra = this.data.ExtraLevel[CoreSkillEnhancement].Extra;
            const extraIds = Object.keys(extra).map(Number);
            extraIds.forEach((id) => {
                // 攻击力百分比
                if (![12102].includes(id))
                    return;
                all_properties.push({
                    property_name: id2zh[id.toString().slice(0, 3)],
                    property_id: id,
                    base: get_base(id, extra[id].Value || 0)
                });
            });
        }
        for (const property of all_properties) {
            const propId = +property.property_id.toString().slice(0, 3);
            if (property.base.includes('%')) {
                if (this.keepPercent.includes(propId)) {
                    properties[propId].add += +property.base.replace('%', '');
                }
                else {
                    properties[propId].add += properties[propId].base * +property.base.replace('%', '') / 100;
                }
            }
            else {
                properties[propId].add += +property.base;
            }
        }
        // 格式化前特殊处理
        const sp = special[this.info.id];
        if (sp && sp.initial_before_format) {
            sp.initial_before_format(properties, this);
        }
        // 格式化、计算final
        if (isToFixed) {
            for (const propId in properties) {
                const property = properties[propId];
                if (this.keepPercent.includes(+propId)) {
                    property.add = +property.add.toFixed(1);
                    property.final = +(property.base + property.add).toFixed(1);
                }
                else if (propId === '305') { // 能量自动回复
                    property.add = +(Math.trunc(property.add * 100) / 100).toFixed(2);
                    property.final = +(property.base + property.add).toFixed(2);
                }
                else if (propId === '111') { // 生命值
                    property.add = Math.ceil(property.add);
                    property.final = property.base + property.add;
                }
                else {
                    property.add = Math.trunc(property.add);
                    property.final = property.base + property.add;
                }
            }
        }
        else {
            for (const propId in properties) {
                const property = properties[propId];
                property.final = property.base + property.add;
            }
        }
        // 命破特殊处理
        if (this.info.avatar_profession === 6) {
            delete properties[231], delete properties[232];
            delete properties[305];
            const sheerForce = Math.trunc(properties[111].final * 1 / 10) +
                Math.trunc(properties[121].final * 3 / 10);
            properties[19] = {
                property_name: '贯穿力',
                property_id: 19,
                base: 0,
                add: sheerForce,
                final: sheerForce
            };
            properties[20] = {
                property_name: '闪能自动累积',
                property_id: 20,
                base: 2,
                add: 0,
                final: 2
            };
        }
        // 格式化后特殊处理
        if (sp && sp.initial_after_format) {
            sp.initial_after_format(properties, this);
        }
        return Object.values(properties);
    }
}
exports.Property = Property;
class Skill {
    static main(skills, rank) {
        const result = skills.map(skill => ({
            level: skill.Level,
            skill_type: skill.Index,
            items: []
        }));
        // >=3额外提升2，>=5额外提升4
        const rankLevel = rank >= 3 ? (rank >= 5 ? 4 : 2) : 0;
        if (rankLevel) {
            result.forEach(skill => {
                if (skill.skill_type === 5)
                    return;
                skill.level += rankLevel;
            });
        }
        return result;
    }
}
exports.Skill = Skill;
function parseInfo(enkaAvatar) {
    const info = {};
    info.id = enkaAvatar.Id;
    const data = PartnerId2Data[info.id];
    if (!data)
        return;
    info.name_mi18n = data.name;
    info.level = enkaAvatar.Level;
    info.full_name_mi18n = data.full_name;
    info.element_type = +data.ElementType;
    info.camp_name_mi18n = data.Camp;
    info.avatar_profession = +data.WeaponType;
    info.rarity = data.Rarity;
    info.rank = enkaAvatar.TalentLevel;
    info.us_full_name = data.en_name;
    /** 特殊元素属性 */
    info.sub_element_type = ({
        1091: 1, // 星见雅
        1371: 2, // 仪玄
        1431: 4, // 叶瞬光
    })[info.id] || 0;
    info.group_icon_path = '';
    info.hollow_icon_path = '';
    info.role_vertical_painting_url = '';
    info.vertical_painting_color = '';
    info.role_square_url = '';
    info.skin_id = enkaAvatar.SkinId;
    return info;
}
function Enka2Mys(enkaAvatars, __isToFixed__ = true) {
    isToFixed = __isToFixed__;
    const avatars = Array.isArray(enkaAvatars) ? enkaAvatars : [enkaAvatars];
    const results = [];
    for (const enkaAvatar of avatars) {
        try {
            const info = parseInfo(enkaAvatar);
            if (!info) {
                throw `角色数据缺失: ${enkaAvatar.Id}`;
            }
            const avatar = info;
            avatar.ranks = [];
            avatar.equip = Equip.main(enkaAvatar.EquippedList);
            avatar.weapon = Weapon.main(enkaAvatar.Weapon);
            avatar.properties = Property.main(info, avatar.equip, avatar.weapon, enkaAvatar);
            avatar.skills = Skill.main(enkaAvatar.SkillLevelList, enkaAvatar.TalentLevel);
            results.push(avatar);
        }
        catch (error) {
            logger.warn(`Enka数据失败 ID: ${enkaAvatar.Id}\n`, error);
            continue;
        }
    }
    return Array.isArray(enkaAvatars) ? results : results[0];
}
/** 特殊处理 */
const special = {
    1121: {
        id: 1121,
        name: '本',
        initial_before_format: (properties, { enkaAvatar }) => {
            const core = [0.4, 0.46, 0.52, 0.6, 0.66, 0.72, 0.8];
            const { CoreSkillEnhancement } = enkaAvatar;
            const value = (core[CoreSkillEnhancement] || 0) * Math.trunc(properties[131].base + properties[131].add);
            properties[121].add += Math.trunc(value);
        }
    },
    1441: {
        id: 1441,
        name: '狛野真斗',
        initial_after_format: (properties) => {
            properties[20] = {
                property_name: '闪能自动累积',
                property_id: 20,
                base: 0,
                add: 0,
                final: 0
            };
        }
    },
};

return exports;})();
const {AvatarProperties}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AvatarProperties = void 0;
class AvatarProperties {
    getProperty(name) {
        return this.properties.find(property => property.property_name === name);
    }
    get basic_properties() {
        const data = {
            hpmax: this.getProperty('生命值'),
            attack: this.getProperty('攻击力'),
            def: this.getProperty('防御力'),
            breakstun: this.getProperty('冲击力'),
            crit: this.getProperty('暴击率'),
            critdam: this.getProperty('暴击伤害'),
            elementabnormalpower: this.getProperty('异常掌控'),
            elementmystery: this.getProperty('异常精通'),
            sheerforce: this.getProperty('贯穿力'),
            laceration: this.getProperty('锐暴伤害'),
            penratio: this.getProperty('穿透率'),
            sprecover: this.getProperty('能量自动回复'),
            adrenalineaccumulate: this.getProperty('闪能自动累积'),
            sharpnessaccumulate: this.getProperty('锐能自动累积'),
            /** 属性增伤 */
            dmgbonus: this.properties.find(property => property.property_id == element.idToPropertyId(this.element_type)),
        };
        return data;
    }
    get base_properties() {
        if (this._base_properties)
            return this._base_properties;
        const basic_properties = this.basic_properties;
        const get = (name) => {
            const data = basic_properties[name];
            return Number((data === null || data === void 0 ? void 0 : data.base) || (data === null || data === void 0 ? void 0 : data.final) || 0);
        };
        return this._base_properties = {
            HP: get('hpmax'),
            ATK: get('attack'),
            DEF: get('def'),
            Impact: get('breakstun'),
            AnomalyMastery: get('elementabnormalpower'),
            AnomalyProficiency: get('elementmystery'),
            EnergyRegen: get('sprecover')
        };
    }
    get initial_properties() {
        var _a;
        if (this._initial_properties)
            return this._initial_properties;
        const basic_properties = this.basic_properties;
        const get = (name) => {
            if (!basic_properties[name])
                return 0;
            const data = basic_properties[name].final;
            return Number(data.includes('%') ? +data.replace('%', '') / 100 : data);
        };
        return this._initial_properties = {
            HP: get('hpmax'),
            ATK: get('attack'),
            DEF: get('def'),
            Impact: get('breakstun'),
            CRITRate: get('crit'),
            CRITDMG: get('critdam'),
            /** 异常掌控 */
            AnomalyMastery: get('elementabnormalpower'),
            /** 异常精通 */
            AnomalyProficiency: get('elementmystery'),
            Pen: ((_a = this.equip) === null || _a === void 0 ? void 0 : _a.reduce((prev, curr) => prev + curr.get_property(23203), 0)) || 0,
            PenRatio: get('penratio'),
            EnergyRegen: get('sprecover')
        };
    }
    get equip_score() {
        var _a;
        if (!((_a = this.equip) === null || _a === void 0 ? void 0 : _a.length))
            return 0;
        if (this.scoreWeight) {
            let score = 0;
            for (const equip of this.equip) {
                score += equip.score || 0;
            }
            return score;
        }
        return 0;
    }
    get equip_comment() {
        const score = this.equip_score;
        if (score < 80) {
            return 'C';
        }
        if (score < 120) {
            return 'B';
        }
        if (score < 160) {
            return 'A';
        }
        if (score < 180) {
            return 'S';
        }
        if (score < 200) {
            return 'SS';
        }
        if (score < 220) {
            return 'SSS';
        }
        if (score < 280) {
            return 'ACE';
        }
        if (score >= 280) {
            return 'MAX';
        }
        return 'C';
    }
}
exports.AvatarProperties = AvatarProperties;

return exports;})();
const avatarPipeline=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.avatar_calc = avatar_calc;
exports.weapon_buff = weapon_buff;
exports.set_buff = set_buff;
const weakMapCalc = new WeakMap();
function avatar_calc(avatar) {
    if (weakMapCalc.has(avatar))
        return weakMapCalc.get(avatar);
    const models = calcFnc.character[avatar.id];
    if (!models)
        return;
    let m = models[models.length - 1];
    if (models.length > 1) {
        for (const model of models) {
            if (model.rule && model.rule(avatar)) {
                m = model;
                break;
            }
        }
    }
    debug(`${avatar.name_mi18n} 伤害计算规则：${m.name} by ${m.author}`);
    const buffM = new BuffManager(avatar);
    const calc = new Calculator(buffM);
    weakMapCalc.set(avatar, calc);
    debug('initial_properties', avatar.initial_properties);
    weapon_buff(avatar.weapon, buffM);
    set_buff(avatar.equip, buffM);
    if (m.buffs) {
        // 预筛选影画buff
        const vaildBuffs = m.buffs.filter(v => v.source ?
            (v.source !== '影画' || typeof v.check !== 'number' || v.check <= avatar.rank) :
            (!/^\d影/.test(v.name) || +v.name[0] <= avatar.rank));
        buffM.new(vaildBuffs);
    }
    if (m.skills)
        calc.new(m.skills);
    if (m.calc)
        m.calc(buffM, calc, avatar);
    debug(`Buff*${buffM.buffs.length}：`, buffM.buffs);
    return calc;
}
function weapon_buff(weapon, buffM) {
    const name = weapon === null || weapon === void 0 ? void 0 : weapon.name;
    if (!name)
        return;
    debug('武器：' + name);
    const m = calcFnc.weapon[name];
    if (!m)
        return;
    buffM.default({ name, source: '音擎' });
    if (m.buffs)
        buffM.new(m.buffs);
    if (m.calc)
        m.calc(buffM, weapon.star);
    buffM.default({});
}
function set_buff(equips, buffM) {
    if (!equips || equips.length === 0)
        return;
    buffM.default({ name: '', source: '套装' });
    const setCount = {};
    for (const equip of equips) {
        if (equip.equipment_type == 5) {
            // 属伤加成
            const index = [31503, 31603, 31703, 31803, , 31903].indexOf(equip.main_properties[0].property_id);
            if (index > -1 && elementEnum[index + 200]) {
                // @ts-expect-error
                buffM.new({
                    name: '驱动盘5号位',
                    type: '增伤',
                    value: Number(equip.main_properties[0].base.replace('%', '')) / 100,
                    element: elementEnum[index + 200]
                });
            }
        }
        const suit_name = equip.equip_suit.name;
        setCount[suit_name] = (setCount[suit_name] || 0) + 1;
    }
    buffM.setCount = setCount;
    for (const [name, count] of Object.entries(setCount)) {
        if (count < 2)
            continue;
        debug(`套装：${name}*${count}`);
        const m = calcFnc.set[name];
        if (!m)
            continue;
        buffM.default('name', name);
        if (m.buffs) {
            // 预筛选套装buff
            const vaildBuffs = m.buffs.filter(v => (v.source && v.source !== '套装') || typeof v.check !== 'number' || v.check <= count);
            buffM.new(vaildBuffs);
        }
        if (m.calc)
            m.calc(buffM, count);
    }
    buffM.default({});
}
function debug(...args) {
    if (!settings.getConfig('config').damage_debug_log)
        return;
    logger.debug(...args);
}

return exports;})();
const {rarityEnum,professionEnum}=runtime;
const {idToName,nameToId}=property;
const scoreFnc={};
const char={aliasToId:name=>characterAliases[name]??null,idToData:id=>referenceMaps.PartnerId2Data[id]};
const {baseValueData,formatScoreWeight,getEquipPropertyEnhanceCount}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getEquipPropertyEnhanceCount = exports.baseValueData = void 0;
exports.formatScoreWeight = formatScoreWeight;
exports.baseValueData = getMapData('EquipBaseValue');
const elementType2propId = (elementType) => [31503, 31603, 31703, 31803, 32303, 31903][elementType - 200];
function formatScoreWeight(oriScoreWeight, charID) {
    var _a;
    if (!oriScoreWeight)
        return false;
    if (Array.isArray(oriScoreWeight))
        return oriScoreWeight;
    if (typeof oriScoreWeight !== 'object')
        return false;
    const weight = {};
    for (const propName in oriScoreWeight) {
        if (!oriScoreWeight[propName] && oriScoreWeight[propName] !== 0)
            continue;
        let propID;
        if (charID && propName === '属性伤害加成') {
            propID = elementType2propId(+((_a = char.idToData(charID)) === null || _a === void 0 ? void 0 : _a.ElementType));
        }
        else {
            propID = +propName || nameToId(propName);
        }
        if (!propID)
            continue;
        weight[propID] = oriScoreWeight[propName];
    }
    ;
    return weight;
}
/**
 * 获取词条强化次数
 * @param propertyID 属性id
 * @param value 属性值
 */
const getEquipPropertyEnhanceCount = (propertyID, value) => {
    const baseValue = exports.baseValueData[propertyID];
    const numericValue = +value.replace('%', '');
    return Math.trunc(numericValue / baseValue - 1 || 0);
};
exports.getEquipPropertyEnhanceCount = getEquipPropertyEnhanceCount;

return exports;})();
const ZZZScore=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const equipScore = getMapData('EquipScore', false);
for (const charName in equipScore) {
    const charID = +charName || char.aliasToId(charName);
    if (!charID) {
        logger.warn(`驱动盘评分：未找到角色${charName}的角色ID`);
        delete equipScore[charName];
        continue;
    }
    equipScore[charID] = equipScore[charName];
    delete equipScore[charName];
}
/** 主词条可能属性 */
const mainStats = getMapData('EquipMainStats');
/** 副词条可能属性 */
const subStats = Object.keys(baseValueData).map(Number);
class Score {
    constructor(equip, weight) {
        this.equip = equip;
        this.weight = weight;
        this.partition = this.equip.equipment_type;
        this.userMainStat = this.equip.main_properties[0].property_id;
    }
    debug(...args) {
        if (!settings.getConfig('config').score_debug_log)
            return;
        logger.debug(...args);
    }
    /** 等级系数 */
    get_level_multiplier() {
        return (0.25 + +this.equip.level * 0.05) || 1;
    }
    /** 品质系数 */
    get_rarity_multiplier() {
        switch (rarityEnum[this.equip.rarity]) {
            case rarityEnum.S:
                return 1;
            case rarityEnum.A:
                return 2 / 3;
            case rarityEnum.B:
                return 1 / 3;
            default:
                return 1;
        }
    }
    /** 理论最大词条数 */
    get_max_count() {
        /** 权重最大的4个副词条 */
        const subMaxStats = subStats
            .filter(p => p !== this.userMainStat && this.weight[p])
            .sort((a, b) => this.weight[b] - this.weight[a]).slice(0, 4);
        if (!subMaxStats.length)
            return 0;
        this.debug(`[${this.partition}号位]理论副词条：` + subMaxStats.map(idToName).reduce((a, p, i) => a + `${p}*${this.weight[subMaxStats[i]].toFixed(2)} `, ''));
        let count = this.weight[subMaxStats[0]] * 6; // 权重最大副词条强化五次
        subMaxStats.slice(1).forEach(p => count += this.weight[p] || 0); // 其他词条各计入一次
        this.debug(`[${this.partition}号位]理论词条数：${logger.blue(count)}`);
        return count;
    }
    /** 实际词条数 */
    get_actual_count() {
        let count = 0;
        for (const prop of this.equip.properties) {
            const propID = prop.property_id;
            const weight = this.weight[propID];
            if (weight) {
                this.debug(`[${this.partition}号位]实际副词条：${idToName(propID)} ${logger.green(prop.count + 1)}*${weight}`);
                count += weight * (prop.count + 1);
            }
        }
        this.debug(`[${this.partition}号位]实际词条数：${logger.blue(count)}`);
        return count;
    }
    /** 计算驱动盘得分 */
    get_score() {
        const rarity_multiplier = this.get_rarity_multiplier();
        const level_multiplier = this.get_level_multiplier();
        const actual_count = this.get_actual_count();
        const max_count = this.get_max_count();
        if (max_count === 0)
            return 0;
        // 123号位
        if (this.partition <= 3) {
            const min_score = 12 * level_multiplier * rarity_multiplier;
            if (actual_count === 0) {
                // 1…1个有效副词条都没有吗？真是拿你没办法呢~给点主词条的分吧~❤️杂鱼~❤️杂鱼~❤️
                return min_score;
            }
            const score = actual_count / max_count * level_multiplier * rarity_multiplier * 55;
            this.debug(`[${this.partition}号位] ${logger.magenta(`${actual_count} / ${max_count} * ${level_multiplier} * ${rarity_multiplier} * 55 = ${score}`)}`);
            return Math.max(score, min_score);
        }
        // 456号位
        const mainMaxStat = mainStats[this.partition]
            .filter(p => this.weight[p])
            .sort((a, b) => this.weight[b] - this.weight[a])[0];
        const mainScore = mainMaxStat ? 12 * (this.weight[this.userMainStat] || 0) / this.weight[mainMaxStat] : 12;
        const subScore = actual_count / max_count * 43;
        const score = (mainScore + subScore) * level_multiplier * rarity_multiplier;
        this.debug(`[${this.partition}号位] ${logger.magenta(`(${mainScore} + ${subScore}) * ${level_multiplier} * ${rarity_multiplier} = ${score}`)}`);
        return score;
    }
    static main(equip, weight) {
        try {
            return new Score(equip, weight).get_score();
        }
        catch (err) {
            logger.error('角色驱动盘评分计算错误：', err);
            return 0;
        }
    }
    static getFinalWeight(avatar) {
        var _a;
        let def_weight = equipScore[avatar.id];
        // 无预设权重（新角色），选择相应基本规则
        if (!def_weight) {
            switch (avatar.avatar_profession) {
                case professionEnum.强攻:
                    def_weight = ['主C·双爆'];
                    break;
                case professionEnum.击破:
                    def_weight = ['冲击·双爆', '冲击·攻击', '冲击·异常'];
                    break;
                case professionEnum.异常:
                    def_weight = ['主C·异常', '辅助·异常'];
                    break;
                case professionEnum.支援:
                case professionEnum.防护:
                    def_weight = ['辅助·双爆', '辅助·异常'];
                    break;
                case professionEnum.命破:
                    def_weight = ['命破·双爆'];
                    break;
                case professionEnum.锋御:
                    def_weight = ['锋御·双爆'];
                    break;
            }
        }
        /** 选择第一个符合条件的规则，若皆不符合则选择第一个有效规则 */
        const delRules = (rules) => {
            var _a, _b;
            if (rules.length === 1) {
                rule_name = rules[0];
                final_weight = (_a = predefinedWeights[rules[0]]) === null || _a === void 0 ? void 0 : _a.value;
            }
            else {
                for (const name of rules) {
                    if ((_b = predefinedWeights[name]) === null || _b === void 0 ? void 0 : _b.rule(avatar)) {
                        rule_name = name;
                        final_weight = predefinedWeights[name].value;
                        break;
                    }
                }
                if (!final_weight) {
                    for (const name of rules) {
                        if (predefinedWeights[name]) {
                            rule_name = name;
                            final_weight = predefinedWeights[name].value;
                            break;
                        }
                    }
                }
            }
            final_weight = { ...final_weight };
        };
        let rule_name = '默认', final_weight;
        if (Array.isArray(def_weight)) {
            delRules(def_weight);
        }
        else if (def_weight === null || def_weight === void 0 ? void 0 : def_weight.rules) {
            const { rules, ...rest } = def_weight;
            delRules(rules);
            if (Object.keys(rest).length) {
                rule_name += '·改';
                Object.assign(final_weight, rest);
            }
        }
        else {
            final_weight = def_weight;
        }
        // console.log(avatar.name_mi18n, 'default_final_weight', final_weight)
        final_weight = formatScoreWeight(final_weight, avatar.id);
        const calc_weight = scoreFnc[avatar.id] && scoreFnc[avatar.id](avatar);
        if (calc_weight) {
            rule_name = calc_weight[0];
            final_weight = { ...final_weight, ...formatScoreWeight(calc_weight[1], avatar.id) };
        }
        // 小生命、小攻击、小防御动态映射为大生命、大攻击、大防御相对于基础属性的等效权重
        for (const [small, big, name] of [[11103, 11102, 'HP'], [12103, 12102, 'ATK'], [13103, 13102, 'DEF']]) {
            if (final_weight[big]) {
                (_a = final_weight[small]) !== null && _a !== void 0 ? _a : (final_weight[small] = +(baseValueData[small] * 100 / (baseValueData[big] * avatar.base_properties[name]) * final_weight[big]).toFixed(2));
            }
        }
        // console.log(avatar.name_mi18n, rule_name, final_weight)
        return [rule_name, final_weight];
    }
}
exports.default = Score;
/** 预设权重规则 */
const predefinedWeights = {
    主C·双爆: {
        rule: (avatar) => {
            const { ATK, CRITRate, CRITDMG, AnomalyMastery, AnomalyProficiency } = avatar.initial_properties;
            return ATK > 2400 && CRITRate * 2 + CRITDMG >= 2.2 && AnomalyMastery < 150 && AnomalyProficiency < 200;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0.75,
            "防御力百分比": 0,
            "冲击力": 0,
            "暴击率": 1,
            "暴击伤害": 1,
            "穿透率": 1,
            "穿透值": 0.25,
            "能量自动回复": 0,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    主C·异常: {
        rule: (avatar) => {
            const { ATK, CRITRate, CRITDMG, AnomalyMastery, AnomalyProficiency } = avatar.initial_properties;
            if (CRITRate * 2 + CRITDMG >= 2)
                return false;
            if (ATK < 2400)
                return false;
            if (AnomalyMastery >= 180 && AnomalyProficiency >= 200)
                return true;
            if (AnomalyMastery >= 120 && AnomalyProficiency >= 300)
                return true;
            if (AnomalyMastery >= 150 && AnomalyProficiency >= 250)
                return true;
            return false;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0.75,
            "防御力百分比": 0,
            "冲击力": 0,
            "暴击率": 0,
            "暴击伤害": 0,
            "穿透率": 1,
            "穿透值": 0.25,
            "能量自动回复": 0,
            "异常精通": 1,
            "异常掌控": 1,
            "属性伤害加成": 1
        }
    },
    命破·双爆: {
        rule: (avatar) => {
            return true;
        },
        value: {
            "生命值百分比": 0.5,
            "攻击力百分比": 0.25,
            "防御力百分比": 0,
            "冲击力": 0,
            "暴击率": 1,
            "暴击伤害": 1,
            "穿透率": 0,
            "穿透值": 0,
            "能量自动回复": 0,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    锋御·双爆: {
        rule: () => true,
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0,
            "防御力百分比": 1,
            "冲击力": 0,
            "暴击率": 1,
            "暴击伤害": 0.75,
            "穿透率": 1,
            "穿透值": 0.25,
            "能量自动回复": 0,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    辅助·双爆: {
        rule: (avatar) => {
            const { CRITRate, CRITDMG, AnomalyProficiency } = avatar.initial_properties;
            return CRITRate * 2 + CRITDMG >= 1.5 && AnomalyProficiency < 200;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0.75,
            "防御力百分比": 0,
            "冲击力": 0,
            "暴击率": 1,
            "暴击伤害": 1,
            "穿透率": 0.75,
            "穿透值": 0.25,
            "能量自动回复": 1,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    辅助·攻击: {
        rule: (avatar) => {
            const { CRITRate, CRITDMG } = avatar.initial_properties;
            return CRITRate * 2 + CRITDMG >= 1.5;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 1,
            "防御力百分比": 0,
            "冲击力": 0,
            "暴击率": 1,
            "暴击伤害": 0.75,
            "穿透率": 0.75,
            "穿透值": 0.25,
            "能量自动回复": 1,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    辅助·异常: {
        rule: (avatar) => {
            const { CRITRate, CRITDMG, AnomalyProficiency } = avatar.initial_properties;
            return CRITRate * 2 + CRITDMG < 2 && AnomalyProficiency >= 200;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0.75,
            "防御力百分比": 0,
            "冲击力": 0,
            "暴击率": 0,
            "暴击伤害": 0,
            "穿透率": 0.75,
            "穿透值": 0.25,
            "能量自动回复": 1,
            "异常精通": 1,
            "异常掌控": 1,
            "属性伤害加成": 1
        }
    },
    冲击·双爆: {
        rule: (avatar) => {
            const { CRITRate, CRITDMG, AnomalyMastery, AnomalyProficiency } = avatar.initial_properties;
            return CRITRate * 2 + CRITDMG >= 1.5 && AnomalyMastery < 150 && AnomalyProficiency < 200;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0.75,
            "防御力百分比": 0,
            "冲击力": 1,
            "暴击率": 1,
            "暴击伤害": 1,
            "穿透率": 0.75,
            "穿透值": 0.25,
            "能量自动回复": 1,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    冲击·攻击: {
        rule: (avatar) => {
            const { ATK, CRITRate, CRITDMG, AnomalyMastery, AnomalyProficiency } = avatar.initial_properties;
            return ATK > 2000 && CRITRate * 2 + CRITDMG >= 1 && AnomalyMastery < 150 && AnomalyProficiency < 200;
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 1,
            "防御力百分比": 0,
            "冲击力": 1,
            "暴击率": 1,
            "暴击伤害": 0.75,
            "穿透率": 0.75,
            "穿透值": 0.25,
            "能量自动回复": 1,
            "异常精通": 0,
            "异常掌控": 0,
            "属性伤害加成": 1
        }
    },
    冲击·异常: {
        rule: (avatar) => {
            const { CRITRate, CRITDMG, AnomalyMastery, AnomalyProficiency } = avatar.initial_properties;
            return CRITRate * 2 + CRITDMG < 2 && (AnomalyMastery >= 150 || AnomalyProficiency >= 200);
        },
        value: {
            "生命值百分比": 0,
            "攻击力百分比": 0.75,
            "防御力百分比": 0,
            "冲击力": 1,
            "暴击率": 0,
            "暴击伤害": 0,
            "穿透率": 0.75,
            "穿透值": 0.25,
            "能量自动回复": 1,
            "异常精通": 1,
            "异常掌控": 1,
            "属性伤害加成": 1
        }
    },
};

return exports;})().default;
const {EquipGrade}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EquipGrade = void 0;
class EquipGrade {
    constructor(score) { this.score = score; }
    get comment() {
        if (this.score === false) {
            return false;
        }
        if (this.score <= 12) {
            return 'C';
        }
        if (this.score < 20) {
            return 'B';
        }
        if (this.score < 28) {
            return 'A';
        }
        if (this.score < 32) {
            return 'S';
        }
        if (this.score < 36) {
            return 'SS';
        }
        if (this.score < 40) {
            return 'SSS';
        }
        if (this.score < 48) {
            return 'ACE';
        }
        if (this.score >= 48) {
            return 'MAX';
        }
        return false;
    }
}
exports.EquipGrade = EquipGrade;

return exports;})();
const {Property}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Property = void 0;
class Property {
    constructor(data) {
        const { property_name, property_id, base, add, final } = data;
        this.property_name = property_name;
        this.property_id = property_id;
        this.base = base;
        this.add = add;
        this.final = final;
    }
}
exports.Property = Property;

return exports;})();
const {Skill}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Skill = exports.SkillItem = void 0;
class SkillItem {
    constructor(title, text) {
        this.title = title;
        this.text = text;
    }
}
exports.SkillItem = SkillItem;
class Skill {
    constructor(data) {
        const { level, skill_type, items } = data;
        this.level = level;
        this.skill_type = skill_type;
        this.items = items;
    }
}
exports.Skill = Skill;

return exports;})();
const {ZZZAvatarInfo}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ZZZAvatarInfo = exports.Rank = exports.Weapon = exports.Equip = exports.EquipMainProperty = exports.EquipProperty = void 0;
const Score = ZZZScore;
class EquipProperty {
    constructor(data) {
        const { property_name, property_id, base } = data;
        this.property_name = property_name;
        this.property_id = property_id;
        this.base = base;
        this.base_score = 0;
        this.classname = property.idToClassName(property_id);
        /** 词条强化次数 */
        this.count = getEquipPropertyEnhanceCount(property_id, base);
    }
}
exports.EquipProperty = EquipProperty;
class EquipMainProperty {
    constructor(data) {
        const { property_name, property_id, base } = data;
        this.property_name = property_name;
        this.property_id = property_id;
        this.base = base;
        this.classname = property.idToClassName(property_id);
    }
    get short_name() {
        if (this.property_name.includes('属性伤害加成')) {
            return this.property_name.replace('属性伤害加成', '伤加成');
        }
        if (this.property_name === '能量自动回复') {
            return '能量回复';
        }
        return this.property_name;
    }
}
exports.EquipMainProperty = EquipMainProperty;
class Equip {
    constructor(data) {
        this.score = false;
        const { id, level, name, icon, rarity, properties, main_properties, equip_suit, equipment_type, } = data;
        this.id = id;
        this.level = level;
        this.name = name;
        this.icon = icon;
        this.rarity = rarity;
        this.properties = properties.map(item => new EquipProperty(item));
        this.main_properties = main_properties.map(item => new EquipMainProperty(item));
        this.equip_suit = equip_suit;
        this.equipment_type = equipment_type;
    }
    get_property(id) {
        var _a;
        const result = ((_a = this.properties.find(item => item.property_id === id)) === null || _a === void 0 ? void 0 : _a.base) || '0';
        return Number(result);
    }
    get_score(weight) {
        if (!weight) {
            this.score = false;
            return this.score;
        }
        this.properties.forEach(item => item.base_score = weight[item.property_id] || 0);
        this.score = Score.main(this, weight);
        return this.score;
    }
    get comment() {
        if (this.score === false) {
            return false;
        }
        if (this.score <= 12) {
            return 'C';
        }
        if (this.score < 20) {
            return 'B';
        }
        if (this.score < 28) {
            return 'A';
        }
        if (this.score < 32) {
            return 'S';
        }
        if (this.score < 36) {
            return 'SS';
        }
        if (this.score < 40) {
            return 'SSS';
        }
        if (this.score < 48) {
            return 'ACE';
        }
        if (this.score >= 48) {
            return 'MAX';
        }
        return false;
    }
}
exports.Equip = Equip;
class Weapon {
    constructor(data) {
        const { id, level, name, star, icon, rarity, properties, main_properties, talent_title, talent_content, profession, } = data;
        this.id = id;
        this.level = level;
        this.name = name;
        this.star = star;
        this.icon = icon;
        this.rarity = rarity;
        this.properties = properties.map(item => new EquipProperty(item));
        this.main_properties = main_properties.map(item => new EquipMainProperty(item));
        this.talent_title = talent_title;
        this.talent_content = talent_content;
        this.profession = profession;
        this.level_rank = Math.floor(level / 10);
    }
}
exports.Weapon = Weapon;
class Rank {
    constructor(data) {
        const { id, name, desc, pos, is_unlocked } = data;
        this.id = id;
        this.name = name;
        this.desc = desc;
        this.pos = pos;
        this.is_unlocked = is_unlocked;
    }
}
exports.Rank = Rank;
class ZZZAvatarInfo {
    constructor(data) {
        var _a, _b, _c;
        const { id, level, name_mi18n, full_name_mi18n, element_type, sub_element_type, camp_name_mi18n, avatar_profession, rarity, group_icon_path, hollow_icon_path, equip, weapon, properties, skills, rank, ranks, role_vertical_painting_url, isNew, skin_id, } = data;
        this.id = id;
        this.level = level;
        this.name_mi18n = name_mi18n;
        this.full_name_mi18n = full_name_mi18n;
        this.element_type = element_type;
        this.sub_element_type = sub_element_type;
        this.camp_name_mi18n = camp_name_mi18n;
        this.avatar_profession = avatar_profession;
        this.rarity = rarity;
        this.group_icon_path = group_icon_path;
        this.hollow_icon_path = hollow_icon_path;
        this.equip =
            (equip &&
                (Array.isArray(equip)
                    ? equip.map(equip => new Equip(equip))
                    : [new Equip(equip)])) ||
                [];
        this.weapon = weapon ? new Weapon(weapon) : null;
        this.properties =
            properties && properties.map(property => new Property(property));
        this.skills = skills && skills.map(skill => new Skill(skill));
        this.rank = rank;
        this.ranks = ranks && ranks.map(rank => new Rank(rank));
        this.ranks_num = rank;
        this.element_str = element.idToName(element_type);
        this.sub_element_str = element.idToName(element_type, sub_element_type);
        this.role_vertical_painting_url = role_vertical_painting_url;
        this.isNew = isNew || false;
        this.skin_id = +(skin_id || ((_c = (_b = (_a = this.role_vertical_painting_url) === null || _a === void 0 ? void 0 : _a.match) === null || _b === void 0 ? void 0 : _b.call(_a, /role_vertical_painting_\d+_(\d+).png$/)) === null || _c === void 0 ? void 0 : _c[1]) || 0);
        this.level_rank = Math.floor(this.level / 10);
        const weight = Score.getFinalWeight(this);
        this.weightRule = weight[0];
        this.scoreWeight = weight[1];
        for (const equip of this.equip) {
            equip.get_score(this.scoreWeight);
        }
    }
    getProperty(name) {
        return this.properties.find(property => property.property_name === name);
    }
    get basic_properties() {
        const data = {
            hpmax: this.getProperty('生命值'),
            attack: this.getProperty('攻击力'),
            def: this.getProperty('防御力'),
            breakstun: this.getProperty('冲击力'),
            crit: this.getProperty('暴击率'),
            critdam: this.getProperty('暴击伤害'),
            elementabnormalpower: this.getProperty('异常掌控'),
            elementmystery: this.getProperty('异常精通'),
            sheerforce: this.getProperty('贯穿力'),
            laceration: this.getProperty('锐暴伤害'),
            penratio: this.getProperty('穿透率'),
            sprecover: this.getProperty('能量自动回复'),
            adrenalineaccumulate: this.getProperty('闪能自动累积'),
            sharpnessaccumulate: this.getProperty('锐能自动累积'),
            /** 属性增伤 */
            dmgbonus: this.properties.find(property => property.property_id == element.idToPropertyId(this.element_type)),
        };
        return data;
    }
    get base_properties() {
        if (this._base_properties)
            return this._base_properties;
        const basic_properties = this.basic_properties;
        const get = (name) => {
            const data = basic_properties[name];
            return Number((data === null || data === void 0 ? void 0 : data.base) || (data === null || data === void 0 ? void 0 : data.final) || 0);
        };
        return this._base_properties = {
            HP: get('hpmax'),
            ATK: get('attack'),
            DEF: get('def'),
            Impact: get('breakstun'),
            AnomalyMastery: get('elementabnormalpower'),
            AnomalyProficiency: get('elementmystery'),
            EnergyRegen: get('sprecover')
        };
    }
    get initial_properties() {
        var _a;
        if (this._initial_properties)
            return this._initial_properties;
        const basic_properties = this.basic_properties;
        const get = (name) => {
            if (!basic_properties[name])
                return 0;
            const data = basic_properties[name].final;
            return Number(data.includes('%') ? +data.replace('%', '') / 100 : data);
        };
        return this._initial_properties = {
            HP: get('hpmax'),
            ATK: get('attack'),
            DEF: get('def'),
            Impact: get('breakstun'),
            CRITRate: get('crit'),
            CRITDMG: get('critdam'),
            /** 异常掌控 */
            AnomalyMastery: get('elementabnormalpower'),
            /** 异常精通 */
            AnomalyProficiency: get('elementmystery'),
            Pen: ((_a = this.equip) === null || _a === void 0 ? void 0 : _a.reduce((prev, curr) => prev + curr.get_property(23203), 0)) || 0,
            PenRatio: get('penratio'),
            EnergyRegen: get('sprecover')
        };
    }
    get equip_score() {
        var _a;
        if (!((_a = this.equip) === null || _a === void 0 ? void 0 : _a.length))
            return 0;
        if (this.scoreWeight) {
            let score = 0;
            for (const equip of this.equip) {
                score += equip.score || 0;
            }
            return score;
        }
        return 0;
    }
    get equip_comment() {
        const score = this.equip_score;
        if (score < 80) {
            return 'C';
        }
        if (score < 120) {
            return 'B';
        }
        if (score < 160) {
            return 'A';
        }
        if (score < 180) {
            return 'S';
        }
        if (score < 200) {
            return 'SS';
        }
        if (score < 220) {
            return 'SSS';
        }
        if (score < 280) {
            return 'ACE';
        }
        if (score >= 280) {
            return 'MAX';
        }
        return 'C';
    }
}
exports.ZZZAvatarInfo = ZZZAvatarInfo;

return exports;})();
