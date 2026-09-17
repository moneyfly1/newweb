package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cboard/v2/internal/database"
	"cboard/v2/internal/models"
	"cboard/v2/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 客户连续输错密码被限制后，后台必须能「看到」并「解封」。
// 这里钉住整条链路：失败记录累计 → 列表里能看到锁定 → 解封后记录被清除。
func setupSecurityTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	oldDB := database.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.LoginAttempt{}, &models.VerificationAttempt{}, &models.SystemConfig{}, &models.AuditLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	database.DB = db
	utils.InvalidateSettingsCache()
	t.Cleanup(func() {
		database.DB = oldDB
		utils.InvalidateSettingsCache()
	})
	return db
}

func setIntSetting(t *testing.T, db *gorm.DB, key string, value int) {
	t.Helper()
	if err := db.Create(&models.SystemConfig{Key: key, Value: itoa(value)}).Error; err != nil {
		t.Fatalf("create setting %s: %v", key, err)
	}
	utils.InvalidateSettingsCache()
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	out := ""
	for v > 0 {
		out = string(rune('0'+v%10)) + out
		v /= 10
	}
	if neg {
		out = "-" + out
	}
	return out
}

func newTestContext(method, path, body string, params gin.Params) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	return c, w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, w.Body.String())
	}
	return out
}

func TestLoginLockVisibleAndUnlockable(t *testing.T) {
	db := setupSecurityTestDB(t)
	setIntSetting(t, db, "max_login_attempts", 5)
	setIntSetting(t, db, "login_lockout_minutes", 30)

	user := models.User{Username: "locked_customer", Email: "locked@example.com", Password: "x", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	const ip = "203.0.113.9"
	for i := 0; i < 5; i++ {
		addr := ip
		db.Create(&models.LoginAttempt{Username: "locked@example.com", IPAddress: &addr, Success: false})
	}

	// 1) 列表里能看到这个锁定（账号 + IP + 失败次数）
	c, w := newTestContext(http.MethodGet, "/api/v1/admin/security/login-limits", "", nil)
	AdminListLoginLimits(c)
	if w.Code != http.StatusOK {
		t.Fatalf("列表接口状态 %d: %s", w.Code, w.Body.String())
	}
	body := decodeBody(t, w)
	data := body["data"].(map[string]any)
	if data["lockout_enabled"] != true {
		t.Errorf("配置为 5 次/30 分钟时应视为启用: %v", data["lockout_note"])
	}
	locks := data["locked_accounts"].([]any)
	if len(locks) != 1 {
		t.Fatalf("应列出 1 条锁定，实际 %d 条: %s", len(locks), w.Body.String())
	}
	first := locks[0].(map[string]any)
	if first["username"] != "locked@example.com" || first["ip_address"] != ip {
		t.Errorf("锁定条目内容不对: %v", first)
	}
	if first["fail_count"].(float64) != 5 {
		t.Errorf("失败次数应为 5: %v", first["fail_count"])
	}

	// 2) 按用户 ID 解封：失败记录被清除，账号不再被锁
	c2, w2 := newTestContext(http.MethodPost, "/api/v1/admin/security/unlock",
		`{"user_id":`+itoa(int(user.ID))+`}`, nil)
	AdminUnlockLogin(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("解封接口状态 %d: %s", w2.Code, w2.Body.String())
	}
	res := decodeBody(t, w2)["data"].(map[string]any)
	if res["deleted_login_attempts"].(float64) != 5 {
		t.Errorf("应删除 5 条失败记录: %v", res["deleted_login_attempts"])
	}

	var remaining int64
	db.Model(&models.LoginAttempt{}).Where("username = ?", "locked@example.com").Count(&remaining)
	if remaining != 0 {
		t.Errorf("解封后仍有 %d 条失败记录", remaining)
	}

	// 3) 再查列表：锁定消失
	c3, w3 := newTestContext(http.MethodGet, "/api/v1/admin/security/login-limits", "", nil)
	AdminListLoginLimits(c3)
	locks3 := decodeBody(t, w3)["data"].(map[string]any)["locked_accounts"].([]any)
	if len(locks3) != 0 {
		t.Errorf("解封后不应再有锁定记录: %v", locks3)
	}
}

// 配置为 0 时必须明确告诉管理员「锁定没生效」，而不是显示成已开启。
func TestLoginLockDisabledIsReported(t *testing.T) {
	db := setupSecurityTestDB(t)
	setIntSetting(t, db, "max_login_attempts", 2000)
	setIntSetting(t, db, "login_lockout_minutes", 0)

	addr := "198.51.100.7"
	for i := 0; i < 3; i++ {
		db.Create(&models.LoginAttempt{Username: "someone@example.com", IPAddress: &addr, Success: false})
	}

	c, w := newTestContext(http.MethodGet, "/api/v1/admin/security/login-limits", "", nil)
	AdminListLoginLimits(c)
	data := decodeBody(t, w)["data"].(map[string]any)
	if data["lockout_enabled"] != false {
		t.Errorf("锁定时长为 0 时应报告为未启用")
	}
	note, _ := data["lockout_note"].(string)
	if !strings.Contains(note, "未启用") {
		t.Errorf("应给出「未启用」提示，实际: %q", note)
	}
	if len(data["locked_accounts"].([]any)) != 0 {
		t.Errorf("未启用锁定时不应列出锁定账号")
	}
	// 但失败记录仍应出现在「近 24 小时失败」里，方便客服排查
	if len(data["recent_failures"].([]any)) == 0 {
		t.Errorf("近 24 小时失败列表不应为空")
	}
}

// 只填 IP 也要能解封（客户没登录、只知道来源 IP 的场景）
func TestUnlockByIPOnly(t *testing.T) {
	db := setupSecurityTestDB(t)
	setIntSetting(t, db, "max_login_attempts", 3)
	setIntSetting(t, db, "login_lockout_minutes", 30)

	addr := "198.51.100.88"
	for i := 0; i < 4; i++ {
		db.Create(&models.LoginAttempt{Username: "a@example.com", IPAddress: &addr, Success: false})
	}
	db.Create(&models.LoginAttempt{Username: "b@example.com", IPAddress: &addr, Success: false})

	c, w := newTestContext(http.MethodPost, "/api/v1/admin/security/unlock", `{"ip_address":"198.51.100.88"}`, nil)
	AdminUnlockLogin(c)
	if w.Code != http.StatusOK {
		t.Fatalf("解封失败 %d: %s", w.Code, w.Body.String())
	}
	var remaining int64
	db.Model(&models.LoginAttempt{}).Where("ip_address = ?", addr).Count(&remaining)
	if remaining != 0 {
		t.Errorf("按 IP 解封后仍有 %d 条记录", remaining)
	}
}

// 参数全空必须报错，避免误清全部记录
func TestUnlockRequiresTarget(t *testing.T) {
	setupSecurityTestDB(t)
	c, w := newTestContext(http.MethodPost, "/api/v1/admin/security/unlock", `{}`, nil)
	AdminUnlockLogin(c)
	if w.Code == http.StatusOK {
		t.Errorf("没有任何目标时不应返回成功: %s", w.Body.String())
	}
}

// 客服只拿得到客户邮箱时的完整链路：按邮箱查 → 看到「已锁定 + 剩余时间 + 来源 IP」→ 解封 → 立即能登录。
func TestLookupByEmailShowsLockAndSourceIP(t *testing.T) {
	db := setupSecurityTestDB(t)
	setIntSetting(t, db, "max_login_attempts", 5)
	setIntSetting(t, db, "login_lockout_minutes", 30)

	user := models.User{Username: "kelly", Email: "kelly@example.com", Password: "x", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	const ip = "198.51.100.24"
	for i := 0; i < 5; i++ {
		addr := ip
		db.Create(&models.LoginAttempt{Username: "kelly@example.com", IPAddress: &addr, Success: false})
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/security/lookup?identifier=kelly@example.com", nil)
	AdminLookupLoginLimit(c)

	data := decodeBody(t, w)["data"].(map[string]any)
	if data["found"] != true {
		t.Fatalf("应能按邮箱找到用户: %s", w.Body.String())
	}
	if data["limited"] != true {
		t.Errorf("5 次失败应显示为已锁定")
	}
	if data["remaining_seconds"].(float64) <= 0 {
		t.Errorf("应给出剩余锁定时间")
	}
	ips := data["source_ips"].([]any)
	if len(ips) != 1 || ips[0].(map[string]any)["ip_address"] != ip {
		t.Errorf("应列出失败来源 IP，实际 %v", ips)
	}
	if len(data["notes"].([]any)) == 0 {
		t.Errorf("应给出客服可读的说明")
	}

	// 邮箱大小写不敏感
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	c2.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/security/lookup?identifier=Kelly@Example.com", nil)
	AdminLookupLoginLimit(c2)
	if decodeBody(t, w2)["data"].(map[string]any)["found"] != true {
		t.Errorf("邮箱查询应大小写不敏感")
	}
}

// 账号被禁用（is_active=false）不是登录限制，必须明确提示，否则客服会一直纠结为什么解封了还登不上。
func TestLookupReportsDisabledAccount(t *testing.T) {
	db := setupSecurityTestDB(t)
	setIntSetting(t, db, "max_login_attempts", 5)
	setIntSetting(t, db, "login_lockout_minutes", 30)

	user := models.User{Username: "disabled_user", Email: "disabled@example.com", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	// is_active 列有 default true，Create 时零值会被默认值覆盖，必须显式更新
	if err := db.Model(&user).Update("is_active", false).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/security/lookup?identifier=disabled@example.com", nil)
	AdminLookupLoginLimit(c)

	data := decodeBody(t, w)["data"].(map[string]any)
	if data["found"] != true {
		t.Fatalf("应找到用户: %s", w.Body.String())
	}
	u := data["user"].(map[string]any)
	if u["is_active"] != false {
		t.Errorf("应返回账号禁用状态")
	}
	found := false
	for _, n := range data["notes"].([]any) {
		if strings.Contains(n.(string), "禁用") {
			found = true
		}
	}
	if !found {
		t.Errorf("应提示账号已被禁用，实际 %v", data["notes"])
	}
}

// 查不到用户时也必须能给出「没有这个用户」的明确结论，而不是空结果。
func TestLookupUnknownUser(t *testing.T) {
	setupSecurityTestDB(t)
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/security/lookup?identifier=nobody@example.com", nil)
	AdminLookupLoginLimit(c)
	data := decodeBody(t, w)["data"].(map[string]any)
	if data["found"] != false {
		t.Errorf("不存在的邮箱应为 found=false")
	}
	if len(data["notes"].([]any)) == 0 {
		t.Errorf("应有「没有找到这个邮箱」的说明")
	}
}
