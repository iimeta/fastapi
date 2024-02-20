package model

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
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

	result, err := dao.Model.FindOne(ctx, bson.M{"model": m, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Model{
		Id:                 result.Id,
		Corp:               result.Corp,
		Name:               result.Name,
		Model:              result.Model,
		Type:               result.Type,
		PromptRatio:        result.PromptRatio,
		CompletionRatio:    result.CompletionRatio,
		DataFormat:         result.DataFormat,
		IsEnableModelAgent: result.IsEnableModelAgent,
		ModelAgents:        result.ModelAgents,
		IsPublic:           result.IsPublic,
		Remark:             result.Remark,
		Status:             result.Status,
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
					Id:                 v.Id,
					Corp:               v.Corp,
					Name:               v.Name,
					Model:              v.Model,
					Type:               v.Type,
					PromptRatio:        v.PromptRatio,
					CompletionRatio:    v.CompletionRatio,
					DataFormat:         v.DataFormat,
					IsEnableModelAgent: v.IsEnableModelAgent,
					ModelAgents:        v.ModelAgents,
					IsPublic:           v.IsPublic,
					Remark:             v.Remark,
					Status:             v.Status,
				}, nil
			}
		}

		for _, v := range models {
			if v.Model == m {
				return &model.Model{
					Id:                 v.Id,
					Corp:               v.Corp,
					Name:               v.Name,
					Model:              v.Model,
					Type:               v.Type,
					PromptRatio:        v.PromptRatio,
					CompletionRatio:    v.CompletionRatio,
					DataFormat:         v.DataFormat,
					IsEnableModelAgent: v.IsEnableModelAgent,
					ModelAgents:        v.ModelAgents,
					IsPublic:           v.IsPublic,
					Remark:             v.Remark,
					Status:             v.Status,
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
					Id:                 v.Id,
					Corp:               v.Corp,
					Name:               v.Name,
					Model:              v.Model,
					Type:               v.Type,
					PromptRatio:        v.PromptRatio,
					CompletionRatio:    v.CompletionRatio,
					DataFormat:         v.DataFormat,
					IsEnableModelAgent: v.IsEnableModelAgent,
					ModelAgents:        v.ModelAgents,
					IsPublic:           v.IsPublic,
					Remark:             v.Remark,
					Status:             v.Status,
				}, nil
			}
		}

		for _, v := range models {
			if v.Model == m {
				return &model.Model{
					Id:                 v.Id,
					Corp:               v.Corp,
					Name:               v.Name,
					Model:              v.Model,
					Type:               v.Type,
					PromptRatio:        v.PromptRatio,
					CompletionRatio:    v.CompletionRatio,
					DataFormat:         v.DataFormat,
					IsEnableModelAgent: v.IsEnableModelAgent,
					ModelAgents:        v.ModelAgents,
					IsPublic:           v.IsPublic,
					Remark:             v.Remark,
					Status:             v.Status,
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
		"status": 1,
	}

	results, err := dao.Model.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Model, 0)
	for _, result := range results {
		items = append(items, &model.Model{
			Id:                 result.Id,
			Corp:               result.Corp,
			Name:               result.Name,
			Model:              result.Model,
			Type:               result.Type,
			PromptRatio:        result.PromptRatio,
			CompletionRatio:    result.CompletionRatio,
			DataFormat:         result.DataFormat,
			IsEnableModelAgent: result.IsEnableModelAgent,
			ModelAgents:        result.ModelAgents,
			Remark:             result.Remark,
			Status:             result.Status,
		})
	}

	return items, nil
}

// 变更订阅
func (s *sModel) Subscribe(ctx context.Context, msg string) error {

	model := new(entity.Model)
	err := gjson.Unmarshal([]byte(msg), &model)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}
	fmt.Println(gjson.MustEncodeString(model))

	return nil
}
