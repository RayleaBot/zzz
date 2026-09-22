// Fixed miao-plugin 7f6f1c84; MIT, see LICENSES. Generated numeric modules.
const Elem=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const elemAlias = {
    anemo: '风,蒙德',
    geo: '岩,璃月',
    electro: '雷,电,雷电,稻妻',
    dendro: '草,须弥',
    pyro: '火,纳塔',
    hydro: '水,枫丹',
    cryo: '冰,至冬'
};
const elemAliasSR = {
    fire: '火',
    ice: '冰',
    wind: '风',
    elec: '雷',
    phy: '物理',
    quantum: '量子',
    imaginary: '虚数'
};
// 元素属性映射, 名称=>elem
let elemMap = {};
let elemMapSR = {};
// 标准元素名
let elemTitleMap = {};
let elemTitleMapSR = elemAliasSR;
lodash.forEach(elemAlias, (txt, key) => {
    elemMap[key] = key;
    elemTitleMap[key] = txt[0];
    Data.eachStr(txt, (t) => (elemMap[t] = key));
});
lodash.forEach(elemAliasSR, (txt, key) => {
    elemMapSR[key] = key;
    elemMapSR[txt] = key;
});
const Elem = {
    // 根据名称获取元素key
    elem(elem = '', defElem = '', game = 'gs') {
        elem = elem.toLowerCase();
        return (game === 'gs' ? elemMap : elemMapSR)[elem] || defElem;
    },
    // 根据key获取元素名
    elemName(elem = '', defName = '') {
        return elemTitleMap[Elem.elem(elem)] || defName;
    },
    // 从字符串中匹配元素
    matchElem(name = '', defElem = '', withName = false) {
        const elemReg = new RegExp(`^(${lodash.keys(elemMap).join('|')})`);
        let elemRet = elemReg.exec(name);
        let elem = (elemRet && elemRet[1]) ? Elem.elem(elemRet[1]) : defElem;
        if (elem) {
            if (withName) {
                return {
                    elem,
                    name: name.replace(elemReg, '')
                };
            }
            return elem;
        }
        return '';
    },
    eachElem(fn, game = 'gs') {
        lodash.forEach(game === 'gs' ? elemTitleMap : elemTitleMapSR, (title, key) => {
            fn(key, title);
        });
    },
    isElem(elem = '', game = 'gs') {
        return !!(game === 'gs' ? elemMap : elemMapSR)[elem];
    },
    sameElem(key1, key2, game = 'gs') {
        let map = (game === 'gs' ? elemMap : elemMapSR);
        return map[key1] === map[key2];
    }
};
exports.default = Elem;

return exports;})().default;
Format=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
let Format = {
    ...Elem,
    int: function (d) {
        return parseInt(d);
    },
    comma: function (num, fix = 0) {
        num = parseFloat((num * 1).toFixed(fix));
        let [integer, decimal] = String.prototype.split.call(num, '.');
        let re = new RegExp(`\\d(?=(\\d{${Cfg.get('commaGroup', 3)}})+$)`, 'g');
        integer = integer.replace(re, '$&,'); // 正则先行断言 = /\d(?=(\d{3})+$)/g
        return `${integer}${fix > 0 ? '.' + (decimal || lodash.repeat('0', fix)) : ''}`;
    },
    pct: function (num, fix = 1) {
        return (num * 1).toFixed(fix) + '%';
    },
    percent: function (num, fix = 1) {
        return Format.pct(num * 100, fix);
    }
};
exports.default = Format;

return exports;})().default;
const AttrItem=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
class AttrItem {
    constructor(ds) {
        this.base = ds.base * 1 || 0;
        this.plus = ds.plus * 1 || 0;
        this.pct = ds.pct * 1 || 0;
        this.inc = ds.inc * 1 || 0;
    }
    static create(ds) {
        return new AttrItem(ds);
        /*
        return {
          base: ds.base * 1 || 0,
          plus: ds.plus * 1 || 0,
          pct: ds.pct * 1 || 0,
          inc: ds.inc * 1 || 0
        } */
    }
    toString() {
        return (this.base || 0) + (this.plus || 0) + ((this.base || 0) * (this.pct || 0) / 100);
    }
}
exports.default = AttrItem;

return exports;})().default;
const {erType,erTitle,eleBaseDmg,breakBaseDmg,cryBaseDmg,elationBaseDmg}=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.elationBaseDmg = exports.breakBaseDmg = exports.cryBaseDmg = exports.eleBaseDmg = exports.stellarConductFactor = exports.erTitle = exports.erType = void 0;
// 元素反应类型及基数
exports.erType = {
    // 增幅反应
    vaporize: { type: 'pct', num: ({ element }) => element === '水' ? 2 : 1.5, title: '蒸发' },
    melt: { type: 'pct', num: ({ element }) => element === '火' ? 2 : 1.5, title: '融化' },
    // 结晶护盾
    crystallize: { type: 'shield', num: () => 1, title: '结晶' },
    // 剧变反应
    burning: { type: 'fusion', num: () => 1, title: '燃烧' },
    superConduct: { type: 'fusion', num: () => 6, title: '超导' },
    swirl: { type: 'fusion', num: () => 2.4, title: '扩散' },
    electroCharged: { type: 'fusion', num: () => 8, title: '感电' },
    shatter: { type: 'fusion', num: () => 12, title: '碎冰' },
    overloaded: { type: 'fusion', num: () => 11, title: '超载' },
    bloom: { type: 'fusion', num: () => 8, title: '绽放' },
    burgeon: { type: 'fusion', num: () => 12, title: '烈绽放' },
    hyperBloom: { type: 'fusion', num: () => 12, title: '超绽放' },
    // 激化反应
    aggravate: { type: 'bonus', num: () => 4.6, title: '超激化' },
    spread: { type: 'bonus', num: () => 5.0, title: '蔓激化' },
    // 月反应
    lunarBloom: { type: 'lunar', num: () => 1, title: '月绽放' },
    lunarCharged: { type: 'lunar', num: ({ talent }) => talent === 'fy' ? 7.2 : 3, title: '月感电' },
    lunarCrystallize: { type: 'lunar', num: ({ talent }) => talent === 'fy' ? 3.84 : 1.6, title: '月结晶' },
    // 星反应
    stellarConduct: { type: 'stellar', num: ({ params }) => exports.stellarConductFactor[params.stellarConductCount], title: '星超导' },
    stellarSwirl: { type: 'stellar', num: ({ talent }) => talent === 'fy' ? 3 : 1, title: '星扩散' },
    stellarVortex: { type: 'stellar', num: ({ params }) => params.stellarVortexCount >= 3 ? 12 : 8, title: '星辉冰旋（反应星扩散·冰）' },
    // 击破持续伤害
    shock: { type: 'breakDot', num: () => 2.0, title: '触电' },
    burn: { type: 'breakDot', num: () => 1.0, title: '灼烧' },
    windShear: { type: 'breakDot', num: () => 1.0, title: '风化' },
    bleed: { type: 'breakDot', num: () => 1.0, title: '裂伤' },
    // 击破附加伤害
    entanglement: { type: 'breakPlus', num: () => 0.6, title: '纠缠' },
    // 击破伤害
    ligntningBreak: { type: 'break', num: () => 1.0, title: '雷击破' },
    fireBreak: { type: 'break', num: () => 2.0, title: '火击破' },
    windBreak: { type: 'break', num: () => 1.5, title: '风击破' },
    physicalBreak: { type: 'break', num: () => 2.0, title: '物理击破' },
    quantumBreak: { type: 'break', num: () => 0.5, title: '量子击破' },
    imaginaryBreak: { type: 'break', num: () => 0.5, title: '虚数击破' },
    iceBreak: { type: 'break', num: () => 1.0, title: '冰击破' },
    // 超击破伤害
    superBreak: { type: 'break', num: () => 1.0, title: '超击破' },
    // 欢愉伤害
    elationDmg: { type: 'elation', num: () => 1.0, title: '欢愉' }
};
let erTmp = {};
lodash.forEach(exports.erType, (er, key) => {
    erTmp[er.title] = key;
});
exports.erTitle = erTmp;
// 星超导基础倍率，依赖叠层计数
// 1层对应 1.45，满层12层对应 2.0
exports.stellarConductFactor = [1, 1.45, 1.5, 1.55, 1.6, 1.65, 1.7, 1.75, 1.8, 1.85, 1.9, 1.95, 2.0];
// 各等级精通基础伤害
exports.eleBaseDmg = {
    1: 4.291,
    2: 4.634,
    3: 4.976,
    4: 5.319,
    5: 5.661,
    6: 6.162,
    7: 6.660,
    8: 7.217,
    9: 7.842,
    10: 8.536,
    11: 9.300,
    12: 10.165,
    13: 11.112,
    14: 12.141,
    15: 13.437,
    16: 14.770,
    17: 16.105,
    18: 17.431,
    19: 18.781,
    20: 20.146,
    21: 21.528,
    22: 22.926,
    23: 24.311,
    24: 25.703,
    25: 27.102,
    26: 28.300,
    27: 29.526,
    28: 30.745,
    29: 32.432,
    30: 34.073,
    31: 35.668,
    32: 37.257,
    33: 38.854,
    34: 40.456,
    35: 42.277,
    36: 44.130,
    37: 46.018,
    38: 47.927,
    39: 49.889,
    40: 51.846,
    41: 53.850,
    42: 56.041,
    43: 58.376,
    44: 60.838,
    45: 64.016,
    46: 67.136,
    47: 70.382,
    48: 73.753,
    49: 77.267,
    50: 80.900,
    51: 84.189,
    52: 87.633,
    53: 91.121,
    54: 94.655,
    55: 99.650,
    56: 104.100,
    57: 108.597,
    58: 113.238,
    59: 118.152,
    60: 123.221,
    61: 128.392,
    62: 134.776,
    63: 141.378,
    64: 148.135,
    65: 156.111,
    66: 162.868,
    67: 169.874,
    68: 176.949,
    69: 184.168,
    70: 191.410,
    71: 198.693,
    72: 206.169,
    73: 212.789,
    74: 219.436,
    75: 228.557,
    76: 236.687,
    77: 244.853,
    78: 252.806,
    79: 261.198,
    80: 269.361,
    81: 277.499,
    82: 285.744,
    83: 294.092,
    84: 302.546,
    85: 313.459,
    86: 322.238,
    87: 331.371,
    88: 340.864,
    89: 351.274,
    90: 361.713,
    95: 390.367,
    100: 418.522
};
// 各等级结晶护盾基础吸收量
exports.cryBaseDmg = {
    1: 91.18,
    2: 98.71,
    3: 106.24,
    4: 113.76,
    5: 121.29,
    6: 128.82,
    7: 136.35,
    8: 143.88,
    9: 151.41,
    10: 158.94,
    11: 169.99,
    12: 181.08,
    13: 192.19,
    14: 204.05,
    15: 215.94,
    16: 227.86,
    17: 247.69,
    18: 267.54,
    19: 287.43,
    20: 303.83,
    21: 320.23,
    22: 336.63,
    23: 352.32,
    24: 368.01,
    25: 383.70,
    26: 394.43,
    27: 405.18,
    28: 415.95,
    29: 426.74,
    30: 437.54,
    31: 450.60,
    32: 463.70,
    33: 476.85,
    34: 491.13,
    35: 502.55,
    36: 514.01,
    37: 531.41,
    38: 549.98,
    39: 568.58,
    40: 585.00,
    41: 605.67,
    42: 626.39,
    43: 646.05,
    44: 665.76,
    45: 685.50,
    46: 700.84,
    47: 723.33,
    48: 745.87,
    49: 768.44,
    50: 786.79,
    51: 809.54,
    52: 832.33,
    53: 855.16,
    54: 878.04,
    55: 899.48,
    56: 919.36,
    57: 946.04,
    58: 974.76,
    59: 1003.58,
    60: 1030.08,
    61: 1056.64,
    62: 1085.25,
    63: 1113.92,
    64: 1149.26,
    65: 1178.06,
    66: 1200.22,
    67: 1227.66,
    68: 1257.24,
    69: 1284.92,
    70: 1314.75,
    71: 1342.67,
    72: 1372.75,
    73: 1396.32,
    74: 1427.31,
    75: 1458.37,
    76: 1482.34,
    77: 1511.91,
    78: 1541.55,
    79: 1569.15,
    80: 1596.15,
    81: 1622.42,
    82: 1648.07,
    83: 1666.38,
    84: 1684.68,
    85: 1702.98,
    86: 1726.10,
    87: 1754.67,
    88: 1785.87,
    89: 1817.14,
    90: 1851.06
};
// 各等级击破基础伤害
exports.breakBaseDmg = {
    1: 54.00,
    2: 58.00,
    3: 62.00,
    4: 67.53,
    5: 70.51,
    6: 73.52,
    7: 76.57,
    8: 79.64,
    9: 82.74,
    10: 85.87,
    11: 91.49,
    12: 97.07,
    13: 102.59,
    14: 108.06,
    15: 113.47,
    16: 118.84,
    17: 124.15,
    18: 129.41,
    19: 134.62,
    20: 139.77,
    21: 149.33,
    22: 158.80,
    23: 168.18,
    24: 177.46,
    25: 186.65,
    26: 195.75,
    27: 204.75,
    28: 213.66,
    29: 222.48,
    30: 231.20,
    31: 246.43,
    32: 261.18,
    33: 275.47,
    34: 289.32,
    35: 302.73,
    36: 315.71,
    37: 328.29,
    38: 340.47,
    39: 352.26,
    40: 363.67,
    41: 408.12,
    42: 451.79,
    43: 494.68,
    44: 536.82,
    45: 578.22,
    46: 618.92,
    47: 658.91,
    48: 698.23,
    49: 736.89,
    50: 774.90,
    51: 871.06,
    52: 964.87,
    53: 1056.42,
    54: 1145.79,
    55: 1233.06,
    56: 1318.30,
    57: 1401.58,
    58: 1482.96,
    59: 1562.52,
    60: 1640.31,
    61: 1752.32,
    62: 1861.90,
    63: 1969.12,
    64: 2074.07,
    65: 2176.80,
    66: 2277.39,
    67: 2375.91,
    68: 2472.42,
    69: 2566.97,
    70: 2659.64,
    71: 2780.30,
    72: 2898.60,
    73: 3014.60,
    74: 3128.37,
    75: 3239.98,
    76: 3349.47,
    77: 3456.92,
    78: 3562.38,
    79: 3665.91,
    80: 3767.55
};
exports.elationBaseDmg = {
    1: 108.00,
    2: 116.00,
    3: 124.00,
    4: 135.05,
    5: 141.02,
    6: 147.05,
    7: 153.11,
    8: 159.28,
    9: 165.48,
    10: 171.74,
    11: 182.99,
    12: 194.14,
    13: 205.12,
    14: 216.12,
    15: 226.95,
    16: 237.68,
    17: 248.30,
    18: 258.82,
    19: 269.23,
    20: 279.54,
    21: 298.66,
    22: 317.60,
    23: 336.34,
    24: 354.92,
    25: 373.30,
    26: 391.49,
    27: 409.50,
    28: 427.32,
    29: 444.95,
    30: 462.40,
    31: 492.86,
    32: 522.36,
    33: 550.95,
    34: 578.64,
    35: 605.45,
    36: 631.43,
    37: 656.58,
    38: 680.94,
    39: 704.51,
    40: 727.33,
    41: 816.25,
    42: 903.58,
    43: 989.36,
    44: 1073.64,
    45: 1156.45,
    46: 1237.83,
    47: 1317.83,
    48: 1396.47,
    49: 1473.78,
    50: 1549.80,
    51: 1742.12,
    52: 1929.74,
    53: 2112.84,
    54: 2291.58,
    55: 2466.12,
    56: 2636.59,
    57: 2803.15,
    58: 2965.92,
    59: 3125.04,
    60: 3280.61,
    61: 3504.64,
    62: 3723.80,
    63: 3938.25,
    64: 4148.13,
    65: 4353.60,
    66: 4554.78,
    67: 4751.82,
    68: 4944.83,
    69: 5133.95,
    70: 5319.28,
    71: 5560.61,
    72: 5797.20,
    73: 6029.21,
    74: 6256.76,
    75: 6479.95,
    76: 6698.95,
    77: 6913.85,
    78: 7124.77,
    79: 7331.82,
    80: 7535.11
};

return exports;})();
const DmgMastery=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const defaultParams = {
    stellarConductCount: 12, // 星超导叠层计数，若相关calc.js未传入参数，则默认为最大12层
    stellarVortexCount: 6, // 星辉风旋系数，若相关calc.js未传入参数，则默认为最大6层
};
let DmgMastery = {
    getMultiple(type, mastery = 0) {
        let typeCfg = erType[type];
        if (typeCfg.type === 'pct') {
            return (25 / 9) * mastery / (mastery + 1400);
        }
        else if (typeCfg.type === 'fusion') {
            return 16 * mastery / (mastery + 2000);
        }
        else if (typeCfg.type === 'lunar' || typeCfg.type === 'stellar') {
            return 6 * mastery / (mastery + 2000);
        }
        else if (typeCfg.type === 'bonus') {
            return 5 * mastery / (mastery + 1200);
        }
        else if (typeCfg.type === 'shield') {
            return (40 / 9) * mastery / (mastery + 1400);
        }
        return 0;
    },
    getBasePct(type, element, talent, params = {}) {
        let typeCfg = erType[type];
        if (typeCfg) {
            const args = {};
            args.element = element;
            args.talent = talent;
            args.params = { ...defaultParams, ...params };
            return typeCfg.num(args) || 1;
        }
        return 1;
    }
};
exports.default = DmgMastery;

return exports;})().default;
const DmgAttr=(()=>{const exports={};
"use strict";
/*
* 伤害计算 - 属性计算
* */
Object.defineProperty(exports, "__esModule", { value: true });
let DmgAttr = {
    // 计算并返回指定属性值
    getAttrValue(ds = {}) {
        return (ds.base || 0) + (ds.plus || 0) + ((ds.base || 0) * (ds.pct || 0) / 100);
    },
    // 获取profile对应attr属性值
    getAttr({ originalAttr, attr, weapon, char, game = 'gs' }) {
        let ret = {};
        if (originalAttr) {
            ret = lodash.merge({}, originalAttr);
        }
        // 基础属性
        lodash.forEach('atk,def,hp,speed'.split(','), (key) => {
            ret[key] = AttrItem.create((originalAttr === null || originalAttr === void 0 ? void 0 : originalAttr[key]) || {
                base: attr[`${key}Base`] * 1 || 0,
                plus: attr[key] * 1 - attr[`${key}Base`] * 1 || 0,
                pct: 0
            });
        });
        lodash.forEach((game === 'gs' ? 'mastery,recharge,cpct,cdmg,heal,dmg,phy,coloringDmg' : 'recharge,cpct,cdmg,heal,dmg,enemydmg,effPct,effDef,stance,joy').split(','), (key) => {
            ret[key] = AttrItem.create((originalAttr === null || originalAttr === void 0 ? void 0 : originalAttr[key]) || {
                base: attr[key] * 1 || 0, // 基础值
                plus: 0, // 加成值
                pct: 0, // 百分比加成
                inc: 0 // 提高：护盾增效&治疗增效
            });
        });
        // 技能属性记录
        lodash.forEach((game === 'gs' ? 'a,a2,a3,e,q,nightsoul' : 'a,a2,a3,e,e2,xe,q,q2,q3,t,t2,me,me2,mt,mt2,dot,break,elation').split(','), (key) => {
            ret[key] = ret[key] || {
                pct: 0, // 倍率加成
                multi: 0, // 独立倍率乘区加成，宵宫E等
                plus: 0, // 伤害值提高
                dmg: 0, // 伤害提高
                enemydmg: 0, // 承受伤害提高
                cpct: 0, // 暴击提高
                cdmg: 0, // 爆伤提高
                elevated: 0, // 擢升
                merrymakes: 0, // 增笑
                def: 0, // 防御降低
                ignore: 0 // 无视防御
            };
        });
        ret.enemy = ret.enemy || {
            def: 0, // 降低防御
            ignore: 0, // 无视防御
            phy: 0 // 物理防御
        };
        ret.shield = AttrItem.create((originalAttr === null || originalAttr === void 0 ? void 0 : originalAttr.shield) || {
            base: 100, // 基础
            plus: 0, // 护盾强效
            inc: 100 // 吸收倍率
        });
        if (!originalAttr) {
            ret.weapon = weapon; // 武器
            ret.weaponTypeName = char.weaponTypeName; // 武器类型
            ret.element = Format.elemName(char.elem); // 元素类型
            ret.refine = ((weapon.affix || ret.refine || 1) * 1 - 1) || 0; // 武器精炼
            ret.multi = 0; // 倍率独立乘区
            ret.kx = 0; // 敌人抗性降低
            ret.staticAttr = attr.staticAttr;
            if (game === 'gs') {
                ret.elevated = 0; // 擢升
                ret.vaporize = 0; // 蒸发
                ret.melt = 0; // 融化
                ret.burning = 0; // 燃烧
                ret.crystallize = 0; // 结晶
                ret.superConduct = 0; // 超导
                ret.swirl = 0; // 扩散
                ret.electroCharged = 0; // 感电
                ret.shatter = 0; // 碎冰
                ret.overloaded = 0; // 超载
                ret.bloom = 0; // 绽放
                ret.burgeon = 0; // 烈绽放
                ret.hyperBloom = 0; // 超绽放
                ret.aggravate = 0; // 超激化
                ret.spread = 0; // 蔓激化
                ret.lunarCharged = 0; // 月感电
                ret.lunarBloom = 0; // 月绽放
                ret.lunarCrystallize = 0; // 月结晶
                ret.stellarConduct = 0; // 星超导
                ret.stellarSwirl = 0; // 星扩散
                ret.fykx = 0; // 敌人反应抗性降低
                ret.fyplus = 0; // 反应伤害值提升（数值/不受精通加成）
                ret.fypct = 0; // 反应基础伤害值提升（百分比/受精通加成）
            }
            else if (game === 'sr') {
                ret.sp = char.sp * 1;
                // 超击破
                ret.superBreak = {
                    ignore: 0 // 无视防御
                };
                ret.merrymakes = 0; // 增笑
                ret.punchline = 0; // 笑点
            }
        }
        return ret;
    },
    // 获取数据集
    getDs(attr, meta, params) {
        return {
            ...meta,
            attr,
            params,
            refine: attr.refine,
            weaponTypeName: attr.weaponTypeName,
            element: Format.elemName(attr.element) || attr.element, // 计算属性
            calc: DmgAttr.getAttrValue
        };
    },
    // 计算属性
    calcAttr({ originalAttr, buffs, meta, artis, params = {}, incAttr = '', reduceAttr = '', talent = '', game = 'gs' }) {
        let attr = DmgAttr.getAttr({ originalAttr, game });
        attr.characterName = meta.characterName;
        let msg = [];
        let { attrMap } = Meta.getMeta(game, 'arti');
        if (incAttr && attrMap[incAttr]) {
            let aCfg = attrMap[incAttr];
            attr[incAttr][aCfg.calc] += aCfg.value;
        }
        if (reduceAttr && attrMap[reduceAttr]) {
            let aCfg = attrMap[reduceAttr];
            attr[reduceAttr][aCfg.calc] -= aCfg.value;
        }
        lodash.forEach(buffs, (buff) => {
            meta.mastery = meta.mastery || buff.mastery; // 先反应
        });
        lodash.forEach(buffs, (buff) => {
            let ds = DmgAttr.getDs(attr, meta, params);
            ds.currentTalent = talent;
            ds.artis = artis;
            if (buff.isStatic) {
                return;
            }
            // 如果存在rule，则进行计算
            if (buff.check && !buff.check(ds)) {
                return;
            }
            if (buff.cons) {
                if (ds.cons * 1 < buff.cons * 1) {
                    return;
                }
            }
            if (!lodash.isUndefined(buff.maxCons)) {
                if (ds.cons * 1 > buff.maxCons * 1) {
                    return;
                }
            }
            if (buff.tree) {
                if (!ds.trees[`10${buff.tree}`]) {
                    return;
                }
            }
            let title = typeof buff.title === 'function' ? buff.title(ds) : buff.title;
            if (buff.mastery) {
                let mKey = {
                    vaporize: '蒸发', melt: '融化', swirl: '扩散'
                };
                let mKey2 = {
                    aggravate: '超激化', spread: '蔓激化'
                };
                let mastery = Math.max(0, attr.mastery.base + attr.mastery.plus);
                buff.data = buff.data || {};
                let key = buff.mastery;
                if (mKey[key]) {
                    buff.data['_' + key] = DmgMastery.getMultiple(key, mastery) * 100;
                }
                else if (mKey2[key]) {
                    let eleNum = DmgMastery.getBasePct(key, attr.element, talent);
                    let eleBase = 1 + attr[key] / 100 + DmgMastery.getMultiple(key, mastery);
                    eleBase *= eleBaseDmg[ds.level];
                    buff.data['_' + key] = DmgMastery.getMultiple(key, mastery) * 100;
                    buff.data['_' + key + 'num'] = eleNum * eleBase;
                }
            }
            lodash.forEach(buff.data, (val, key) => {
                if (lodash.isFunction(val)) {
                    val = val(ds);
                }
                if (!val && val !== 0) {
                    return;
                }
                title = title.replace(`[${key}]`, Format.comma(val, 1));
                // 技能提高
                let tRet = /^(a|a2|a3|e|e2|q|q2|q3|t|t2|me|xe|xe2|mt|dot|break|nightsoul)(Def|Ignore|Dmg|Enemydmg|Plus|Pct|Cpct|Cdmg|Multi|Elevated|Merrymakes)$/.exec(key);
                if (tRet) {
                    attr[tRet[1]][tRet[2].toLowerCase()] += val * 1 || 0;
                    return;
                }
                let aRet = /^(mastery|cpct|cdmg|heal|recharge|dmg|enemydmg|phy|coloringDmg|shield|speed|stance|joy)(Plus|Pct|Inc)?$/.exec(key);
                if (aRet) {
                    attr[aRet[1]][aRet[2] ? aRet[2].toLowerCase() : 'plus'] += val * 1 || 0;
                    return;
                }
                let bRet = /^(hp|def|atk)(Base|Plus|Pct|Inc)?$/.exec(key);
                if (bRet) {
                    attr[bRet[1]][bRet[2] ? bRet[2].toLowerCase() : 'plus'] += val * 1 || 0;
                    // hp、atk、def的基础值增加时（例如玛薇卡2命在夜魂加持状态下时，基础攻击力提高200）
                    if (bRet[2] === 'Base')
                        attr[bRet[1]].plus += val * attr.staticAttr[bRet[1]].pct / 100 || 0;
                    return;
                }
                if (key === 'enemyDef') {
                    attr.enemy.def += val * 1 || 0;
                    return;
                }
                if (key === 'ignore' || key === 'enemyIgnore') {
                    attr.enemy.ignore += val * 1 || 0;
                    return;
                }
                if (['vaporize', 'melt', 'crystallize', 'burning', 'superConduct', 'swirl', 'electroCharged', 'shatter', 'overloaded', 'bloom', 'burgeon', 'hyperBloom', 'aggravate', 'spread', 'elevated', 'lunarCharged', 'lunarBloom', 'lunarCrystallize', 'stellarConduct', 'stellarSwirl', 'kx', 'fykx', 'multi', 'fyplus', 'fypct', 'merrymakes', 'punchline'].includes(key)) {
                    attr[key] += val * 1 || 0;
                    return;
                }
                let sRet = /^(superBreak)(Ignore)$/.exec(key);
                if (sRet) {
                    attr[sRet[1]][sRet[2].toLowerCase()] += val * 1 || 0;
                }
            });
            msg.push(title);
        });
        return {
            attr, msg
        };
    }
};
exports.default = DmgAttr;

return exports;})().default;
const DmgCalc=(()=>{const exports={};
"use strict";
/*
* 伤害计算 - 计算伤害
* */
Object.defineProperty(exports, "__esModule", { value: true });
let DmgCalc = {
    calcRet(fnArgs = {}, data = {}) {
        let { pctNum, // 技能倍率
        talent, // 天赋类型
        ele, // 元素反应
        basicNum, // 基础数值
        mode, // 模式
        dynamicData, // 动态伤害计算数据
        params // 伤害计算所需额外参数
         } = fnArgs;
        let { dynamicDmg = 0, // 动态增伤
        dynamicPhy = 0, // 动态物伤
        dynamicCpct = 0, // 动态暴击率
        dynamicCdmg = 0, // 动态暴击伤害
        dynamicEnemydmg = 0 // 动态易伤
         } = dynamicData;
        let { ds, // 数据集
        attr, // 属性
        level, // 面板数据
        enemyLv, // 敌人等级
        showDetail = false, // 是否展示详情
        game } = data;
        let calc = ds.calc;
        let { atk, dmg, phy, coloringDmg, cdmg, cpct, enemydmg } = attr;
        // 攻击区
        let atkNum = calc(atk);
        // 倍率独立乘区
        let multiNum = attr.multi / 100;
        let fyplus = attr.fyplus;
        let fypct = attr.fypct / 100;
        // 增伤区
        let elevatedNum = attr.elevated / 100;
        let dmgNum = (1 + dmg.base / 100 + dmg.plus / 100 + dynamicDmg / 100);
        // 增笑
        let merrymakesNum = 1;
        // 笑点
        let punchlineNum = 0;
        // 欢愉度
        let joyNum = 1;
        if (game === 'sr') {
            merrymakesNum = 1 + attr.merrymakes / 100;
            punchlineNum = attr.punchline;
            joyNum = 1 + attr.joy.base / 100 + attr.joy.plus / 100;
        }
        if (ele === 'phy') {
            dmgNum = (1 + phy.base / 100 + phy.plus / 100 + dynamicPhy / 100);
        }
        if (ele === 'coloringDmg') {
            let dmgPct = attr.staticAttr.dmg.plus / 100;
            if (dmgPct > 0) {
                dmgNum = (dmgNum - dmgPct) < 1 ? 1 : (dmgNum - dmgPct);
            }
            dmgNum += (coloringDmg.base / 100 + coloringDmg.plus / 100);
        }
        if (/^scene,.*/.test(ele) || /.*,scene$/.test(ele) || ele === 'scene') {
            let dmgPct = attr.staticAttr.dmg.plus / 100;
            if (dmgPct > 0) {
                dmgNum = (dmgNum - dmgPct) < 1 ? 1 : (dmgNum - dmgPct);
            }
            if (ele !== 'scene') {
                ele = ele.replace(/(,)?scene(,)?/g, '');
            }
        }
        // 易伤区
        let enemydmgNum = 1;
        if (game === 'sr') {
            enemydmgNum = 1 + enemydmg.base / 100 + enemydmg.plus / 100 + dynamicEnemydmg / 100;
        }
        // 暴击区
        let cpctNum = cpct.base / 100 + cpct.plus / 100 + dynamicCpct / 100;
        // 爆伤区
        let cdmgNum = cdmg.base / 100 + cdmg.plus / 100 + dynamicCdmg / 100;
        let enemyDef = attr.enemy.def / 100;
        let enemyIgnore = attr.enemy.ignore / 100;
        let plusNum = 0;
        pctNum = pctNum / 100;
        if (talent) {
            lodash.forEach(talent.split(','), (t) => {
                if (attr[t]) {
                    let ds = attr[t];
                    pctNum += ds.pct / 100;
                    dmgNum += ds.dmg / 100;
                    enemydmgNum += game === 'gs' ? 0 : ds.enemydmg / 100;
                    cpctNum += ds.cpct / 100;
                    cdmgNum += ds.cdmg / 100;
                    enemyDef += ds.def / 100;
                    enemyIgnore += ds.ignore / 100;
                    multiNum += ds.multi / 100;
                    plusNum += ds.plus;
                }
            });
        }
        // TODO
        if (ele === 'superBreak') {
            enemyIgnore += attr.superBreak.ignore / 100;
        }
        // 防御区
        let defNum = (level + 100) / ((level + 100) + (enemyLv + 100) * (1 - enemyDef) * (1 - enemyIgnore));
        if (game === 'sr') {
            let enemyDefdown = enemyDef + enemyIgnore <= 1 ? enemyDef + enemyIgnore : 1;
            defNum = (200 + level * 10) / ((200 + level * 10) + (200 + enemyLv * 10) * (1 - enemyDefdown));
        }
        // 抗性区
        let kx = attr.kx;
        let kNum = 0.9;
        if (game === 'sr') {
            kNum = 1 + (kx / 100);
        }
        else {
            if (ele === 'swirl') {
                kx = attr.fykx;
            }
            else if (ele === 'coloringDmg') {
                kx = (attr.kx || 0) + (attr.fykx || 0);
            }
            kx = 10 - (kx || 0);
            if (kx >= 75) {
                kNum = 1 / (1 + 3 * kx / 100);
            }
            else if (kx >= 0) {
                kNum = (100 - kx) / 100;
            }
            else {
                kNum = 1 - kx / 200;
            }
        }
        // 减伤区
        let dmgReduceNum = 1;
        if (game === 'sr') {
            dmgReduceNum = 0.9;
        }
        cpctNum = Math.max(0, Math.min(1, cpctNum));
        if (cpctNum === 0) {
            cdmgNum = 0;
        }
        const isEle = ele !== false && ele !== 'phy' && ele !== 'scene' && ele !== 'coloringDmg';
        // 反应区
        let eleNum = 1;
        let eleBase = 1;
        if (game === 'gs') {
            eleNum = isEle ? DmgMastery.getBasePct(ele, attr.element, talent, params) : 1;
            eleBase = isEle ? 1 + attr[ele] / 100 + DmgMastery.getMultiple(ele, calc(attr.mastery)) : 1;
        }
        let stanceNum = 1;
        if (game === 'sr') {
            switch (ele) {
                case 'shock':
                case 'burn':
                case 'windShear':
                case 'bleed':
                case 'entanglement':
                case 'lightningBreak':
                case 'fireBreak':
                case 'windBreak':
                case 'physicalBreak':
                case 'quantumBreak':
                case 'imaginaryBreak':
                case 'iceBreak':
                case 'superBreak': {
                    eleNum = DmgMastery.getBasePct(ele, attr.element, talent, params);
                    stanceNum = 1 + calc(attr.stance) / 100;
                    break;
                }
                default:
                    break;
            }
        }
        let dmgBase = (mode === 'basic') ? basicNum * (1 + multiNum) + plusNum : atkNum * pctNum * (1 + multiNum) + plusNum;
        let ret = {};
        switch (ele) {
            case 'vaporize':
            case 'melt': {
                ret = {
                    dmg: dmgBase * dmgNum * (1 + cdmgNum) * defNum * kNum * eleBase * eleNum,
                    avg: dmgBase * dmgNum * (1 + cpctNum * cdmgNum) * defNum * kNum * eleBase * eleNum
                };
                break;
            }
            case 'burning':
            case 'superConduct':
            case 'swirl':
            case 'electroCharged':
            case 'shatter':
            case 'overloaded':
            case 'bloom':
            case 'burgeon':
            case 'hyperBloom': {
                ret = { avg: (eleBaseDmg[level] * (1 + fypct) * eleBase * eleNum + fyplus) * kNum };
                break;
            }
            case 'lunarBloom':
            case 'lunarCharged':
            case 'lunarCrystallize':
            case 'stellarConduct':
            case 'stellarSwirl': {
                let base = dmgBase ? dmgBase : eleBaseDmg[level];
                ret = {
                    avg: (base * (1 + fypct) * eleBase * eleNum + fyplus) * (1 + elevatedNum) * kNum * (1 + cpctNum * cdmgNum),
                    dmg: (base * (1 + fypct) * eleBase * eleNum + fyplus) * (1 + elevatedNum) * kNum * (1 + cdmgNum)
                };
                break;
            }
            case 'crystallize': {
                eleBase *= cryBaseDmg[level];
                ret = { avg: eleBase * (calc(attr.shield) / 100) * (attr.shield.inc / 100) };
                break;
            }
            case 'aggravate':
            case 'spread': {
                eleBase *= eleBaseDmg[level];
                dmgBase += eleBase * eleNum;
                ret = {
                    dmg: dmgBase * dmgNum * (1 + cdmgNum) * defNum * kNum,
                    avg: dmgBase * dmgNum * (1 + cpctNum * cdmgNum) * defNum * kNum
                };
                break;
            }
            // 技能持续伤害 = 伤害值乘区 * 增伤区 * 易伤区 * 防御区 * 抗性区 * 减伤区
            case 'skillDot': {
                ret = {
                    avg: dmgBase * dmgNum * enemydmgNum * defNum * kNum * dmgReduceNum
                };
                break;
            }
            // 未计算：1. 层数(风化、纠缠) 2. 韧性条系数(击破、纠缠) 3. 削韧值(超击破)、超击破伤害提高
            // 常规击破伤害均需要计算减伤区（即按韧性条存在处理） 特例：阮梅终结技/秘技击破伤害不计算减伤/波提欧
            // 击破伤害 = 基础伤害 * 属性击破伤害系数 * (1+击破特攻%) * 易伤区 * 防御区 * 抗性区 * 减伤区 * (敌方韧性+2)/4 * 层数系数
            // 击破持续伤害 = 基础伤害 * 属性持续伤害系数 * (1+击破特攻%) * 易伤区 * 防御区 * 抗性区 * 减伤区 * 层数系数
            // 超击破伤害 = 基础伤害 * (1+击破特攻%) * 易伤区 * 防御区 * 抗性区 * 减伤区 * (1+超击破伤害提高) * 技能最终削韧值
            // 技能最终削韧值 = 技能基础削韧值 ×（1＋削韧值提高）×（1＋弱点击破效率提高）
            // 超击破伤害提高 截至2.2版本该乘区仅能由同谐开拓者提供，与常规增伤区无关，暂时只在calc.js中手动计算
            case 'shock':
            case 'burn':
            case 'windShear':
            case 'bleed':
            case 'entanglement':
            case 'lightningBreak':
            case 'fireBreak':
            case 'windBreak':
            case 'physicalBreak':
            case 'quantumBreak':
            case 'imaginaryBreak':
            case 'iceBreak':
            case 'superBreak': {
                let breakBase = 1;
                breakBase *= breakBaseDmg[level];
                ret = {
                    avg: breakBase * eleNum * stanceNum * enemydmgNum * defNum * kNum * dmgReduceNum
                };
                break;
            }
            case 'elation': {
                let elationBase = 1 * elationBaseDmg[level];
                pctNum += multiNum;
                let punchlineCals = 1 + punchlineNum * 5 / (punchlineNum + 240);
                ret = {
                    avg: elationBase * pctNum * merrymakesNum * joyNum * punchlineCals * (1 + cpctNum * cdmgNum) * defNum * kNum * dmgReduceNum * enemydmgNum,
                    dmg: elationBase * pctNum * merrymakesNum * joyNum * punchlineCals * (1 + cdmgNum) * defNum * kNum * dmgReduceNum * enemydmgNum
                };
                break;
            }
            default: {
                ret = {
                    dmg: dmgBase * dmgNum * enemydmgNum * (1 + cdmgNum) * defNum * kNum * dmgReduceNum,
                    avg: dmgBase * dmgNum * enemydmgNum * (1 + cpctNum * cdmgNum) * defNum * kNum * dmgReduceNum
                };
            }
        }
        if (showDetail) {
            console.log('Attr', attr);
            console.log({ mode, dmgBase, atkNum, pctNum, multiNum, plusNum, dmgNum, enemydmgNum, stanceNum, cpctNum, cdmgNum, defNum, eleNum, kNum, dmgReduceNum });
            console.log('Ret', ret);
        }
        return ret;
    },
    getDmgFn(data) {
        let { showDetail, attr, ds, game } = data;
        let { calc } = ds;
        let dmgFn = function (pctNum = 0, talent = false, ele = false, basicNum = 0, mode = 'talent', dynamicData = false, params = false) {
            if (ele) {
                ele = erTitle[ele] || ele;
            }
            if (game === 'sr') {
                // 星铁meta数据天赋为百分比前数字
                pctNum = pctNum * 100;
            }
            return DmgCalc.calcRet({ pctNum, talent, ele, basicNum, mode, dynamicData, params }, data);
        };
        dmgFn.basic = function (basicNum = 0, talent = false, ele = false, dynamicData = false, params = false) {
            return dmgFn(0, talent, ele, basicNum, 'basic', dynamicData, params);
        };
        dmgFn.reaction = function (ele = false, params = false, talent = 'fy') {
            switch (ele) {
                // 击破持续伤害
                case 'shock':
                case 'burn':
                case 'windShear':
                case 'bleed': {
                    talent = 'dot';
                    break;
                }
                // 击破伤害
                case 'superBreak':
                case 'lightningBreak':
                case 'fireBreak':
                case 'windBreak':
                case 'physicalBreak':
                case 'quantumBreak':
                case 'imaginaryBreak':
                case 'iceBreak': {
                    talent = 'break';
                    break;
                }
                default:
                    break;
            }
            return dmgFn(0, talent, ele, 0, 'basic', false, params);
        };
        dmgFn.dynamic = function (pctNum = 0, talent = false, dynamicData = false, ele = false, params = false) {
            return dmgFn(pctNum, talent, ele, 0, 'talent', dynamicData, params);
        };
        // 计算治疗
        dmgFn.heal = function (num) {
            if (showDetail) {
                console.log(num, calc(attr.heal), attr.heal.inc);
            }
            return {
                avg: num * (1 + calc(attr.heal) / 100 + attr.heal.inc / 100)
            };
        };
        // 计算护盾
        dmgFn.shield = function (num) {
            if (showDetail) {
                console.log(num, calc(attr.shield), calc(attr.shield.inc));
            }
            return {
                avg: num * (calc(attr.shield) / 100) * (attr.shield.inc / 100)
            };
        };
        // 扩散方法
        dmgFn.swirl = function () {
            return dmgFn(0, 'fy', 'swirl');
        };
        return dmgFn;
    }
};
exports.default = DmgCalc;

return exports;})().default;
const AttrData=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const baseAttr = {
    gs: 'atk,def,hp,mastery,recharge,cpct,cdmg,dmg,phy,heal,shield,coloringDmg'.split(','),
    sr: 'atk,def,hp,speed,recharge,cpct,cdmg,dmg,heal,stance,effPct,effDef,joy'.split(',')
};
let attrReg = {
    gs: new RegExp(`^(${baseAttr.gs.join('|')})(Base|Plus|Pct|Inc)$`),
    sr: new RegExp(`^(${baseAttr.sr.join('|')})(Base|Plus|Pct|Inc)$`)
};
class AttrData extends Base {
    constructor(char, data = null) {
        super();
        this.char = char;
        this.game = char.game;
        this.init(data, this.game);
    }
    static create(char, data = null) {
        return new AttrData(char, data);
    }
    init(data) {
        // 基础属性
        this._attr = {};
        this._base = {};
        let attr = this._attr;
        let base = this._base;
        lodash.forEach(baseAttr[this.game], (key) => {
            attr[key] = {
                base: 0,
                plus: 0,
                pct: 0
            };
            base[key] = 0;
        });
        if (data) {
            this.setAttr(data, true);
        }
    }
    /**
     * getter
     *
     * @param key
     * @returns {*|number}
     * @private
     */
    _get(key) {
        let attr = this._attr;
        if (baseAttr[this.game].includes(key)) {
            let a = attr[key];
            return a.base * (1 + a.pct / 100) + a.plus;
        }
        let testRet = attrReg[this.game].exec(key);
        if (testRet && testRet[1] && testRet[2]) {
            let key = testRet[1];
            let key2 = testRet[2].toLowerCase();
            return attr[key][key2] || 0;
        }
    }
    /**
     * 添加或追加Attr数据
     * @param key
     * @param val
     * @param isBase
     * @returns {boolean}
     */
    addAttr(key, val, isBase = false) {
        let attr = this._attr;
        let base = this._base;
        if (this.isSr && Format.isElem(key, this.game)) {
            if (Format.sameElem(this.char.elem, key, this.game)) {
                key = 'dmg';
            }
        }
        if (baseAttr[this.game].includes(key)) {
            attr[key].plus += val * 1;
            if (isBase) {
                base[key] = (base[key] || 0) + val * 1;
            }
            return true;
        }
        let testRet = attrReg[this.game].exec(key);
        if (testRet && testRet[1] && testRet[2]) {
            let key = testRet[1];
            let key2 = testRet[2].toLowerCase();
            attr[key][key2] = attr[key][key2] || 0;
            attr[key][key2] += val * 1;
            if (key2 === 'base' || isBase) {
                base[key] = (base[key] || 0) + val * 1;
            }
            return true;
        }
        return false;
    }
    /**
     * 设置属性
     * @param data
     * @param withBase：带有base数据的初始化设置，会将atk/hp/def视作结果数据而非plus数据
     */
    setAttr(data, withBase = false) {
        if (withBase) {
            lodash.forEach(['hp', 'def', 'atk', 'speed'], (key) => {
                let base = `${key}Base`;
                if (data[key] && data[base]) {
                    data[`${key}Plus`] = data[key] - data[base];
                    delete data[key];
                }
            });
        }
        lodash.forEach(data, (val, key) => {
            if (this.isSr && Format.isElem(key, this.game)) {
                if (this.char.elem === Format.elem(key, '', this.game)) {
                    this.addAttr('dmg', val);
                }
            }
            else {
                this.addAttr(key, val);
            }
        });
    }
    getAttr() {
        let ret = {};
        lodash.forEach(baseAttr[this.game], (key) => {
            ret[key] = this[key];
            if (['hp', 'atk', 'def', 'speed'].includes(key)) {
                ret[`${key}Base`] = this[`${key}Base`];
            }
        });
        ret._calc = true;
        /**
         * 提取并赋值角色静态基础数值
         * 包括但不限于圣遗物主副词条之和
         * 圣遗物常驻加成，如追忆2、角斗士2等
         * 武器副词条
         * 角色突破加成等等
         */
        ret.staticAttr = this._attr;
        return ret;
    }
    getBase() {
        return this._base;
    }
}
exports.default = AttrData;

return exports;})().default;
const Attr=(()=>{const exports={};
"use strict";
/**
 * 面板属性计算
 * @type {{}}
 */
Object.defineProperty(exports, "__esModule", { value: true });
class Attr extends Base {
    constructor(profile) {
        super();
        this.profile = profile;
        this.game = profile.game;
    }
    /**
     * 静态调用入口
     * @param profile
     * @returns {Attr}
     */
    static create(profile) {
        return new Attr(profile);
    }
    static calcPromote(lv, game = 'gs') {
        let lvs = game === 'gs' ? [1, 20, 40, 50, 60, 70, 80, 90, 100] : [1, 20, 30, 40, 50, 60, 70, 80];
        let promote = 0;
        for (let idx = 0; idx <= lvs.length - 1; idx++) {
            if (lv >= lvs[idx] && lv <= lvs[idx + 1]) {
                return promote;
            }
            promote++;
        }
        return promote;
    }
    /**
     * 面板属性计算
     * @returns {{}}
     */
    calc() {
        let profile = this.profile;
        this.attr = AttrData.create(profile.char, {});
        if (this.isGs) {
            this.addAttr('recharge', 100, true);
            this.addAttr('cpct', 5, true);
            this.addAttr('cdmg', 50, true);
        }
        this.setCharAttr();
        this.setWeaponAttr();
        this.setArtisAttr();
        return this.attr.getAttr();
    }
    getBase() {
        return this.attr.getBase();
    }
    addAttr(key, val, isBase = false) {
        this.attr.addAttr(key, val, isBase);
    }
    // 计算角色属性
    setCharAttr() {
        var _a, _b;
        let { char, level, promote, trees } = this.profile;
        let metaAttr = ((_a = char.detail) === null || _a === void 0 ? void 0 : _a.attr) || {};
        let self = this;
        if (this.isSr) {
            // 星铁面板属性
            let attr = char.getLvAttr(level, promote);
            lodash.forEach(attr, (v, k) => {
                k = k + (['hp', 'atk', 'def', 'speed'].includes(k) ? 'Base' : '');
                self.addAttr(k, v, true);
            });
            let tree = ((_b = char.detail) === null || _b === void 0 ? void 0 : _b.tree) || {};
            lodash.forEach(trees || [], (tid) => {
                let tCfg = tree[tid];
                if (tCfg) {
                    let key = tCfg.key;
                    if (['atk', 'hp', 'def'].includes(key)) {
                        key = key + 'Pct';
                    }
                    self.addAttr(key, tCfg.value);
                }
            });
            return;
        }
        let { keys = {}, details = {} } = metaAttr;
        let lvLeft = 0;
        let lvRight = 0;
        let lvStep = [1, 20, 40, 50, 60, 70, 80, 90, 100];
        let currPromote = 0;
        for (let idx = 0; idx < lvStep.length - 1; idx++) {
            if (currPromote === promote) {
                if (level >= lvStep[idx] && level <= lvStep[idx + 1]) {
                    lvLeft = lvStep[idx];
                    lvRight = lvStep[idx + 1];
                    break;
                }
            }
            currPromote++;
        }
        let detailLeft = details[lvLeft + '+'] || details[lvLeft] || {};
        let detailRight = details[lvRight] || {};
        let getLvData = (idx, step = false) => {
            let valueLeft = detailLeft[idx];
            let valueRight = detailRight[idx];
            if (!step) {
                return valueLeft * 1 + ((valueRight - valueLeft) * (level - lvLeft) / (lvRight - lvLeft));
            }
            else {
                return valueLeft * 1 + ((valueRight - valueLeft) * Math.floor((level - lvLeft) / 5) / Math.round(((lvRight - lvLeft) / 5)));
            }
        };
        this.addAttr('hpBase', getLvData(0), true);
        this.addAttr('atkBase', getLvData(1), true);
        this.addAttr('defBase', getLvData(2), true);
        this.addAttr(keys[3], getLvData(3, true), !/(hp|atk|def)/.test(keys[3]));
        // 特有角色自带属性
        const Characters = [
            { id: 10000119, attrs: { mastery: 200 } }, // 菈乌玛: +200 元素精通
            { id: 10000122, attrs: { mastery: 100 } }, // 奈芙尔: +100 元素精通
            { id: 10000005, attrs: { mastery: 15, hpBase: 50, atkBase: 7, cpct: 10, cdmg: 20, recharge: 20, hpPct: 20, atkPct: 20, defPct: 20 } },
            { id: 10000007, attrs: { mastery: 15, hpBase: 50, atkBase: 7, cpct: 10, cdmg: 20, recharge: 20, hpPct: 20, atkPct: 20, defPct: 20 } }
            // 在丝柯克传说任务磷星之章中，完成特训后，旅行者的元素精通永久增加15点
            // 在完成魔神任务空月之歌·第八幕获得衣装复地重天后，将为各个元素的旅行者解锁全新的天赋
            // 旅行者完成与一种元素的共鸣后，将会获得额外的强化效果，该效果对所有元素类型的旅行者生效
            // 后续角色兼容
        ];
        const charBuff = Characters.find(c => c.id === char.id);
        if (charBuff) {
            for (const key in charBuff.attrs) {
                if (key.endsWith('Pct')) {
                    this.addAttr(key, charBuff.attrs[key]);
                }
                else {
                    this.addAttr(key, charBuff.attrs[key], true);
                }
            }
        }
        let charBuffs = char.getCalcRule();
        lodash.forEach(charBuffs.buffs, (buff) => {
            if (!buff.isStatic) {
                return true;
            }
            if (buff) {
                lodash.forEach(buff.data, (val, key) => {
                    this.addAttr(key, val);
                });
            }
        });
    }
    /**
     * 计算武器属性
     */
    setWeaponAttr() {
        var _a, _b, _c;
        let wData = ((_a = this.profile) === null || _a === void 0 ? void 0 : _a.weapon) || {};
        if (!wData || !wData.name) {
            return false;
        }
        let weapon = Weapon.get((wData === null || wData === void 0 ? void 0 : wData.name) || (wData === null || wData === void 0 ? void 0 : wData.id), this.game);
        if (!weapon) {
            return false;
        }
        let wCalcRet = weapon.calcAttr(wData.level, wData.promote);
        let self = this;
        let char = this.profile.char;
        if (this.isSr) {
            // 星铁面板属性
            lodash.forEach(wCalcRet, (v, k) => {
                k = k + (['hp', 'atk', 'def'].includes(k) ? 'Base' : '');
                self.addAttr(k, v, true);
            });
            // 检查武器类型
            if (weapon.type === char.weapon) {
                // todo sr&gs 统一
                let wBuffs = weapon.getWeaponAffixBuffs(wData.affix, true);
                lodash.forEach(wBuffs, (buff) => {
                    lodash.forEach(buff.data || [], (v, k) => {
                        self.addAttr(k, v);
                    });
                });
            }
            return;
        }
        // 原神属性
        if (wCalcRet) {
            this.addAttr('atkBase', wCalcRet.atkBase);
            this.addAttr((_b = wCalcRet.attr) === null || _b === void 0 ? void 0 : _b.key, (_c = wCalcRet.attr) === null || _c === void 0 ? void 0 : _c.value);
        }
        let { weaponBuffs } = Meta.getMeta('gs', 'weapon');
        let wBuffs = weaponBuffs[weapon.name] || [];
        if (lodash.isPlainObject(wBuffs)) {
            wBuffs = [wBuffs];
        }
        let affix = wData.affix || 1;
        lodash.forEach(wBuffs, (buff) => {
            if (!buff.isStatic) {
                return true;
            }
            if (buff) {
                lodash.forEach(buff.refine, (r, key) => {
                    this.addAttr(key, r[affix - 1] * (buff.buffCount || 1));
                });
            }
        });
    }
    /**
     * 计算圣遗物属性
     */
    setArtisAttr() {
        let profile = this.profile;
        let artis = profile === null || profile === void 0 ? void 0 : profile.artis;
        // 计算圣遗物词条
        artis.forEach((arti) => {
            this.calcArtisAttr(arti.main, profile.char);
            lodash.forEach(arti.attrs, (ds) => {
                this.calcArtisAttr(ds, profile.char);
            });
        });
        // 计算圣遗物静态加成
        artis.eachArtisSet((set, num) => {
            let buffs = ArtifactSet.getArtisSetBuff(set.name, num, this.game);
            if (!buffs)
                return true;
            lodash.forEach(buffs, (buff) => {
                if (!buff.isStatic) {
                    return true;
                }
                if (buff.elem && !profile.char.isElem(buff.elem)) {
                    return true;
                }
                lodash.forEach(buff.data, (val, key) => {
                    this.addAttr(key, val);
                });
            });
        });
    }
    /**
     * 计算单条圣遗物词缀
     * @param ds
     * @param char
     * @param autoPct
     * @returns {boolean}
     */
    calcArtisAttr(ds, char) {
        if (!ds) {
            return false;
        }
        let key = ds.key;
        if (Format.isElem(key)) {
            if (char.elem === key) {
                key = 'dmg';
            }
            else if (['electro', 'pyro', 'hydro', 'cryo'].includes(key)) {
                key = 'coloringDmg';
            }
        }
        if (!key) {
            return false;
        }
        if (['atk', 'hp', 'def'].includes(key)) {
            key = key + 'Pct';
        }
        this.attr.addAttr(key, ds.value * 1);
    }
}
exports.default = Attr;

return exports;})().default;
const ArtisMarkCfg=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const weaponCfg = {
    磐岩结绿: {
        attr: 'hp',
        abbr: '绿剑',
        max: 30,
        min: 15
    },
    猎人之径: {
        attr: 'mastery'
    },
    薙草之稻光: {
        attr: 'recharge',
        abbr: '薙刀'
    },
    护摩之杖: {
        attr: 'hp',
        abbr: '护摩',
        max: 18,
        min: 10
    }
};
const ArtisMarkCfg = {
    getCharArtisCfg(profile) {
        let { attr, weapon, elem, char, artis, game } = profile;
        let { isGs } = char;
        let { usefulAttr } = Meta.getMeta(game, 'arti');
        let rule = function (title, attrWeight) {
            return {
                title,
                attrWeight
            };
        };
        let def = function (attrWeight, title = []) {
            let weight = lodash.extend({}, attrWeight || usefulAttr[char.name] || {});
            let check = (key, max = 75, maxPlus = 75, isWeapon = true) => {
                let original = weight[key] || 0;
                if (original < max) {
                    let plus = isWeapon ? maxPlus * (1 + weapon.affix / 5) / 2 : maxPlus;
                    weight[key] = Math.min(Math.round(original + plus), max);
                    return true;
                }
                return false;
            };
            /* maxAffix_attr 精5武器权重
            minAffix_attr 精1武器权重 */
            let weaponCheck = (key, maxAffix_attr = 20, minAffix_attr = 10, max = 100) => {
                let original = weight[key] || 0;
                if (original == max) {
                    return false;
                }
                else {
                    let plus = minAffix_attr + (maxAffix_attr - minAffix_attr) * (weapon.affix - 1) / 4;
                    weight[key] = Math.min(Math.round(original + plus), max);
                    return true;
                }
            };
            let wn = (weapon === null || weapon === void 0 ? void 0 : weapon.name) || '';
            if (isGs) {
                // 对原神一些特殊情况做适配与判定
                // 增加攻击力或直接伤害类武器判定
                if (weight.atk > 0 && weaponCfg[wn]) {
                    let wCfg = weaponCfg[wn];
                    if (weaponCheck(wCfg.attr, wCfg.max || 20, wCfg.min || 10)) {
                        title.push(wCfg.abbr || wn);
                    }
                }
                // 圣遗物判定，如果是绝缘4，将充能权重拉高至沙漏圣遗物当前最高权重齐平
                let maxWeight = Math.max(weight.atk || 0, weight.hp || 0, weight.def || 0, weight.mastery || 0);
                if (artis.is('绝缘4') && check('recharge', maxWeight, 75, false)) {
                    title.push('绝缘4');
                }
                // 西风系列武器判定。如果携带西风系列武器，则暴击权重强制提高至 100
                if (/^西风(长枪|大剑|剑|猎弓|秘典)$/.test(wn) && weight.cpct < 100) {
                    weight['cpct'] = 100;
                    title.push('西风');
                }
            }
            title = title.length > 0 ? title.join('|') : '通用';
            return {
                title: `${char.abbr}-${title}`,
                attrWeight: weight
            };
        };
        let charRule = char.getArtisCfg() || function ({ def }) {
            let defaultAttrWeight = isGs ? { atk: 75, cpct: 100, cdmg: 100, dmg: 100, phy: 100 } : { atk: 75, cpct: 100, cdmg: 100, dmg: 100, speed: 100 };
            return def(usefulAttr[char.name] || defaultAttrWeight);
        };
        if (charRule) {
            return charRule({ attr, elem, artis, rule, def, weapon, cons: profile.cons });
        }
    },
    getCfg(profile) {
        let { char } = profile;
        let { game } = char;
        let { attrWeight, title } = ArtisMarkCfg.getCharArtisCfg(profile);
        let attrs = {};
        let baseAttr = char.baseAttr || { hp: 14000, atk: 230, def: 700 };
        let { attrMap } = Meta.getMeta(game, 'arti');
        lodash.forEach(attrMap, (attr, key) => {
            let k = attr.base || '';
            let weight = attrWeight[k || key];
            if (!weight || weight * 1 === 0) {
                return true;
            }
            let ret = {
                ...attr, weight, fixWeight: weight, mark: weight / attr.value
            };
            if (!k) {
                ret.mark = weight / attr.value;
            }
            else {
                let plus = k === 'atk' ? 520 : 0;
                ret.mark = weight / attrMap[k].value / (baseAttr[k] + plus) * 100;
                ret.fixWeight = weight * attr.value / attrMap[k].value / (baseAttr[k] + plus) * 100;
            }
            attrs[key] = ret;
        });
        let posMaxMark = ArtisMark.getMaxMark(attrs, game);
        // 返回内容待梳理简化
        return {
            attrs,
            classTitle: title,
            posMaxMark
        };
    }
};
exports.default = ArtisMarkCfg;

return exports;})().default;
const ArtisMark=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
let ArtisMark = {
    getKeyTitleMap(game = 'gs') {
        let ret = {};
        let { attrMap } = Meta.getMeta(game, 'arti');
        lodash.forEach(attrMap, (ds, key) => {
            ret[key] = ds.title;
        });
        Format.eachElem((key, name) => {
            ret[key] = `${name}伤加成`;
        }, game);
        return ret;
    },
    formatAttr(ds, game = 'gs') {
        if (!ds) {
            return {};
        }
        if (!ds.value) {
            return {};
        }
        return {
            key: ds.key || '',
            value: ds.value || ''
        };
    },
    /**
     * 格式化圣遗物词条
     * @param ds
     * @param charAttrCfg
     * @param isMain
     * @param game
     * @returns {{title: *, value: string}|*[]}
     */
    formatArti(ds, charAttrCfg = false, isMain = false, game = 'gs') {
        var _a, _b;
        // 若为attr数组
        if (ds[0] && (ds[0].title || ds[0].key)) {
            let ret = [];
            lodash.forEach(ds, (d) => {
                let arti = ArtisMark.formatArti(d, charAttrCfg, isMain, game);
                ret.push(arti);
            });
            return ret;
        }
        let key = ds.key;
        let isDmg = Format.isElem(key, game);
        let val = ds.value || ds[1];
        let num = ds.value || ds[1];
        if (!key || key === 'undefined') {
            return {};
        }
        let arrCfg = Meta.getMeta(game, 'arti', 'attrMap')[isDmg ? 'dmg' : key];
        val = Format[(arrCfg === null || arrCfg === void 0 ? void 0 : arrCfg.format) || 'comma'](val, 1);
        let ret = {
            key,
            value: val,
            upNum: ds.upNum || 0,
            eff: ds.eff || 0
        };
        if (charAttrCfg) {
            let mark = ((_a = charAttrCfg[key]) === null || _a === void 0 ? void 0 : _a.mark) * num || 0;
            if (isDmg) {
                mark = ((_b = charAttrCfg.dmg) === null || _b === void 0 ? void 0 : _b.mark) * num || 0;
            }
            if (isMain) {
                mark = mark / 4 + 0.01;
                ret.key = key;
            }
            ret.mark = Format.comma(mark || 0);
            ret._mark = mark || 0;
        }
        ret.eff = ret.eff ? Format.comma(ret.eff / (game === 'gs' ? 0.85 : 0.9), 1) : '-';
        return ret;
    },
    formatArtiAttrs(ds, charAttrCfg = false, game = 'gs') {
        let ret = [];
        lodash.forEach(ds, (d) => {
            let arti = ArtisMark.formatArti(d, charAttrCfg, false, game);
            ret.push(arti);
        });
        return ret;
    },
    // 获取评分档位
    getMarkClass(mark) {
        let pct = mark;
        let scoreMap = [['D', 7], ['C', 14], ['B', 21], ['A', 28], ['S', 35], ['SS', 42], ['SSS', 49], ['ACE', 56], ['MAX', 70]];
        for (let idx = 0; idx < scoreMap.length; idx++) {
            if (pct < scoreMap[idx][1]) {
                return scoreMap[idx][0];
            }
        }
    },
    // 获取位置分数
    getMark({ charCfg, idx, arti, elem = '', game = 'gs', id }) {
        var _a, _b, _c;
        let ret = 0;
        let mAttr = arti.main;
        let sAttr = arti.attrs;
        let { attrs, posMaxMark } = charCfg;
        let key = mAttr === null || mAttr === void 0 ? void 0 : mAttr.key;
        if (!key) {
            return 0;
        }
        let fixPct = 1;
        idx = idx * 1;
        if (idx >= 3) {
            let mainKey = key;
            if (key !== 'recharge') {
                let dmgIdx = { gs: 4, sr: 5 };
                if (idx === dmgIdx[game]) {
                    // 对法尔伽做特殊处理————所有异色属伤杯，在圣遗物评分时，均视为风伤杯
                    if (Format.sameElem(elem, key, game) || id === 10000128) {
                        mainKey = 'dmg';
                    }
                }
                let mMax = posMaxMark['m' + idx];
                fixPct = mMax > 0 ? Math.max(0, Math.min(1, (((_a = attrs[mainKey]) === null || _a === void 0 ? void 0 : _a.weight) || 0) / mMax)) : 1;
                if (game === 'gs') {
                    if (['atk', 'hp', 'def'].includes(mainKey) && ((_b = attrs[mainKey]) === null || _b === void 0 ? void 0 : _b.weight) >= 75) {
                        fixPct = 1;
                    }
                }
            }
            ret += (((_c = attrs[mainKey]) === null || _c === void 0 ? void 0 : _c.mark) || 0) * (mAttr.value || 0) / 4;
        }
        lodash.forEach(sAttr, (ds) => {
            var _a;
            ret += (((_a = attrs[ds.key]) === null || _a === void 0 ? void 0 : _a.mark) || 0) * (ds.value || 0);
        });
        let pMax = posMaxMark[idx];
        return pMax > 0 ? ret * (1 + fixPct) / 2 / pMax * 66 : 0;
    },
    // 获取位置最高分
    getMaxMark(attrs, game = 'gs') {
        let ret = {};
        let { mainAttr, subAttr } = Meta.getMeta(game, 'arti');
        for (let idx = 1; idx <= (game === 'gs' ? 5 : 6); idx++) {
            let totalMark = 0;
            let mMark = 0;
            let mAttr = '';
            if (idx === 1) {
                mAttr = 'hpPlus';
            }
            else if (idx === 2) {
                mAttr = 'atkPlus';
            }
            else if (idx >= 3) {
                let mainCandidates = ArtisMark.getMaxAttr(attrs, mainAttr[idx]);
                if (mainCandidates.length > 0) {
                    mAttr = mainCandidates[0];
                    mMark = attrs[mAttr].fixWeight;
                    totalMark += mMark * 2;
                }
                else {
                    mAttr = mainAttr[idx][0];
                }
            }
            let sAttr = ArtisMark.getMaxAttr(attrs, subAttr, 4, mAttr);
            lodash.forEach(sAttr, (attr, aIdx) => {
                totalMark += attrs[attr].fixWeight * (aIdx === 0 ? 6 : 1);
            });
            ret[idx] = totalMark;
            ret['m' + idx] = mMark;
        }
        return ret;
    },
    // 获取最高分的属性
    getMaxAttr(attrs = {}, list2 = [], maxLen = 1, banAttr = '') {
        let tmp = [];
        lodash.forEach(list2, (attr) => {
            if (attr === banAttr)
                return;
            if (!attrs[attr])
                return;
            tmp.push({ attr, mark: attrs[attr].fixWeight });
        });
        tmp = lodash.sortBy(tmp, 'mark');
        tmp = tmp.reverse();
        let ret = [];
        lodash.forEach(tmp, (ds) => ret.push(ds.attr));
        return ret.slice(0, maxLen);
    },
    getMarkDetail(profile, withDetail = true) {
        if (!profile.isProfile) {
            return {};
        }
        let charCfg = ArtisMarkCfg.getCfg(profile);
        let artisRet = {};
        let setCount = {};
        let totalMark = 0;
        let { game, artis, elem, id } = profile;
        artis.forEach((arti, idx) => {
            let mark = ArtisMark.getMark({ charCfg, idx, arti, elem, game, id });
            totalMark += mark;
            setCount[arti.set] = (setCount[arti.set] || 0) + 1;
            artisRet[idx] = {
                _mark: mark,
                mark: Format.comma(mark, 1),
                markClass: ArtisMark.getMarkClass(mark)
            };
            if (withDetail) {
                let artifact = Artifact.get(arti, game);
                artisRet[idx] = {
                    ...artifact.getData('name,abbr,set:setName,img'),
                    level: arti.level,
                    main: ArtisMark.formatArti(arti.main, charCfg.attrs, true, game),
                    attrs: ArtisMark.formatArtiAttrs(arti.attrs, charCfg.attrs, game),
                    ...artisRet[idx]
                };
            }
        });
        let setData = artis.getSetData();
        artis.mark = totalMark;
        artis.markClass = ArtisMark.getMarkClass(totalMark / (profile.isGs ? 5 : 6));
        let ret = {
            classTitle: charCfg.classTitle,
            artis: artisRet,
            mark: Format.comma(totalMark, 1),
            _mark: artis.mark,
            markClass: artis.markClass,
            ...Data.getData(setData, 'sets,names,imgs')
        };
        if (withDetail) {
            ret.charWeight = lodash.mapValues(charCfg.attrs, ds => ds.weight);
        }
        return ret;
    }
};
exports.default = ArtisMark;

return exports;})().default;
