# 资料来源

本插件的固定资料由 `game-plugin-kit/scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留角色、装备名称、属性、技能文字与材料；不执行上游脚本，不包含完整伤害计算或全部素材。

绝区零资料来自 ZZZ-Plugin dev，固定提交 fb66219cec0294e1834bacdf0033b2d43a9ccaf4；插件按 AGPL-3.0-only 分发，保留 LICENSES/ZZZ-Plugin-AGPL-3.0.txt。分发时同时提供此插件与实际构建依赖的对应源码。

公开展柜来自 [Enka.Network](https://github.com/EnkaNetwork/API-docs/blob/master/api.md)，按响应 TTL 缓存。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件及编译期业务库沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## 装备评分

评分核心按固定参考原生实现：miao `ArtisMark`/`ArtisMarkCfg` 与 ZZZ `Score`。转换脚本 `game-plugin-kit/scripts/import-score-data.py` 只读取静态数字对象与 JSON，不运行原插件。对应版权许可保留在 `LICENSES/`，评分结果带规则版本。

原神、星铁采用 miao 提交 `7f6f1c84c89102bc6b1c58c8e1b06c61f4642161` 的基础权重；角色专属 `artis.js` 已由受限语法转换器编译为 Go，并与原始纯函数进行固定输入对照。绝区零采用 dev 提交 `fb66219cec0294e1834bacdf0033b2d43a9ccaf4` 的十个显式预设；动态模式已原生适配三份角色覆盖函数和条件预设选择，不在插件运行期执行 JS。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存完整的绝区零计算运行时：`data.js`（上游 ID 映射）、`common.js`（Calculator 与 BuffManager）、`buffs.js`（音擎与驱动盘效果）、手写的 `bootstrap.js` 与 `runner.js`，以及角色脚本 `characters/<ID>-<角色名>.js`。除手写的两个文件外均由相邻库的 `scripts/bundle-zzz-calculation.mjs` 从 ZZZ-Plugin dev 固定快照生成。这些文件只编入本插件。

## 模拟与固定资料

材料、卡池与固定生日从同一 miao 固定快照的 material/info 数据转换，不联网补造日程。娱乐模拟的默认曲线来自 [Miao-Yunzai 固定提交](https://github.com/yoimiya-kokomi/Miao-Yunzai/tree/40cc2103efba1fbb279b768e3f5345d45372d357) 和 [StarRail-plugin 固定提交](https://github.com/TsukinaKasumi/StarRail-plugin/tree/090e411cf9be28b1644721fab54eab0ad283668f)，转换为 Go，不在运行时加载原插件。只向原神和星铁开放模拟入口；不声明为当前官方概率。

GPL-3.0 和 Apache-2.0 许可分别保留在 `LICENSES/Miao-Yunzai-GPL-3.0.txt` 与 `LICENSES/StarRail-plugin-Apache-2.0.txt`，对应的转换器、数据和原生实现随源码分发。转换模拟数据需要已有 Python/PyYAML，正常构建使用已生成数据，无需该再生成步骤。


绝区零卡池历史使用 [GachaClock 固定提交](https://github.com/iaoongin/GachaClock/tree/99d16c10bfeeb5f885e9cb42861c50f993f3a746) 的 JSON 数据，保留 `LICENSES/GachaClock-MIT.txt`。110 条收录中 54 个起点按参考规则推算并标记；当前快照末期结束日为 2026-05-05，不能代表最新官方排期。仅转换文字和日期，没有下载或分发关联图片。原神/星铁仍使用 miao 同一固定快照的数据。


原魔属性使用 [Atlas 固定提交](https://github.com/Nwflower/atlas/tree/016e49357666e0823791abdf28fbb3b2efe68225) 的文字数值资料：556 项条目、200 级曲线、102 项修饰因子，仅在原神界面开放。保留 `LICENSES/Atlas-GPL-3.0.txt`，转换器为编译期库的 `scripts/import-enemy-data.py`；未引入其外部图鉴图片仓库。攻略合集编号来自上述固定 Yunzai/StarRail/ZZZ 参考，运行时匿名读取官方文字与原图链接，不打包攻略图。

## 计算脚本的兼容修正

固定来源为 ZZZ-Plugin dev `fb66219cec0294e1834bacdf0033b2d43a9ccaf4`，许可见 `LICENSES/ZZZ-Plugin-AGPL-3.0.txt`。

- 柏妮思六影的内层“灼烧每段”直接调用 `calc_skill`，绕过 `new` 对异常类型的识别，因而查询不存在的技能倍率。打包器只为该内层对象补上 `isAnomalyDMG=true`，沿用原灼烧倍率、18 次系数及条件。
- 计算器会把当前技能的临时属性写回 `skill.props`，Buff 注册也会修改规则对象。基准、候选和后续请求各自复制固定角色/音擎/套装定义，防止露西等角色的技能缓存污染另一套配装。
- 使用原始属性格式化和元素映射模块，明确区分百分数与比例。内部日志不输出数据；遇到未处理的规则错误/缺失倍率立即失败，不沿用参考的静默跳过行为生成不完整结果。

## 音擎补充与自定义条件

`scripts/calc-supplements/weapons.js` 为固定描述完整的 33 项音擎补充被动，使用五档原始数值，不外推后续版本。现有可计算音擎资料由 67 项扩展为 100 项；14004 的名称为占位符且无效果描述，不伪造被动。锋御的四份音擎使用独立数值/技能范围测试，因为同快照没有对应职业角色计算脚本。

非伤害的能量、失衡、防护、积蓄与持续时间保留完整说明；比格气缸附加伤害只显示描述可确定的防御倍率基础值，未指定的元素不作猜测。血髓秘匣读取截断前暴击率后再计算增伤上限；元素、职业和技能限定保留，未来异常类型不冒充已有异常。

自定义条件只修改候选情境，基准不变。队友由用户选择并填写实际增益，不自动假定角色等级或装备；原神/星铁按原乘区应用，绝区零的百分比基础属性用明确函数转换，避免把 100% 当固定一点。局内来源开关保留静态面板，绝区零保留驱动盘主属性及两件常驻属性。无额外条件的旧参考向量保持一致。

官方面板导出复用突破匹配，仅在内部 `resolve_identity` 输入时返回匹配出的角色/武器突破；普通试算输出保持一致。星铁 HP/攻击/防御词条的两组编号按明确的百分号单位区分百分比与固定值，兼容参考和实际接口编号族。交换格式中的强化档位按官方显示值和次数近似还原，不声明为真实升级顺序。
