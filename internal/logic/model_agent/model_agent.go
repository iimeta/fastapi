package model

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"go.mongodb.org/mongo-driver/bson"
)

type sModelAgent struct {
	modelAgentsMap *gmap.StrAnyMap
	roundRobinMap  *gmap.StrAnyMap
}

func init() {
	service.RegisterModelAgent(New())
}

func New() service.IModelAgent {
	return &sModelAgent{
		modelAgentsMap: gmap.NewStrAnyMap(true),
		roundRobinMap:  gmap.NewStrAnyMap(true),
	}
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
		"status": 1,
	}

	results, err := dao.ModelAgent.Find(ctx, filter, "-weight")
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

// 挑选模型代理
func (s *sModelAgent) PickModelAgent(ctx context.Context, m *model.Model) (modelAgent *model.ModelAgent, err error) {

	var modelAgents []*model.ModelAgent
	var roundRobin *util.RoundRobin

	modelAgentsValue := s.modelAgentsMap.Get(m.Id)
	roundRobinValue := s.roundRobinMap.Get(m.Id)

	if modelAgentsValue != nil {
		modelAgents = modelAgentsValue.([]*model.ModelAgent)
	}

	if len(modelAgents) == 0 {

		modelAgents, err = s.List(ctx, m.ModelAgents)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if len(modelAgents) == 0 {
			return nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT
		}

		s.modelAgentsMap.Set(m.Id, modelAgents)
	}

	if roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.roundRobinMap.Set(m.Id, roundRobin)
	}

	return modelAgents[roundRobin.Index(len(modelAgents))], nil
}

// 移除模型代理
func (s *sModelAgent) RemoveModelAgent(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent) {

	modelAgentsValue := s.modelAgentsMap.Get(m.Id)
	if modelAgentsValue != nil {

		modelAgents := modelAgentsValue.([]*model.ModelAgent)

		if len(modelAgents) > 0 {

			newModelAgents := make([]*model.ModelAgent, 0)

			for _, agent := range modelAgents {
				if agent.Id != modelAgent.Id {
					newModelAgents = append(newModelAgents, agent)
				}
			}

			s.modelAgentsMap.Set(m.Id, newModelAgents)
		}

		if err := dao.ModelAgent.UpdateById(ctx, modelAgent.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 记录错误模型代理
func (s *sModelAgent) RecordErrorModelAgent(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent) {

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT, m.Model), modelAgent.Id, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	_, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT, m.Model), gtime.Now().EndOfDay().Time)
	if err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {
		s.RemoveModelAgent(ctx, m, modelAgent)
	}
}
