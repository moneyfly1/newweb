package software_sync

import (
	"testing"

	"cboard/v2/internal/services/ghrelease"
)

// 用自研客户端真实的 Release 资产名钉子：四个平台都必须能匹配到正确文件，
// 且 SHA256SUMS-*.txt 这类说明文件绝不能被当成安装包。
// （资产名取自 moneyfly004/Mclash v0.0.1 实际发布内容）
func TestMclashAssetMatching(t *testing.T) {
	release := &ghrelease.Release{
		TagName: "v0.0.1",
		Assets: []ghrelease.Asset{
			{Name: "Mclash-android-0.0.1.aab"},
			{Name: "Mclash-android-arm64-v8a-0.0.1.apk"},
			{Name: "Mclash-android-armeabi-v7a-0.0.1.apk"},
			{Name: "Mclash-android-x86_64-0.0.1.apk"},
			{Name: "Mclash-macos-arm64-0.0.1.dmg"},
			{Name: "Mclash-macos-universal-0.0.1.dmg"},
			{Name: "Mclash-macos-x64-0.0.1.dmg"},
			{Name: "Mclash-setup-0.0.1.exe"},
			{Name: "Mclash-windows-x64-portable-0.0.1.zip"},
			{Name: "SHA256SUMS-android.txt"},
			{Name: "SHA256SUMS-macos-arm64.txt"},
			{Name: "SHA256SUMS-macos-universal.txt"},
			{Name: "SHA256SUMS-macos-x64.txt"},
			{Name: "SHA256SUMS-windows.txt"},
		},
	}

	sw := FindSoftwareByConfigKey("client_mclash_windows_url")
	if sw == nil {
		t.Fatal("Mclash 未注册进自动下载目录")
	}
	if len(sw.Targets) != 4 {
		t.Fatalf("Mclash 应有 4 个平台目标，实际 %d", len(sw.Targets))
	}

	want := map[string]string{
		"client_mclash_windows_url":   "Mclash-setup-0.0.1.exe",
		"client_mclash_macos_url":     "Mclash-macos-x64-0.0.1.dmg",
		"client_mclash_macos_arm_url": "Mclash-macos-arm64-0.0.1.dmg",
		"client_mclash_android_url":   "Mclash-android-arm64-v8a-0.0.1.apk",
	}
	for i := range sw.Targets {
		target := sw.Targets[i]
		asset, err := FindAssetFor(release, &target)
		if err != nil {
			t.Errorf("%s 应能匹配到安装包，实际报错: %v", target.ConfigKey, err)
			continue
		}
		if want[target.ConfigKey] != asset.Name {
			t.Errorf("%s 匹配错误: got %s want %s", target.ConfigKey, asset.Name, want[target.ConfigKey])
		}
	}

	// 校验文件绝不能被匹配成下载包
	for i := range sw.Targets {
		asset, err := FindAssetFor(release, &sw.Targets[i])
		if err == nil && len(asset.Name) >= 6 && asset.Name[:6] == "SHA256" {
			t.Errorf("%s 误匹配到校验文件: %s", sw.Targets[i].ConfigKey, asset.Name)
		}
	}
}

// 只有版本号/无资产时不能整条链路失效：有 tag 也要有版本号
func TestMclashFallbackWhenNoAssets(t *testing.T) {
	empty := &ghrelease.Release{TagName: "v0.0.1"}
	target := Target{ConfigKey: "client_mclash_windows_url", OS: "windows", Arch: "x64", Preferred: mclashWinSetup, Patterns: mclashWinPortable}
	if _, err := FindAssetFor(empty, &target); err == nil {
		t.Error("没有资产时应返回错误，由调用方回退到 Releases 页面")
	}
}
