package session_keep

import (
	"context"
	"crypto/md5"
	"fmt"
	"regexp"
	"sync"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
)

var regexCache sync.Map

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

	// 预解析请求体 JSON, 避免重复解析
	var bodyJson *gjson.Json
	if needsBodyParsing(cfg) {
		if j, err := gjson.LoadJson(r.GetBody()); err == nil {
			bodyJson = j
		}
	}

	for i := range cfg.Rules {
		rule := &cfg.Rules[i]

		if len(rule.ModelRegex) > 0 && !matchAnyRegex(modelName, rule.ModelRegex) {
			continue
		}

		if len(rule.PathRegex) > 0 && !matchAnyRegex(path, rule.PathRegex) {
			continue
		}

		value := extractKeyValue(ctx, bodyJson, rule.KeySources)
		if value == "" {
			continue
		}

		return buildRuleSessionKey(userId, rule, modelName, value)
	}

	if cfg.EnableSystemPromptHash {
		if hash := extractSystemPromptHash(bodyJson); hash != "" {
			return buildSysHashSessionKey(userId, modelName, hash)
		}
	}

	return nil
}

func needsBodyParsing(cfg *common.ModelAgentSessionKeep) bool {

	if cfg.EnableSystemPromptHash {
		return true
	}

	for i := range cfg.Rules {
		for j := range cfg.Rules[i].KeySources {
			if cfg.Rules[i].KeySources[j].Type == "body" {
				return true
			}
		}
	}

	return false
}

func extractKeyValue(ctx context.Context, bodyJson *gjson.Json, sources []common.SessionKeepKeySource) string {

	r := g.RequestFromCtx(ctx)

	for _, src := range sources {
		switch src.Type {
		case "body":
			if bodyJson == nil {
				continue
			}
			val := bodyJson.Get(src.Key).String()
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

func extractSystemPromptHash(bodyJson *gjson.Json) string {

	if bodyJson == nil {
		return ""
	}

	messages := bodyJson.GetJsons("messages")
	if len(messages) == 0 {
		return ""
	}

	for _, msg := range messages {
		if msg.Get("role").String() == "system" {
			content := msg.Get("content").String()
			if content != "" {
				h := md5.Sum([]byte(content))
				return fmt.Sprintf("%x", h)
			}
		}
	}

	return ""
}

func matchAnyRegex(value string, patterns []string) bool {
	for _, pattern := range patterns {
		re, ok := regexCache.Load(pattern)
		if !ok {
			compiled, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			re, _ = regexCache.LoadOrStore(pattern, compiled)
		}
		if re.(*regexp.Regexp).MatchString(value) {
			return true
		}
	}
	return false
}
