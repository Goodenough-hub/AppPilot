package blog

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// verifyHMAC 用 secret 对 body 计算 HMAC-SHA256，与 Authorization 头比较。
// 头格式：Bearer <hex-digest>。
func verifyHMAC(secret, body []byte, authHeader string) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))
	got := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	return hmac.Equal([]byte(want), []byte(got))
}

// SignCallback 生成回调签名（供测试与文档校验），生产由 Actions 侧计算。
func SignCallback(secret, body []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	return "Bearer " + hex.EncodeToString(mac.Sum(nil))
}
