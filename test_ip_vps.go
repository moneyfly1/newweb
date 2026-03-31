package main

import (
	"fmt"
	"cboard/v2/internal/utils"
)

func main() {
	testIPs := []string{
		// IPv4
		"223.104.110.4",     // 中国
		"1.198.218.245",     // 中国
		"8.8.8.8",           // 美国
		"114.114.114.114",   // 中国

		// IPv6
		"2001:4860:4860::8888",  // Google DNS
		"2400:3200::1",          // 阿里 DNS
		"240e:1f:1::1",          // 中国电信
	}

	fmt.Println("测试 IP 地理位置查询（IPv4 + IPv6）：\n")
	for _, ip := range testIPs {
		location := utils.GetIPLocation(ip)
		if location == "" {
			location = "[空字符串]"
		}
		fmt.Printf("IP: %-30s => %s\n", ip, location)
	}
}
