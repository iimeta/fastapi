package common

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk/v2"
	sconsts "github.com/iimeta/fastapi-sdk/v2/consts"
	"github.com/iimeta/fastapi-sdk/v2/general"
	"github.com/iimeta/fastapi-sdk/v2/openai"
	"github.com/iimeta/fastapi-sdk/v2/options"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func NewAdapter(ctx context.Context, mak *MAK, isLong bool) (adapter sdk.Adapter) {

	if mak.Path == "" {
		defer func() {
			if general, isGeneral := adapter.(*general.General); isGeneral {
				if general.Path == "" {
					general.Path = g.RequestFromCtx(ctx).URL.Path
					if gstr.HasSuffix(general.BaseUrl, "/v1beta") {
						general.Path = general.Path[7:]
					} else if gstr.HasSuffix(general.BaseUrl, "/v1") {
						general.Path = general.Path[3:]
					}
				}
			}
		}()
	}

	options := &options.AdapterOptions{
		Provider:             GetProviderCode(ctx, mak.Provider),
		Model:                mak.RealModel.Model,
		Key:                  mak.RealKey,
		BaseUrl:              mak.BaseUrl,
		Path:                 mak.Path,
		Stream:               isLong,
		Action:               g.RequestFromCtx(ctx).GetRouter("action", "").String(),
		Timeout:              config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl:             config.Cfg.Http.ProxyUrl,
		ReqPassthroughParams: getReqPassthroughParams(mak.Passthrough),
		ResPassthroughParams: getResPassthroughParams(mak.Passthrough),
		PassthroughHeader:    getPassthroughHeaders(ctx, mak.Passthrough),
	}

	if mak.Passthrough != nil && slices.Contains(mak.Passthrough.ReqParams, "req_path") {
		options.Path = g.RequestFromCtx(ctx).URL.Path
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	if mak.RealModel.IsEnablePresetConfig {
		options.IsSupportSystemRole = &mak.RealModel.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &mak.RealModel.PresetConfig.IsSupportStream
	}

	g.RequestFromCtx(ctx).SetCtxVar("passthrough", mak.Passthrough)

	return sdk.NewAdapter(ctx, options)
}

func NewAdapterOfficial(ctx context.Context, mak *MAK, isLong bool) (adapter sdk.AdapterOfficial) {

	if mak.Path == "" {
		defer func() {
			if general, isGeneral := adapter.(*general.General); isGeneral {
				if general.Path == "" {
					general.Path = g.RequestFromCtx(ctx).URL.Path
					if gstr.HasSuffix(general.BaseUrl, "/v1beta") {
						general.Path = general.Path[7:]
					} else if gstr.HasSuffix(general.BaseUrl, "/v1") {
						general.Path = general.Path[3:]
					}
				}
			}
		}()
	}

	options := &options.AdapterOptions{
		Provider:             GetProviderCode(ctx, mak.Provider),
		Model:                mak.RealModel.Model,
		Key:                  mak.RealKey,
		BaseUrl:              mak.BaseUrl,
		Path:                 mak.Path,
		Stream:               isLong,
		Action:               g.RequestFromCtx(ctx).GetRouter("action", "").String(),
		Timeout:              config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl:             config.Cfg.Http.ProxyUrl,
		ReqPassthroughParams: []string{"req_data"},
		PassthroughHeader:    getPassthroughHeaders(ctx, mak.Passthrough),
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	if mak.RealModel.IsEnablePresetConfig {
		options.IsSupportSystemRole = &mak.RealModel.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &mak.RealModel.PresetConfig.IsSupportStream
	}

	officialPassthrough := &EffectivePassthrough{ResParams: []string{"res_data"}}
	if mak.Passthrough != nil {
		if slices.Contains(mak.Passthrough.ResParams, "res_header") {
			officialPassthrough.ResParams = append(officialPassthrough.ResParams, "res_header")
			officialPassthrough.ResHeaderMode = mak.Passthrough.ResHeaderMode
			officialPassthrough.ResHeaderList = mak.Passthrough.ResHeaderList
		}
	}
	g.RequestFromCtx(ctx).SetCtxVar("passthrough", officialPassthrough)

	return sdk.NewAdapterOfficial(ctx, options)
}

func NewAdapterOpenAI(ctx context.Context, mak *MAK, isLong bool) *openai.OpenAI {

	options := &options.AdapterOptions{
		Provider:             sconsts.PROVIDER_OPENAI,
		Model:                mak.RealModel.Model,
		Key:                  mak.RealKey,
		BaseUrl:              mak.BaseUrl,
		Path:                 mak.Path,
		Stream:               isLong,
		Action:               g.RequestFromCtx(ctx).GetRouter("action", "").String(),
		Timeout:              config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl:             config.Cfg.Http.ProxyUrl,
		ReqPassthroughParams: []string{"req_data"},
		PassthroughHeader:    getPassthroughHeaders(ctx, mak.Passthrough),
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	if mak.RealModel.IsEnablePresetConfig {
		options.IsSupportSystemRole = &mak.RealModel.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &mak.RealModel.PresetConfig.IsSupportStream
	}

	openaiPassthrough := &EffectivePassthrough{ResParams: []string{"res_data"}}
	if mak.Passthrough != nil {
		if slices.Contains(mak.Passthrough.ResParams, "res_header") {
			openaiPassthrough.ResParams = append(openaiPassthrough.ResParams, "res_header")
			openaiPassthrough.ResHeaderMode = mak.Passthrough.ResHeaderMode
			openaiPassthrough.ResHeaderList = mak.Passthrough.ResHeaderList
		}
	}
	g.RequestFromCtx(ctx).SetCtxVar("passthrough", openaiPassthrough)

	return openai.NewAdapter(ctx, options)
}

func NewRealtimeClient(ctx context.Context, model *model.Model, key, baseUrl, path string) *sdk.RealtimeClient {
	return sdk.NewRealtimeClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
}

func NewModerationClient(ctx context.Context, m *model.Model, key, baseUrl, path string, passthrough *EffectivePassthrough) *sdk.ModerationClient {

	g.RequestFromCtx(ctx).SetCtxVar("passthrough", passthrough)

	return sdk.NewModerationClient(ctx, m.Model, key, baseUrl, path, config.Cfg.Base.ShortTimeout*time.Second, config.Cfg.Http.ProxyUrl, getReqPassthroughParams(passthrough), getPassthroughHeaders(ctx, passthrough))
}

func NewConverter(ctx context.Context, provider string) sdk.Converter {
	return sdk.NewConverter(ctx, &options.AdapterOptions{Provider: provider})
}

func GetProviderCode(ctx context.Context, providerId string) string {

	provider, err := service.Provider().GetCache(ctx, providerId)
	if err != nil || provider == nil {
		provider, err = service.Provider().GetAndSaveCache(ctx, providerId)
	}

	if err != nil {
		logger.Error(ctx, err)
		return providerId
	}

	if provider != nil {
		return provider.Code
	}

	return providerId
}

func getReqPassthroughParams(pt *EffectivePassthrough) []string {
	if pt == nil {
		return nil
	}
	return pt.ReqParams
}

func getResPassthroughParams(pt *EffectivePassthrough) []string {
	if pt == nil {
		return nil
	}
	return pt.ResParams
}

func getPassthroughHeaders(ctx context.Context, pt *EffectivePassthrough) map[string]string {
	if pt == nil || !slices.Contains(pt.ReqParams, "req_header") {
		return nil
	}
	headers := make(map[string]string)
	request := g.RequestFromCtx(ctx)
	for k, v := range request.Header {
		key := strings.ToLower(k)
		if pt.ReqHeaderMode == 1 {
			if !slices.Contains(ReqReservedHeaders, key) {
				headers[k] = v[0]
			}
		} else if pt.ReqHeaderMode == 2 {
			for _, allowed := range pt.ReqHeaderList {
				if strings.EqualFold(k, allowed) {
					headers[k] = v[0]
					break
				}
			}
		}
	}
	return headers
}
