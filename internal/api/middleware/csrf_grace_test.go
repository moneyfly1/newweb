package middleware

import "testing"

// 轮换后旧 token 必须在宽限期内仍然有效（客户端并发写请求不会被轮换打死），
// 而胡乱拼的 token 一律拒绝。
func TestCSRFGraceAfterRotate(t *testing.T) {
	const uid = 999999
	old := generateCSRFToken(uid)
	if old == "" {
		t.Fatal("expected a token")
	}
	rotateCSRFToken(uid)
	if !validateCSRFToken(uid, old) {
		t.Fatal("轮换后旧 token 应仍在宽限期内可用")
	}
	fresh := generateCSRFToken(uid)
	if fresh == old {
		t.Fatal("轮换应生成新 token")
	}
	if !validateCSRFToken(uid, fresh) {
		t.Fatal("新 token 必须可用")
	}
	if validateCSRFToken(uid, "not-a-real-token") {
		t.Fatal("伪造 token 必须被拒绝")
	}
	if validateCSRFToken(uid, "") {
		t.Fatal("空 token 必须被拒绝")
	}
}
