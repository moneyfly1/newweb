package handlers

import "testing"

// 设备详情（型号/系统/品牌）以前**永远补不上**：首次登记时客户端没发
// x-device-model，落库的是空串；老代码用 `field == nil` 判缺失，于是
// 线上两台 Mclash 桌面端的 device_model / os_version 一直是 ''。
// 这里钉住「空串也算缺失、只补不覆盖」的口径。
func TestDeviceDetailUpdates(t *testing.T) {
	str := func(s string) *string { return &s }

	t.Run("空串也补齐（这是线上真实形态）", func(t *testing.T) {
		got := deviceDetailUpdates(
			"Mac16,10", "Apple", "macOS", "26.6",
			str(""), str(""), str("macOS"), str(""),
		)
		if got["device_model"] != "Mac16,10" {
			t.Errorf("空串型号必须补上，实际 %v", got["device_model"])
		}
		if got["device_brand"] != "Apple" {
			t.Errorf("空串品牌必须补上，实际 %v", got["device_brand"])
		}
		if got["os_version"] != "26.6" {
			t.Errorf("空串系统版本必须补上，实际 %v", got["os_version"])
		}
		if _, ok := got["os_name"]; ok {
			t.Error("已有值的字段不该被改写")
		}
	})

	t.Run("NULL 同样补齐", func(t *testing.T) {
		got := deviceDetailUpdates("Mac16,10", "", "", "", nil, nil, nil, nil)
		if got["device_model"] != "Mac16,10" {
			t.Errorf("NULL 型号必须补上，实际 %v", got["device_model"])
		}
	})

	t.Run("已有值（可能是后台人工填的）不被覆盖", func(t *testing.T) {
		got := deviceDetailUpdates(
			"Windows 11 Pro", "LENOVO", "Windows", "10.0",
			str("ThinkPad X1 Carbon"), str("Lenovo"), str("Windows"), str("11"),
		)
		if len(got) != 0 {
			t.Errorf("不该产生任何更新，实际 %v", got)
		}
	})

	t.Run("客户端没发头时什么都不写（不能把空值写进去）", func(t *testing.T) {
		got := deviceDetailUpdates("", "  ", "", "", nil, nil, nil, nil)
		if len(got) != 0 {
			t.Errorf("没有上报就不该更新，实际 %v", got)
		}
	})

	t.Run("两端都是空白串时视为无值", func(t *testing.T) {
		if !blankStrPtr(nil) || !blankStrPtr(str("")) || !blankStrPtr(str("   ")) {
			t.Error("nil / 空串 / 空白串都算缺失")
		}
		if blankStrPtr(str("macOS")) {
			t.Error("有值不算缺失")
		}
	})
}
