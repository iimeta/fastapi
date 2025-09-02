package common

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi-sdk"
	"github.com/iimeta/fastapi-sdk/anthropic"
	"github.com/iimeta/fastapi-sdk/google"
	"github.com/iimeta/fastapi-sdk/openai"
	"github.com/iimeta/fastapi-sdk/options"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func NewAdapter(ctx context.Context, mak *MAK, isLong bool) sdk.Adapter {

	options := &options.AdapterOptions{
		Corp:                    GetCorpCode(ctx, mak.Corp),
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

	return sdk.NewAdapter(ctx, options.Corp, options)
}

func NewGoogleAdapter(ctx context.Context, mak *MAK, isLong bool) *google.Google {

	options := &options.AdapterOptions{
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

	return google.NewAdapter(ctx, options)
}

func NewAnthropicAdapter(ctx context.Context, mak *MAK, isLong bool) *anthropic.Anthropic {

	options := &options.AdapterOptions{
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

	return anthropic.NewAdapter(ctx, options)
}

func NewRealtimeAdapter(ctx context.Context, model *model.Model, key, baseUrl, path string) *sdk.RealtimeClient {
	return sdk.NewRealtimeClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl)
}

func NewOpenAIAdapter(ctx context.Context, mak *MAK, isLong bool) *openai.OpenAI {

	if mak.Path == "" {
		mak.Path = "/responses"
	}

	options := &options.AdapterOptions{
		Model:    mak.RealModel.Model,
		Key:      mak.RealKey,
		BaseUrl:  mak.BaseUrl,
		Path:     mak.Path,
		Timeout:  config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl: config.Cfg.Http.ProxyUrl,
	}

	if isLong {
		options.Timeout = config.Cfg.Base.LongTimeout * time.Second
	}

	return openai.NewAdapter(ctx, options)
}

func NewModerationClient(ctx context.Context, model *model.Model, key, baseUrl, path string) *sdk.ModerationClient {
	return sdk.NewModerationClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Base.ShortTimeout, config.Cfg.Http.ProxyUrl)
}

func NewConverter(ctx context.Context, corp string) sdk.Converter {
	return sdk.NewConverter(ctx, corp)
}

func GetCorpCode(ctx context.Context, corpId string) string {

	corp, err := service.Corp().GetCacheCorp(ctx, corpId)
	if err != nil || corp == nil {
		corp, err = service.Corp().GetCorpAndSaveCache(ctx, corpId)
	}

	if err != nil {
		logger.Error(ctx, err)
		return corpId
	}

	if corp != nil {
		return corp.Code
	}

	return corpId
}
