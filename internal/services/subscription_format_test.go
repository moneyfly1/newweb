package services

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"cboard/v2/internal/models"
)

func testNode(name, typ, config string) models.Node {
	return models.Node{
		Name:     name,
		Type:     typ,
		Status:   "online",
		Config:   &config,
		IsActive: true,
	}
}

func sampleSubscriptionFormatNodes() []models.Node {
	ss := "ss://YWVzLTEyOC1nY206c2VjcmV0@ss.example.com:8388#SS%20Node"
	trojan := "trojan://trojan-pass@trojan.example.com:443?sni=trojan.example.com&type=ws&host=cdn.example.com&path=%2Ftrojan#Trojan%20Node"
	vmess := "vmess://eyJ2IjoiMiIsInBzIjoiVk1lc3MgV1MiLCJhZGQiOiJ2bWVzcy5leGFtcGxlLmNvbSIsInBvcnQiOiI0NDMiLCJpZCI6IjExMTExMTExLTExMTEtMTExMS0xMTExLTExMTExMTExMTExMSIsImFpZCI6IjAiLCJzY3kiOiJhdXRvIiwibmV0Ijoid3MiLCJ0eXBlIjoibm9uZSIsImhvc3QiOiJjZG4uZXhhbXBsZS5jb20iLCJwYXRoIjoiL3ZtZXNzIiwidGxzIjoidGxzIiwic25pIjoidm1lc3MuZXhhbXBsZS5jb20ifQ=="
	vless := "vless://22222222-2222-2222-2222-222222222222@vless.example.com:443?encryption=none&security=tls&sni=vless.example.com&type=ws&host=cdn.example.com&path=%2Fvless&flow=xtls-rprx-vision#VLESS%20WS"
	info := "ss://YWVzLTEyOC1nY206aW5mbw==@baidu.com:1234#Info"

	return []models.Node{
		testNode("SS 节点, 香港", "ss", ss),
		testNode("Trojan 节点", "trojan", trojan),
		testNode("VMess WS", "vmess", vmess),
		testNode("VLESS WS", "vless", vless),
		testNode("Info Node", "ss", info),
	}
}

func TestGenerateShadowrocketBase64ReturnsUniversalLinks(t *testing.T) {
	nodes := sampleSubscriptionFormatNodes()
	out := GenerateShadowrocketBase64(nodes)
	decoded, err := base64.StdEncoding.DecodeString(out)
	if err != nil {
		t.Fatalf("Shadowrocket output is not valid base64: %v", err)
	}
	body := string(decoded)
	if !strings.Contains(body, "ss://") || !strings.Contains(body, "trojan://") || !strings.Contains(body, "vmess://") || !strings.Contains(body, "vless://") {
		t.Fatalf("Shadowrocket output should preserve universal proxy links, got:\n%s", body)
	}
}

func TestGenerateSurgeConfigSkipsUnsupportedAndInfoNodes(t *testing.T) {
	out := GenerateSurgeConfig(sampleSubscriptionFormatNodes(), "CBoard")
	for _, want := range []string{
		"[Proxy]",
		"SS 节点 香港 = ss, ss.example.com, 8388, encrypt-method=aes-128-gcm, password=secret",
		"Trojan 节点 = trojan, trojan.example.com, 443, password=trojan-pass, sni=trojan.example.com, ws=true, ws-path=/trojan, ws-headers=Host:cdn.example.com",
		"VMess WS = vmess, vmess.example.com, 443, username=11111111-1111-1111-1111-111111111111, encrypt-method=chacha20-ietf-poly1305, sni=vmess.example.com, ws=true, ws-path=/vmess, ws-headers=Host:cdn.example.com",
		"[Proxy Group]",
		"Proxy = select, AutoTest, DIRECT, SS 节点 香港, Trojan 节点, VMess WS",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Surge output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "baidu.com") || strings.Contains(out, "VLESS WS") {
		t.Fatalf("Surge output should skip info and unsupported nodes:\n%s", out)
	}
}

func TestGenerateQuantumultXConfigHasValidServerSection(t *testing.T) {
	out := GenerateQuantumultXConfig(sampleSubscriptionFormatNodes())
	for _, want := range []string{
		"[server_local]",
		"shadowsocks=ss.example.com:8388, method=aes-128-gcm, password=secret, tag=SS 节点 香港",
		"trojan=trojan.example.com:443, password=trojan-pass, over-tls=true",
		"vmess=vmess.example.com:443",
		"vless=vless.example.com:443",
		"[filter_local]",
		"FINAL,Proxy",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Quantumult X output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "baidu.com") {
		t.Fatalf("Quantumult X output should skip info nodes:\n%s", out)
	}
}

func TestGenerateLoonConfigHasValidProxySection(t *testing.T) {
	out := GenerateLoonConfig(sampleSubscriptionFormatNodes(), "CBoard")
	for _, want := range []string{
		"[Proxy]",
		"SS 节点 香港 = Shadowsocks,ss.example.com,8388,aes-128-gcm,secret",
		"Trojan 节点 = Trojan,trojan.example.com,443,trojan-pass,over-tls=true,tls-name=trojan.example.com",
		"VMess WS = VMESS,vmess.example.com,443,auto,11111111-1111-1111-1111-111111111111,over-tls=true,tls-name=vmess.example.com,transport=ws,path=/vmess,host=cdn.example.com",
		"VLESS WS = VLESS,vless.example.com,443,22222222-2222-2222-2222-222222222222,over-tls=true,tls-name=vless.example.com",
		"[Rule]",
		"FINAL,Proxy",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("Loon output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "baidu.com") {
		t.Fatalf("Loon output should skip info nodes:\n%s", out)
	}
}

func TestGenerateSingBoxConfigPreservesOutboundTransports(t *testing.T) {
	out := GenerateSingBoxConfig(sampleSubscriptionFormatNodes())

	var cfg struct {
		Outbounds []map[string]interface{} `json:"outbounds"`
		Route     map[string]interface{}   `json:"route"`
		DNS       map[string]interface{}   `json:"dns"`
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("Sing-box output is not valid JSON: %v\n%s", err, out)
	}
	if len(cfg.Outbounds) == 0 {
		t.Fatal("Sing-box output should contain outbounds")
	}
	if cfg.Route == nil || cfg.DNS == nil {
		t.Fatalf("Sing-box output should contain route and dns sections:\n%s", out)
	}

	outbounds := map[string]map[string]interface{}{}
	for _, outbound := range cfg.Outbounds {
		if tag, _ := outbound["tag"].(string); tag != "" {
			outbounds[tag] = outbound
		}
	}
	for _, tag := range []string{"SS 节点_ 香港", "Trojan 节点", "VMess WS", "VLESS WS", "direct", "block", "dns-out"} {
		if _, ok := outbounds[tag]; !ok {
			t.Fatalf("Sing-box output missing outbound tag %q:\n%s", tag, out)
		}
	}

	assertTransport := func(tag, wantType, wantPath string) {
		t.Helper()
		transport, _ := outbounds[tag]["transport"].(map[string]interface{})
		if transport == nil {
			t.Fatalf("Sing-box outbound %q missing transport: %#v", tag, outbounds[tag])
		}
		if got, _ := transport["type"].(string); got != wantType {
			t.Fatalf("Sing-box outbound %q transport type: want %q got %q", tag, wantType, got)
		}
		if wantPath != "" {
			if got, _ := transport["path"].(string); got != wantPath {
				t.Fatalf("Sing-box outbound %q transport path: want %q got %q", tag, wantPath, got)
			}
		}
	}
	assertTransport("VMess WS", "ws", "/vmess")
	assertTransport("Trojan 节点", "ws", "/trojan")
	assertTransport("VLESS WS", "ws", "/vless")

	if got, _ := outbounds["VLESS WS"]["flow"].(string); got != "xtls-rprx-vision" {
		t.Fatalf("Sing-box VLESS outbound should preserve flow, got %q", got)
	}
}
