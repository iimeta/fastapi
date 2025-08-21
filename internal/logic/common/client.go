package common

import (
	"context"
	"time"

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

func NewAdapter(ctx context.Context, corp string, model *model.Model, key, baseUrl, path string) (sdk.Adapter, error) {

	options := &options.AdapterOptions{
		Corp:     GetCorpCode(ctx, corp),
		Model:    model.Model,
		Key:      key,
		BaseUrl:  baseUrl,
		Path:     path,
		Timeout:  config.Cfg.Http.Timeout * time.Second,
		ProxyUrl: config.Cfg.Http.ProxyUrl,
	}

	if model.IsEnablePresetConfig {
		options.IsSupportSystemRole = &model.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &model.PresetConfig.IsSupportStream
	}

	return sdk.NewAdapter(ctx, options), nil
}

func NewGoogleAdapter(ctx context.Context, model *model.Model, key, baseUrl, path string) (*google.Google, error) {

	options := &options.AdapterOptions{
		Model:    model.Model,
		Key:      key,
		BaseUrl:  baseUrl,
		Path:     path,
		Timeout:  config.Cfg.Http.Timeout * time.Second,
		ProxyUrl: config.Cfg.Http.ProxyUrl,
	}

	if model.IsEnablePresetConfig {
		options.IsSupportSystemRole = &model.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &model.PresetConfig.IsSupportStream
	}

	return google.NewAdapter(ctx, options), nil
}

func NewAnthropicAdapter(ctx context.Context, model *model.Model, key, baseUrl, path string) (*anthropic.Anthropic, error) {

	options := &options.AdapterOptions{
		Model:    model.Model,
		Key:      key,
		BaseUrl:  baseUrl,
		Path:     path,
		Timeout:  config.Cfg.Http.Timeout * time.Second,
		ProxyUrl: config.Cfg.Http.ProxyUrl,
	}

	if model.IsEnablePresetConfig {
		options.IsSupportSystemRole = &model.PresetConfig.IsSupportSystemRole
		options.IsSupportStream = &model.PresetConfig.IsSupportStream
	}

	return anthropic.NewAdapter(ctx, options), nil
}

func NewRealtimeAdapter(ctx context.Context, model *model.Model, key, baseUrl, path string) (*sdk.RealtimeClient, error) {
	return sdk.NewRealtimeClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Http.ProxyUrl), nil
}

func NewOpenAIAdapter(ctx context.Context, model *model.Model, key, baseUrl, path string) (*openai.OpenAI, error) {

	if path == "" {
		path = "/responses"
	}

	options := &options.AdapterOptions{
		Model:    model.Model,
		Key:      key,
		BaseUrl:  baseUrl,
		Path:     path,
		Timeout:  config.Cfg.Http.Timeout * time.Second,
		ProxyUrl: config.Cfg.Http.ProxyUrl,
	}

	return openai.NewAdapter(ctx, options), nil
}

func NewModerationClient(ctx context.Context, model *model.Model, key, baseUrl, path string) (*sdk.ModerationClient, error) {
	return sdk.NewModerationClient(ctx, model.Model, key, baseUrl, path, config.Cfg.Http.Timeout, config.Cfg.Http.ProxyUrl), nil
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
