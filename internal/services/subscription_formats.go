package services

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"cboard/v2/internal/models"
)

// GenerateSurgeConfig generates Surge-compatible proxy list
func GenerateSurgeConfig(nodes []models.Node, siteName string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s Surge Config\n", siteName))
	sb.WriteString("[Proxy]\n")
	sb.WriteString("DIRECT = direct\n")
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		line := convertNodeToSurgeLine(node)
		if line != "" {
			sb.WriteString(line + "\n")
		}
	}
	sb.WriteString("\n[Proxy Group]\n")
	sb.WriteString("Proxy = select, ")
	var names []string
	for _, node := range nodes {
		if node.Config != nil && *node.Config != "" {
			names = append(names, node.Name)
		}
	}
	sb.WriteString(strings.Join(names, ", "))
	sb.WriteString("\n")
	return sb.String()
}

func convertNodeToSurgeLine(node models.Node) string {
	if node.Config == nil {
		return ""
	}
	config := *node.Config
	// For SS nodes
	if strings.HasPrefix(config, "ss://") {
		return convertSSToSurge(node.Name, config)
	}
	// For trojan nodes
	if strings.HasPrefix(config, "trojan://") {
		return convertTrojanToSurge(node.Name, config)
	}
	return ""
}

func convertSSToSurge(name, config string) string {
	// ss://BASE64@host:port#name or ss://method:password@host:port#name
	config = strings.TrimPrefix(config, "ss://")
	// Remove fragment
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	// Try to parse
	var method, password, host, port string
	if atIdx := strings.LastIndex(config, "@"); atIdx >= 0 {
		userInfo := config[:atIdx]
		serverInfo := config[atIdx+1:]
		// Decode userInfo if base64
		if decoded, err := base64.RawURLEncoding.DecodeString(userInfo); err == nil {
			userInfo = string(decoded)
		} else if decoded, err := base64.StdEncoding.DecodeString(userInfo); err == nil {
			userInfo = string(decoded)
		}
		parts := strings.SplitN(userInfo, ":", 2)
		if len(parts) == 2 {
			method = parts[0]
			password = parts[1]
		}
		hostPort := strings.SplitN(serverInfo, ":", 2)
		if len(hostPort) == 2 {
			host = hostPort[0]
			port = hostPort[1]
		}
	}
	if method == "" || host == "" {
		return ""
	}
	return fmt.Sprintf("%s = ss, %s, %s, encrypt-method=%s, password=%s", name, host, port, method, password)
}

func convertTrojanToSurge(name, config string) string {
	config = strings.TrimPrefix(config, "trojan://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	u, err := url.Parse("trojan://" + config)
	if err != nil {
		return ""
	}
	password := u.User.Username()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	sni := u.Query().Get("sni")
	if sni == "" {
		sni = host
	}
	return fmt.Sprintf("%s = trojan, %s, %s, password=%s, sni=%s", name, host, port, password, sni)
}

// GenerateShadowrocketBase64 generates Shadowrocket-compatible base64 subscription
// Shadowrocket uses the same base64 format as universal, so this is an alias
func GenerateShadowrocketBase64(nodes []models.Node) string {
	return GenerateUniversalBase64(nodes)
}

// GenerateQuantumultXConfig generates QuantumultX server_remote format
func GenerateQuantumultXConfig(nodes []models.Node) string {
	var lines []string
	for _, node := range nodes {
		if node.Config == nil || *node.Config == "" {
			continue
		}
		config := *node.Config
		if strings.HasPrefix(config, "ss://") {
			line := convertSSToQuantumultX(node.Name, config)
			if line != "" {
				lines = append(lines, line)
			}
		} else if strings.HasPrefix(config, "trojan://") {
			line := convertTrojanToQuantumultX(node.Name, config)
			if line != "" {
				lines = append(lines, line)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func convertSSToQuantumultX(name, config string) string {
	config = strings.TrimPrefix(config, "ss://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	var method, password, host, port string
	if atIdx := strings.LastIndex(config, "@"); atIdx >= 0 {
		userInfo := config[:atIdx]
		serverInfo := config[atIdx+1:]
		if decoded, err := base64.RawURLEncoding.DecodeString(userInfo); err == nil {
			userInfo = string(decoded)
		} else if decoded, err := base64.StdEncoding.DecodeString(userInfo); err == nil {
			userInfo = string(decoded)
		}
		parts := strings.SplitN(userInfo, ":", 2)
		if len(parts) == 2 {
			method = parts[0]
			password = parts[1]
		}
		hostPort := strings.SplitN(serverInfo, ":", 2)
		if len(hostPort) == 2 {
			host = hostPort[0]
			port = hostPort[1]
		}
	}
	if method == "" || host == "" {
		return ""
	}
	return fmt.Sprintf("shadowsocks=%s:%s, method=%s, password=%s, tag=%s", host, port, method, password, name)
}

func convertTrojanToQuantumultX(name, config string) string {
	config = strings.TrimPrefix(config, "trojan://")
	if idx := strings.Index(config, "#"); idx >= 0 {
		config = config[:idx]
	}
	u, err := url.Parse("trojan://" + config)
	if err != nil {
		return ""
	}
	password := u.User.Username()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return fmt.Sprintf("trojan=%s:%s, password=%s, over-tls=true, tls-verification=false, tag=%s", host, port, password, name)
}
