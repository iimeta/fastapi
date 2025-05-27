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

func NewClient(ctx context.Context, corp string, model *model.Model, key, baseURL, path string) (sdk.Client, error) {

	if model.IsEnablePresetConfig {
		return sdk.NewClient(ctx, GetCorpCode(ctx, corp), model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, config.Cfg.Http.ProxyUrl), nil
	}

	return sdk.NewClient(ctx, GetCorpCode(ctx, corp), model.Model, key, baseURL, path, nil, config.Cfg.Http.ProxyUrl), nil
}

func NewGoogleClient(ctx context.Context, model *model.Model, key, baseURL, path string) (*google.Client, error) {

	if model.IsEnablePresetConfig {
		return google.NewClient(ctx, model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, config.Cfg.Http.ProxyUrl), nil
	}

	return google.NewClient(ctx, model.Model, key, baseURL, path, nil, config.Cfg.Http.ProxyUrl), nil
}

func NewAnthropicClient(ctx context.Context, model *model.Model, key, baseURL, path string) (*anthropic.Client, error) {

	if model.IsEnablePresetConfig {
		return anthropic.NewClient(ctx, model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, config.Cfg.Http.ProxyUrl), nil
	}

	isSupportSystemRole := true

	return anthropic.NewClient(ctx, model.Model, key, baseURL, path, &isSupportSystemRole, config.Cfg.Http.ProxyUrl), nil
}

func NewRealtimeClient(ctx context.Context, model *model.Model, key, baseURL, path string) (*sdk.RealtimeClient, error) {
	return sdk.NewRealtimeClient(ctx, model.Model, key, baseURL, path, config.Cfg.Http.ProxyUrl), nil
}

func NewOpenAIClient(ctx context.Context, model *model.Model, key, baseURL, path string) (*openai.Client, error) {

	if path == "" {
		path = "/responses"
	}

	if model.IsEnablePresetConfig {
		return openai.NewClient(ctx, model.Model, key, baseURL, path, &model.PresetConfig.IsSupportSystemRole, config.Cfg.Http.ProxyUrl), nil
	}

	isSupportSystemRole := true

	return openai.NewClient(ctx, model.Model, key, baseURL, path, &isSupportSystemRole, config.Cfg.Http.ProxyUrl), nil
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
