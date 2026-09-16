package services

import "testing"

// 自有客户端识别回归（用户反馈：网站认不出 Mclash 客户端、设备数量也不准）。
//
// 背景：Mclash 的 UA 是 `Mclash/<版本> platform/<os> mihomo/<内核版本>`，
// 里面同时带 `mihomo` 关键词。旧规则里没有 mclash，于是：
//   - detectSoftware 命中 `mihomo` → 设备被记成 "Mihomo"（内核名）；
//   - detectOS 认不出 `platform/macos` → OSName=Unknown、device_type=unknown；
//   - 特征不足 2 个 → 指纹退化成 sha256(整个 UA)，而 UA 带版本号
//     → 每升级一个版本就多出一台"新设备"，设备数量自然不对。
func TestParseUserAgentRecognizesMclash(t *testing.T) {
	cases := []struct {
		name     string
		ua       string
		wantSoft string
		wantVer  string
		wantOS   string
		wantType string
	}{
		{
			name:     "macOS 桌面端",
			ua:       "Mclash/0.0.7 platform/macos mihomo/1.19.31",
			wantSoft: "Mclash",
			wantVer:  "0.0.7",
			wantOS:   "macOS",
			wantType: "desktop",
		},
		{
			name:     "Windows 桌面端",
			ua:       "Mclash/0.0.7 platform/windows mihomo/1.19.31",
			wantSoft: "Mclash",
			wantVer:  "0.0.7",
			wantOS:   "Windows",
			wantType: "desktop",
		},
		{
			name:     "Android 移动端",
			ua:       "Mclash/0.0.7 platform/android mihomo/1.19.31",
			wantSoft: "Mclash",
			wantVer:  "0.0.7",
			wantOS:   "Android",
			wantType: "mobile",
		},
		{
			name:     "旧版本号（升级后仍需认得出是同一个客户端）",
			ua:       "Mclash/1.0.0.1 platform/macos mihomo/1.19.31",
			wantSoft: "Mclash",
			wantVer:  "1.0.0.1",
			wantOS:   "macOS",
			wantType: "desktop",
		},
		{
			name:     "参考客户端 MoneyFly",
			ua:       "MoneyFly/1.2.3 (Windows NT 10.0; Win64; x64)",
			wantSoft: "MoneyFly",
			wantVer:  "1.2.3",
			wantOS:   "Windows",
			wantType: "desktop",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := ParseUserAgent(tc.ua)
			if info.SoftwareName != tc.wantSoft {
				t.Errorf("SoftwareName = %q, want %q", info.SoftwareName, tc.wantSoft)
			}
			if info.SoftwareVersion != tc.wantVer {
				t.Errorf("SoftwareVersion = %q, want %q", info.SoftwareVersion, tc.wantVer)
			}
			if info.OSName != tc.wantOS {
				t.Errorf("OSName = %q, want %q", info.OSName, tc.wantOS)
			}
			if info.DeviceType != tc.wantType {
				t.Errorf("DeviceType = %q, want %q", info.DeviceType, tc.wantType)
			}
		})
	}
}

// 指纹必须与**版本无关**：否则每次发版都会多一台设备。
func TestDeviceFingerprintStableAcrossVersions(t *testing.T) {
	a := GenerateDeviceFingerprint("Mclash/0.0.6 platform/macos mihomo/1.19.31", "")
	b := GenerateDeviceFingerprint("Mclash/0.0.7 platform/macos mihomo/1.19.31", "")
	if a != b {
		t.Fatalf("同一台机器不同版本应当得到相同指纹\n a=%s\n b=%s", a, b)
	}

	// 不同平台必须是不同设备
	c := GenerateDeviceFingerprint("Mclash/0.0.7 platform/windows mihomo/1.19.31", "")
	if a == c {
		t.Fatal("macOS 与 Windows 不应被算作同一台设备")
	}
}

// 客户端上报的设备 id（x-hwid / X-App-Device-Id）必须给出稳定指纹。
func TestDeviceFingerprintByAppDeviceID(t *testing.T) {
	a := GenerateDeviceFingerprint("MoneyFly-App-Device:abc-123", "")
	b := GenerateDeviceFingerprint("MoneyFly-App-Device:abc-123", "")
	if a != b {
		t.Fatal("同一个设备 id 必须得到相同指纹")
	}
	c := GenerateDeviceFingerprint("MoneyFly-App-Device:other-id", "")
	if a == c {
		t.Fatal("不同设备 id 不应得到相同指纹")
	}
}
