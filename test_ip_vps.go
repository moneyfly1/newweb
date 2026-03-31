package main

import (
	"fmt"
	"cboard/v2/internal/utils"
)

func main() {
	testIPs := []string{
		"223.104.110.4",
		"1.198.218.245",
		"8.8.8.8",
	}

	fmt.Println("测试 IP 地理位置查询：\n")
	for _, ip := range testIPs {
		location := utils.GetIPLocation(ip)
		if location == "" {
			location = "[空字符串]"
		}
		fmt.Printf("IP: %-20s => %s\n", ip, location)
	}
}
