# 资料来源

本插件的固定资料由 `scripts/import-reference-data.py` 从已下载参考 JSON 转换，源提交记录在 `internal/assets/catalog.json`。转换保留代理人、音擎与邦布的名称、稀有度、属性、特性、描述与基础属性，不包含素材图片；内置别名由 `scripts/import-aliases.py` 取自上游 `defSet/alias.yaml`。伤害与评分由打包的上游脚本计算，见下文。

绝区零资料来自 ZZZ-Plugin dev，固定提交 fb66219cec0294e1834bacdf0033b2d43a9ccaf4；插件按 AGPL-3.0-only 分发，保留 LICENSES/ZZZ-Plugin-AGPL-3.0.txt。分发时同时提供此插件与实际构建依赖的对应源码。

没有账号时的面板来自 [Enka.Network](https://github.com/EnkaNetwork/API-docs/blob/master/api.md)（ZZZ-Plugin 的默认 enkaApi），按响应 TTL 缓存，由打包的 ZZZ-Plugin `model/Enka/formater` 转为官方格式。`templates/panel-list/` 按 ZZZ-Plugin 的 `panel/list` 与 `panel/refresh` 改写（两页合为一个模板）。抽卡导入导出遵循 [UIGF](https://uigf.org/en/standards/uigf.html)，尚未申请兼容性认证。

本插件沿用 RayleaBot SDK 的 AGPL-3.0 许可，安装包管理页提供对应源码下载。上游数据的原许可声明另行保留。

## 装备评分

评分运行 dev 提交 `fb66219cec0294e1834bacdf0033b2d43a9ccaf4` 的 `Score`（含预设权重与按属性选择的规则）、`lib/score` 与角色专属 `score.js`，驱动盘与整套评级沿用上游 `Equip` 与角色模型的评级阈值。`EquipScore.json`、`EquipMainStats.json` 与 `EquipBaseValue.json` 随 `data.js` 打包，`internal/assets/calc/scores/<ID>-<角色名>.js` 由 `scripts/bundle-reference-calculation.mjs` 从 `score.js` 生成；尚无伤害脚本的代理人也进入计算目录，只用于评分。回归向量在 `internal/assets/testdata/score-vectors.json`。评分不等于伤害计算或队伍收益。

## 自动参考计算

`internal/assets/calc/` 保存完整的绝区零计算运行时：`data.js`（上游 ID 映射）、`common.js`（Calculator、BuffManager 与“伤害”用来读取面板的 `ZZZAvatarInfo` 等代理人模型，去掉下载图片的方法）、`buffs.js`（音擎与驱动盘效果）、手写的 `bootstrap.js` 与 `runner.js`，以及角色脚本 `characters/<ID>-<角色名>.js`。除手写的两个文件外均由 `scripts/bundle-reference-calculation.mjs` 从 ZZZ-Plugin dev 固定快照生成。`scripts/generate-reference-build-vectors.mjs` 以合成代理人对照打包后的计算与上游流程，结果写入 `internal/assets/testdata/calc-vectors.json`。

聊天“面板”图的伤害统计与“伤害”都运行 `runner.js` 的 `runDamage`，与 ZZZ-Plugin 的 panel/card 与 apps/damage 相同：由保存的官方面板构造 `ZZZAvatarInfo`，`avatar_calc` 计算各技能伤害；面板图另列角色规则标为在面板显示的增益（`calc_showInPanel_buffs`），“伤害”再以 `calc_sub_differences` 与 `calc_main_differences` 比较词条，词条按角色评分权重选取，因此同时加载该角色的伤害与评分脚本。上游没有效果的音擎在这里同样没有效果，下文的音擎补充不参与。上游对序号只判断是否大于技能数，写最后一个技能之后的序号时没有可画的伤害而出错，这里取最后一个技能。

## 卡池资料

绝区零卡池历史与 ZZZ-Plugin 一样运行时读取 [GachaClock](https://github.com/iaoongin/GachaClock) 的 `spider/data/zzz/history.json`（缓存一天，读取失败时沿用上次结果），并按上游 processData 补上缺少的 1.0 上半卡池、推算“版本更新后”的起点；管理页读不到时使用随插件的[固定提交](https://github.com/iaoongin/GachaClock/tree/99d16c10bfeeb5f885e9cb42861c50f993f3a746)快照（由 `scripts/import-banner-data.py` 转换），保留 `LICENSES/GachaClock-MIT.txt`。缺少起点的卡池按参考规则推算并标记；2026-09-22 时上游数据的末期结束日仍为 2026-05-05。插件只转换文字和日期，不下载关联图片；聊天中的卡池回复与上游一样以图片消息段发送 GachaClock 给出的卡池图片地址（哔哩哔哩绝区零 WIKI），由聊天平台获取。

## 计算脚本的兼容修正

固定来源为 ZZZ-Plugin dev `fb66219cec0294e1834bacdf0033b2d43a9ccaf4`，许可见 `LICENSES/ZZZ-Plugin-AGPL-3.0.txt`。

- 柏妮思六影的内层“灼烧每段”直接调用 `calc_skill`，绕过 `new` 对异常类型的识别，因而查询不存在的技能倍率。打包器只为该内层对象补上 `isAnomalyDMG=true`，沿用原灼烧倍率、18 次系数及条件。
- 计算器会把当前技能的临时属性写回 `skill.props`，Buff 注册也会修改规则对象。基准、候选和后续请求各自复制固定角色/音擎/套装定义，防止露西等角色的技能缓存污染另一套配装。
- 使用原始属性格式化和元素映射模块，明确区分百分数与比例。内部日志不输出数据；遇到未处理的规则错误/缺失倍率立即失败，不沿用参考的静默跳过行为生成不完整结果。

## 音擎补充与自定义条件

`scripts/calc-supplements/weapons.js` 为固定描述完整的 33 项音擎补充被动，使用五档原始数值，不外推后续版本；它们只用于管理页的配装试算（`runBuild`），聊天回复不使用。现有可计算音擎资料由 67 项扩展为 100 项；14004 的名称为占位符且无效果描述，不伪造被动。锋御的四份音擎使用独立数值/技能范围测试，因为同快照没有对应职业角色计算脚本。

非伤害的能量、失衡、防护、积蓄与持续时间保留完整说明；比格气缸附加伤害只显示描述可确定的防御倍率基础值，未指定的元素不作猜测。血髓秘匣读取截断前暴击率后再计算增伤上限；元素、职业和技能限定保留，未来异常类型不冒充已有异常。

自定义条件只修改候选情境，基准不变。队友由用户选择并填写实际增益，不自动假定角色等级或装备；百分比基础属性用明确函数转换，避免把 100% 当固定一点。局内来源开关保留静态面板、驱动盘主属性及两件常驻属性。无额外条件的旧参考向量保持一致。

## 图片模板

`templates/note/` 按 ZZZ-Plugin 的 note 模板改写；`templates/panel/`、`templates/damage/`、`templates/gacha/`、`templates/abyss/`、`templates/deadly/`、`templates/holo-boss/`、`templates/void-front/`、`templates/tower/`、`templates/card/`、`templates/training/`、`templates/help/`、`templates/monthly/`、`templates/monthly-collect/`、`templates/hollow-zero/`、`templates/lost-void/`、`templates/zenkov/` 与 `templates/exploration/` 的样式由 ZZZ-Plugin 的 common/style、common/layout、panel/card、panel/damage、gachalog、abyss、deadly、holoBoss、voidFrontBattle、climbingTower、card、proficiency、help、monthly、monthly/collect、hollowZero/hollowZero、hollowZeroS2、zenkov/detail（已包含 zenkov/index 的全部样式，两页共用）与 explorationDetail 样式转换而来，抽卡分析的统计、UP 判定与欧非评级同上游 anaylizeGachaLog（均为 AGPL-3.0，见 `LICENSES/ZZZ-Plugin-AGPL-3.0.txt`）。图片地址改为宿主渲染资源；角色、音擎、邦布与驱动盘图与上游一样出图时从 ZZZeroUID 镜像按需下载，抽卡分析的代理人方形头像从官方图片地址按需下载（上游的最后一级来源），名称按 ZZZ-Plugin 的映射表换算。上游三个 SVG 属性图标不是可接受的渲染资源，不显示。上游 common/style 另从相邻的原神插件读取 tttgbnumber 与 HYWenHei-55W 字体，伤害页与之相同，两者取自“原神插件图片”素材来源。战绩里的代理人、邦布、首领与增益图是官方图片地址，首次出图时按需缓存到“官方图片缓存”。日历同 ZZZ-Plugin 发送官方“活动日历”公告中的第一张图片（`templates/calendar/` 只负责显示该图），没有这篇公告时以文字列出公告。战绩末尾与上游一样提示本人在本群排名中的显示或隐藏状态。`templates/rank-*/` 的九个排名页由 ZZZ-Plugin 的 rank 样式（式舆、危局、绝境、拟境、临界与爬塔四个赛季）转换而来，排序规则同上游 rank.js。`templates/skills/` 与 `templates/cinema/` 按 ZZZ-Plugin 的 skills 与 cinema 页面改写，代理人资料与上游一样运行时读取第三方 static.nanoka.cc（先读 manifest.json 取最新版本，版本与资料各缓存十二小时，请求不带账号信息），意象影画图取自同一站点，头像使用上游备用的官方方形头像。模板用到的图片、字体与资料文件取自各上游的固定提交，由 `scripts/bundle-artwork.py` 复制到插件包的 `assets/` 随插件分发（许可同各上游，见 `LICENSES/`）；管理员可通过“素材更新”从上游仓库下载新版本，下载的文件优先使用。“下载全部资源”另按 ZZZ-Plugin 的做法，为映射表中的每个代理人、音擎、驱动盘套装与邦布预先下载模板按需使用的图片，为此随包分发 ZZZ-Plugin 的 `BangbooId2Data.json`；上游的圆形立绘与 Nanoka 资料文件这里没有模板使用，改为下载影画页用的意象影画图。“删除全部资源”与上游一样只删除按需下载的图片缓存，素材仓库的下载与随包文件保留。

`templates/uid-list/` 按 Miao-Yunzai 原神插件的 html/user/uid-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），页面框架与样式沿用 miao-plugin 的 common/layout/elem 与 common/common.css（MIT，见 `LICENSES/miao-plugin-MIT.txt`），同 miao 的 1.4 倍缩放；图片地址改为宿主渲染资源。与上游一样，每个 UID 都使用 miao 的通用头像与原神插件的绝区零横幅；为此新增两个素材来源：只取 `resources/common/` 的“喵喵插件图片”，以及只取 `resources/ZZZero/img/other/`、邦布列表用到的 `resources/ZZZero/img/buddy/`、`resources/img/icon/`、公告页用到的 `resources/html/mysNews/`、`resources/html/mysNews-list/`、`resources/font/tttgbnumber.ttf` 与伤害页用到的 `resources/font/HYWenHei-55W.ttf` 的“原神插件图片”。

`templates/buddy/` 按 Yunzai 原神插件的 ZZZero/html/buddy 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），同上游的 1.5 倍缩放；邦布图片、稀有度条与数字字体取自“原神插件图片”素材来源，上游尚无图片的邦布改用 ZZZeroUID 镜像的方形头像。八个及以下邦布时与上游一样使用窄版页面，模板以 `fit_width` 按卡片宽度出图。

`templates/news/` 与 `templates/news-list/` 按 Yunzai 原神插件的 html/mysNews 与 html/mysNews-list 改写（GPL-3.0，见 `LICENSES/Miao-Yunzai-GPL-3.0.txt`），公告、资讯、活动、米游社搜索、帖子与预估按其 mysNews 的规则出图；详情页的样式去掉了米游社编辑器、加载、提示与回复控件等页面用不到的规则。米游社正文是任意 HTML，而渲染器可以联网，因此正文按页面样式用到的标签、类名与颜色字号等样式重建，图片只经渲染资源引用（官方图片缓存），链接与脚本不保留。与上游不同之处：正文中没有地址的超链图片按帖子的 structured_content 补上（上游显示为空白）；纯图片帖显示全部图片（上游只显示最后一张）；上游按 4000 像素分段截图，这里出一整张长图；二维码由 go-qrcode 生成（MIT，见 `LICENSES/go-qrcode-MIT.txt`），指向帖子所在游戏的米游社地址（上游固定为原神路径）。页面的米游社标志、图标字体、列表页背景与数字字体取自“原神插件图片”素材来源。
