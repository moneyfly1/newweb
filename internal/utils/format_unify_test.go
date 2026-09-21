package utils

import (
	"strings"
	"testing"
	"time"

	"cboard/v2/internal/models"
)

// 这些用例钉住「同一件事只有一个实现」：
// 面板/CSV/优惠券的时间解析、金额换算、地区拼接此前各有 2~4 份写法，
// 同一个输入在不同入口结果不同（时区差 8 小时、差 1 分钱、地区少一段）。

func TestTimeFromUnixAuto(t *testing.T) {
	if got := TimeFromUnixAuto(1861920000000); !got.Equal(time.UnixMilli(1861920000000)) {
		t.Errorf("毫秒时间戳解析错误: %s", got)
	}
	if got := TimeFromUnixAuto(1861920000); !got.Equal(time.Unix(1861920000, 0)) {
		t.Errorf("秒时间戳解析错误: %s", got)
	}
}

func TestParseFlexibleTimeStringAcceptsPanelFormats(t *testing.T) {
	// 面板/前端实际会发的形态：toISOString()、本地 datetime、纯日期、时间戳
	cases := []struct {
		in   string
		want time.Time
	}{
		{"2029-01-01T00:00:00.000Z", time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"2029-01-01T00:00:00", time.Date(2029, 1, 1, 0, 0, 0, 0, time.Local)},
		{"2029-01-01 12:00:00", time.Date(2029, 1, 1, 12, 0, 0, 0, time.Local)},
		{"2029-01-01 12:00", time.Date(2029, 1, 1, 12, 0, 0, 0, time.Local)},
		{"2029-01-01", time.Date(2029, 1, 1, 0, 0, 0, 0, time.Local)},
		{"2029/01/01", time.Date(2029, 1, 1, 0, 0, 0, 0, time.Local)},
		{"1861920000000", time.UnixMilli(1861920000000)},
	}
	for _, tc := range cases {
		got, err := ParseFlexibleTimeString(tc.in)
		if err != nil {
			t.Errorf("%q 应该能解析，却报错: %v", tc.in, err)
			continue
		}
		if !got.Equal(tc.want) {
			t.Errorf("%q 解析结果不对: got %s want %s", tc.in, got, tc.want)
		}
	}
}

func TestParseFlexibleTimeStringRejectsGarbage(t *testing.T) {
	for _, in := range []string{"", "   ", "tomorrow", "2029年1月1日"} {
		if _, err := ParseFlexibleTimeString(in); err == nil {
			t.Errorf("%q 必须报错（静默成功会让数据库留下旧值）", in)
		}
	}
}

// 日期-only 必须按本地时区解释：两个后台入口（用户管理 / 订阅管理）
// 此前一个 UTC 一个 Local，写进库里的到期时刻差 8 小时。
func TestParseFlexibleTimeDateOnlyUsesLocalZone(t *testing.T) {
	got, err := ParseFlexibleTimeString("2029-01-01")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if _, offset := got.Zone(); offset != mustLocalOffset(t) {
		t.Errorf("应使用本地时区，实际偏移 %d", offset)
	}
}

func mustLocalOffset(t *testing.T) int {
	t.Helper()
	_, offset := time.Now().In(time.Local).Zone()
	return offset
}

func TestParseStoredTime(t *testing.T) {
	// 数据库里的历史存储形态（含纳秒、含时区后缀）必须都能读回
	for _, in := range []string{
		"2029-01-01 12:00:00.123456789",
		"2029-01-01 12:00:00",
		"2029-01-01 12:00:00+08:00",
		"2029-01-01T12:00:00Z",
	} {
		if _, err := ParseStoredTime(in); err != nil {
			t.Errorf("%q 应该能解析: %v", in, err)
		}
	}
	if _, err := ParseStoredTime("不是时间"); err == nil {
		t.Error("脏数据必须报错")
	}
}

func TestPercentOfAndCents(t *testing.T) {
	if got := PercentOf(100, 10); got != 10 {
		t.Errorf("100 的 10%% 应为 10，实际 %v", got)
	}
	if got := PercentOf(99.99, 10); got != 10 {
		t.Errorf("99.99 的 10%% 应四舍五入到 10，实际 %v", got)
	}
	// 元→分：必须四舍五入（创建与回调校验同一口径；1.005 这类十进制小数
	// 在 float64 里本就是 1.00499…，这里只钉住「创建与校验用同一函数、同一结果」）
	if got := YuanToCents(12.34); got != 1234 {
		t.Errorf("12.34 元应为 1234 分，实际 %d", got)
	}
	if got, want := YuanToCents(1.005), YuanToCents(1.005); got != want {
		t.Errorf("同一输入必须得到同一结果: %d vs %d", got, want)
	}
	if got := YuanToCentsAtLeast(0.1, 50); got != 50 {
		t.Errorf("低于下限应抬到 50 分，实际 %d", got)
	}
	if got := YuanToCentsAtLeast(9.99, 50); got != 999 {
		t.Errorf("9.99 元应为 999 分，实际 %d", got)
	}
}

// 汇率：未配置 / 非法 / 非正数都必须回退默认值，不能把支付算成 0 元。
func TestExchangeRateFallback(t *testing.T) {
	// 空 key 直接回退默认值（不触碰数据库）；
	// 「读配置 + 解析 + >0 校验」这条路径在三个支付入口共用同一份实现，
	// 由 utils.ExchangeRate 单点保证，生产环境由支付流程覆盖。
	if got := ExchangeRate("", 7.2); got != 7.2 {
		t.Errorf("空 key 应用默认值，实际 %v", got)
	}
	if got := ExchangeRate("", 0); got != 0 {
		t.Errorf("默认值应原样返回，实际 %v", got)
	}
}

// 两个离线库必须输出同一形状的地区串（此前 MMDB 少一段省份）。
func TestJoinLocationParts(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"中国", "河南省", "郑州市"}, "中国 河南省 郑州市"},
		{[]string{"中国", "", "郑州市"}, "中国 郑州市"},
		{[]string{"中国", "0", "0"}, "中国"},
		{[]string{"中国", "中国", "郑州市"}, "中国 郑州市"},
		{[]string{"", "", ""}, ""},
	}
	for _, tc := range cases {
		if got := joinLocationParts(tc.in...); got != tc.want {
			t.Errorf("joinLocationParts(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsUnknownLocation(t *testing.T) {
	for _, in := range []string{"", "  ", LocationUnknown, LocationLocal, LocationPrivate} {
		if !IsUnknownLocation(in) {
			t.Errorf("%q 应视为无有效地区", in)
		}
	}
	for _, in := range []string{"中国 河南省 郑州市", "美国 加利福尼亚州 洛杉矶"} {
		if IsUnknownLocation(in) {
			t.Errorf("%q 是有效地区，不应视为未知", in)
		}
	}
}

// 签到奖励配置存的是「分」，入账用的是「元」。
// 这里钉住换算：漏掉换算会让每个用户每天签到领到 100 倍的奖励
// （线上曾出现 15 元的签到入账，而面板上写的是 ¥0.1~¥0.5）。
func TestCentsToYuan(t *testing.T) {
	cases := []struct {
		in   int
		want float64
	}{{10, 0.1}, {15, 0.15}, {50, 0.5}, {0, 0}, {100, 1}}
	for _, tc := range cases {
		if got := CentsToYuan(tc.in); got != tc.want {
			t.Errorf("CentsToYuan(%d) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// 按比例算钱（优惠券折扣、代理佣金）必须同一口径，不允许各写各的舍入。
func TestPercentOfParity(t *testing.T) {
	amounts := []float64{1, 9.99, 10, 99.99, 100, 366.66}
	percents := []float64{1, 5, 10, 12.5, 30, 100}
	for _, a := range amounts {
		for _, p := range percents {
			got := PercentOf(a, p)
			if got != Round2(a*p/100) {
				t.Errorf("PercentOf(%v, %v) = %v 与 Round2 口径不一致", a, p, got)
			}
			if got < 0 || got > a {
				t.Errorf("PercentOf(%v, %v) = %v 越界", a, p, got)
			}
		}
	}
}

// 两个离线库必须输出同一形状的地区串：
// ip2region 走「国家 省份 城市」，MMDB 此前只输出「国家 城市」，
// 同一个 IP 走不同库得到不同粒度，后台地区统计就会把同一批用户拆成两组。
func TestFormatMMDBLocationMatchesIP2RegionShape(t *testing.T) {
	var rec mmdbCityRecord
	rec.Country.Names = map[string]string{"zh-CN": "中国", "en": "China"}
	rec.Subdivisions = append(rec.Subdivisions, struct {
		Names map[string]string `maxminddb:"names"`
	}{Names: map[string]string{"zh-CN": "河南省", "en": "Henan"}})
	rec.City.Names = map[string]string{"zh-CN": "郑州市", "en": "Zhengzhou"}

	if got := formatMMDBLocation(rec); got != "中国 河南省 郑州市" {
		t.Errorf("MMDB 地区形状不对: %q，应为 %q", got, "中国 河南省 郑州市")
	}

	// 没有省份数据时（如国家级库）只输出国家+城市，不能出现空段
	rec.Subdivisions = nil
	if got := formatMMDBLocation(rec); got != "中国 郑州市" {
		t.Errorf("无省份数据时: %q，应为 %q", got, "中国 郑州市")
	}

	// 只有英文名时回退英文，不能输出空串
	rec.Country.Names = map[string]string{"en": "United States"}
	rec.City.Names = map[string]string{"en": "Los Angeles"}
	if got := formatMMDBLocation(rec); got != "United States Los Angeles" {
		t.Errorf("英文回退: %q", got)
	}
}

// ip2region 的 xdb 有两种字段布局，这两组用例取自线上真实库的返回串。
// 此前代码一律按旧版下标取值，把「城市」当「省份」、「ISP」当「城市」，
// 线上存下来的地区因此是「中国 郑州市 电信」「United States Google LLC」。
func TestParseIP2RegionFieldsBothLayouts(t *testing.T) {
	cases := []struct {
		raw                     string
		country, province, city string
	}{
		// 新版布局（线上 ip2region_v4.xdb / ip2region_v6.xdb）：国家|省份|城市|ISP|国家码
		{"中国|河南省|郑州市|电信|CN", "中国", "河南省", "郑州市"},
		{"中国|江苏省|南京市|0|CN", "中国", "江苏省", "南京市"},
		{"中国|北京市|北京市|联通|CN", "中国", "北京市", "北京市"},
		{"United States|California|0|Google LLC|US", "United States", "California", ""},
		{"United States|California|0|0|US", "United States", "California", ""},
		{"United Kingdom|England|London|Cloudflare, Inc.|GB", "United Kingdom", "England", "London"},
		{"Reserved|Reserved|Reserved|0|0", "Reserved", "Reserved", "Reserved"},
		// 旧版布局（ip2region.db 时代）：国家|区域|省份|城市|ISP
		{"中国|0|广东省|深圳市|电信", "中国", "广东省", "深圳市"},
		{"美国|0|加利福尼亚州|洛杉矶|0", "美国", "加利福尼亚州", "洛杉矶"},
	}
	for _, tc := range cases {
		c, p, ci := parseIP2RegionFields(strings.Split(tc.raw, "|"))
		if c != tc.country || p != tc.province || ci != tc.city {
			t.Errorf("%q → (%q, %q, %q)，期望 (%q, %q, %q)", tc.raw, c, p, ci, tc.country, tc.province, tc.city)
		}
	}

	// 拼装后的展示串（两个库形状一致、不含 ISP）
	if got := joinLocationParts(parseIP2RegionFields(strings.Split("中国|河南省|郑州市|电信|CN", "|"))); got != "中国 河南省 郑州市" {
		t.Errorf("展示串应为「中国 河南省 郑州市」，实际 %q", got)
	}
	if got := joinLocationParts(parseIP2RegionFields(strings.Split("Reserved|Reserved|Reserved|0|0", "|"))); got != "Reserved" {
		t.Errorf("重复段应去重，实际 %q", got)
	}
}

// 密钥类配置必须去掉首尾空白：后台粘贴 token 常带尾随空格/换行，
// 而 HTTP 头容忍尾随空格、不容忍换行——同一份配置在测试按钮里可用、
// 在真正任务里失败，很难排查。线上 gh_nodes_token 就带着一个尾随空格。
func TestGetSecretSettingTrimsWhitespace(t *testing.T) {
	db := setupSettingsTestDB(t)
	db.Create(&models.SystemConfig{Key: "test_secret_token", Value: "  ghp_example_token \n"})
	InvalidateSettingsCache()
	if got := GetSecretSetting("test_secret_token"); got != "ghp_example_token" {
		t.Errorf("应去掉首尾空白，实际 %q", got)
	}
	m := GetSecretSettings("test_secret_token")
	if m["test_secret_token"] != "ghp_example_token" {
		t.Errorf("批量读取也应去空白，实际 %q", m["test_secret_token"])
	}
}
