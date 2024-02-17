package model

import (
	"context"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sModelAgent struct{}

func init() {
	service.RegisterModelAgent(New())
}

func New() service.IModelAgent {
	return &sModelAgent{}
}

// 根据模型代理ID获取模型代理信息
func (s *sModelAgent) GetModelAgent(ctx context.Context, id string) (*model.ModelAgent, error) {

	modelAgent, err := dao.ModelAgent.FindOne(ctx, bson.M{"_id": id, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.ModelAgent{
		Id:      modelAgent.Id,
		Name:    modelAgent.Name,
		BaseUrl: modelAgent.BaseUrl,
		Path:    modelAgent.Path,
		Weight:  modelAgent.Weight,
		Remark:  modelAgent.Remark,
		Status:  modelAgent.Status,
	}, nil
}

// 模型代理列表
func (s *sModelAgent) List(ctx context.Context, ids []string) ([]*model.ModelAgent, error) {

	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}

	results, err := dao.ModelAgent.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.ModelAgent, 0)
	for _, result := range results {
		items = append(items, &model.ModelAgent{
			Id:      result.Id,
			Name:    result.Name,
			BaseUrl: result.BaseUrl,
			Path:    result.Path,
			Weight:  result.Weight,
			Remark:  result.Remark,
			Status:  result.Status,
		})
	}

	return items, nil
}
