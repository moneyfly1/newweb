package services

import (
	"strings"
)

func DetectRegion(name string) string {
	lower := strings.ToLower(name)

	// Use word boundary matching to avoid false positives
	// Check for exact matches or matches with word boundaries
	regionMap := map[string][]string{
		"香港":    {"香港", " hk ", " hk-", "-hk-", "-hk ", "hong kong", "hongkong", "🇭🇰"},
		"美国":    {"美国", " us ", " us-", "-us-", "-us ", "usa", "united states", "america", "🇺🇸"},
		"日本":    {"日本", " jp ", " jp-", "-jp-", "-jp ", "japan", "tokyo", "🇯🇵"},
		"新加坡":   {"新加坡", " sg ", " sg-", "-sg-", "-sg ", "singapore", "🇸🇬"},
		"台湾":    {"台湾", " tw ", " tw-", "-tw-", "-tw ", "taiwan", "🇹🇼"},
		"韩国":    {"韩国", " kr ", " kr-", "-kr-", "-kr ", "korea", "seoul", "🇰🇷"},
		"英国":    {"英国", " uk ", " uk-", "-uk-", "-uk ", "united kingdom", "london", "🇬🇧"},
		"德国":    {"德国", " de ", " de-", "-de-", "-de ", "germany", "🇩🇪"},
		"法国":    {"法国", " fr ", " fr-", "-fr-", "-fr ", "france", "🇫🇷"},
		"加拿大":   {"加拿大", " ca ", " ca-", "-ca-", "-ca ", "canada", "🇨🇦"},
		"澳大利亚":  {"澳大利亚", "澳", " au ", " au-", "-au-", "-au ", "australia", "🇦🇺"},
		"俄罗斯":   {"俄罗斯", " ru ", " ru-", "-ru-", "-ru ", "russia", "🇷🇺"},
		"印度":    {"印度", " in ", " in-", "-in-", "-in ", "india", "🇮🇳"},
		"马来西亚":  {"马来西亚", "大马", " my ", " my-", "-my-", "-my ", "malaysia", "🇲🇾"},
		"菲律宾":   {"菲律宾", " ph ", " ph-", "-ph-", "-ph ", "philippines", "🇵🇭"},
		"柬埔寨":   {"柬埔寨", " kh ", " kh-", "-kh-", "-kh ", "cambodia", "🇰🇭"},
		"越南":    {"越南", " vn ", " vn-", "-vn-", "-vn ", "vietnam", "🇻🇳"},
		"泰国":    {"泰国", " th ", " th-", "-th-", "-th ", "thailand", "🇹🇭"},
		"印度尼西亚": {"印度尼西亚", "印尼", " id ", " id-", "-id-", "-id ", "indonesia", "🇮🇩"},
		"土耳其":   {"土耳其", " tr ", " tr-", "-tr-", "-tr ", "turkey", "🇹🇷"},
		"巴西":    {"巴西", " br ", " br-", "-br-", "-br ", "brazil", "🇧🇷"},
		"荷兰":    {"荷兰", " nl ", " nl-", "-nl-", "-nl ", "netherlands", "🇳🇱"},
		"意大利":   {"意大利", " it ", " it-", "-it-", "-it ", "italy", "🇮🇹"},
		"西班牙":   {"西班牙", " es ", " es-", "-es-", "-es ", "spain", "🇪🇸"},
		"瑞士":    {"瑞士", " ch ", " ch-", "-ch-", "-ch ", "switzerland", "🇨🇭"},
		"瑞典":    {"瑞典", " se ", " se-", "-se-", "-se ", "sweden", "🇸🇪"},
		"波兰":    {"波兰", " pl ", " pl-", "-pl-", "-pl ", "poland", "🇵🇱"},
		"阿联酋":   {"阿联酋", " ae ", " ae-", "-ae-", "-ae ", "uae", "🇦🇪"},
		"新西兰":   {"新西兰", " nz ", " nz-", "-nz-", "-nz ", "new zealand", "🇳🇿"},
		"南非":    {"南非", " za ", " za-", "-za-", "-za ", "south africa", "🇿🇦"},
		"爱尔兰":   {"爱尔兰", " ie ", " ie-", "-ie-", "-ie ", "ireland", "🇮🇪"},
		"墨西哥":   {"墨西哥", " mx ", " mx-", "-mx-", "-mx ", "mexico", "🇲🇽"},
		"阿根廷":   {"阿根廷", " ar ", " ar-", "-ar-", "-ar ", "argentina", "🇦🇷"},
		"哥伦比亚":  {"哥伦比亚", " co ", " co-", "-co-", "-co ", "colombia", "🇨🇴"},
		"智利":    {"智利", " cl ", " cl-", "-cl-", "-cl ", "chile", "🇨🇱"},
		"埃及":    {"埃及", " eg ", " eg-", "-eg-", "-eg ", "egypt", "🇪🇬"},
		"以色列":   {"以色列", " il ", " il-", "-il-", "-il ", "israel", "🇮🇱"},
		"乌克兰":   {"乌克兰", " ua ", " ua-", "-ua-", "-ua ", "ukraine", "🇺🇦"},
		"罗马尼亚":  {"罗马尼亚", " ro ", " ro-", "-ro-", "-ro ", "romania", "🇷🇴"},
		"匈牙利":   {"匈牙利", " hu ", " hu-", "-hu-", "-hu ", "hungary", "🇭🇺"},
		"捷克":    {"捷克", " cz ", " cz-", "-cz-", "-cz ", "czech", "🇨🇿"},
		"希腊":    {"希腊", " gr ", " gr-", "-gr-", "-gr ", "greece", "🇬🇷"},
		"葡萄牙":   {"葡萄牙", " pt ", " pt-", "-pt-", "-pt ", "portugal", "🇵🇹"},
		"芬兰":    {"芬兰", " fi ", " fi-", "-fi-", "-fi ", "finland", "🇫🇮"},
		"挪威":    {"挪威", " no ", " no-", "-no-", "-no ", "norway", "🇳🇴"},
		"丹麦":    {"丹麦", " dk ", " dk-", "-dk-", "-dk ", "denmark", "🇩🇰"},
		"奥地利":   {"奥地利", " at ", " at-", "-at-", "-at ", "austria", "🇦🇹"},
		"比利时":   {"比利时", " be ", " be-", "-be-", "-be ", "belgium", "🇧🇪"},
		"缅甸":    {"缅甸", " mm ", " mm-", "-mm-", "-mm ", "myanmar", "🇲🇲"},
		"老挝":    {"老挝", " la ", " la-", "-la-", "-la ", "laos", "🇱🇦"},
		"巴基斯坦":  {"巴基斯坦", " pk ", " pk-", "-pk-", "-pk ", "pakistan", "🇵🇰"},
		"孟加拉":   {"孟加拉", " bd ", " bd-", "-bd-", "-bd ", "bangladesh", "🇧🇩"},
		"蒙古":    {"蒙古", " mn ", " mn-", "-mn-", "-mn ", "mongolia", "🇲🇳"},
		"哈萨克斯坦": {"哈萨克斯坦", " kz ", " kz-", "-kz-", "-kz ", "kazakhstan", "🇰🇿"},
	}

	// Add spaces around the name for boundary matching
	paddedLower := " " + lower + " "

	for region, keywords := range regionMap {
		for _, keyword := range keywords {
			// For keywords with spaces (word boundaries), check in padded string
			if strings.HasPrefix(keyword, " ") || strings.HasSuffix(keyword, " ") {
				if strings.Contains(paddedLower, keyword) {
					return region
				}
			} else {
				// For other keywords, use regular contains
				if strings.Contains(lower, keyword) {
					return region
				}
			}
		}
	}

	return "其他"
}

// VmessLinkToClashMap parses a vmess:// link into a Clash-compatible proxy map.
