package model

import (
	"cmp"
	"context"
	"fmt"
	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"go.mongodb.org/mongo-driver/bson"
	"slices"
)

type sModelAgent struct {
	modelAgentsMap              *gmap.StrAnyMap
	modelAgentsRoundRobinMap    *gmap.StrAnyMap
	modelAgentKeysMap           *gmap.StrAnyMap
	modelAgentKeysRoundRobinMap *gmap.StrAnyMap
	modelAgentCacheMap          *gmap.StrAnyMap
	modelAgentKeysCacheMap      *gmap.StrAnyMap
}

func init() {
	service.RegisterModelAgent(New())
}

func New() service.IModelAgent {
	return &sModelAgent{
		modelAgentsMap:              gmap.NewStrAnyMap(true),
		modelAgentsRoundRobinMap:    gmap.NewStrAnyMap(true),
		modelAgentKeysMap:           gmap.NewStrAnyMap(true),
		modelAgentKeysRoundRobinMap: gmap.NewStrAnyMap(true),
		modelAgentCacheMap:          gmap.NewStrAnyMap(true),
		modelAgentKeysCacheMap:      gmap.NewStrAnyMap(true),
	}
}

// 根据模型代理ID获取模型代理信息
func (s *sModelAgent) GetModelAgent(ctx context.Context, id string) (*model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetModelAgent time: %d", gtime.TimestampMilli()-now)
	}()

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

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent List time: %d", gtime.TimestampMilli()-now)
	}()

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

// 根据模型代理ID获取密钥列表
func (s *sModelAgent) GetModelAgentKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetModelAgentKeys time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "status": 1, "model_agents": bson.M{"$in": []string{id}}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:           result.Id,
			AppId:        result.AppId,
			Corp:         result.Corp,
			Key:          result.Key,
			Type:         result.Type,
			Models:       result.Models,
			IsLimitQuota: result.IsLimitQuota,
			Quota:        result.Quota,
			IpWhitelist:  result.IpWhitelist,
			IpBlacklist:  result.IpBlacklist,
			Remark:       result.Remark,
			Status:       result.Status,
		})
	}

	return items, nil
}

// 挑选模型代理
func (s *sModelAgent) PickModelAgent(ctx context.Context, m *model.Model) (modelAgent *model.ModelAgent, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "PickModelAgent time: %d", gtime.TimestampMilli()-now)
	}()

	var modelAgents []*model.ModelAgent
	var roundRobin *util.RoundRobin

	modelAgentsValue := s.modelAgentsMap.Get(m.Id)
	roundRobinValue := s.modelAgentsRoundRobinMap.Get(m.Id)

	if modelAgentsValue != nil {
		modelAgents = modelAgentsValue.([]*model.ModelAgent)
	}

	if len(modelAgents) == 0 {

		modelAgents, err = s.GetCacheList(ctx, m.ModelAgents...)
		if err != nil {

			modelAgents, err = s.List(ctx, m.ModelAgents)
			if err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			if err = s.SaveCacheList(ctx, modelAgents); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
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
		s.modelAgentsRoundRobinMap.Set(m.Id, roundRobin)
	}

	return modelAgents[roundRobin.Index(len(modelAgents))], nil
}

// 移除模型代理
func (s *sModelAgent) RemoveModelAgent(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RemoveModelAgent time: %d", gtime.TimestampMilli()-now)
	}()

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

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RecordErrorModelAgent time: %d", gtime.TimestampMilli()-now)
	}()

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

// 挑选模型代理密钥
func (s *sModelAgent) PickModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent) (key *model.Key, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "PickModelAgentKey time: %d", gtime.TimestampMilli()-now)
	}()

	var keys []*model.Key
	var roundRobin *util.RoundRobin

	keysValue := s.modelAgentKeysMap.Get(modelAgent.Id)
	roundRobinValue := s.modelAgentKeysRoundRobinMap.Get(modelAgent.Id)

	if keysValue != nil {
		keys = keysValue.([]*model.Key)
	}

	if len(keys) == 0 {

		keys, err = s.GetCacheModelAgentKeys(ctx, modelAgent.Id)
		if err != nil {

			keys, err = s.GetModelAgentKeys(ctx, modelAgent.Id)
			if err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			if err = s.SaveCacheModelAgentKeys(ctx, modelAgent.Id, keys); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

		if len(keys) == 0 {
			return nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY
		}

		s.modelAgentKeysMap.Set(modelAgent.Id, keys)
	}

	if roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.modelAgentKeysRoundRobinMap.Set(modelAgent.Id, roundRobin)
	}

	return keys[roundRobin.Index(len(keys))], nil
}

// 移除模型代理密钥
func (s *sModelAgent) RemoveModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RemoveModelAgentKey time: %d", gtime.TimestampMilli()-now)
	}()

	keysValue := s.modelAgentKeysMap.Get(modelAgent.Id)
	if keysValue != nil {

		keys := keysValue.([]*model.Key)

		if len(keys) > 0 {

			newKeys := make([]*model.Key, 0)

			for _, k := range keys {
				if k.Id != key.Id {
					newKeys = append(newKeys, k)
				}
			}

			s.modelAgentKeysMap.Set(modelAgent.Id, newKeys)
		}

		if err := dao.Key.UpdateById(ctx, key.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 记录错误模型代理密钥
func (s *sModelAgent) RecordErrorModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RecordErrorModelAgentKey time: %d", gtime.TimestampMilli()-now)
	}()

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT_KEY, modelAgent.Id), key.Key, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	_, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT_KEY, modelAgent.Id), gtime.Now().EndOfDay().Time)
	if err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {
		s.RemoveModelAgentKey(ctx, modelAgent, key)
	}
}

// 保存模型代理列表到缓存
func (s *sModelAgent) SaveCacheList(ctx context.Context, modelAgents []*model.ModelAgent) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, modelAgent := range modelAgents {
		fields[modelAgent.Id] = modelAgent
		s.modelAgentCacheMap.Set(modelAgent.Id, modelAgent)
	}

	if len(fields) > 0 {
		_, err := redis.HSet(ctx, consts.API_MODEL_AGENTS_KEY, fields)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的模型代理列表
func (s *sModelAgent) GetCacheList(ctx context.Context, ids ...string) ([]*model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.ModelAgent, 0)

	for _, id := range ids {
		modelAgentCacheValue := s.modelAgentCacheMap.Get(id)
		if modelAgentCacheValue != nil {
			items = append(items, modelAgentCacheValue.(*model.ModelAgent))
		}
	}

	if len(items) == len(ids) {
		return items, nil
	}

	reply, err := redis.HMGet(ctx, consts.API_MODEL_AGENTS_KEY, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		if len(items) != 0 {
			return items, nil
		}
		return nil, errors.New("modelAgents is nil")
	}

	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.ModelAgent)
		err = gjson.Unmarshal([]byte(str), &result)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if s.modelAgentCacheMap.Get(result.Id) != nil {
			continue
		}

		if result.Status == 1 {
			items = append(items, result)
			s.modelAgentCacheMap.Set(result.Id, result)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("modelAgents is nil")
	}

	return items, nil
}

// 保存模型代理密钥列表到缓存
func (s *sModelAgent) SaveCacheModelAgentKeys(ctx context.Context, id string, keys []*model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent SaveCacheModelAgentKeys time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, key := range keys {
		fields[key.Id] = key
	}

	if len(fields) > 0 {

		_, err := redis.HSet(ctx, fmt.Sprintf(consts.API_MODEL_AGENT_KEYS_KEY, id), fields)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.modelAgentKeysCacheMap.Set(id, keys)
	}

	return nil
}

// 获取缓存中的模型代理密钥列表
func (s *sModelAgent) GetCacheModelAgentKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetCacheModelAgentKeys time: %d", gtime.TimestampMilli()-now)
	}()

	modelAgentKeysCacheValue := s.modelAgentKeysCacheMap.Get(id)
	if modelAgentKeysCacheValue != nil {
		return modelAgentKeysCacheValue.([]*model.Key), nil
	}

	reply, err := redis.HVals(ctx, fmt.Sprintf(consts.API_MODEL_AGENT_KEYS_KEY, id))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		return nil, errors.New("modelAgentKeys is nil")
	}

	items := make([]*model.Key, 0)
	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.Key)
		err = gjson.Unmarshal([]byte(str), &result)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if result.Status == 1 {
			items = append(items, result)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("modelAgentKeys is nil")
	}

	slices.SortFunc(items, func(k1, k2 *model.Key) int {
		return cmp.Compare(k1.Id, k2.Id)
	})

	s.modelAgentKeysCacheMap.Set(id, items)

	return items, nil
}

// 变更订阅
func (s *sModelAgent) Subscribe(ctx context.Context, msg string) error {

	modelAgent := new(entity.ModelAgent)
	err := gjson.Unmarshal([]byte(msg), &modelAgent)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}
	fmt.Println(gjson.MustEncodeString(modelAgent))

	return nil
}
