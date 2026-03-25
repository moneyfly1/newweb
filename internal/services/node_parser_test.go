package services

import (
	"testing"
)

func TestParseSubscriptionContentSupportsClashYAML(t *testing.T) {
	content := `proxies:
  - name: 香港一区
    type: vmess
    server: example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
    alterId: 0
    cipher: auto
    udp: true
    tls: true
    network: ws
    ws-opts:
      path: /ws
      headers:
        Host: cdn.example.com
  - name: 美国二区
    type: trojan
    server: trojan.example.com
    port: 443
    password: secret
    sni: trojan.example.com
`

	nodes, err := ParseSubscriptionContent(content)
	if err != nil {
		t.Fatalf("ParseSubscriptionContent returned error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Type != "vmess" {
		t.Fatalf("expected first node type vmess, got %s", nodes[0].Type)
	}
	if nodes[0].Config == nil || *nodes[0].Config == "" {
		t.Fatal("expected first node config to be populated")
	}
	if nodes[1].Type != "trojan" {
		t.Fatalf("expected second node type trojan, got %s", nodes[1].Type)
	}
}

func TestParseSubscriptionContentSupportsLinks(t *testing.T) {
	content := "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4NDQz#test-ss\n"

	nodes, err := ParseSubscriptionContent(content)
	if err != nil {
		t.Fatalf("ParseSubscriptionContent returned error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != "ss" {
		t.Fatalf("expected node type ss, got %s", nodes[0].Type)
	}
}

func TestExtractDomainPortFromNodeLink(t *testing.T) {
	tests := []struct {
		name    string
		link    string
		domain  string
		port    int
		wantErr bool
	}{
		{
			name:   "vmess json link",
			link:   "vmess://eyJhZGQiOiJ2bWVzcy5leGFtcGxlLmNvbSIsInBvcnQiOiI4NDQzIiwicHMiOiJ0ZXN0IiwiaWQiOiIxMTExMTExMS0xMTExLTExMTEtMTExMS0xMTExMTExMTExMTEiLCJhaWQiOiIwIiwibmV0Ijoid3MiLCJ0eXBlIjoibm9uZSIsImhvc3QiOiIiLCJwYXRoIjoiLyIsInRscyI6InRscyJ9",
			domain: "vmess.example.com",
			port:   8443,
		},
		{
			name:   "vless link",
			link:   "vless://11111111-1111-1111-1111-111111111111@vless.example.com:2096?security=tls#test",
			domain: "vless.example.com",
			port:   2096,
		},
		{
			name:   "trojan link",
			link:   "trojan://secret@trojan.example.com:443?sni=trojan.example.com#test",
			domain: "trojan.example.com",
			port:   443,
		},
		{
			name:   "shadowsocks link",
			link:   "ss://YWVzLTI1Ni1nY206cGFzc0BleGFtcGxlLmNvbTo4NDQz#test-ss",
			domain: "example.com",
			port:   8443,
		},
		{
			name:    "invalid link",
			link:    "not-a-valid-link",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domain, port, err := ExtractDomainPortFromNodeLink(tt.link)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ExtractDomainPortFromNodeLink returned error: %v", err)
			}
			if domain != tt.domain {
				t.Fatalf("expected domain %s, got %s", tt.domain, domain)
			}
			if port != tt.port {
				t.Fatalf("expected port %d, got %d", tt.port, port)
			}
		})
	}
}

func TestSSRLinkToClashMapPreservesRemarksAndGroup(t *testing.T) {
	link := "ssr://Y24wOS5zb21ldGhpbmdzdHJhbmdlcy5jb206ODIxNDpvcmlnaW46Y2hhY2hhMjAtaWV0ZjpodHRwX3NpbXBsZTpjR0Z6YzNkay8_cmVtYXJrcz01cGF3NVlxZzVaMmhNVFEmcHJvdG9wYXJhbT0mb2Jmc3BhcmFtPU1qYzNPVFF0VTJsclpXMXBibWN3TURFdVpHOTNibXh2WVdRdWJXbGpjbTl6YjJaMExtTnZiUSZncm91cD1TVkJNUXk1V1NWQQ"

	proxy, err := SSRLinkToClashMap(link, "")
	if err != nil {
		t.Fatalf("SSRLinkToClashMap returned error: %v", err)
	}
	if got := proxy["name"]; got != "新加坡14" {
		t.Fatalf("expected name 新加坡14, got %v", got)
	}
	if got := proxy["group"]; got != "IPLC.VIP" {
		t.Fatalf("expected group IPLC.VIP, got %v", got)
	}
	if got := proxy["server"]; got != "cn09.somethingstranges.com" {
		t.Fatalf("expected server cn09.somethingstranges.com, got %v", got)
	}
	if got := proxy["port"]; got != 8214 {
		t.Fatalf("expected port 8214, got %v", got)
	}
	if got := proxy["obfs-param"]; got != "27794-Sikeming001.download.microsoft.com" {
		t.Fatalf("expected obfs-param to be preserved, got %v", got)
	}
}

func TestSanitizeNodeNameRemovesInvalidWhitespace(t *testing.T) {
	got := sanitizeNodeName("  新加坡\r\nVIP\t节点  ")
	if got != "新加坡 VIP 节点" {
		t.Fatalf("expected sanitized name, got %q", got)
	}
}
