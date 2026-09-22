package app

import "slices"

func overseasGameRegion(game, region string) bool {
	return slices.Contains(map[string][]string{"genshin": {"os_usa", "os_euro", "os_asia", "os_cht"}, "starrail": {"prod_official_usa", "prod_official_eur", "prod_official_asia", "prod_official_cht"}, "zzz": {"prod_gf_us", "prod_gf_eu", "prod_gf_jp", "prod_gf_sg"}}[game], region)
}
func syncRegionAllowed(game, region string) bool {
	return overseasGameRegion(game, region) || game == "genshin" && slices.Contains([]string{"cn_gf01", "cn_qd01"}, region) || game == "zzz" && region == "prod_gf_cn"
}
