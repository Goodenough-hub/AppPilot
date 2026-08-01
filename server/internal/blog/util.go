package blog

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

var (
	ErrUserNotFound  = errors.New("blog user not found")
	ErrUserExists    = errors.New("blog username already exists")
	ErrDraftNotFound = errors.New("draft not found")
	ErrConflict      = errors.New("optimistic lock conflict")
)

// slugRegexp 允许小写字母、数字、连字符与中文（Han），全站唯一。
// 不以连字符开头/结尾，不连续连字符。
var slugRegexp = regexp.MustCompile(`^[a-z0-9\p{Han}]+(?:-[a-z0-9\p{Han}]+)*$`)

// ValidSlug 校验 slug：非空、只含小写字母/数字/连字符、不以连字符开头或结尾。
func ValidSlug(slug string) bool {
	return slug != "" && slugRegexp.MatchString(slug)
}

func encodeID(id int64) string {
	return strconv.FormatInt(id, 10)
}

// blogUserID 从 gin.Context 取 BlogAuthRequired 注入的 blog 用户 ID。
func blogUserID(c *gin.Context) int64 {
	v, _ := c.Get("blogUserID")
	id, _ := v.(int64)
	return id
}

func parseIDParam(c *gin.Context, key string) (int64, bool) {
	s := c.Param(key)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid id"})
		return 0, false
	}
	return id, true
}

// strPtr 返回 *string，供请求体与测试构造指针字段。
func strPtr(s string) *string { return &s }
