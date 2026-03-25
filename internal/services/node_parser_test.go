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
