package model

import (
	"context"
	"github.com/iimeta/fastapi/internal/dao"
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

	res, err := dao.Model.FindOne(ctx, bson.M{"model": m})
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

// 模型列表
func (s *sModel) List(ctx context.Context) ([]*model.Model, error) {

	filter := bson.M{}

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
