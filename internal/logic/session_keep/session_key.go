package session_keep

import (
	"crypto/md5"
	"fmt"
	"strconv"
	"strings"

	"github.com/iimeta/fastapi/v2/internal/model/common"
)

func buildUserSessionKey(userId int, modelName string) *common.SessionKey {
	return &common.SessionKey{
		Raw:    fmt.Sprintf("u:%d:m:%s", userId, modelName),
		UserId: userId,
		Mode:   "user",
	}
}

func buildRuleSessionKey(userId int, rule *common.SessionKeepRule, modelName, extractedValue string) *common.SessionKey {
	transformed := applyTransform(extractedValue, rule.Transform)
	return &common.SessionKey{
		Raw:    fmt.Sprintf("r:%d:%s:%s:%s", userId, rule.Name, modelName, transformed),
		UserId: userId,
		Mode:   "rule",
	}
}

func buildSysHashSessionKey(userId int, modelName, hash string) *common.SessionKey {
	return &common.SessionKey{
		Raw:    fmt.Sprintf("r:%d:syshash:%s:%s", userId, modelName, hash),
		UserId: userId,
		Mode:   "rule",
	}
}

func recoverSessionKey(raw string) *common.SessionKey {

	if strings.HasPrefix(raw, "u:") {
		parts := strings.SplitN(raw, ":", 4)
		if len(parts) >= 4 {
			userId, _ := strconv.Atoi(parts[1])
			if userId > 0 {
				return &common.SessionKey{Raw: raw, UserId: userId, Mode: "user"}
			}
		}
		return nil
	}

	if strings.HasPrefix(raw, "r:") {
		parts := strings.SplitN(raw, ":", 3)
		if len(parts) >= 3 {
			userId, _ := strconv.Atoi(parts[1])
			if userId > 0 {
				return &common.SessionKey{Raw: raw, UserId: userId, Mode: "rule"}
			}
		}
		return nil
	}

	return nil
}

func applyTransform(value, transform string) string {

	if transform == "" || transform == "none" {
		return safeLen(value)
	}

	if transform == "md5" {
		return md5Short(value)
	}

	if strings.HasPrefix(transform, "prefix:") {
		nStr := strings.TrimPrefix(transform, "prefix:")
		n, err := strconv.Atoi(nStr)
		if err == nil && n > 0 && n < len(value) {
			return safeLen(value[:n])
		}
		return safeLen(value)
	}

	return safeLen(value)
}

func md5Short(value string) string {
	h := md5.Sum([]byte(value))
	return fmt.Sprintf("%x", h)
}

func safeLen(value string) string {
	if len(value) > 64 {
		return md5Short(value)
	}
	return value
}
