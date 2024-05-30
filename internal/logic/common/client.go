package common

import (
	"context"
	sdk "github.com/iimeta/fastapi-sdk"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func NewClient(ctx context.Context, model *model.Model, key, baseURL, path string) (sdk.Chat, error) {
	return sdk.NewClient(ctx, GetCorpCode(ctx, model.Corp), model.Model, key, baseURL, path, config.Cfg.Http.ProxyUrl), nil
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
