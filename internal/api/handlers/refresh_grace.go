package handlers

import (
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"cboard/v2/internal/utils"
)

// 刷新令牌「轮换宽限期」。
//
// 背景（对齐 myweb refresh_grace）：/auth/refresh 一旦成功就把旧 refresh_token
// 立刻拉黑。只要第一次刷新的响应在途中丢了（客户端进程被杀、断网、换网络），
// 客户端手里就只剩一个已拉黑的旧 token —— 下一次刷新必然 401「Refresh Token 已失效」，
// 用户被迫重新登录（部分客户端直接登出回登录页）。
//
// 这里让轮换**幂等**：旧 token 在轮换后 60 秒内再次提交时，直接返回上一次签发的
// 同一对令牌（既不签发新的、也不报 401）。窗口一过，旧 token 依然按黑名单拒绝。
const refreshRotateGracePeriod = 60 * time.Second

// 缓存上限：超出后整体清空，避免异常流量把内存撑爆。
const refreshRotateMaxEntries = 20000

type rotatedTokenPair struct {
	accessToken  string
	refreshToken string
	expireAt     time.Time
}

var (
	rotatedRefreshCache sync.Map
	rotatedJanitorOnce  sync.Once
)

// storeRotatedRefreshResult 记录「旧 token 哈希 → 刚签发的新令牌对」。
func storeRotatedRefreshResult(oldHash, accessToken, newRefreshToken string) {
	rotatedJanitorOnce.Do(startRotatedRefreshJanitor)

	if countRotatedRefreshEntries() >= refreshRotateMaxEntries {
		rotatedRefreshCache.Range(func(k, _ any) bool {
			rotatedRefreshCache.Delete(k)
			return true
		})
	}
	rotatedRefreshCache.Store(oldHash, &rotatedTokenPair{
		accessToken:  accessToken,
		refreshToken: newRefreshToken,
		expireAt:     time.Now().Add(refreshRotateGracePeriod),
	})
}

// lookupRotatedRefreshResult 命中 = 宽限期内的重复提交 → 重放上次结果。
func lookupRotatedRefreshResult(oldHash string) (accessToken, refreshToken string, ok bool) {
	val, found := rotatedRefreshCache.Load(oldHash)
	if !found {
		return "", "", false
	}
	pair, isPair := val.(*rotatedTokenPair)
	if !isPair || time.Now().After(pair.expireAt) {
		rotatedRefreshCache.Delete(oldHash)
		return "", "", false
	}
	return pair.accessToken, pair.refreshToken, true
}

// deleteRotatedRefreshResult 主动清除某条宽限记录（登出时用：登出必须立刻生效）。
func deleteRotatedRefreshResult(oldHash string) {
	rotatedRefreshCache.Delete(oldHash)
}

func countRotatedRefreshEntries() int {
	n := 0
	rotatedRefreshCache.Range(func(_, _ any) bool {
		n++
		return n < refreshRotateMaxEntries
	})
	return n
}

func startRotatedRefreshJanitor() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			rotatedRefreshCache.Range(func(k, v any) bool {
				if pair, ok := v.(*rotatedTokenPair); ok && now.After(pair.expireAt) {
					rotatedRefreshCache.Delete(k)
				}
				return true
			})
		}
	}()
}

// logRefreshReject 记录「刷新被服务端拒绝」的原因，便于定位「用户无故退出」。
func logRefreshReject(c *gin.Context, reason string, userID uint) {
	utils.LogInfo("刷新令牌被拒绝: reason=%s user_id=%d ip=%s ua=%s",
		reason, userID, c.ClientIP(), truncateUA(c.GetHeader("User-Agent")))
}

// logRefreshReplay 记录「轮换重放」：说明客户端确实丢了响应，但会话被救回来了。
func logRefreshReplay(c *gin.Context, userID uint, since time.Duration) {
	utils.LogInfo("刷新令牌轮换重放: user_id=%d 距轮换=%s ip=%s ua=%s",
		userID, since.Round(time.Second), c.ClientIP(), truncateUA(c.GetHeader("User-Agent")))
}

// truncateUA 截断 UA 到 60 字符，避免日志被超长 UA 撑爆。
func truncateUA(ua string) string {
	r := []rune(strings.TrimSpace(ua))
	if len(r) <= 60 {
		return string(r)
	}
	return string(r[:60])
}
