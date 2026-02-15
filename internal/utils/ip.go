package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type IPInfo struct {
	Country string `json:"country"`
	Region  string `json:"regionName"`
	City    string `json:"city"`
	Query   string `json:"query"`
	Status  string `json:"status"`
}

// GetIPLocation returns a location string for the given IP address.
// Uses the free ip-api.com service. Returns empty string on failure.
func GetIPLocation(ip string) string {
	if ip == "" || ip == "127.0.0.1" || ip == "::1" {
		return "本地"
	}

	// Check for private IP ranges
	if isPrivateIP(ip) {
		return "本地网络"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN&fields=status,country,regionName,city,query", ip))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return ""
	}
	if info.Status != "success" {
		return ""
	}

	location := info.Country
	if info.Region != "" && info.Region != info.Country {
		location += " " + info.Region
	}
	if info.City != "" && info.City != info.Region {
		location += " " + info.City
	}
	return location
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip string) bool {
	// Check for common private IP prefixes
	privateRanges := []string{
		"10.",
		"192.168.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.",
		"172.28.", "172.29.", "172.30.", "172.31.",
		"169.254.", // Link-local
		"fc00:",    // IPv6 private
		"fd00:",    // IPv6 private
		"fe80:",    // IPv6 link-local
	}

	for _, prefix := range privateRanges {
		if len(ip) >= len(prefix) && ip[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
