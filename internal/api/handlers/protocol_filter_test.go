package handlers

import (
	"testing"

	"cboard/v2/internal/models"
)

// 协议开关必须能识别同一协议的两种写法，否则客户导入的节点会「凭空消失」：
// 线上协议开关里存的是 socks5，而部分导入路径写出的类型是 socks，
// 结果 SOCKS 节点在 Clash 订阅里被静默过滤掉（后台也没有任何提示）。
func TestFilterNodesByProtocolAliases(t *testing.T) {
	nodes := []models.Node{
		{Name: "s1", Type: "socks5"},
		{Name: "s2", Type: "socks"},
		{Name: "h1", Type: "hysteria2"},
		{Name: "h2", Type: "hy2"},
		{Name: "w1", Type: "wireguard"},
		{Name: "w2", Type: "wg"},
		{Name: "v1", Type: "vless"},
	}

	// 开关里写 socks5 → socks 也应通过
	got := FilterNodesByProtocol(nodes, map[string]bool{"socks5": true})
	if len(got) != 2 {
		t.Fatalf("socks5 开关应同时放行 socks5 与 socks，实际 %d 个: %+v", len(got), names(got))
	}

	// 反向：开关里写 socks → socks5 也应通过
	got = FilterNodesByProtocol(nodes, map[string]bool{"socks": true})
	if len(got) != 2 {
		t.Fatalf("socks 开关应同时放行 socks5 与 socks，实际 %d 个", len(got))
	}

	// hysteria2 / hy2
	got = FilterNodesByProtocol(nodes, map[string]bool{"hysteria2": true})
	if len(got) != 2 {
		t.Fatalf("hysteria2 开关应放行 hysteria2 与 hy2，实际 %d 个", len(got))
	}

	// wireguard / wg
	got = FilterNodesByProtocol(nodes, map[string]bool{"wireguard": true})
	if len(got) != 2 {
		t.Fatalf("wireguard 开关应放行 wireguard 与 wg，实际 %d 个", len(got))
	}

	// nil 开关 = 不过滤
	if got = FilterNodesByProtocol(nodes, nil); len(got) != len(nodes) {
		t.Errorf("开关为 nil 时不应过滤，实际 %d 个", len(got))
	}

	// 不在开关里的协议仍应被排除
	got = FilterNodesByProtocol(nodes, map[string]bool{"vless": true})
	if len(got) != 1 || got[0].Name != "v1" {
		t.Errorf("只勾选 vless 时应只放行 vless，实际 %+v", names(got))
	}
}

func names(nodes []models.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Name+"/"+n.Type)
	}
	return out
}
