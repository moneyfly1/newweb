package models

import "testing"

// 「固定在线」用于只在部分地区可达的节点（例如仅中国境内可连的住宅 IP）：
// 服务端探测永远失败，若按探测结果过滤，这些节点就永远下发不出去，
// 而客户在国内其实是能用的。
func TestNodeDeliverableWhere(t *testing.T) {
	where, args := NodeDeliverableWhere()
	if where == "" || len(args) != 3 {
		t.Fatalf("查询条件异常: %q %v", where, args)
	}
	if args[0] != true || args[1] != NodeStatusOnline || args[2] != true {
		t.Errorf("条件参数不对（应为 启用 + 在线或固定在线）: %v", args)
	}
}
