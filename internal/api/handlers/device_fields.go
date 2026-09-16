package handlers

import "strings"

// blankStrPtr 指针为空、或指向空白串，都算「没有值」。
//
// 为什么要专门判断：设备详情字段（device_model / device_brand / os_name /
// os_version / software_version / device_type）在首次登记时如果客户端没上报，
// 落库的是**空串**而不是 NULL。老代码用 `field == nil` 判「缺失」，
// 于是这些字段**永远补不上** —— 线上实测两台 Mclash 桌面端的
// device_model / device_brand / os_version 全是 ”（面板与客户端都显示空）。
func blankStrPtr(p *string) bool {
	return p == nil || strings.TrimSpace(*p) == ""
}

// deviceDetailUpdates 依据本次请求头，算出「需要补写」的设备详情。
//
// 口径：**只补空，不改已有值**。已有值可能是后台人工填的（备注/机型纠正），
// 客户端不该把它冲掉；而空值（含空串）说明这台设备登记时没报上来，可以自愈。
//
// 头名与客户端（`HwidUtils.getHwidHeaders`）一致：
// x-device-model / x-device-brand / x-device-os / x-ver-os。
func deviceDetailUpdates(
	model string,
	brand string,
	osName string,
	osVersion string,
	curModel *string,
	curBrand *string,
	curOSName *string,
	curOSVersion *string,
) map[string]interface{} {
	updates := map[string]interface{}{}
	if v := strings.TrimSpace(model); v != "" && blankStrPtr(curModel) {
		updates["device_model"] = v
	}
	if v := strings.TrimSpace(brand); v != "" && blankStrPtr(curBrand) {
		updates["device_brand"] = v
	}
	if v := strings.TrimSpace(osName); v != "" && blankStrPtr(curOSName) {
		updates["os_name"] = v
	}
	if v := strings.TrimSpace(osVersion); v != "" && blankStrPtr(curOSVersion) {
		updates["os_version"] = v
	}
	return updates
}
