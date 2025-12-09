package common

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/general"
	"github.com/iimeta/fastapi-sdk/openai"
	"github.com/iimeta/fastapi-sdk/options"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func NewAdapter(ctx context.Context, mak *MAK, isLong bool) (adapter sdk.Adapter) {

	if mak.Path == "" {
		defer func() {
			if general, isGeneral := adapter.(*general.General); isGeneral {
				if general.Path == "" {
					general.Path = g.RequestFromCtx(ctx).RequestURI
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
		Provider:                GetProviderCode(ctx, mak.Provider),
		Model:                   mak.RealModel.Model,
		Key:                     mak.RealKey,
		BaseUrl:                 mak.BaseUrl,
		Path:                    mak.Path,
		Timeout:                 config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl:                config.Cfg.Http.ProxyUrl,
		IsOfficialFormatRequest: mak.ReqModel.RequestDataFormat == 2,
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	if mak.RealModel.IsEnablePresetConfig {
		options.IsSupportSystemRole = &mak.RealModel.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &mak.RealModel.PresetConfig.IsSupportStream
	}

	g.RequestFromCtx(ctx).SetCtxVar("is_official_format_response", mak.ReqModel.ResponseDataFormat == 2)

	return sdk.NewAdapter(ctx, options)
}

func NewOfficialAdapter(ctx context.Context, mak *MAK, isLong bool) (adapter sdk.Adapter) {

	if mak.Path == "" {
		defer func() {
			if general, isGeneral := adapter.(*general.General); isGeneral {
				if general.Path == "" {
					general.Path = g.RequestFromCtx(ctx).RequestURI
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
		Provider:                GetProviderCode(ctx, mak.Provider),
		Model:                   mak.RealModel.Model,
		Key:                     mak.RealKey,
		BaseUrl:                 mak.BaseUrl,
		Path:                    mak.Path,
		Timeout:                 config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl:                config.Cfg.Http.ProxyUrl,
		IsOfficialFormatRequest: true,
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	if mak.RealModel.IsEnablePresetConfig {
		options.IsSupportSystemRole = &mak.RealModel.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &mak.RealModel.PresetConfig.IsSupportStream
	}

	g.RequestFromCtx(ctx).SetCtxVar("is_official_format_response", true)

	return sdk.NewAdapter(ctx, options)
}

func NewOpenAIAdapter(ctx context.Context, mak *MAK, isLong bool) *openai.OpenAI {

	options := &options.AdapterOptions{
		Provider:                sconsts.PROVIDER_OPENAI,
		Model:                   mak.RealModel.Model,
		Key:                     mak.RealKey,
		BaseUrl:                 mak.BaseUrl,
		Path:                    mak.Path,
		Timeout:                 config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl:                config.Cfg.Http.ProxyUrl,
		IsOfficialFormatRequest: true,
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	if mak.RealModel.IsEnablePresetConfig {
		options.IsSupportSystemRole = &mak.RealModel.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &mak.RealModel.PresetConfig.IsSupportStream
	}

	g.RequestFromCtx(ctx).SetCtxVar("is_official_format_response", true)

	return openai.NewAdapter(ctx, options)
}

func NewRealtimeAdapter(ctx context.Context, model *model.Model, key, baseUrl, path string) *sdk.RealtimeClient {
	return sdk.NewRealtimeClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
}

func NewModerationClient(ctx context.Context, model *model.Model, key, baseUrl, path string) *sdk.ModerationClient {
	return sdk.NewModerationClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Base.ShortTimeout*time.Second, config.Cfg.Http.ProxyUrl)
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
