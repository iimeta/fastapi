package session_keep

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
)

func (s *sSessionKeepModelAgent) ResolveSessionKey(ctx context.Context, modelName string, cfg *common.ModelAgentSessionKeep) *common.SessionKey {

	userId := service.Session().GetUserId(ctx)
	if userId <= 0 {
		userId = service.Session().GetUserId(g.RequestFromCtx(ctx).GetCtx())
	}

	if userId <= 0 {
		return nil
	}

	mode := cfg.Mode
	if mode == "" || mode == "user" {
		return buildUserSessionKey(userId, modelName)
	}

	return matchRuleAndBuildKey(ctx, userId, modelName, cfg)
}

func matchRuleAndBuildKey(ctx context.Context, userId int, modelName string, cfg *common.ModelAgentSessionKeep) *common.SessionKey {

	r := g.RequestFromCtx(ctx)
	path := r.URL.Path

	for i := range cfg.Rules {
		rule := &cfg.Rules[i]

		if len(rule.ModelRegex) > 0 && !matchAnyRegex(modelName, rule.ModelRegex) {
			continue
		}

		if len(rule.PathRegex) > 0 && !matchAnyRegex(path, rule.PathRegex) {
			continue
		}

		value := extractKeyValue(ctx, rule.KeySources)
		if value == "" {
			continue
		}

		return buildRuleSessionKey(userId, rule, modelName, value)
	}

	if cfg.EnableSystemPromptHash {
		if hash := extractSystemPromptHash(ctx); hash != "" {
			return buildSysHashSessionKey(userId, modelName, hash)
		}
	}

	return nil
}

func extractKeyValue(ctx context.Context, sources []common.SessionKeepKeySource) string {

	r := g.RequestFromCtx(ctx)

	for _, src := range sources {
		switch src.Type {
		case "body":
			j, err := gjson.LoadJson(r.GetBody())
			if err != nil {
				continue
			}
			val := j.Get(src.Key).String()
			if val != "" {
				return val
			}
		case "header":
			val := r.GetHeader(src.Key)
			if val != "" {
				return val
			}
		}
	}

	return ""
}

func extractSystemPromptHash(ctx context.Context) string {

	r := g.RequestFromCtx(ctx)

	j, err := gjson.LoadJson(r.GetBody())
	if err != nil {
		return ""
	}

	messages := j.GetJsons("messages")
	if len(messages) == 0 {
		return ""
	}

	for _, msg := range messages {
		if msg.Get("role").String() == "system" {
			content := msg.Get("content").String()
			if content != "" {
				h := md5.Sum([]byte(content))
				return fmt.Sprintf("%x", h)[:8]
			}
		}
	}

	return ""
}

func matchAnyRegex(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, err := regexp.MatchString(pattern, value); err == nil && matched {
			return true
		}
	}
	return false
}
