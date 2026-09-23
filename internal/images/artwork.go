package images

import (
	rayleabot "github.com/RayleaBot/RayleaBot/sdk/go"
	"github.com/RayleaBot/plugin-zzz/internal/app"
)

// The tables below map the images named in each stylesheet, converted from
// ZZZ-Plugin's stylesheets, to their repository paths; stylesheets read them
// as --render-resource-<id>.

// commonArtwork is what common/style, common/layout and the shared fonts use.
var commonArtwork = [][2]string{
	{"common-images-RANK_A", "resources/common/images/RANK_A.png"},
	{"common-images-RANK_B", "resources/common/images/RANK_B.png"},
	{"common-images-RANK_S", "resources/common/images/RANK_S.png"},
	{"common-images-Rarity_A", "resources/common/images/Rarity_A.png"},
	{"common-images-Rarity_B", "resources/common/images/Rarity_B.png"},
	{"common-images-Rarity_C", "resources/common/images/Rarity_C.png"},
	{"common-images-Rarity_S", "resources/common/images/Rarity_S.png"},
	{"common-images-Rarity_X", "resources/common/images/Rarity_X.png"},
	{"common-images-SuitBg", "resources/common/images/SuitBg.png"},
	{"common-images-UIDBg", "resources/common/images/UIDBg.png"},
	{"common-images-bg", "resources/common/images/bg.jpg"},
	{"common-images-element-AuricInk", "resources/common/images/element/AuricInk.png"},
	{"common-images-element-Electric", "resources/common/images/element/Electric.png"},
	{"common-images-element-Ether", "resources/common/images/element/Ether.png"},
	{"common-images-element-Fire", "resources/common/images/element/Fire.png"},
	{"common-images-element-Frost", "resources/common/images/element/Frost.png"},
	{"common-images-element-HonedEdge", "resources/common/images/element/HonedEdge.png"},
	{"common-images-element-Ice", "resources/common/images/element/Ice.png"},
	{"common-images-element-Lumiflux", "resources/common/images/element/Lumiflux.png"},
	{"common-images-element-Physical", "resources/common/images/element/Physical.png"},
	{"common-images-element-Wind", "resources/common/images/element/Wind.png"},
	{"common-images-max_score-149500", "resources/common/images/max_score/149500.png"},
	{"common-images-max_score-150000", "resources/common/images/max_score/150000.png"},
	{"common-images-max_score-182000", "resources/common/images/max_score/182000.png"},
	{"common-images-max_score-195000", "resources/common/images/max_score/195000.png"},
	{"common-images-max_score-481000", "resources/common/images/max_score/481000.png"},
	{"common-images-max_score-50000", "resources/common/images/max_score/50000.png"},
	{"common-images-max_score-65000", "resources/common/images/max_score/65000.png"},
	{"common-images-max_score-663000", "resources/common/images/max_score/663000.png"},
	{"common-images-profession-IconAnomaly", "resources/common/images/profession/IconAnomaly.png"},
	{"common-images-profession-IconArmorer", "resources/common/images/profession/IconArmorer.png"},
	{"common-images-profession-IconAttack", "resources/common/images/profession/IconAttack.png"},
	{"common-images-profession-IconDefense", "resources/common/images/profession/IconDefense.png"},
	{"common-images-profession-IconRupture", "resources/common/images/profession/IconRupture.png"},
	{"common-images-profession-IconStun", "resources/common/images/profession/IconStun.png"},
	{"common-images-profession-IconSupport", "resources/common/images/profession/IconSupport.png"},
	{"common-images-property-IconAttack", "resources/common/images/property/IconAttack.png"},
	{"common-images-property-IconBreakStun", "resources/common/images/property/IconBreakStun.png"},
	{"common-images-property-IconCrit", "resources/common/images/property/IconCrit.png"},
	{"common-images-property-IconCritDam", "resources/common/images/property/IconCritDam.png"},
	{"common-images-property-IconDef", "resources/common/images/property/IconDef.png"},
	{"common-images-property-IconDungeonBuffEther", "resources/common/images/property/IconDungeonBuffEther.png"},
	{"common-images-property-IconElementAbnormalPower", "resources/common/images/property/IconElementAbnormalPower.png"},
	{"common-images-property-IconElementMystery", "resources/common/images/property/IconElementMystery.png"},
	{"common-images-property-IconFire", "resources/common/images/property/IconFire.png"},
	{"common-images-property-IconHpMax", "resources/common/images/property/IconHpMax.png"},
	{"common-images-property-IconIce", "resources/common/images/property/IconIce.png"},
	{"common-images-property-IconPenRatio", "resources/common/images/property/IconPenRatio.png"},
	{"common-images-property-IconPenValue", "resources/common/images/property/IconPenValue.png"},
	{"common-images-property-IconPhysDmg", "resources/common/images/property/IconPhysDmg.png"},
	{"common-images-property-IconSheerForce", "resources/common/images/property/IconSheerForce.png"},
	{"common-images-property-IconSpGetRatio", "resources/common/images/property/IconSpGetRatio.png"},
	{"common-images-property-IconSpMax", "resources/common/images/property/IconSpMax.png"},
	{"common-images-property-IconSpRecover", "resources/common/images/property/IconSpRecover.png"},
	{"common-images-property-IconThunder", "resources/common/images/property/IconThunder.png"},
	{"common-images-property-IconWind", "resources/common/images/property/IconWind.png"},
	{"common-images-rank_bg-1", "resources/common/images/rank_bg/1.png"},
	{"common-images-rank_bg-2", "resources/common/images/rank_bg/2.png"},
	{"common-images-rank_bg-3", "resources/common/images/rank_bg/3.png"},
	{"common-images-rank_bg-4", "resources/common/images/rank_bg/4.png"},
	{"common-images-rank_bg-5", "resources/common/images/rank_bg/5.png"},
	{"common-images-rating-A", "resources/common/images/rating/A.png"},
	{"common-images-rating-A-corner", "resources/common/images/rating/A-corner.png"},
	{"common-images-rating-B", "resources/common/images/rating/B.png"},
	{"common-images-rating-B-corner", "resources/common/images/rating/B-corner.png"},
	{"common-images-rating-S", "resources/common/images/rating/S.png"},
	{"common-images-rating-S-corner", "resources/common/images/rating/S-corner.png"},
	{"common-images-rating-SP", "resources/common/images/rating/SP.png"},
	{"common-images-skills-assist", "resources/common/images/skills/assist.webp"},
	{"common-images-skills-basic", "resources/common/images/skills/basic.webp"},
	{"common-images-skills-chain", "resources/common/images/skills/chain.webp"},
	{"common-images-skills-core", "resources/common/images/skills/core.webp"},
	{"common-images-skills-dodge", "resources/common/images/skills/dodge.webp"},
	{"common-images-skills-special", "resources/common/images/skills/special.webp"},
	{"common-images-team-01", "resources/common/images/team/01.png"},
	{"common-images-team-02", "resources/common/images/team/02.png"},
	{"common-images-team-03", "resources/common/images/team/03.png"},
	{"zzz", "resources/common/fonts/inpinhongmengti.ttf"},
}

// commonFonts are the fonts ZZZ-Plugin's common style loads besides its own:
// the number and Chinese fonts of the Yunzai 原神插件 beside it, which every
// page's text falls back to after zzz.
var commonFonts = [][2]string{
	{"tttgbnumber", "resources/font/tttgbnumber.ttf"},
	{"HYWenHei-55W", "resources/font/HYWenHei-55W.ttf"},
}

// fontResources are the common fonts for the pages that list their resources
// themselves.
func fontResources(context app.ImageContext) []rayleabot.RenderImageResource {
	resources := []rayleabot.RenderImageResource{}
	for _, font := range commonFonts {
		if resource, ok := context.ArtworkResource(font[0], "yunzai-genshin", font[1]); ok {
			resources = append(resources, resource)
		}
	}
	return resources
}

// panelArtwork is what panel/card adds.
var panelArtwork = [][2]string{
	{"panel-images-BgFrame01", "resources/panel/images/BgFrame01.png"},
	{"panel-images-CurseBG04", "resources/panel/images/CurseBG04.png"},
	{"panel-images-CurseBG08", "resources/panel/images/CurseBG08.png"},
	{"panel-images-empty_equip_03", "resources/panel/images/empty_equip_03.png"},
	{"panel-images-empty_equip_07", "resources/panel/images/empty_equip_07.png"},
	{"panel-images-equip_bg", "resources/panel/images/equip_bg.png"},
	{"panel-images-ranks-1", "resources/panel/images/ranks/1.png"},
	{"panel-images-ranks-2", "resources/panel/images/ranks/2.png"},
	{"panel-images-ranks-3", "resources/panel/images/ranks/3.png"},
	{"panel-images-ranks-4", "resources/panel/images/ranks/4.png"},
	{"panel-images-ranks-5", "resources/panel/images/ranks/5.png"},
	{"panel-images-ranks-6", "resources/panel/images/ranks/6.png"},
	{"panel-images-skill_bg", "resources/panel/images/skill_bg.png"},
	{"panel-images-star-0", "resources/panel/images/star/0.png"},
	{"panel-images-star-1", "resources/panel/images/star/1.png"},
	{"panel-images-star-2", "resources/panel/images/star/2.png"},
	{"panel-images-star-3", "resources/panel/images/star/3.png"},
	{"panel-images-star-4", "resources/panel/images/star/4.png"},
	{"panel-images-star-5", "resources/panel/images/star/5.png"},
	{"panel-images-weapon_bg", "resources/panel/images/weapon_bg.png"},
}

// gachaArtwork is what gachalog adds.
var gachaArtwork = [][2]string{
	{"gachalog-images-IconTabUP", "resources/gachalog/images/IconTabUP.png"},
	{"gachalog-images-bg1", "resources/gachalog/images/bg1.png"},
	{"gachalog-images-bg2", "resources/gachalog/images/bg2.png"},
	{"gachalog-images-bg3", "resources/gachalog/images/bg3.png"},
	{"gachalog-images-emoji-1", "resources/gachalog/images/emoji/1.png"},
	{"gachalog-images-emoji-10", "resources/gachalog/images/emoji/10.png"},
	{"gachalog-images-emoji-11", "resources/gachalog/images/emoji/11.png"},
	{"gachalog-images-emoji-12", "resources/gachalog/images/emoji/12.png"},
	{"gachalog-images-emoji-13", "resources/gachalog/images/emoji/13.png"},
	{"gachalog-images-emoji-14", "resources/gachalog/images/emoji/14.png"},
	{"gachalog-images-emoji-15", "resources/gachalog/images/emoji/15.png"},
	{"gachalog-images-emoji-16", "resources/gachalog/images/emoji/16.png"},
	{"gachalog-images-emoji-2", "resources/gachalog/images/emoji/2.png"},
	{"gachalog-images-emoji-3", "resources/gachalog/images/emoji/3.png"},
	{"gachalog-images-emoji-4", "resources/gachalog/images/emoji/4.png"},
	{"gachalog-images-emoji-5", "resources/gachalog/images/emoji/5.png"},
	{"gachalog-images-emoji-6", "resources/gachalog/images/emoji/6.png"},
	{"gachalog-images-emoji-7", "resources/gachalog/images/emoji/7.png"},
	{"gachalog-images-emoji-8", "resources/gachalog/images/emoji/8.png"},
	{"gachalog-images-emoji-9", "resources/gachalog/images/emoji/9.png"},
}
