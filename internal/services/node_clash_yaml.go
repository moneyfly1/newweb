package services

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"cboard/v2/internal/models"
	"gopkg.in/yaml.v3"
)

func GenerateClashYAML(nodes []models.Node) string {
	return GenerateClashYAMLWithDomain(nodes, "", "")
}

// GenerateClashYAMLWithDomain generates Clash YAML using the template file (uploads/config/temp.yaml).
// subscriptionName is used for the YAML `name` field (e.g. "到期: 2026-03-15").

func GenerateClashYAMLWithDomain(nodes []models.Node, siteDomain string, subscriptionName string) string {
	var proxies []map[string]interface{}
	var proxyNames []string
	var infoNames []string
	usedNames := make(map[string]bool)

	for _, n := range nodes {
		if n.Config == nil || *n.Config == "" {
			continue
		}
		name := sanitizeNodeName(n.Name)
		if name == "" {
			name = fmt.Sprintf("%s Node", strings.ToUpper(n.Type))
		}
		origName := name
		counter := 1
		for usedNames[name] {
			name = fmt.Sprintf("%s_%d", origName, counter)
			counter++
		}
		usedNames[name] = true

		m, err := NodeConfigToClashMap(n.Type, *n.Config, name)
		if err != nil {
			continue
		}
		proxies = append(proxies, m)
		proxyNames = append(proxyNames, name)

		if server, ok := m["server"].(string); ok && server == "baidu.com" {
			infoNames = append(infoNames, name)
		}
	}

	// Real proxy names (exclude info nodes) for auto-select groups
	infoSet := make(map[string]bool)
	for _, n := range infoNames {
		infoSet[n] = true
	}
	var realNames []string
	for _, n := range proxyNames {
		if !infoSet[n] {
			realNames = append(realNames, n)
		}
	}

	// Try template-based generation
	if result := generateFromTemplate(proxies, proxyNames, realNames, subscriptionName); result != "" {
		return result
	}

	// Fallback: generate default YAML
	return generateDefaultClashYAML(proxies, proxyNames, realNames, siteDomain, subscriptionName)
}

// GenerateStashYAMLWithDomain generates Stash YAML using stash_temp.yaml template.
// Falls back to Clash YAML if the Stash template is not found.

func GenerateStashYAMLWithDomain(nodes []models.Node, siteDomain string, subscriptionName string) string {
	var proxies []map[string]interface{}
	var proxyNames []string
	var infoNames []string
	usedNames := make(map[string]bool)

	for _, n := range nodes {
		if n.Config == nil || *n.Config == "" {
			continue
		}
		name := sanitizeNodeName(n.Name)
		if name == "" {
			name = fmt.Sprintf("%s Node", strings.ToUpper(n.Type))
		}
		origName := name
		counter := 1
		for usedNames[name] {
			name = fmt.Sprintf("%s_%d", origName, counter)
			counter++
		}
		usedNames[name] = true
		m, err := NodeConfigToClashMap(n.Type, *n.Config, name)
		if err != nil {
			continue
		}
		proxies = append(proxies, m)
		proxyNames = append(proxyNames, name)
		if server, ok := m["server"].(string); ok && server == "baidu.com" {
			infoNames = append(infoNames, name)
		}
	}

	infoSet := make(map[string]bool)
	for _, n := range infoNames {
		infoSet[n] = true
	}
	var realNames []string
	for _, n := range proxyNames {
		if !infoSet[n] {
			realNames = append(realNames, n)
		}
	}

	if result := generateFromTemplateFile("uploads/config/stash_temp.yaml", proxies, proxyNames, realNames, subscriptionName); result != "" {
		return result
	}
	// Fall back to Clash YAML
	return GenerateClashYAMLWithDomain(nodes, siteDomain, subscriptionName)
}

// generateFromTemplate loads uploads/config/temp.yaml and injects proxies + updates proxy-groups.

func generateFromTemplate(proxies []map[string]interface{}, allNames, realNames []string, subscriptionName string) string {
	return generateFromTemplateFile("uploads/config/temp.yaml", proxies, allNames, realNames, subscriptionName)
}


func generateFromTemplateFile(templatePath string, proxies []map[string]interface{}, allNames, realNames []string, subscriptionName string) string {
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return ""
	}

	var templateConfig yaml.Node
	if err := yaml.Unmarshal(data, &templateConfig); err != nil {
		return ""
	}

	// templateConfig is a Document node; the actual mapping is its first child
	if templateConfig.Kind != yaml.DocumentNode || len(templateConfig.Content) == 0 {
		return ""
	}
	root := templateConfig.Content[0]
	if root.Kind != yaml.MappingNode {
		return ""
	}

	// Build proxies YAML using our ordered writer for deterministic output
	var proxiesSB strings.Builder
	for _, p := range proxies {
		writeClashProxy(&proxiesSB, p)
	}
	var proxiesNode yaml.Node
	if err := yaml.Unmarshal([]byte("proxies:\n"+proxiesSB.String()), &proxiesNode); err != nil {
		return ""
	}

	// Inject subscription name as YAML "name" field (used by Clash clients as profile display name)
	if subscriptionName != "" {
		nameFound := false
		for i := 0; i < len(root.Content)-1; i += 2 {
			if root.Content[i].Value == "name" {
				root.Content[i+1].Value = subscriptionName
				nameFound = true
				break
			}
		}
		if !nameFound {
			// Prepend name field to the root mapping
			root.Content = append([]*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "name", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: subscriptionName, Tag: "!!str"},
			}, root.Content...)
		}
	}

	// Walk the root mapping and update proxies + proxy-groups
	for i := 0; i < len(root.Content)-1; i += 2 {
		keyNode := root.Content[i]
		valNode := root.Content[i+1]

		if keyNode.Value == "proxies" {
			// Replace proxies value with our generated proxies
			if proxiesNode.Kind == yaml.DocumentNode && len(proxiesNode.Content) > 0 {
				mappingNode := proxiesNode.Content[0]
				if mappingNode.Kind == yaml.MappingNode && len(mappingNode.Content) >= 2 {
					*valNode = *mappingNode.Content[1]
				}
			}
		}

		if keyNode.Value == "proxy-groups" && valNode.Kind == yaml.SequenceNode {
			updateProxyGroupsYAML(valNode, allNames, realNames)
		}

		// 为 Sparkle 等客户端：模板中的 profile 增加自动更新间隔（小时）
		if keyNode.Value == "profile" && valNode.Kind == yaml.MappingNode {
			injectProfileUpdateInterval(valNode, 24)
		}
	}

	output, err := yaml.Marshal(&templateConfig)
	if err != nil {
		return ""
	}
	return unescapeUnicode(string(output))
}

// injectProfileUpdateInterval sets profile.update-interval (hours) for Clash/Sparkle 自动更新.

func injectProfileUpdateInterval(profileNode *yaml.Node, hours int) {
	if profileNode.Kind != yaml.MappingNode {
		return
	}
	val := strconv.Itoa(hours)
	for j := 0; j < len(profileNode.Content)-1; j += 2 {
		if profileNode.Content[j].Value == "update-interval" {
			profileNode.Content[j+1].Value = val
			return
		}
	}
	profileNode.Content = append(profileNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "update-interval", Tag: "!!str"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: val, Tag: "!!str"},
	)
}

// updateProxyGroupsYAML updates proxy-groups in the YAML node tree.

func updateProxyGroupsYAML(groupsNode *yaml.Node, allNames, realNames []string) {
	// Collect group names
	groupNames := make(map[string]bool)
	for _, g := range groupsNode.Content {
		if g.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j < len(g.Content)-1; j += 2 {
			if g.Content[j].Value == "name" {
				groupNames[g.Content[j+1].Value] = true
			}
		}
	}

	for _, g := range groupsNode.Content {
		if g.Kind != yaml.MappingNode {
			continue
		}
		var gType string
		var proxiesIdx int = -1
		for j := 0; j < len(g.Content)-1; j += 2 {
			if g.Content[j].Value == "type" {
				gType = g.Content[j+1].Value
			}
			if g.Content[j].Value == "proxies" {
				proxiesIdx = j + 1
			}
		}
		if proxiesIdx < 0 || (gType != "select" && gType != "url-test" && gType != "fallback" && gType != "load-balance") {
			continue
		}

		// Collect special entries (DIRECT, REJECT, group references)
		var specials []string
		oldVal := g.Content[proxiesIdx]
		if oldVal.Kind == yaml.SequenceNode {
			for _, item := range oldVal.Content {
				if item.Kind == yaml.ScalarNode {
					if item.Value == "DIRECT" || item.Value == "REJECT" || groupNames[item.Value] {
						specials = append(specials, item.Value)
					}
				}
			}
		}

		// Build new proxies list
		var newItems []*yaml.Node
		for _, s := range specials {
			newItems = append(newItems, &yaml.Node{Kind: yaml.ScalarNode, Value: s, Tag: "!!str"})
		}
		names := allNames
		if gType != "select" && len(realNames) > 0 {
			names = realNames
		}
		for _, n := range names {
			newItems = append(newItems, &yaml.Node{Kind: yaml.ScalarNode, Value: n, Tag: "!!str"})
		}

		g.Content[proxiesIdx] = &yaml.Node{
			Kind:    yaml.SequenceNode,
			Tag:     "!!seq",
			Content: newItems,
		}
	}
}

// unescapeUnicode converts \UXXXXXXXX and \uXXXX escape sequences back to actual Unicode characters.

func unescapeUnicode(s string) string {
	result := s
	// Handle \UXXXXXXXX (8-digit)
	for {
		idx := strings.Index(result, "\\U")
		if idx < 0 || idx+10 > len(result) {
			break
		}
		hexStr := result[idx+2 : idx+10]
		codePoint, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			// Not a valid escape, skip
			result = result[:idx] + "U" + result[idx+2:]
			continue
		}
		result = result[:idx] + string(rune(codePoint)) + result[idx+10:]
	}
	// Handle \uXXXX (4-digit)
	for {
		idx := strings.Index(result, "\\u")
		if idx < 0 || idx+6 > len(result) {
			break
		}
		hexStr := result[idx+2 : idx+6]
		codePoint, err := strconv.ParseInt(hexStr, 16, 32)
		if err != nil {
			result = result[:idx] + "u" + result[idx+2:]
			continue
		}
		result = result[:idx] + string(rune(codePoint)) + result[idx+6:]
	}
	return result
}

// updateProxyGroups injects proxy names into each group, preserving special entries.
// generateDefaultClashYAML is the fallback when no template file exists.

func generateDefaultClashYAML(proxies []map[string]interface{}, allNames, realNames []string, siteDomain, subscriptionName string) string {
	var sb strings.Builder

	// When no real nodes exist, fall back to allNames to avoid empty proxy groups
	autoNames := realNames
	if len(autoNames) == 0 {
		autoNames = allNames
	}

	if subscriptionName != "" {
		sb.WriteString(fmt.Sprintf("name: %s\n", escapeYAML(subscriptionName)))
	}
	sb.WriteString("mixed-port: 7890\n")
	sb.WriteString("allow-lan: true\n")
	sb.WriteString("bind-address: '*'\n")
	sb.WriteString("mode: rule\n")
	sb.WriteString("log-level: info\n")
	sb.WriteString("ipv6: false\n")
	sb.WriteString("external-controller: 127.0.0.1:9090\n")
	sb.WriteString("find-process-mode: always\n")
	sb.WriteString("unified-delay: true\n")
	sb.WriteString("tcp-concurrent: true\n")
	sb.WriteString("\n")
	sb.WriteString("profile:\n")
	sb.WriteString("  store-selected: true\n")
	sb.WriteString("  store-fake-ip: true\n")
	sb.WriteString("  update-interval: 24\n")
	sb.WriteString("\n")
	sb.WriteString("dns:\n")
	sb.WriteString("  enable: true\n")
	sb.WriteString("  listen: 0.0.0.0:1053\n")
	sb.WriteString("  ipv6: false\n")
	sb.WriteString("  enhanced-mode: fake-ip\n")
	sb.WriteString("  fake-ip-range: 198.18.0.1/16\n")
	sb.WriteString("  fake-ip-filter:\n")
	sb.WriteString("    - '*.lan'\n")
	sb.WriteString("    - '*.local'\n")
	sb.WriteString("    - localhost.ptlogin2.qq.com\n")
	sb.WriteString("    - '+.msftconnecttest.com'\n")
	sb.WriteString("    - '+.msftncsi.com'\n")
	sb.WriteString("  default-nameserver:\n")
	sb.WriteString("    - 223.5.5.5\n")
	sb.WriteString("    - 119.29.29.29\n")
	sb.WriteString("  nameserver:\n")
	sb.WriteString("    - https://dns.alidns.com/dns-query\n")
	sb.WriteString("    - https://doh.pub/dns-query\n")
	sb.WriteString("  fallback:\n")
	sb.WriteString("    - https://1.1.1.1/dns-query\n")
	sb.WriteString("    - https://dns.google/dns-query\n")
	sb.WriteString("  fallback-filter:\n")
	sb.WriteString("    geoip: true\n")
	sb.WriteString("    geoip-code: CN\n")
	sb.WriteString("\n")

	sb.WriteString("proxies:\n")
	for _, p := range proxies {
		writeClashProxy(&sb, p)
	}

	// 17 个代理组（与老项目 goweb 模板一致）
	grpSelect := "🚀 节点选择"
	grpAuto := "♻️ 自动选择"
	grpFallover := "🔰 故障转移"
	grpBalance := "🔮 负载均衡"
	grpDirect := "🎯 全球直连"
	grpBlock := "🛑 全球拦截"
	grpFish := "🐟 漏网之鱼"
	grpApple := "📱 苹果服务"
	grpMicrosoft := "🍎 微软服务"
	grpGoogle := "🔍 谷歌服务"
	grpTelegram := "📲 电报消息"
	grpOpenAI := "🤖 OpenAI"
	grpStreamIntl := "📺 国际流媒体"
	grpStreamCN := "📺 国内流媒体"
	grpForeign := "🌐 国外网站"
	grpChina := "🇨🇳 国内网站"
	grpLocal := "🏠 本地网络"

	sb.WriteString("\nproxy-groups:\n")

	// 1. 🚀 节点选择
	sb.WriteString("  - name: " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - " + escapeYAML(grpFallover) + "\n")
	sb.WriteString("      - " + escapeYAML(grpBalance) + "\n")
	sb.WriteString("      - DIRECT\n")
	for _, name := range allNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 2. ♻️ 自动选择
	sb.WriteString("  - name: " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("    type: url-test\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    tolerance: 50\n")
	sb.WriteString("    proxies:\n")
	for _, name := range autoNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 3. 🔰 故障转移
	sb.WriteString("  - name: " + escapeYAML(grpFallover) + "\n")
	sb.WriteString("    type: fallback\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    proxies:\n")
	for _, name := range autoNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 4. 🔮 负载均衡
	sb.WriteString("  - name: " + escapeYAML(grpBalance) + "\n")
	sb.WriteString("    type: load-balance\n")
	sb.WriteString("    url: http://www.gstatic.com/generate_204\n")
	sb.WriteString("    interval: 300\n")
	sb.WriteString("    strategy: consistent-hashing\n")
	sb.WriteString("    proxies:\n")
	for _, name := range autoNames {
		sb.WriteString("      - " + escapeYAML(name) + "\n")
	}

	// 5. 🎯 全球直连
	sb.WriteString("  - name: " + escapeYAML(grpDirect) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")

	// 6. 🛑 全球拦截
	sb.WriteString("  - name: " + escapeYAML(grpBlock) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - REJECT\n")
	sb.WriteString("      - DIRECT\n")

	// 7. 🐟 漏网之鱼
	sb.WriteString("  - name: " + escapeYAML(grpFish) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")

	// 8. 📱 苹果服务
	sb.WriteString("  - name: " + escapeYAML(grpApple) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 9. 🍎 微软服务
	sb.WriteString("  - name: " + escapeYAML(grpMicrosoft) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 10. 🔍 谷歌服务
	sb.WriteString("  - name: " + escapeYAML(grpGoogle) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 11. 📲 电报消息
	sb.WriteString("  - name: " + escapeYAML(grpTelegram) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 12. 🤖 OpenAI
	sb.WriteString("  - name: " + escapeYAML(grpOpenAI) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 13. 📺 国际流媒体
	sb.WriteString("  - name: " + escapeYAML(grpStreamIntl) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 14. 📺 国内流媒体
	sb.WriteString("  - name: " + escapeYAML(grpStreamCN) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")

	// 15. 🌐 国外网站
	sb.WriteString("  - name: " + escapeYAML(grpForeign) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")
	sb.WriteString("      - " + escapeYAML(grpAuto) + "\n")
	sb.WriteString("      - DIRECT\n")

	// 16. 🇨🇳 国内网站
	sb.WriteString("  - name: " + escapeYAML(grpChina) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")

	// 17. 🏠 本地网络
	sb.WriteString("  - name: " + escapeYAML(grpLocal) + "\n")
	sb.WriteString("    type: select\n")
	sb.WriteString("    proxies:\n")
	sb.WriteString("      - DIRECT\n")
	sb.WriteString("      - " + escapeYAML(grpSelect) + "\n")

	sb.WriteString("\nrules:\n")
	if siteDomain != "" {
		d := siteDomain
		for _, prefix := range []string{"https://", "http://"} {
			d = strings.TrimPrefix(d, prefix)
		}
		d = strings.TrimRight(d, "/")
		sb.WriteString("  - DOMAIN-SUFFIX," + d + "," + grpDirect + "\n")
	}
	sb.WriteString("  - DOMAIN-SUFFIX,local," + grpLocal + "\n")
	sb.WriteString("  - IP-CIDR,127.0.0.0/8," + grpLocal + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,172.16.0.0/12," + grpLocal + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,192.168.0.0/16," + grpLocal + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,10.0.0.0/8," + grpLocal + ",no-resolve\n")
	// 苹果
	sb.WriteString("  - DOMAIN-SUFFIX,apple.com," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,icloud.com," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,apple.news," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,apple.ae," + grpApple + "\n")
	sb.WriteString("  - DOMAIN-KEYWORD,apple," + grpApple + "\n")
	// 微软
	sb.WriteString("  - DOMAIN-SUFFIX,microsoft.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,windows.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,live.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,office.com," + grpMicrosoft + "\n")
	sb.WriteString("  - DOMAIN-KEYWORD,microsoft," + grpMicrosoft + "\n")
	// 谷歌
	sb.WriteString("  - DOMAIN-SUFFIX,google.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,gstatic.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,youtube.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,googleapis.com," + grpGoogle + "\n")
	sb.WriteString("  - DOMAIN-KEYWORD,google," + grpGoogle + "\n")
	// 电报
	sb.WriteString("  - DOMAIN-SUFFIX,telegram.org," + grpTelegram + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,t.me," + grpTelegram + "\n")
	sb.WriteString("  - IP-CIDR,91.108.4.0/22," + grpTelegram + ",no-resolve\n")
	sb.WriteString("  - IP-CIDR,149.154.160.0/20," + grpTelegram + ",no-resolve\n")
	// OpenAI
	sb.WriteString("  - DOMAIN-SUFFIX,openai.com," + grpOpenAI + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,chatgpt.com," + grpOpenAI + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,ai.com," + grpOpenAI + "\n")
	// 国际流媒体
	sb.WriteString("  - DOMAIN-SUFFIX,netflix.com," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,netflix.net," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,disneyplus.com," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,hbo.com," + grpStreamIntl + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,spotify.com," + grpStreamIntl + "\n")
	// 国内流媒体
	sb.WriteString("  - DOMAIN-SUFFIX,iqiyi.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,bilibili.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,youku.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,tencentvideo.com," + grpStreamCN + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,qq.com," + grpStreamCN + "\n")
	// 国内网站直连
	sb.WriteString("  - GEOIP,CN," + grpChina + "\n")
	// 国外网站
	sb.WriteString("  - GEOIP,!CN," + grpForeign + "\n")
	// 广告拦截
	sb.WriteString("  - DOMAIN-KEYWORD,adservice," + grpBlock + "\n")
	sb.WriteString("  - DOMAIN-SUFFIX,doubleclick.net," + grpBlock + "\n")
	sb.WriteString("  - MATCH," + grpFish + "\n")

	return sb.String()
}
