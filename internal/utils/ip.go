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
