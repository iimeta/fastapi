package common

import (
	"context"

	"github.com/iimeta/fastapi-sdk"
	"github.com/iimeta/fastapi-sdk/anthropic"
	"github.com/iimeta/fastapi-sdk/google"
	"github.com/iimeta/fastapi-sdk/openai"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func NewAdapter(ctx context.Context, corp string, model *model.Model, key, baseURL, path string) (sdk.Adapter, error) {

	if model.IsEnablePresetConfig {
		return sdk.NewAdapter(ctx, GetCorpCode(ctx, corp), model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, &model.PresetConfig.IsSupportStream, config.Cfg.Http.ProxyUrl), nil
	}

	return sdk.NewAdapter(ctx, GetCorpCode(ctx, corp), model.Model, key, baseURL, path, nil, nil, config.Cfg.Http.ProxyUrl), nil
}

func NewGoogleAdapter(ctx context.Context, model *model.Model, key, baseURL, path string) (*google.Google, error) {

	if model.IsEnablePresetConfig {
		return google.NewAdapter(ctx, model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, &model.PresetConfig.IsSupportStream, config.Cfg.Http.ProxyUrl), nil
	}

	return google.NewAdapter(ctx, model.Model, key, baseURL, path, nil, nil, config.Cfg.Http.ProxyUrl), nil
}

func NewAnthropicAdapter(ctx context.Context, model *model.Model, key, baseURL, path string) (*anthropic.Anthropic, error) {

	if model.IsEnablePresetConfig {
		return anthropic.NewAdapter(ctx, model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, &model.PresetConfig.IsSupportStream, config.Cfg.Http.ProxyUrl), nil
	}

	isSupportSystemRole := true
	isSupportStream := true

	return anthropic.NewAdapter(ctx, model.Model, key, baseURL, path, &isSupportSystemRole, &isSupportStream, config.Cfg.Http.ProxyUrl), nil
}

func NewRealtimeAdapter(ctx context.Context, model *model.Model, key, baseURL, path string) (*sdk.RealtimeClient, error) {
	return sdk.NewRealtimeClient(ctx, model.Model, key, baseURL, path, config.Cfg.Http.ProxyUrl), nil
}

func NewOpenAIAdapter(ctx context.Context, model *model.Model, key, baseURL, path string) (*openai.OpenAI, error) {

	if path == "" {
		path = "/responses"
	}

	if model.IsEnablePresetConfig {
		return openai.NewAdapter(ctx, model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, &model.PresetConfig.IsSupportStream, config.Cfg.Http.ProxyUrl), nil
	}

	isSupportSystemRole := true
	isSupportStream := true

	return openai.NewAdapter(ctx, model.Model, key, baseURL, path, &isSupportSystemRole, &isSupportStream, config.Cfg.Http.ProxyUrl), nil
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
