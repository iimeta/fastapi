package model

import (
	"context"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sModel struct{}

func init() {
	service.RegisterModel(New())
}

func New() service.IModel {
	return &sModel{}
}

// 根据model获取模型信息
func (s *sModel) GetModel(ctx context.Context, m string) (*model.Model, error) {

	res, err := dao.Model.FindOne(ctx, bson.M{"model": m, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Model{
		Id:              res.Id,
		Corp:            res.Corp,
		Name:            res.Name,
		Model:           res.Model,
		Type:            res.Type,
		PromptRatio:     res.PromptRatio,
		CompletionRatio: res.CompletionRatio,
		DataFormat:      res.DataFormat,
		BaseUrl:         res.BaseUrl,
		Path:            res.Path,
		Proxy:           res.Proxy,
		IsPublic:        res.IsPublic,
		Remark:          res.Remark,
		Status:          res.Status,
	}, nil
}

// 根据model和secretKey获取模型信息
func (s *sModel) GetModelBySecretKey(ctx context.Context, m, secretKey string) (*model.Model, error) {

	key, err := service.Key().GetKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(key.Models) > 0 {

		models, err := s.List(ctx, key.Models)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		for _, v := range models {
			if v.Name == m {
				return &model.Model{
					Id:              v.Id,
					Corp:            v.Corp,
					Name:            v.Name,
					Model:           v.Model,
					Type:            v.Type,
					PromptRatio:     v.PromptRatio,
					CompletionRatio: v.CompletionRatio,
					DataFormat:      v.DataFormat,
					BaseUrl:         v.BaseUrl,
					Path:            v.Path,
					Proxy:           v.Proxy,
					IsPublic:        v.IsPublic,
					Remark:          v.Remark,
					Status:          v.Status,
				}, nil
			}
		}

		for _, v := range models {
			if v.Model == m {
				return &model.Model{
					Id:              v.Id,
					Corp:            v.Corp,
					Name:            v.Name,
					Model:           v.Model,
					Type:            v.Type,
					PromptRatio:     v.PromptRatio,
					CompletionRatio: v.CompletionRatio,
					DataFormat:      v.DataFormat,
					BaseUrl:         v.BaseUrl,
					Path:            v.Path,
					Proxy:           v.Proxy,
					IsPublic:        v.IsPublic,
					Remark:          v.Remark,
					Status:          v.Status,
				}, nil
			}
		}

		return nil, errors.ERR_PERMISSION_DENIED
	}

	app, err := service.App().GetApp(ctx, key.AppId)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(app.Models) > 0 {

		models, err := s.List(ctx, app.Models)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		for _, v := range models {
			if v.Name == m {
				return &model.Model{
					Id:              v.Id,
					Corp:            v.Corp,
					Name:            v.Name,
					Model:           v.Model,
					Type:            v.Type,
					PromptRatio:     v.PromptRatio,
					CompletionRatio: v.CompletionRatio,
					DataFormat:      v.DataFormat,
					BaseUrl:         v.BaseUrl,
					Path:            v.Path,
					Proxy:           v.Proxy,
					IsPublic:        v.IsPublic,
					Remark:          v.Remark,
					Status:          v.Status,
				}, nil
			}
		}

		for _, v := range models {
			if v.Model == m {
				return &model.Model{
					Id:              v.Id,
					Corp:            v.Corp,
					Name:            v.Name,
					Model:           v.Model,
					Type:            v.Type,
					PromptRatio:     v.PromptRatio,
					CompletionRatio: v.CompletionRatio,
					DataFormat:      v.DataFormat,
					BaseUrl:         v.BaseUrl,
					Path:            v.Path,
					Proxy:           v.Proxy,
					IsPublic:        v.IsPublic,
					Remark:          v.Remark,
					Status:          v.Status,
				}, nil
			}
		}

		return nil, errors.ERR_PERMISSION_DENIED
	}

	return nil, errors.ERR_PERMISSION_DENIED
}

// 模型列表
func (s *sModel) List(ctx context.Context, ids []string) ([]*model.Model, error) {

	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}

	results, err := dao.Model.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Model, 0)
	for _, result := range results {
		items = append(items, &model.Model{
			Id:              result.Id,
			Corp:            result.Corp,
			Name:            result.Name,
			Model:           result.Model,
			Type:            result.Type,
			PromptRatio:     result.PromptRatio,
			CompletionRatio: result.CompletionRatio,
			DataFormat:      result.DataFormat,
			BaseUrl:         result.BaseUrl,
			Path:            result.Path,
			Proxy:           result.Proxy,
			Remark:          result.Remark,
			Status:          result.Status,
		})
	}

	return items, nil
}
