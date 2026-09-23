var scoreRule=(()=>{const exports={};
"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = default_1;
/** @type {import('#interface').scoreFunction} */
function default_1(avatar) {
    if (avatar.rank >= 4)
        return ['高影画', { "暴击伤害": 0.25 }];
}

return exports;})();
