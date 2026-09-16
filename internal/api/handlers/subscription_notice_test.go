package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 订阅不可用这条链会读 devices / nodes，标准测试库只迁移了 subscription。
// 这里补上，才能真的走到「设备超限」与「正常下发节点」两条分支。
func setupNoticeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := database.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Subscription{},
		&models.SubscriptionReset{},
		&models.SystemConfig{},
		&models.Device{},
		&models.Node{},
		&models.UserCustomNode{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	t.Cleanup(func() {
		database.DB = oldDB
	})
	return db
}

// 「客户到期 / 被禁用 / 设备超过限制 → 禁止连接」这条产品行为的后端侧回归。
//
// 设计：订阅不可用时，这次订阅请求**只下发提示节点**（`📢 官网:` / `❌ 原因:` /
// `💡 解决:` / `💬 客服:`，本体是 baidu.com:1234 的死节点），于是客户端
//  1. 用这份内容**覆盖本地配置档** —— 之前那份能用的节点被抹掉；
//  2. 解析出的节点列表为空（提示节点会被过滤），自动选节点没有候选；
//  3. 客户端的账号门禁据此**拒绝连接**（含托盘/URL scheme/开机自动连接）。
//
// 这里的断言就是钉住「只下发提示节点、且写明原因」这一点：一旦有人改成
// 「照常下发真实节点再让客户端自己判断」，用户就能继续白嫖，本测试会红。
func TestSubscriptionUnavailableStatesEmitOnlyNoticeNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		sub        models.Subscription
		wantName   string
		wantReason string
	}{
		{
			name: "已到期",
			sub: models.Subscription{
				UserID: 1, SubscriptionURL: "expired-token",
				DeviceLimit: 3, CurrentDevices: 1,
				IsActive: true, Status: "active",
				ExpireTime: time.Now().Add(-24 * time.Hour),
			},
			wantName:   "订阅已过期",
			wantReason: "❌ 原因: 订阅已过期",
		},
		{
			name: "被禁用",
			sub: models.Subscription{
				UserID: 1, SubscriptionURL: "disabled-token",
				DeviceLimit: 3, CurrentDevices: 0,
				IsActive: false, Status: "disabled",
				ExpireTime: time.Now().Add(24 * time.Hour),
			},
			wantName:   "订阅已失效",
			wantReason: "❌ 原因: 订阅已失效",
		},
		{
			name: "设备超过限制",
			sub: models.Subscription{
				UserID: 1, SubscriptionURL: "device-full-token",
				DeviceLimit: 1, CurrentDevices: 1,
				IsActive: true, Status: "active",
				ExpireTime: time.Now().Add(24 * time.Hour),
			},
			wantName:   "设备超限",
			wantReason: "❌ 原因: 设备数量超限",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupNoticeTestDB(t)
			sub := tc.sub
			if err := db.Create(&sub).Error; err != nil {
				t.Fatalf("create subscription: %v", err)
			}

			r := gin.New()
			r.GET("/api/v1/client/subscribe", GetSubscription)
			// 非浏览器 UA → 走设备登记/限制判定那条路径
			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/client/subscribe?token="+sub.SubscriptionURL+"&type=clash",
				nil,
			)
			req.Header.Set("User-Agent", "Mclash/0.0.7 platform/windows mihomo/1.19.31")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
			}
			body := w.Body.String()

			if !strings.Contains(body, "name: "+tc.wantName) {
				t.Errorf("订阅名应为 %q（客户端用它显示状态），body=%s", tc.wantName, firstLines(body, 3))
			}
			if !strings.Contains(body, tc.wantReason) {
				t.Errorf("必须写明原因 %q，body=%s", tc.wantReason, firstLines(body, 20))
			}
			if !strings.Contains(body, "💡 解决:") {
				t.Errorf("必须给出解决方式（客户端的拦截弹窗直接展示它），body=%s", firstLines(body, 20))
			}
			// 提示节点的本体必须是死节点：即使客户端硬连也连不通
			if !strings.Contains(body, "server: baidu.com") {
				t.Errorf("提示节点本体应是不可用的占位节点，body=%s", firstLines(body, 20))
			}
			// 明确「没有被回填任何真实节点」
			if strings.Contains(body, "server: 1.2.3.4") {
				t.Errorf("不可用订阅绝不能下发真实节点，body=%s", firstLines(body, 20))
			}
		})
	}
}

// 服务端内部错误（例如设备表查询失败）绝不能谎报「订阅已失效」：
// 付费正常的客户会因此被禁止连接，而且原因说不清、客服无法解释。
func TestSubscriptionServerErrorReportsRetryableReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupNoticeTestDB(t)

	sub := models.Subscription{
		UserID: 1, SubscriptionURL: "server-error-token",
		DeviceLimit: 5, CurrentDevices: 0,
		IsActive: true, Status: "active",
		ExpireTime: time.Now().Add(24 * time.Hour),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	// 制造「查询设备失败」：把 devices 表删掉（模拟表缺失 / 数据库抖动）
	if err := db.Migrator().DropTable(&models.Device{}); err != nil {
		t.Fatalf("drop devices: %v", err)
	}

	r := gin.New()
	r.GET("/api/v1/client/subscribe", GetSubscription)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/client/subscribe?token=server-error-token&type=clash",
		nil,
	)
	req.Header.Set("User-Agent", "Mclash/0.0.7 platform/windows mihomo/1.19.31")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "❌ 原因: 服务暂时不可用") {
		t.Errorf("内部错误应如实返回可重试原因，body=%s", firstLines(body, 20))
	}
	if strings.Contains(body, "订阅已失效") {
		t.Errorf("内部错误不能谎报「订阅已失效」（付费客户会被误拦），body=%s", firstLines(body, 20))
	}
	// 仍然不下发真实节点：宁可拦住，也不放行未登记的设备
	if !strings.Contains(body, "server: baidu.com") {
		t.Errorf("服务端错误时也不该下发真实节点，body=%s", firstLines(body, 20))
	}
}

// 正常订阅必须照常下发真实节点（否则这条「禁止连接」会把付费用户也一起拦掉）。
func TestSubscriptionActiveStillServesRealNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupNoticeTestDB(t)

	sub := models.Subscription{
		UserID: 1, SubscriptionURL: "ok-token",
		DeviceLimit: 5, CurrentDevices: 0,
		IsActive: true, Status: "active",
		ExpireTime: time.Now().Add(720 * time.Hour),
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	// 必须是能通过协议过滤的真实节点配置（与生产下发的写法一致）
	cfg := "ss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@1.2.3.4:8443#%E9%A6%99%E6%B8%AF%E7%BA%BF%E8%B7%AF3"
	node := models.Node{
		Name: "香港线路3", Type: "ss", Status: models.NodeStatusOnline,
		IsActive: true, Config: &cfg,
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}

	r := gin.New()
	r.GET("/api/v1/client/subscribe", GetSubscription)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/client/subscribe?token=ok-token&type=clash",
		nil,
	)
	req.Header.Set("User-Agent", "Mclash/0.0.7 platform/windows mihomo/1.19.31")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, "香港线路3") {
		t.Errorf("正常订阅必须下发真实节点，body=%s", firstLines(body, 20))
	}
	if strings.Contains(body, "❌ 原因:") {
		t.Errorf("正常订阅不该出现错误提示节点，body=%s", firstLines(body, 20))
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
