var characterRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.skills = exports.buffs = void 0;
/** @type {import('#interface').buff[]} */
exports.buffs = [
    {
        name: '6影',
        type: '增伤',
        value: 0.45,
        range: ['A', 'CC']
    }
];
/** @type {import('#interface').skill[]} */
exports.skills = [
    { name: '感电每段', type: '感电' },
    { name: '普攻：落雷', type: 'AX' },
    { name: '闪避反击：迅雷', type: 'CF' },
    { name: '强化特殊技：苍雷斩', type: 'EQ' },
    { name: '连携技：电磁引擎', type: 'RL' },
    { name: '终结技：过载引擎', type: 'RZ' }
];

return exports;})();
