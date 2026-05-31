package common

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

// 有效透传配置(Model ∩ ModelAgent 交集)
type EffectivePassthrough struct {
	ReqParams     []string // 请求透传参数
	ResParams     []string // 响应透传参数
	ReqHeaderMode int      // 请求头透传模式[1:全量, 2:指定]
	ReqHeaderList []string // 请求头透传白名单
	ResHeaderMode int      // 响应头透传模式[1:全量, 2:指定]
	ResHeaderList []string // 响应头透传白名单
}

// 请求保留头(不透传)
var ReqReservedHeaders = []string{
	"authorization", "x-api-key", "api-key", "x-goog-api-key",
	"host", "connection", "keep-alive", "transfer-encoding", "upgrade",
	"content-length", "content-type", "trace-id",
}

// 响应保留头(不透传)
var ResReservedHeaders = []string{
	"transfer-encoding", "connection", "keep-alive",
	"content-length", "content-encoding", "set-cookie", "strict-transport-security",
}

// 计算有效的透传配置, 无ModelAgent时使用Model配置, 有ModelAgent时取交集
func GetEffectivePassthrough(ctx context.Context, m *model.Model, agent *model.ModelAgent) *EffectivePassthrough {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetEffectivePassthrough time: %d", gtime.TimestampMilli()-now)
	}()

	if agent == nil {
		if !m.IsEnableDataPassthrough {
			return nil
		}
		result := &EffectivePassthrough{
			ReqParams: m.ReqPassthroughParams,
			ResParams: m.ResPassthroughParams,
		}
		if len(result.ReqParams) == 0 && len(result.ResParams) == 0 {
			return nil
		}
		if slices.Contains(result.ReqParams, "req_header") {
			result.ReqHeaderMode = m.ReqHeaderPassthroughMode
			result.ReqHeaderList = m.ReqHeaderPassthroughList
		}
		if slices.Contains(result.ResParams, "res_header") {
			result.ResHeaderMode = m.ResHeaderPassthroughMode
			result.ResHeaderList = m.ResHeaderPassthroughList
		}
		return result
	}

	if !m.IsEnableDataPassthrough || !agent.IsEnableDataPassthrough {
		return nil
	}

	result := &EffectivePassthrough{
		ReqParams: intersect(m.ReqPassthroughParams, agent.ReqPassthroughParams),
		ResParams: intersect(m.ResPassthroughParams, agent.ResPassthroughParams),
	}

	if len(result.ReqParams) == 0 && len(result.ResParams) == 0 {
		return nil
	}

	if slices.Contains(result.ReqParams, "req_header") {
		if m.ReqHeaderPassthroughMode == 2 || agent.ReqHeaderPassthroughMode == 2 {
			result.ReqHeaderMode = 2
			result.ReqHeaderList = intersectCaseInsensitive(
				getHeaderList(m.ReqHeaderPassthroughMode, m.ReqHeaderPassthroughList),
				getHeaderList(agent.ReqHeaderPassthroughMode, agent.ReqHeaderPassthroughList),
			)
		} else {
			result.ReqHeaderMode = 1
		}
	}

	if slices.Contains(result.ResParams, "res_header") {
		if m.ResHeaderPassthroughMode == 2 || agent.ResHeaderPassthroughMode == 2 {
			result.ResHeaderMode = 2
			result.ResHeaderList = intersectCaseInsensitive(
				getHeaderList(m.ResHeaderPassthroughMode, m.ResHeaderPassthroughList),
				getHeaderList(agent.ResHeaderPassthroughMode, agent.ResHeaderPassthroughList),
			)
		} else {
			result.ResHeaderMode = 1
		}
	}

	return result
}

func getHeaderList(mode int, list []string) []string {
	if mode == 2 {
		return list
	}
	return nil
}

func intersect(a, b []string) []string {
	var result []string
	for _, v := range a {
		if slices.Contains(b, v) {
			result = append(result, v)
		}
	}
	return result
}

// WritePassthroughHeaders 将上游响应头透传写入客户端响应
func WritePassthroughHeaders(ctx context.Context, pt *EffectivePassthrough, headers http.Header) {

	if pt == nil || headers == nil || !slices.Contains(pt.ResParams, "res_header") {
		return
	}

	r := g.RequestFromCtx(ctx)

	for k, v := range headers {

		key := strings.ToLower(k)

		if slices.Contains(ResReservedHeaders, key) {
			continue
		}

		if pt.ResHeaderMode == 1 {
			r.Response.Header().Set(k, v[0])
		} else if pt.ResHeaderMode == 2 {
			for _, allowed := range pt.ResHeaderList {
				if strings.EqualFold(k, allowed) {
					r.Response.Header().Set(k, v[0])
					break
				}
			}
		}
	}
}

func intersectCaseInsensitive(a, b []string) []string {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	var result []string
	for _, v := range a {
		for _, w := range b {
			if strings.EqualFold(v, w) {
				result = append(result, strings.ToLower(v))
				break
			}
		}
	}
	return result
}
