package app

import "slices"

func overseasGameRegion(region string) bool {
	return slices.Contains([]string{"prod_gf_us", "prod_gf_eu", "prod_gf_jp", "prod_gf_sg"}, region)
}
func syncRegionAllowed(region string) bool {
	return overseasGameRegion(region) || region == "prod_gf_cn"
}

// UIDRegion is the server ZZZ-Plugin's getGameRoles reads from a UID's
// leading digits; other UIDs are on 新艾利都.
func UIDRegion(uid string) string {
	if len(uid) > 8 {
		switch uid[:len(uid)-8] {
		case "10":
			return "prod_gf_us"
		case "15":
			return "prod_gf_eu"
		case "13":
			return "prod_gf_jp"
		case "17":
			return "prod_gf_sg"
		}
	}
	return "prod_gf_cn"
}

// regionTimezone is the UTC offset a region's records are written in.
func regionTimezone(region string) int {
	switch region {
	case "prod_gf_us":
		return -5
	case "prod_gf_eu":
		return 1
	}
	return 8
}
