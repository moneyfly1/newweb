// Package software_sync 定义从 GitHub Release 自动解析客户端安装包直链的软件目录。
// 配置键 client_*_url 的值若为 pan://<键> 或空，则通过 /download/gh?key=<键> 自动解析
// GitHub 最新 Release 对应平台的安装包（VPS 侧 30 分钟缓存 + 国内加速镜像 302 跳转）。
package software_sync

import (
	"fmt"

	"cboard/v2/internal/services/ghrelease"

	"regexp"
)

// Target 一个下载入口（对应一个软件下载配置键）
type Target struct {
	// ConfigKey 软件下载配置键，如 client_v2rayn_url（兼容 CBoard 现有键名）
	ConfigKey string
	// OS 平台标识：windows / macos / android / linux / ios
	OS string
	// Arch 架构标识：x64 / intel / apple / universal
	Arch string
	// Label 展示名称，如 "Windows x64"
	Label string
	// Preferred 优先匹配规则（先尝试，如 arm64-v8a apk）
	Preferred []*regexp.Regexp
	// Patterns 匹配规则（对齐各仓库实际资产命名）
	Patterns []*regexp.Regexp
}

// Software 一个软件
type Software struct {
	// Key 标识
	Key string
	// Name 展示名称
	Name string
	// Repo GitHub 仓库，如 clash-verge-rev/clash-verge-rev
	Repo string
	// Targets 各平台下载目标
	Targets []Target
}

func rx(patterns ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		out = append(out, regexp.MustCompile(p))
	}
	return out
}

// 通用匹配（已对照各仓库真实资产名）
var apkAny = rx(`(?i)\.apk$`)
var apkPreferredArm = rx(`(?i)(arm64|arm64[-_]?v8a)[^.]*\.apk$`)
var dmgIntel = rx(`(?i)^.*(intel|x64|amd64|_x64|-64)\.(dmg|pkg)$`)
var dmgApple = rx(`(?i)^.*(apple|silicon|m[0-9]+|arm64|aarch64|_aarch64).*\.(dmg|pkg)$`)

// Catalog 软件目录：配置键与 CBoard 现有 client_*_url 保持一致
var Catalog = []Software{
	{
		Key: "v2rayn", Name: "V2rayN", Repo: "2dust/v2rayN",
		Targets: []Target{
			{ConfigKey: "client_v2rayn_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*windows-(64|x64)([-.]|$).*\.zip$`)},
			{ConfigKey: "client_v2rayn_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: rx(`(?i)^.*macos-64\.dmg$`)},
			{ConfigKey: "client_v2rayn_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
		},
	},
	{
		Key: "hiddify", Name: "Hiddify", Repo: "hiddify/hiddify-app",
		Targets: []Target{
			{ConfigKey: "client_hiddify_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*x64.*\.exe$`)},
			{ConfigKey: "client_hiddify_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
			{ConfigKey: "client_hiddify_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: rx(`(?i)^.*macos.*\.dmg$`)},
			{ConfigKey: "client_hiddify_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: rx(`(?i)^.*macos.*\.dmg$`)},
			{ConfigKey: "client_hiddify_linux_url", OS: "linux", Arch: "x64", Label: "Linux x64", Patterns: rx(`(?i)^.*linux.*(x64|amd64).*\.(deb|rpm|AppImage|tar\.gz)$`)},
			{ConfigKey: "client_hiddify_linux_arm_url", OS: "linux", Arch: "arm64", Label: "Linux arm64", Patterns: rx(`(?i)^.*linux.*(arm64|aarch64).*\.(deb|rpm|AppImage|tar\.gz)$`)},
		},
	},
	{
		Key: "clash-verge", Name: "Clash Verge", Repo: "clash-verge-rev/clash-verge-rev",
		Targets: []Target{
			{ConfigKey: "client_clashverge_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*x64.*\.(exe|msi)$`)},
			{ConfigKey: "client_clashverge_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: dmgIntel},
			{ConfigKey: "client_clashverge_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
			{ConfigKey: "client_clashverge_linux_url", OS: "linux", Arch: "x64", Label: "Linux x64", Patterns: rx(`(?i)^.*linux.*(x64|amd64).*\.(deb|rpm|AppImage)$`)},
			{ConfigKey: "client_clashverge_linux_arm_url", OS: "linux", Arch: "arm64", Label: "Linux arm64", Patterns: rx(`(?i)^.*linux.*(arm64|aarch64).*\.(deb|rpm|AppImage)$`)},
		},
	},
	{
		Key: "clash-part", Name: "Clash Party", Repo: "mihomo-party-org/clash-party",
		Targets: []Target{
			{ConfigKey: "client_clashparty_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*windows.*(x64|[^a-z]64).*\.exe$`)},
			{ConfigKey: "client_clashparty_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: dmgIntel},
			{ConfigKey: "client_clashparty_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
			{ConfigKey: "client_clashparty_linux_url", OS: "linux", Arch: "x64", Label: "Linux x64", Patterns: rx(`(?i)^.*linux.*(x64|amd64).*\.(deb|rpm|AppImage)$`)},
			{ConfigKey: "client_clashparty_linux_arm_url", OS: "linux", Arch: "arm64", Label: "Linux arm64", Patterns: rx(`(?i)^.*linux.*(arm64|aarch64).*\.(deb|rpm|AppImage)$`)},
		},
	},
	{
		Key: "flclash", Name: "FlClash", Repo: "chen08209/FlClash",
		Targets: []Target{
			{ConfigKey: "client_flclash_windows_url", OS: "windows", Arch: "x64", Label: "Windows x64", Patterns: rx(`(?i)^.*(x64|amd64).*\.exe$`)},
			{ConfigKey: "client_flclash_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
			{ConfigKey: "client_flclash_macos_url", OS: "macos", Arch: "intel", Label: "macOS Intel", Patterns: dmgIntel},
			{ConfigKey: "client_flclash_macos_arm_url", OS: "macos", Arch: "apple", Label: "macOS Apple 芯片", Patterns: dmgApple},
			{ConfigKey: "client_flclash_linux_url", OS: "linux", Arch: "x64", Label: "Linux x64", Patterns: rx(`(?i)^.*linux.*(x64|amd64).*\.(deb|rpm|AppImage)$`)},
			{ConfigKey: "client_flclash_linux_arm_url", OS: "linux", Arch: "arm64", Label: "Linux arm64", Patterns: rx(`(?i)^.*linux.*(arm64|aarch64).*\.(deb|rpm|AppImage)$`)},
		},
	},
	{
		Key: "v2rayng", Name: "V2rayNG", Repo: "2dust/v2rayNG",
		Targets: []Target{
			{ConfigKey: "client_v2rayng_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
		},
	},
	{
		Key: "clash-meta", Name: "Clash Meta", Repo: "MetaCubeX/ClashMetaForAndroid",
		Targets: []Target{
			{ConfigKey: "client_clash_android_url", OS: "android", Arch: "universal", Label: "Android APK", Preferred: apkPreferredArm, Patterns: apkAny},
		},
	},
}

// FindSoftwareByConfigKey 按配置键找所属软件（含 Repo）
func FindSoftwareByConfigKey(configKey string) *Software {
	for i := range Catalog {
		for j := range Catalog[i].Targets {
			if Catalog[i].Targets[j].ConfigKey == configKey {
				return &Catalog[i]
			}
		}
	}
	return nil
}

// FindTarget 按配置键找目标
func FindTarget(configKey string) *Target {
	for i := range Catalog {
		for j := range Catalog[i].Targets {
			if Catalog[i].Targets[j].ConfigKey == configKey {
				return &Catalog[i].Targets[j]
			}
		}
	}
	return nil
}

// FindAssetFor 在 Release 资产中按目标匹配规则挑选最合适的文件
func FindAssetFor(release *ghrelease.Release, t *Target) (*ghrelease.Asset, error) {
	if release == nil {
		return nil, fmt.Errorf("无 Release 数据")
	}
	// 先尝试 Preferred 规则
	if len(t.Preferred) > 0 {
		for _, asset := range release.Assets {
			for _, re := range t.Preferred {
				if re.MatchString(asset.Name) {
					return &asset, nil
				}
			}
		}
	}
	// 再尝试 Patterns
	for _, asset := range release.Assets {
		for _, re := range t.Patterns {
			if re.MatchString(asset.Name) {
				return &asset, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到匹配的下载文件（平台: %s, 架构: %s）", t.OS, t.Arch)
}
