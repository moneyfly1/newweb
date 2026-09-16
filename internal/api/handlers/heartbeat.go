package handlers

import (
	"strings"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/services"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
)

// heartbeatWriteGap 心跳写库去抖：窗口内的重复心跳不再写库，避免高频写放大（SQLite）。
const heartbeatWriteGap = 20 * time.Second

// heartbeatSuggestedInterval 返回给客户端的建议心跳间隔（秒）。
const heartbeatSuggestedInterval = 120

// ClientHeartbeat 自有客户端（Mclash / MoneyFly / ClashMi）在线心跳。
//
// 背景：订阅拉取是周期性/手动行为，无法反映「用户此刻是否在用」——
// 用户正常上网走代理时并不会有任何请求到服务端，所以只能靠客户端主动上报。
//
// 设计：
//   - 只更新「已登记设备」的心跳，不创建新设备（心跳不应占用设备配额）；
//   - 设备指纹口径与订阅拉取完全一致（X-App-Device-Id → app_device_id → x-hwid），
//     否则心跳与订阅会落在两条不同的设备记录上；
//   - 写库带 20 秒去抖，客户端即使高频重试也不会造成写放大。
//
// 请求（任选其一携带 token，设备标识必须带）：
//
//	POST /api/v1/client/heartbeat?token=<订阅token>
//	Header: X-App-Device-Id: <hwid>   （或 x-hwid / ?device_id=）
//	Body(JSON, 可选): {"token":"<订阅token>","device_id":"<hwid>"}
func ClientHeartbeat(c *gin.Context) {
	db := database.GetDB()

	// ---- 1. 订阅 token：query > header > JSON body，兼容不同客户端实现 ----
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = strings.TrimSpace(c.GetHeader("X-Subscription-Token"))
	}
	var body struct {
		Token    string `json:"token"`
		DeviceID string `json:"device_id"`
	}
	if c.Request.Body != nil {
		_ = c.ShouldBindJSON(&body)
	}
	if token == "" {
		token = strings.TrimSpace(body.Token)
	}
	sub, err := findSubscriptionByAccessToken(db, token)
	if err != nil {
		utils.Unauthorized(c, "订阅不存在或已失效")
		return
	}

	// ---- 2. 设备标识：与订阅拉取保持同一优先级 ----
	deviceKey := strings.TrimSpace(c.GetHeader("X-App-Device-Id"))
	if deviceKey == "" {
		deviceKey = strings.TrimSpace(c.Query("app_device_id"))
	}
	if deviceKey == "" {
		deviceKey = strings.TrimSpace(c.GetHeader("x-hwid"))
	}
	if deviceKey == "" {
		deviceKey = strings.TrimSpace(body.DeviceID)
	}
	if deviceKey == "" {
		utils.BadRequest(c, "缺少设备标识（X-App-Device-Id / x-hwid）")
		return
	}

	// ---- 3. 定位已登记设备 ----
	fingerprint := services.GenerateDeviceFingerprint("MoneyFly-App-Device:"+deviceKey, "")
	var device models.Device
	if err := db.Where("subscription_id = ? AND device_fingerprint = ? AND is_active = ?", sub.ID, fingerprint, true).
		First(&device).Error; err != nil {
		// 未登记设备不自动创建（避免绕过设备数限制），提示客户端先拉取一次订阅完成登记
		utils.Success(c, gin.H{
			"online":     false,
			"registered": false,
			"interval":   heartbeatSuggestedInterval,
			"message":    "设备尚未登记，请先拉取一次订阅",
		})
		return
	}

	// ---- 4. 写入心跳（带去抖）----
	now := time.Now()
	updates := map[string]interface{}{"last_heartbeat": now}
	if ip := utils.GetRealClientIP(c); ip != "" {
		updates["ip_address"] = ip
	}
	// 仅当距上次心跳超过去抖窗口才写库；条件更新天然幂等，避免并发重复写
	if err := db.Model(&models.Device{}).
		Where("id = ? AND (last_heartbeat IS NULL OR last_heartbeat < ?)", device.ID, now.Add(-heartbeatWriteGap)).
		Updates(updates).Error; err != nil {
		utils.InternalError(c, "心跳写入失败")
		return
	}

	// ---- 4.5 补齐设备详情（型号/系统/品牌）----
	// 客户端一直在心跳里发 x-device-model / x-device-brand / x-device-os /
	// x-ver-os（HwidUtils.getHwidHeaders），但以前这里收下就丢了 ——
	// 面板里的「型号」因此永远是空的（用户实测反馈）。
	// 只补空字段，不覆盖已有值，也**不影响指纹**（指纹只看 Did）。
	if detail := deviceDetailUpdates(
		c.GetHeader("x-device-model"),
		c.GetHeader("x-device-brand"),
		c.GetHeader("x-device-os"),
		c.GetHeader("x-ver-os"),
		device.DeviceModel,
		device.DeviceBrand,
		device.OSName,
		device.OSVersion,
	); len(detail) > 0 {
		// 不放进上面那条带去抖的更新里：去抖窗口内的心跳会被跳过，
		// 补详情没必要等，而且失败也不该让心跳报错（下一次心跳会再补）。
		if err := db.Model(&models.Device{}).Where("id = ?", device.ID).
			Updates(detail).Error; err != nil {
			utils.LogError("[heartbeat] 补写设备详情失败 device=%d err=%v", device.ID, err)
		}
	}

	// ---- 5. 返回建议间隔，客户端据此自适应（服务端可随时调整）----
	utils.Success(c, gin.H{
		"online":      true,
		"registered":  true,
		"interval":    heartbeatSuggestedInterval,
		"server_time": now.Unix(),
	})
}
