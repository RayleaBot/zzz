package app

import "slices"

func overseasGameRegion(region string) bool {
	return slices.Contains([]string{"prod_gf_us", "prod_gf_eu", "prod_gf_jp", "prod_gf_sg"}, region)
}
func syncRegionAllowed(region string) bool {
	return overseasGameRegion(region) || region == "prod_gf_cn"
}
