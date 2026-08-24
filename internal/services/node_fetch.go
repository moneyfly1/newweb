package services

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
	"time"
)


const (
	maxResponseSize                     = 10 * 1024 * 1024 // 10MB limit for subscription content
	maxSubscriptionRequestAttempts      = 3
	initialSubscriptionRequestRetryWait = 2 * time.Second
)

type subscriptionRequestProfile struct {
	Name      string
	UserAgent string
	Accept    string
}


var subscriptionRequestProfiles = []subscriptionRequestProfile{
	{Name: "v2rayN", UserAgent: "v2rayN/6.23", Accept: "*/*"},
	{Name: "ClashForWindows", UserAgent: "ClashforWindows/0.20.39", Accept: "*/*"},
	{Name: "ClashMeta", UserAgent: "Clash.Meta/1.18.9", Accept: "*/*"},
	{Name: "ClashForAndroid", UserAgent: "ClashForAndroid/2.5.12", Accept: "*/*"},
	{Name: "sing-box", UserAgent: "sing-box/1.10.0", Accept: "*/*"},
	{Name: "Shadowrocket", UserAgent: "Shadowrocket/1990 CFNetwork/1496.0.7 Darwin/23.5.0", Accept: "*/*"},
	{Name: "QuantumultX", UserAgent: "Quantumult%20X/1.5.6", Accept: "*/*"},
	{Name: "Surge", UserAgent: "Surge iOS/5.9.0", Accept: "*/*"},
	{Name: "Stash", UserAgent: "Stash/2.5.4", Accept: "*/*"},
	{Name: "Clash", UserAgent: "clash-verge/v1.7.7", Accept: "*/*"},
	{Name: "curl", UserAgent: "curl/8.0.1", Accept: "*/*"},
	{Name: "browser", UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", Accept: "*/*"},
}


func FetchSubscriptionContent(urlStr string) (string, error) {
	return FetchSubscriptionContentWithProxy(urlStr, "")
}

// FetchSubscriptionContentWithProxy fetches and base64-decodes subscription content from a URL.

func FetchSubscriptionContentWithProxy(urlStr string, proxyURL string) (string, error) {
	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Only allow http and https schemes to prevent SSRF
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Prevent access to private IP ranges
	if parsedURL.Hostname() != "" {
		if ips, err := net.LookupIP(parsedURL.Hostname()); err == nil {
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return "", fmt.Errorf("access to private IP addresses is not allowed")
				}
			}
		} else {
			return "", fmt.Errorf("DNS lookup failed for %s: %w", parsedURL.Hostname(), err)
		}
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: subscriptionHTTPTransport(proxyURL),
		// 防止通过重定向绕过 SSRF 检查
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			// 对重定向目标也进行私有 IP 检查
			if req.URL.Hostname() != "" {
				if ips, err := net.LookupIP(req.URL.Hostname()); err == nil {
					for _, ip := range ips {
						if isPrivateIP(ip) {
							return fmt.Errorf("redirect to private IP is not allowed")
						}
					}
				}
			}
			return nil
		},
	}
	requestURL := normalizeSubscriptionRequestURL(parsedURL)
	body, err := fetchSubscriptionBody(client, requestURL)
	if err != nil {
		return "", err
	}

	content := normalizeSubscriptionContent(string(body))
	trimmed := strings.TrimSpace(content)
	if looksLikeBase64Subscription(trimmed) {
		if decoded, err := decodeBase64Flexible(trimmed); err == nil {
			decoded = normalizeSubscriptionContent(decoded)
			if decoded != "" {
				content = decoded
			}
		}
	}
	return content, nil
}


func subscriptionHTTPTransport(proxyURL string) *http.Transport {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		proxyURL = strings.TrimSpace(os.Getenv("SUBSCRIPTION_FETCH_PROXY"))
	}
	if proxyURL != "" {
		if parsedProxyURL, err := url.Parse(proxyURL); err == nil {
			transport.Proxy = http.ProxyURL(parsedProxyURL)
		}
	}

	return transport
}


func normalizeSubscriptionRequestURL(parsedURL *url.URL) string {
	normalized := *parsedURL
	if normalized.RawQuery != "" {
		normalized.RawQuery = normalized.Query().Encode()
	}
	return normalized.String()
}


func fetchSubscriptionBody(client *http.Client, urlStr string) ([]byte, error) {
	var lastErr error
	var errorsByProfile []string
	retryWait := initialSubscriptionRequestRetryWait
	profiles := subscriptionRequestProfilesForURL(urlStr)

	for attempt := 1; attempt <= maxSubscriptionRequestAttempts; attempt++ {
		for _, profile := range profiles {
			req, err := http.NewRequest("GET", urlStr, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set("User-Agent", profile.UserAgent)
			req.Header.Set("Accept", profile.Accept)
			if strings.Contains(urlStr, "gist.githubusercontent.com") {
				req.Header.Set("Connection", "close")
			}

			resp, err := client.Do(req)
			if err != nil {
				lastErr = fmt.Errorf("%s profile request failed: %w", profile.Name, err)
				errorsByProfile = append(errorsByProfile, lastErr.Error())
				continue
			}
			body, readErr := func() ([]byte, error) {
				defer resp.Body.Close()
				if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
					return nil, fmt.Errorf("%s profile status %d", profile.Name, resp.StatusCode)
				}
				limitedReader := io.LimitReader(resp.Body, maxResponseSize)
				return io.ReadAll(limitedReader)
			}()
			if readErr == nil {
				return body, nil
			}
			lastErr = readErr
			errorsByProfile = append(errorsByProfile, readErr.Error())
		}

		if attempt < maxSubscriptionRequestAttempts {
			time.Sleep(retryWait)
			retryWait *= 2
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("subscription request failed after %d attempts: %s", maxSubscriptionRequestAttempts, summarizeSubscriptionRequestErrors(errorsByProfile))
	}
	return nil, fmt.Errorf("subscription request failed")
}


func summarizeSubscriptionRequestErrors(errs []string) string {
	if len(errs) == 0 {
		return "no response"
	}
	seen := make(map[string]bool, len(errs))
	unique := make([]string, 0, len(errs))
	for _, errText := range errs {
		if seen[errText] {
			continue
		}
		seen[errText] = true
		unique = append(unique, errText)
	}
	return strings.Join(unique, "; ")
}


func subscriptionRequestProfilesForURL(urlStr string) []subscriptionRequestProfile {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return subscriptionRequestProfiles
	}

	switch strings.ToLower(parsedURL.Hostname()) {
	case "static.novarelliance.com":
		return prioritizeSubscriptionProfiles([]string{"ClashForWindows"}, subscriptionRequestProfiles)
	default:
		return subscriptionRequestProfiles
	}
}


func prioritizeSubscriptionProfiles(names []string, profiles []subscriptionRequestProfile) []subscriptionRequestProfile {
	if len(names) == 0 || len(profiles) == 0 {
		return profiles
	}

	priorities := make(map[string]int, len(names))
	for i, name := range names {
		priorities[name] = i
	}

	prioritized := make([]subscriptionRequestProfile, 0, len(profiles))
	remaining := make([]subscriptionRequestProfile, 0, len(profiles))
	for _, profile := range profiles {
		if _, ok := priorities[profile.Name]; ok {
			prioritized = append(prioritized, profile)
			continue
		}
		remaining = append(remaining, profile)
	}
	sort.SliceStable(prioritized, func(i, j int) bool {
		return priorities[prioritized[i].Name] < priorities[prioritized[j].Name]
	})
	return append(prioritized, remaining...)
}


func normalizeSubscriptionContent(content string) string {
	content = strings.TrimPrefix(content, "\ufeff")
	return strings.TrimSpace(content)
}


func looksLikeBase64Subscription(content string) bool {
	if content == "" {
		return false
	}
	if strings.Contains(content, "://") || strings.Contains(content, "proxies:") || strings.Contains(content, "proxy-groups:") {
		return false
	}
	if strings.Contains(content, "<html") || strings.Contains(content, "<!DOCTYPE html") {
		return false
	}
	for _, r := range content {
		if r == '\n' || r == '\r' {
			continue
		}
		if r == '=' || r == '+' || r == '/' || r == '-' || r == '_' {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return utf8.ValidString(content)
}

// isPrivateIP checks if an IP address is in a private range

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	// Additional checks for special ranges
	if ip4 := ip.To4(); ip4 != nil {
		// 0.0.0.0/8, 169.254.0.0/16, 224.0.0.0/4
		return ip4[0] == 0 || (ip4[0] == 169 && ip4[1] == 254) || ip4[0] >= 224
	}
	return false
}

// ParseNodeLinks parses multi-line node links into Node models.
