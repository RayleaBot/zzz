const calcFnc={character:{},weapon:{},set:{}};
calcFnc.weapon["「恒等式」-本格"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '防御力',
        value: [0.2, 0.23, 0.26, 0.29, 0.32]
    }
];

return exports;})();
calcFnc.weapon["「月相」-晦"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.15, 0.175, 0.2, 0.225, 0.25]
    }
];

return exports;})();
calcFnc.weapon["「月相」-望"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.12, 0.14, 0.16, 0.18, 0.2],
        range: ['A', 'CC', 'CF']
    }
];

return exports;})();
calcFnc.weapon["「残响」-Ⅰ型"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '冲击力',
        teamTarget: true,
        stackable: false,
        value: [0.08, 0.09, 0.1, 0.11, 0.12]
    }
];

return exports;})();
calcFnc.weapon["「残响」-Ⅱ型"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常掌控',
        teamTarget: true,
        stackable: false,
        value: [10, 12, 13, 15, 16]
    },
    {
        type: '异常精通',
        teamTarget: true,
        stackable: false,
        value: [10, 12, 13, 15, 16]
    }
];

return exports;})();
calcFnc.weapon["「残响」-Ⅲ型"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        teamTarget: true,
        stackable: false,
        value: [0.08, 0.09, 0.1, 0.11, 0.12]
    }
];

return exports;})();
calcFnc.weapon["「湍流」-斧型"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '冲击力',
        value: [0.09, 0.1, 0.11, 0.12, 0.13]
    }
];

return exports;})();
calcFnc.weapon["「灰烬」-钴蓝"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.072, 0.082, 0.093, 0.104, 0.115]
    }
];

return exports;})();
calcFnc.weapon["「电磁暴」-壹式"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常掌控',
        value: [25, 28, 32, 36, 40]
    }
];

return exports;})();
calcFnc.weapon["「电磁暴」-贰式"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常精通',
        value: [25, 28, 32, 36, 40]
    }
];

return exports;})();
calcFnc.weapon["云霓孤光"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '无视抗性',
        value: [0.2, 0.22, 0.24, 0.26, 0.28],
        element: 'Physical'
    },
    {
        type: '增伤',
        value: [0.25, 0.287, 0.325, 0.362, 0.4] // [以太帷幕]直接视为生效
    },
    {
        type: '暴击伤害',
        value: [0.25, 0.287, 0.325, 0.362, 0.4]
    }
];

return exports;})();
calcFnc.weapon["人为刀俎"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '冲击力',
        value: [0.02, 0.023, 0.026, 0.029, 0.032].map(v => v * 8)
    }
];

return exports;})();
calcFnc.weapon["仿制星徽引擎"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.36, 0.41, 0.465, 0.52, 0.575],
        element: 'Physical'
    }
];

return exports;})();
calcFnc.weapon["兔能环"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '生命值',
        value: [0.08, 0.092, 0.104, 0.116, 0.128]
    },
    {
        type: '攻击力',
        value: [0.1, 0.115, 0.13, 0.145, 0.16]
    }
];

return exports;})();
calcFnc.weapon["加农转子"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.075, 0.086, 0.097, 0.108, 0.12]
    }
];

return exports;})();
calcFnc.weapon["十方锻星"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常掌控',
        value: [60, 69, 78, 87, 96]
    },
    {
        type: '增伤',
        value: [0.2, 0.23, 0.26, 0.29, 0.32].map(v => v * 2),
        element: 'Physical'
    }
];

return exports;})();
calcFnc.weapon["千面日陨"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击伤害',
        value: [0.45, 0.5175, 0.585, 0.6525, 0.72]
    },
    {
        type: '无视防御',
        value: [0.25, 0.2875, 0.325, 0.3625, 0.4],
        element: 'Ice',
        range: ['EQ', 'RL', 'RZ'] // 仅持续3s，认为只作用于技能自身
    }
];

return exports;})();
calcFnc.weapon["双生泣星"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常精通',
        value: [30, 34, 38, 42, 48].map(v => v * 4)
    }
];

return exports;})();
calcFnc.weapon["含羞恶面"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.15, 0.175, 0.2, 0.22, 0.24],
        element: 'Ice'
    },
    {
        type: '攻击力',
        value: [0.02, 0.023, 0.026, 0.029, 0.032].map(v => v * 4)
    }
];

return exports;})();
calcFnc.weapon["啜泣摇篮"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        teamTarget: true,
        stackable: false,
        value: [0.1, 0.125, 0.15, 0.175, 0.2]
    },
    {
        type: '增伤',
        teamTarget: true,
        stackable: false,
        value: [0.017, 0.02, 0.025, 0.03, 0.033].map(v => v * 6)
    }
];

return exports;})();
calcFnc.weapon["奔袭獠牙"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        teamTarget: true,
        stackable: false,
        value: [0.18, 0.225, 0.27, 0.315, 0.36]
    }
];

return exports;})();
calcFnc.weapon["好斗的阿炮"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.025, 0.028, 0.032, 0.036, 0.04].map(v => v * 4),
    }
];

return exports;})();
calcFnc.weapon["家政员"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.03, 0.035, 0.04, 0.044, 0.048].map(v => v * 15),
        element: 'Physical',
        range: ['EQ'] // 只持续1s
    }
];

return exports;})();
calcFnc.weapon["嵌合编译器"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.12, 0.15, 0.18, 0.21, 0.24]
    },
    {
        type: '异常精通',
        value: [25, 31, 37, 43, 50].map(v => v * 3)
    }
];

return exports;})();
calcFnc.weapon["幻变魔方"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击伤害',
        value: [0.16, 0.184, 0.208, 0.232, 0.256]
    },
    {
        type: '增伤',
        value: [0.2, 0.23, 0.26, 0.29, 0.32],
        range: ['EQ']
    }
];

return exports;})();
calcFnc.weapon["强音热望"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.06, 0.069, 0.078, 0.087, 0.096].map(v => v * 2)
    }
];

return exports;})();
calcFnc.weapon["德玛拉电池Ⅱ型"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.15, 0.175, 0.2, 0.22, 0.24],
        element: 'Electric'
    }
];

return exports;})();
calcFnc.weapon["心弦夜响"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击伤害',
        value: [0.5, 0.575, 0.65, 0.725, 0.8]
    },
    {
        type: '无视抗性',
        value: [0.125, 0.145, 0.165, 0.185, 0.2].map(v => v * 2),
        element: 'Fire',
        range: ['RL', 'RZ']
    }
];

return exports;})();
calcFnc.weapon["怒目金刚"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: "暴击率",
        value: [0.2, 0.23, 0.26, 0.29, 0.32],
    },
    {
        type: "贯穿增伤",
        value: [0.09, 0.1035, 0.117, 0.1305, 0.144].map(v => v * 2),
        element: "Fire",
    },
];

return exports;})();
calcFnc.weapon["思络成歌"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.125, 0.143, 0.161, 0.179, 0.2].map(v => v * 2),
        teamTarget: true,
        check: ({ avatar, runtime }) => avatar.element_type === runtime.elementEnum.Physical,
        stackable: false,
    },
    {
        type: '攻击力',
        value: [0.1, 0.115, 0.13, 0.145, 0.16],
        teamTarget: true,
        check: ({ avatar, runtime }) => avatar.element_type === runtime.elementEnum.Physical,
        stackable: false,
    }
];

return exports;})();
calcFnc.weapon["拘缚者"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.06, 0.075, 0.09, 0.105, 0.12].map(v => v * 5),
        range: ['A']
    }
];

return exports;})();
calcFnc.weapon["旋钻机-赤轴"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.5, 0.575, 0.65, 0.725, 0.8],
        element: 'Electric',
        range: ['A', 'CC']
    }
];

return exports;})();
calcFnc.weapon["时流贤者"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常精通',
        value: [75, 85, 95, 105, 115]
    },
    {
        type: '异常增伤',
        value: [0.25, 0.275, 0.3, 0.325, 0.35],
        check: ({ calc }) => calc.get_AnomalyProficiency() >= 375,
        range: ['紊乱']
    }
];

return exports;})();
calcFnc.weapon["星徽引擎"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.12, 0.138, 0.156, 0.174, 0.192]
    }
];

return exports;})();
calcFnc.weapon["机巧心种"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击率',
        value: [0.15, 0.17, 0.19, 0.21, 0.23]
    },
    {
        type: '增伤',
        value: [0.125, 0.145, 0.165, 0.185, 0.2].map(v => v * 2),
        element: 'Electric'
    },
    {
        type: '无视防御',
        value: [0.2, 0.23, 0.26, 0.29, 0.32],
        range: ['A', 'RZ']
    }
];

return exports;})();
calcFnc.weapon["正版变身器"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '生命值',
        value: [0.08, 0.09, 0.1, 0.11, 0.125]
    },
    {
        type: '冲击力',
        value: [0.1, 0.115, 0.13, 0.145, 0.16]
    }
];

return exports;})();
calcFnc.weapon["残心青囊"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击率',
        value: [0.1, 0.115, 0.13, 0.145, 0.16]
    },
    {
        type: '增伤',
        value: [0.4, 0.46, 0.52, 0.58, 0.64],
        element: 'Electric',
        range: ['CC']
    },
    {
        type: '暴击率',
        value: [0.1, 0.115, 0.13, 0.145, 0.16]
    }
];

return exports;})();
calcFnc.weapon["淬锋钳刺"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.12, 0.15, 0.18, 0.21, 0.24].map(v => v * 3),
        element: 'Physical'
    }
];

return exports;})();
calcFnc.weapon["深海访客"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.25, 0.315, 0.38, 0.445, 0.5],
        element: 'Ice'
    },
    {
        type: '暴击率',
        value: [0.1, 0.125, 0.15, 0.175, 0.2].map(v => v * 2)
    }
];

return exports;})();
calcFnc.weapon["灼心摇壶"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.035, 0.044, 0.052, 0.061, 0.07].map(v => v * 10)
    },
    {
        type: '异常精通',
        value: [50, 62, 75, 87, 100]
    }
];

return exports;})();
calcFnc.weapon["焰心桂冠"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '冲击力',
        value: [0.25, 0.2875, 0.325, 0.3625, 0.4]
    },
    {
        type: '暴击伤害',
        teamTarget: true,
        stackable: false,
        value: [0.015, 0.0172, 0.0195, 0.0217, 0.024].map(v => v * 20),
        element: ['Ice', 'Fire']
    }
];

return exports;})();
calcFnc.weapon["燃狱齿轮"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '冲击力',
        value: [0.1, 0.125, 0.15, 0.175, 0.2].map(v => v * 2)
    }
];

return exports;})();
calcFnc.weapon["燔火胧夜"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: "增伤",
        value: [0.15, 0.1725, 0.195, 0.2175, 0.24],
        element: "Fire",
    },
    {
        type: "暴击率", // 装备者的生命值降低时
        value: [0.15, 0.1725, 0.195, 0.2175, 0.24],
    },
];

return exports;})();
calcFnc.weapon["牺牲洁纯"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击伤害',
        value: [0.3, 0.345, 0.39, 0.435, 0.48]
    },
    {
        type: '暴击伤害',
        value: [0.1, 0.115, 0.13, 0.145, 0.16].map(v => v * 3),
    },
    {
        type: '增伤',
        value: [0.2, 0.23, 0.26, 0.29, 0.32],
        element: 'Electric'
    }
];

return exports;})();
calcFnc.weapon["狸法七变化"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常掌控',
        value: [30, 34, 39, 43, 48],
        check: ({ avatar }) => avatar.element_type === 200
    },
    {
        type: '异常精通',
        teamTarget: true,
        stackable: false,
        value: [60, 69, 78, 87, 96]
    }
];

return exports;})();
calcFnc.weapon["玉壶青冰"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '冲击力',
        value: [0.007, 0.0088, 0.0105, 0.0122, 0.014].map(v => v * 30)
    },
    {
        type: '增伤',
        teamTarget: true,
        stackable: false,
        value: [0.2, 0.23, 0.26, 0.29, 0.32]
    }
];

return exports;})();
calcFnc.weapon["玲珑妆匣"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        teamTarget: true,
        stackable: false,
        value: [0.1, 0.115, 0.13, 0.145, 0.16].map(v => v * 2)
    }
];

return exports;})();
calcFnc.weapon["电波漫步"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '贯穿力',
        value: [80, 92, 104, 116, 128].map(v => v * 3)
    }
];

return exports;})();
calcFnc.weapon["硫磺石"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.035, 0.044, 0.052, 0.06, 0.07].map(v => v * 8)
    }
];

return exports;})();
calcFnc.weapon["福虓炉炉"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        check: ({ avatar }) => avatar.element_type === 201,
        value: [0.1, 0.115, 0.13, 0.145, 0.16].map(v => v * 2),
        teamTarget: true,
        stackable: false
    }
];

return exports;})();
calcFnc.weapon["索魂影眸"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '无视防御',
        teamTarget: true,
        stackable: false,
        value: [0.25, 0.2875, 0.325, 0.3625, 0.4]
    },
    {
        type: '冲击力',
        value: [0.04, 0.046, 0.052, 0.058, 0.064].map(v => v * 3)
    },
    {
        type: '冲击力',
        value: [0.08, 0.092, 0.104, 0.116, 0.128]
    }
];

return exports;})();
calcFnc.weapon["聚宝箱"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.15, 0.175, 0.2, 0.22, 0.24],
        check: ({ avatar }) => avatar.element_type === 205
    }
];

return exports;})();
calcFnc.weapon["街头巨星"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.15, 0.172, 0.195, 0.217, 0.24].map(v => v * 3),
        range: ['RZ']
    }
];

return exports;})();
calcFnc.weapon["裁纸刀"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.15, 0.173, 0.195, 0.218, 0.24],
        element: 'Physical'
    }
];

return exports;})();
calcFnc.weapon["触电唇彩"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.1, 0.115, 0.13, 0.145, 0.16]
    },
    {
        type: '增伤',
        value: [0.15, 0.175, 0.2, 0.225, 0.25]
    }
];

return exports;})();
calcFnc.weapon["轰鸣座驾"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.08, 0.092, 0.104, 0.116, 0.128]
    },
    {
        type: '异常精通',
        value: [40, 46, 52, 58, 64]
    }
];

return exports;})();
calcFnc.weapon["辉骑面铠"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: "暴击率",
        value: [0.2, 0.23, 0.26, 0.29, 0.32],
    },
    {
        type: "贯穿增伤",
        value: [0.1, 0.115, 0.13, 0.145, 0.16].map(v => v * 2),
        element: "Physics",
    },
];

return exports;})();
calcFnc.weapon["逍遥游球"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击率',
        value: [0.12, 0.135, 0.155, 0.175, 0.2]
    }
];

return exports;})();
calcFnc.weapon["鎏金花信"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.06, 0.069, 0.078, 0.087, 0.096]
    },
    {
        type: '增伤',
        value: [0.15, 0.172, 0.195, 0.218, 0.24],
        range: ['EQ']
    }
];

return exports;})();
calcFnc.weapon["钢铁肉垫"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.2, 0.25, 0.3, 0.35, 0.4],
        element: 'Physical'
    },
    {
        type: '增伤',
        value: [0.25, 0.315, 0.38, 0.44, 0.5]
    }
];

return exports;})();
calcFnc.weapon["防暴者Ⅵ型"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击率',
        value: [0.15, 0.188, 0.226, 0.264, 0.3]
    },
    {
        type: '增伤',
        value: [0.35, 0.435, 0.52, 0.605, 0.7],
        element: 'Ether',
        range: ['A']
    }
];

return exports;})();
calcFnc.weapon["雨林饕客"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: [0.025, 0.028, 0.032, 0.036, 0.04].map(v => v * 10)
    }
];

return exports;})();
calcFnc.weapon["震元奇枢"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: [0.25, 0.287, 0.325, 0.362, 0.4],
        range: ['EQ', 'RZ']
    }
];

return exports;})();
calcFnc.weapon["霰落星殿"]=(()=>{const exports={};
"use strict";
// 函数导出：
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/**
 * @param {import('#interface').BuffManager} buffM
 * @param {number} star 进阶星数
 */
// export function calc(buffM, star) {
//   buffM.new({
//     type: '暴击伤害',
//     value: [0.5, 0.57, 0.65, 0.72, 0.8][star - 1]
//   })
//   buffM.new({
//     type: '增伤',
//     value: [0.2, 0.23, 0.26, 0.29, 0.32][star - 1] * 2,
//     element: 'Ice'
//   })
// }
// 直接导出：
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击伤害',
        value: [0.5, 0.57, 0.65, 0.72, 0.8]
    },
    {
        type: '增伤',
        value: [0.2, 0.23, 0.26, 0.29, 0.32].map(v => v * 2),
        element: 'Ice'
    }
];

return exports;})();
calcFnc.weapon["青溟笼舍"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击率',
        value: [0.2, 0.23, 0.26, 0.29, 0.32]
    },
    {
        type: '增伤',
        value: [0.08, 0.092, 0.104, 0.116, 0.128].map(v => v * 2),
        element: 'Ether'
    },
    {
        type: '贯穿增伤',
        value: [0.1, 0.115, 0.13, 0.145, 0.16].map(v => v * 2),
        element: 'Ether',
        range: ['RZ', 'EQ']
    }
];

return exports;})();
calcFnc.weapon["青漪灵鼎"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: "增伤", // 装备者发动 [强化特殊技] 时
        value: [0.04, 0.046, 0.052, 0.058, 0.064].map(v => v * 3),
    },
    {
        type: "暴击率", // 拥有3层增益效果时
        value: [0.065, 0.075, 0.085, 0.094, 0.104],
    },
];

return exports;})();
calcFnc.weapon["飞鸟星梦"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常精通',
        value: [20, 23, 26, 29, 32].map(v => v * 6)
    }
];

return exports;})();
calcFnc.set["云岿如我"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击率',
        value: 0.04 * 3,
        check: 4
    },
    {
        type: '贯穿增伤',
        value: 0.1,
        check: 4
    }
];

return exports;})();
calcFnc.set["原始朋克"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.15,
        teamTarget: true,
        stackable: false,
        check: 4
    }
];

return exports;})();
calcFnc.set["啄木鸟电音"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: 0.09 * 3,
        check: 4
    }
];

return exports;})();
calcFnc.set["如影相随"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.15,
        check: 2,
        range: ['CC', '追加攻击']
    },
    {
        type: '攻击力',
        value: 0.04 * 3,
        check: 4
    },
    {
        type: '暴击率',
        value: 0.04 * 3,
        check: 4
    }
];

return exports;})();
calcFnc.set["山大王"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '暴击伤害',
        teamTarget: true,
        stackable: false,
        value: ({ calc }) => {
            if (calc.get_CRITRate() >= 0.5) {
                return 0.3;
            }
            return 0.15;
        },
        check: ({ buffM, avatar, runtime }) => buffM.setCount.山大王 >= 4 && avatar.avatar_profession === runtime.professionEnum.击破
    }
];

return exports;})();
calcFnc.set["折枝剑歌"]=(()=>{const exports={};
"use strict";
// 函数导出：
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/**
 * @param {import('#interface').BuffManager} buffM
 * @param {number} count 套装数量
 */
// export function calc(buffM, count) {
//   const name = buffM.defaultBuff.name
//   switch (true) {
//     case (count >= 4):
//       buffM.new({
//         name: name + '4',
//         type: '暴击伤害',
//         value: 0.3,
//         check: ({ buffM, calc }) => calc.get_AnomalyMastery() >= 115
//       })
//       buffM.new({
//         name: name + '4',
//         type: '暴击率',
//         value: 0.12
//       })
//   }
// }
// 直接导出：
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        name: '折枝剑歌4',
        type: '暴击伤害',
        value: 0.3,
        check: ({ buffM, calc }) => buffM.setCount.折枝剑歌 >= 4 && calc.get_AnomalyMastery() >= 115
    },
    {
        type: '暴击率',
        value: 0.12,
        check: 4
    }
];

return exports;})();
calcFnc.set["拂晓生花"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.15,
        check: 2,
        range: ['A']
    },
    {
        type: '增伤',
        value: 0.2,
        check: 4,
        range: ['A']
    },
    {
        type: '增伤',
        value: 0.2,
        check: ({ avatar, buffM, runtime }) => buffM.setCount.拂晓生花 >= 4 && avatar.avatar_profession === runtime.professionEnum.强攻, // 仅强攻角色
        range: ['A']
    }
];

return exports;})();
calcFnc.set["摇摆爵士"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.15,
        teamTarget: true,
        stackable: false,
        check: 4
    }
];

return exports;})();
calcFnc.set["月光骑士颂"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.18,
        teamTarget: true,
        stackable: false,
        check: ({ avatar, buffM, runtime }) => buffM.setCount.月光骑士颂 >= 4 && avatar.avatar_profession === runtime.professionEnum.支援 // 仅支援角色
    }
];

return exports;})();
calcFnc.set["极地重金属"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.1,
        element: 'Ice',
        check: 2
    },
    {
        type: '增伤',
        value: 0.2,
        range: ['A', 'CC'],
        check: 4
    },
    {
        type: '增伤',
        value: 0.2,
        range: ['A', 'CC'],
        check: 4
    }
];

return exports;})();
calcFnc.set["沧浪行歌"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        check: 2,
        type: '增伤',
        value: 0.1,
        element: 'Physical',
    },
    {
        check: 4,
        type: '暴击率', // [以太帷幕]直接视为生效
        value: 0.1,
    },
    {
        check: ({ buffM, avatar, runtime }) => buffM.setCount.沧浪行歌 >= 4 && avatar.avatar_profession === runtime.professionEnum.强攻,
        type: '暴击率',
        value: 0.1,
    },
    {
        check: ({ buffM, avatar, runtime }) => buffM.setCount.沧浪行歌 >= 4 && avatar.avatar_profession === runtime.professionEnum.强攻,
        type: '攻击力',
        value: 0.1,
    }
];

return exports;})();
calcFnc.set["河豚电音"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.2,
        range: ['RZ'],
        check: 4
    },
    {
        type: '攻击力',
        value: 0.15,
        check: 4
    }
];

return exports;})();
calcFnc.set["法厄同之歌"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '异常精通',
        value: 45,
        check: 4
    },
    {
        type: '增伤',
        value: 0.25,
        element: 'Ether',
        check: 4
    }
];

return exports;})();
calcFnc.set["流光咏叹"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        check: 2,
        type: '增伤',
        value: 0.1,
        element: 'Ether',
    },
    {
        check: 4,
        type: '异常精通',
        value: 36,
    },
    {
        check: 4,
        type: '增伤',
        value: 0.25,
    }
];

return exports;})();
calcFnc.set["混沌爵士"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.15,
        element: ['Electric', 'Fire'],
        check: 4
    },
    {
        type: '增伤',
        value: 0.2,
        check: 4,
        range: ['EQ', 'L']
    }
];

return exports;})();
calcFnc.set["混沌重金属"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.1,
        element: 'Ether',
        check: 2
    },
    {
        type: '暴击伤害',
        value: 0.2,
        check: 4
    },
    {
        type: '暴击伤害',
        value: 0.055 * 6,
        check: 4
    }
];

return exports;})();
calcFnc.set["激素朋克"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '攻击力',
        value: 0.25,
        check: 4
    }
];

return exports;})();
calcFnc.set["炎狱重金属"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.1,
        element: 'Fire',
        check: 2
    },
    {
        type: '暴击率',
        value: 0.28,
        check: 4
    }
];

return exports;})();
calcFnc.set["獠牙重金属"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.1,
        check: 2,
        element: 'Physical'
    },
    {
        type: '增伤',
        value: 0.35,
        check: 4
    }
];

return exports;})();
calcFnc.set["雷暴重金属"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.1,
        check: 2,
        element: 'Electric'
    },
    {
        type: '攻击力',
        value: 0.28,
        check: 4
    }
];

return exports;})();
calcFnc.set["静听嘉音"]=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        type: '增伤',
        value: 0.08 * 3,
        check: 4,
        teamTarget: true,
        stackable: false
    }
];

return exports;})();

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

