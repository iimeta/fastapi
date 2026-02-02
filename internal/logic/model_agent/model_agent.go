package model_agent

import (
	"context"
	"fmt"
	"slices"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/cache"
	"github.com/iimeta/fastapi/v2/utility/lb"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
)

type sModelAgent struct {
	modelAgentCache                 *cache.Cache // [模型代理ID]模型代理
	modelAgentKeysCache             *cache.Cache // [模型代理ID][]模型代理密钥列表
	modelAgentKeysRoundRobinCache   *cache.Cache // [模型代理ID]模型代理密钥下标索引
	modelAgentsCache                *cache.Cache // [模型ID][]模型代理列表
	modelAgentsRoundRobinCache      *cache.Cache // [模型ID]模型代理下标索引
	groupModelAgentsCache           *cache.Cache // [分组ID][]模型代理列表
	groupModelAgentsRoundRobinCache *cache.Cache // [分组ID]模型代理下标索引
}

func init() {
	service.RegisterModelAgent(New())
}

func New() service.IModelAgent {
	return &sModelAgent{
		modelAgentCache:                 cache.New(),
		modelAgentKeysCache:             cache.New(),
		modelAgentKeysRoundRobinCache:   cache.New(),
		modelAgentsCache:                cache.New(),
		modelAgentsRoundRobinCache:      cache.New(),
		groupModelAgentsCache:           cache.New(),
		groupModelAgentsRoundRobinCache: cache.New(),
	}
}

// 根据模型代理ID获取模型代理信息
func (s *sModelAgent) GetById(ctx context.Context, id string) (*model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetById time: %d", gtime.TimestampMilli()-now)
	}()

	modelAgent, err := dao.ModelAgent.FindById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.ModelAgent{
		Id:                   modelAgent.Id,
		ProviderId:           modelAgent.ProviderId,
		Name:                 modelAgent.Name,
		BaseUrl:              modelAgent.BaseUrl,
		Path:                 modelAgent.Path,
		Weight:               modelAgent.Weight,
		BillingMethods:       modelAgent.BillingMethods,
		Models:               modelAgent.Models,
		IsEnableModelReplace: modelAgent.IsEnableModelReplace,
		ReplaceModels:        modelAgent.ReplaceModels,
		TargetModels:         modelAgent.TargetModels,
		IsNeverDisable:       modelAgent.IsNeverDisable,
		LbStrategy:           modelAgent.LbStrategy,
		Status:               modelAgent.Status,
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
	}

	results, err := dao.ModelAgent.Find(ctx, filter, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.ModelAgent, 0)
	for _, result := range results {
		items = append(items, &model.ModelAgent{
			Id:                   result.Id,
			ProviderId:           result.ProviderId,
			Name:                 result.Name,
			BaseUrl:              result.BaseUrl,
			Path:                 result.Path,
			Weight:               result.Weight,
			BillingMethods:       result.BillingMethods,
			Models:               result.Models,
			IsEnableModelReplace: result.IsEnableModelReplace,
			ReplaceModels:        result.ReplaceModels,
			TargetModels:         result.TargetModels,
			IsNeverDisable:       result.IsNeverDisable,
			LbStrategy:           result.LbStrategy,
			Status:               result.Status,
		})
	}

	return items, nil
}

// 全部模型代理列表
func (s *sModelAgent) ListAll(ctx context.Context) ([]*model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent ListAll time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.ModelAgent.Find(ctx, filter, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.ModelAgent, 0)
	for _, result := range results {
		items = append(items, &model.ModelAgent{
			Id:                   result.Id,
			ProviderId:           result.ProviderId,
			Name:                 result.Name,
			BaseUrl:              result.BaseUrl,
			Path:                 result.Path,
			Weight:               result.Weight,
			BillingMethods:       result.BillingMethods,
			Models:               result.Models,
			IsEnableModelReplace: result.IsEnableModelReplace,
			ReplaceModels:        result.ReplaceModels,
			TargetModels:         result.TargetModels,
			IsNeverDisable:       result.IsNeverDisable,
			LbStrategy:           result.LbStrategy,
			Status:               result.Status,
		})
	}

	return items, nil
}

// 根据模型代理ID获取密钥列表
func (s *sModelAgent) GetKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetByModelId time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.Key.Find(ctx, bson.M{"model_agents": bson.M{"$in": []string{id}}}, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Key, 0)
	for _, result := range results {
		items = append(items, &model.Key{
			Id:             result.Id,
			ProviderId:     result.ProviderId,
			Key:            result.Key,
			Weight:         result.Weight,
			Models:         result.Models,
			ModelAgents:    result.ModelAgents,
			IsNeverDisable: result.IsNeverDisable,
			UsedQuota:      result.UsedQuota,
			Status:         result.Status,
		})
	}

	return items, nil
}

// 获取模型代理与密钥列表
func (s *sModelAgent) GetModelAgentsAndKeys(ctx context.Context) ([]*model.ModelAgent, map[string][]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetModelAgentsAndKeys time: %d", gtime.TimestampMilli()-now)
	}()

	modelAgents, err := s.ListAll(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return nil, nil, err
	}

	results, err := dao.Key.Find(ctx, bson.M{}, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, nil, err
	}

	modelAgentKeyMap := make(map[string][]*model.Key)
	for _, result := range results {

		key := &model.Key{
			Id:             result.Id,
			ProviderId:     result.ProviderId,
			Key:            result.Key,
			Weight:         result.Weight,
			Models:         result.Models,
			ModelAgents:    result.ModelAgents,
			IsNeverDisable: result.IsNeverDisable,
			UsedQuota:      result.UsedQuota,
			Status:         result.Status,
		}

		for _, modelAgentId := range result.ModelAgents {
			modelAgentKeyMap[modelAgentId] = append(modelAgentKeyMap[modelAgentId], key)
		}
	}

	return modelAgents, modelAgentKeyMap, nil
}

// 挑选模型代理
func (s *sModelAgent) Pick(ctx context.Context, m *model.Model) (int, *model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent Pick time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		modelAgents []*model.ModelAgent
		roundRobin  *lb.RoundRobin
		err         error
	)

	if modelAgentsValue := s.modelAgentsCache.GetVal(ctx, m.Id); modelAgentsValue != nil {
		modelAgents = modelAgentsValue.([]*model.ModelAgent)
	}

	if len(modelAgents) == 0 {

		modelAgents, err = s.GetCacheList(ctx, m.ModelAgents...)
		if err != nil || len(modelAgents) != len(m.ModelAgents) {

			if modelAgents, err = s.List(ctx, m.ModelAgents); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}

			if err = s.SaveCacheList(ctx, modelAgents); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}
		}

		if len(modelAgents) == 0 {
			return 0, nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT
		}

		if err = s.modelAgentsCache.Set(ctx, m.Id, modelAgents, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	modelAgentList := make([]*model.ModelAgent, 0)
	for _, modelAgent := range modelAgents {
		// 过滤被禁用的模型代理
		if modelAgent.Status == 1 {
			modelAgentList = append(modelAgentList, modelAgent)
		}
	}

	if len(modelAgentList) == 0 {
		return 0, nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT
	}

	filterModelAgentList := make([]*model.ModelAgent, 0)
	if len(modelAgentList) > 1 {
		errorModelAgents := service.Session().GetErrorModelAgents(ctx)
		if len(errorModelAgents) > 0 {
			for _, modelAgent := range modelAgentList {
				// 过滤错误的模型代理
				if !slices.Contains(errorModelAgents, modelAgent.Id) {
					filterModelAgentList = append(filterModelAgentList, modelAgent)
				}
			}
		} else {
			filterModelAgentList = modelAgentList
		}
	} else {
		filterModelAgentList = modelAgentList
	}

	if len(filterModelAgentList) == 0 {
		return 0, nil, errors.ERR_ALL_MODEL_AGENT
	}

	// 测试
	if modelAgentId, yes := service.Session().IsTest(ctx); yes {
		for _, modelAgent := range filterModelAgentList {
			if modelAgent.Id == modelAgentId {
				return len(filterModelAgentList), modelAgent, nil
			}
		}
	}

	// 负载策略-权重
	if m.LbStrategy == 2 {
		return len(filterModelAgentList), lb.NewModelAgentWeight(filterModelAgentList).PickModelAgent(), nil
	}

	if roundRobinValue := s.modelAgentsRoundRobinCache.GetVal(ctx, m.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*lb.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = lb.NewRoundRobin()
		if err = s.modelAgentsRoundRobinCache.Set(ctx, m.Id, roundRobin, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	return len(filterModelAgentList), filterModelAgentList[roundRobin.Index(len(filterModelAgentList))], nil
}

// 根据模型挑选分组模型代理
func (s *sModelAgent) PickGroup(ctx context.Context, m *model.Model, group *model.Group) (int, *model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent PickGroup time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		modelAgents []*model.ModelAgent
		roundRobin  *lb.RoundRobin
		err         error
	)

	if modelAgentsValue := s.groupModelAgentsCache.GetVal(ctx, group.Id); modelAgentsValue != nil {
		modelAgents = modelAgentsValue.([]*model.ModelAgent)
	}

	if len(modelAgents) != len(group.ModelAgents) {

		modelAgents, err = s.GetCacheList(ctx, group.ModelAgents...)
		if err != nil || len(modelAgents) != len(group.ModelAgents) {

			if modelAgents, err = s.List(ctx, group.ModelAgents); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}

			if err = s.SaveCacheList(ctx, modelAgents); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}
		}

		if len(modelAgents) == 0 {
			return 0, nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT
		}

		if err = s.groupModelAgentsCache.Set(ctx, group.Id, modelAgents, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	modelAgentList := make([]*model.ModelAgent, 0)
	for _, modelAgent := range modelAgents {
		// 过滤被禁用的模型代理
		if modelAgent.Status == 1 && slices.Contains(modelAgent.Models, m.Id) {
			modelAgentList = append(modelAgentList, modelAgent)
		}
	}

	if len(modelAgentList) == 0 {
		return 0, nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT
	}

	filterModelAgentList := make([]*model.ModelAgent, 0)
	if len(modelAgentList) > 1 {
		errorModelAgents := service.Session().GetErrorModelAgents(ctx)
		if len(errorModelAgents) > 0 {
			for _, modelAgent := range modelAgentList {
				// 过滤错误的模型代理
				if !slices.Contains(errorModelAgents, modelAgent.Id) {
					filterModelAgentList = append(filterModelAgentList, modelAgent)
				}
			}
		} else {
			filterModelAgentList = modelAgentList
		}
	} else {
		filterModelAgentList = modelAgentList
	}

	if len(filterModelAgentList) == 0 {
		return 0, nil, errors.ERR_ALL_MODEL_AGENT
	}

	// 测试
	if modelAgentId, yes := service.Session().IsTest(ctx); yes {
		for _, modelAgent := range filterModelAgentList {
			if modelAgent.Id == modelAgentId {
				return len(filterModelAgentList), modelAgent, nil
			}
		}
	}

	// 负载策略-权重
	if group.LbStrategy == 2 {
		return len(filterModelAgentList), lb.NewModelAgentWeight(filterModelAgentList).PickModelAgent(), nil
	}

	if roundRobinValue := s.groupModelAgentsRoundRobinCache.GetVal(ctx, group.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*lb.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = lb.NewRoundRobin()
		if err = s.groupModelAgentsRoundRobinCache.Set(ctx, group.Id, roundRobin, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	return len(filterModelAgentList), filterModelAgentList[roundRobin.Index(len(filterModelAgentList))], nil
}

// 移除模型代理
func (s *sModelAgent) Remove(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent Remove time: %d", gtime.TimestampMilli()-now)
	}()

	if modelAgentsValue := s.modelAgentsCache.GetVal(ctx, m.Id); modelAgentsValue != nil {

		if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

			newModelAgents := make([]*model.ModelAgent, 0)
			for _, agent := range modelAgents {
				if agent.Id != modelAgent.Id {
					newModelAgents = append(newModelAgents, agent)
				}
			}

			if err := s.modelAgentsCache.Set(ctx, m.Id, newModelAgents, 0); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := dao.ModelAgent.UpdateById(ctx, modelAgent.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 记录错误模型代理
func (s *sModelAgent) RecordError(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent RecordError time: %d", gtime.TimestampMilli()-now)
	}()

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT, m.Model), modelAgent.Id, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT, m.Model), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= config.Cfg.Base.ModelAgentErrDisable {
		s.Disabled(ctx, modelAgent, "Reached the maximum number of errors")
	}
}

// 禁用模型代理
func (s *sModelAgent) Disabled(ctx context.Context, modelAgent *model.ModelAgent, disabledReason string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent Disabled time: %d", gtime.TimestampMilli()-now)
	}()

	// 永不禁用
	if modelAgent.IsNeverDisable {
		return
	}

	modelAgent.Status = 2
	modelAgent.IsAutoDisabled = true
	modelAgent.AutoDisabledReason = disabledReason

	s.UpdateCache(ctx, nil, modelAgent)

	if err := dao.ModelAgent.UpdateById(ctx, modelAgent.Id, bson.M{
		"status":               2,
		"is_auto_disabled":     true,
		"auto_disabled_reason": disabledReason,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 挑选模型代理密钥
func (s *sModelAgent) PickKey(ctx context.Context, modelAgent *model.ModelAgent) (int, *model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent PickKey time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		keys       []*model.Key
		roundRobin *lb.RoundRobin
		err        error
	)

	if keysValue := s.modelAgentKeysCache.GetVal(ctx, modelAgent.Id); keysValue != nil {
		keys = keysValue.([]*model.Key)
	}

	if len(keys) == 0 {

		if keys, err = s.GetCacheKeys(ctx, modelAgent.Id); err != nil {
			if keys, err = s.GetKeys(ctx, modelAgent.Id); err != nil {
				logger.Error(ctx, err)
				return 0, nil, err
			}
		}

		if len(keys) == 0 {
			return 0, nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY
		}

		if err = s.SaveCacheKeys(ctx, modelAgent.Id, keys); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	keyList := make([]*model.Key, 0)
	for _, key := range keys {
		// 过滤被禁用的模型代理密钥
		if key.Status == 1 {
			keyList = append(keyList, key)
		}
	}

	if len(keyList) == 0 {
		return 0, nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY
	}

	filterKeyList := make([]*model.Key, 0)
	if len(keyList) > 1 {
		errorKeys := service.Session().GetErrorKeys(ctx)
		if len(errorKeys) > 0 {
			for _, key := range keyList {
				// 过滤错误的模型代理密钥
				if !slices.Contains(errorKeys, key.Id) {
					filterKeyList = append(filterKeyList, key)
				}
			}
		} else {
			filterKeyList = keyList
		}
	} else {
		filterKeyList = keyList
	}

	if len(filterKeyList) == 0 {
		return 0, nil, errors.ERR_ALL_MODEL_AGENT_KEY
	}

	// 负载策略-权重
	if modelAgent.LbStrategy == 2 {
		return len(filterKeyList), lb.NewKeyWeight(filterKeyList).PickKey(), nil
	}

	if roundRobinValue := s.modelAgentKeysRoundRobinCache.GetVal(ctx, modelAgent.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*lb.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = lb.NewRoundRobin()
		if err = s.modelAgentKeysRoundRobinCache.Set(ctx, modelAgent.Id, roundRobin, 0); err != nil {
			logger.Error(ctx, err)
			return 0, nil, err
		}
	}

	return len(filterKeyList), filterKeyList[roundRobin.Index(len(filterKeyList))], nil
}

// 移除模型代理密钥
func (s *sModelAgent) RemoveKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent RemoveKey time: %d", gtime.TimestampMilli()-now)
	}()

	if keysValue := s.modelAgentKeysCache.GetVal(ctx, modelAgent.Id); keysValue != nil {

		keys := keysValue.([]*model.Key)

		if len(keys) > 0 {

			newKeys := make([]*model.Key, 0)
			for _, k := range keys {
				if k.Id != key.Id {
					newKeys = append(newKeys, k)
				}
			}

			if err := s.modelAgentKeysCache.Set(ctx, modelAgent.Id, newKeys, 0); err != nil {
				logger.Error(ctx, err)
			}
		}

		if err := dao.Key.UpdateById(ctx, key.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 记录错误模型代理密钥
func (s *sModelAgent) RecordErrorKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent RecordErrorKey time: %d", gtime.TimestampMilli()-now)
	}()

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT_KEY, modelAgent.Id), key.Key, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT_KEY, modelAgent.Id), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= config.Cfg.Base.ModelAgentKeyErrDisable {
		s.DisabledKey(ctx, key, "Reached the maximum number of errors")
	}
}

// 禁用模型代理密钥
func (s *sModelAgent) DisabledKey(ctx context.Context, key *model.Key, disabledReason string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent DisabledKey time: %d", gtime.TimestampMilli()-now)
	}()

	// 永不禁用
	if key.IsNeverDisable {
		return
	}

	s.UpdateCacheKey(ctx, nil, &entity.Key{
		Id:                 key.Id,
		ProviderId:         key.ProviderId,
		Key:                key.Key,
		Weight:             key.Weight,
		Models:             key.Models,
		ModelAgents:        key.ModelAgents,
		IsNeverDisable:     key.IsNeverDisable,
		UsedQuota:          key.UsedQuota,
		Status:             2,
		IsAutoDisabled:     true,
		AutoDisabledReason: disabledReason,
	})

	if err := dao.Key.UpdateById(ctx, key.Id, bson.M{
		"status":               2,
		"is_auto_disabled":     true,
		"auto_disabled_reason": disabledReason,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存模型代理列表到缓存
func (s *sModelAgent) SaveCacheList(ctx context.Context, modelAgents []*model.ModelAgent) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	for _, modelAgent := range modelAgents {
		if err := s.modelAgentCache.Set(ctx, modelAgent.Id, modelAgent, 0); err != nil {
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
		if modelAgentCacheValue := s.modelAgentCache.GetVal(ctx, id); modelAgentCacheValue != nil {
			items = append(items, modelAgentCacheValue.(*model.ModelAgent))
		}
	}

	if len(items) == 0 {
		return nil, errors.New("modelAgentsCache is nil")
	}

	return items, nil
}

// 添加模型代理到缓存列表中
func (s *sModelAgent) AddCache(ctx context.Context, newData *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent AddCache time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheList(ctx, []*model.ModelAgent{{
		Id:                   newData.Id,
		ProviderId:           newData.ProviderId,
		Name:                 newData.Name,
		BaseUrl:              newData.BaseUrl,
		Path:                 newData.Path,
		Weight:               newData.Weight,
		BillingMethods:       newData.BillingMethods,
		Models:               newData.Models,
		IsEnableModelReplace: newData.IsEnableModelReplace,
		ReplaceModels:        newData.ReplaceModels,
		TargetModels:         newData.TargetModels,
		IsNeverDisable:       newData.IsNeverDisable,
		LbStrategy:           newData.LbStrategy,
		Status:               newData.Status,
	}}); err != nil {
		logger.Error(ctx, err)
	}

	// 将新增的模型代理添加到对应模型的模型代理列表缓存中(有点绕, 哈哈哈...)
	for _, id := range newData.Models {
		if modelAgentsValue := s.modelAgentsCache.GetVal(ctx, id); modelAgentsValue != nil {
			if err := s.modelAgentsCache.Set(ctx, id, append(modelAgentsValue.([]*model.ModelAgent), newData), 0); err != nil {
				logger.Error(ctx, err)
			}
		}
	}
}

// 更新缓存中的模型代理列表
func (s *sModelAgent) UpdateCache(ctx context.Context, oldData *entity.ModelAgent, newData *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent UpdateCache time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheList(ctx, []*model.ModelAgent{{
		Id:                   newData.Id,
		ProviderId:           newData.ProviderId,
		Name:                 newData.Name,
		BaseUrl:              newData.BaseUrl,
		Path:                 newData.Path,
		Weight:               newData.Weight,
		BillingMethods:       newData.BillingMethods,
		Models:               newData.Models,
		IsEnableModelReplace: newData.IsEnableModelReplace,
		ReplaceModels:        newData.ReplaceModels,
		TargetModels:         newData.TargetModels,
		IsNeverDisable:       newData.IsNeverDisable,
		LbStrategy:           newData.LbStrategy,
		Status:               newData.Status,
		IsAutoDisabled:       newData.IsAutoDisabled,
		AutoDisabledReason:   newData.AutoDisabledReason,
	}}); err != nil {
		logger.Error(ctx, err)
	}

	// 用于处理oldData时判断作用
	newModelAgentModelMap := make(map[string]string)

	// 将变更后的模型代理替换(或添加)到对应模型的模型代理列表缓存中(有点绕, 哈哈哈...)
	for _, id := range newData.Models {

		newModelAgentModelMap[id] = id

		if modelAgentsValue := s.modelAgentsCache.GetVal(ctx, id); modelAgentsValue != nil {

			modelAgents := modelAgentsValue.([]*model.ModelAgent)
			newModelAgents := make([]*model.ModelAgent, 0)
			// 用于处理新添加了模型代理时判断作用
			modelAgentMap := make(map[string]*model.ModelAgent)

			for _, agent := range modelAgents {

				if agent.Id != newData.Id {
					newModelAgents = append(newModelAgents, agent)
					modelAgentMap[agent.Id] = agent
				} else {
					newModelAgents = append(newModelAgents, newData)
					modelAgentMap[newData.Id] = newData
				}
			}

			if modelAgentMap[newData.Id] == nil {
				newModelAgents = append(newModelAgents, newData)
			}

			if err := s.modelAgentsCache.Set(ctx, id, newModelAgents, 0); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	// 将变更后被移除模型的模型代理移除
	if oldData != nil {

		for _, id := range oldData.Models {

			if newModelAgentModelMap[id] == "" {

				if modelAgentsValue := s.modelAgentsCache.GetVal(ctx, id); modelAgentsValue != nil {

					if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

						newModelAgents := make([]*model.ModelAgent, 0)
						for _, agent := range modelAgents {
							if agent.Id != oldData.Id {
								newModelAgents = append(newModelAgents, agent)
							}
						}

						if err := s.modelAgentsCache.Set(ctx, id, newModelAgents, 0); err != nil {
							logger.Error(ctx, err)
						}
					}
				}
			}
		}
	}

	if groups, err := s.groupModelAgentsCache.Keys(ctx); err == nil {
		for _, id := range groups {

			if modelAgentsValue := s.groupModelAgentsCache.GetVal(ctx, id); modelAgentsValue != nil {

				if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

					newModelAgents := make([]*model.ModelAgent, 0)
					for _, agent := range modelAgents {
						if agent.Id != newData.Id {
							newModelAgents = append(newModelAgents, agent)
						} else {
							newModelAgents = append(newModelAgents, newData)
						}
					}

					if err := s.groupModelAgentsCache.Set(ctx, id, newModelAgents, 0); err != nil {
						logger.Error(ctx, err)
					}
				}
			}
		}
	} else {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的模型代理
func (s *sModelAgent) RemoveCache(ctx context.Context, modelAgent *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent RemoveCache time: %d", gtime.TimestampMilli()-now)
	}()

	for _, id := range modelAgent.Models {

		if modelAgentsValue := s.modelAgentsCache.GetVal(ctx, id); modelAgentsValue != nil {

			if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

				newModelAgents := make([]*model.ModelAgent, 0)
				for _, agent := range modelAgents {
					if agent.Id != modelAgent.Id {
						newModelAgents = append(newModelAgents, agent)
					}
				}

				if err := s.modelAgentsCache.Set(ctx, id, newModelAgents, 0); err != nil {
					logger.Error(ctx, err)
				}
			}
		}
	}

	if groups, err := s.groupModelAgentsCache.Keys(ctx); err == nil {
		for _, id := range groups {

			if modelAgentsValue := s.groupModelAgentsCache.GetVal(ctx, id); modelAgentsValue != nil {

				if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

					newModelAgents := make([]*model.ModelAgent, 0)
					for _, agent := range modelAgents {
						if agent.Id != modelAgent.Id {
							newModelAgents = append(newModelAgents, agent)
						}
					}

					if err := s.groupModelAgentsCache.Set(ctx, id, newModelAgents, 0); err != nil {
						logger.Error(ctx, err)
					}
				}
			}
		}
	} else {
		logger.Error(ctx, err)
	}

	if _, err := s.modelAgentCache.Remove(ctx, modelAgent.Id); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存模型代理密钥列表到缓存
func (s *sModelAgent) SaveCacheKeys(ctx context.Context, id string, keys []*model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent SaveCacheKeys keys: %d, time: %d", len(keys), gtime.TimestampMilli()-now)
	}()

	if err := s.modelAgentKeysCache.Set(ctx, id, keys, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的模型代理密钥列表
func (s *sModelAgent) GetCacheKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetCacheKeys time: %d", gtime.TimestampMilli()-now)
	}()

	if modelAgentKeysCacheValue := s.modelAgentKeysCache.GetVal(ctx, id); modelAgentKeysCacheValue != nil {
		return modelAgentKeysCacheValue.([]*model.Key), nil
	}

	return nil, errors.New("modelAgentKeys is nil")
}

// 新增模型代理密钥到缓存列表中
func (s *sModelAgent) CreateCacheKey(ctx context.Context, key *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent CreateCacheKey time: %d", gtime.TimestampMilli()-now)
	}()

	k := &model.Key{
		Id:             key.Id,
		ProviderId:     key.ProviderId,
		Key:            key.Key,
		Weight:         key.Weight,
		Models:         key.Models,
		ModelAgents:    key.ModelAgents,
		IsNeverDisable: key.IsNeverDisable,
		UsedQuota:      key.UsedQuota,
		Status:         key.Status,
	}

	for _, id := range k.ModelAgents {

		if modelAgentKeysValue := s.modelAgentKeysCache.GetVal(ctx, id); modelAgentKeysValue != nil {
			if err := s.SaveCacheKeys(ctx, id, append(modelAgentKeysValue.([]*model.Key), k)); err != nil {
				logger.Error(ctx, err)
			}
		} else {
			if err := s.SaveCacheKeys(ctx, id, []*model.Key{k}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}
}

// 更新缓存中的模型代理密钥
func (s *sModelAgent) UpdateCacheKey(ctx context.Context, oldData *entity.Key, newData *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent UpdateCacheKey time: %d", gtime.TimestampMilli()-now)
	}()

	key := &model.Key{
		Id:                 newData.Id,
		ProviderId:         newData.ProviderId,
		Key:                newData.Key,
		Weight:             newData.Weight,
		Models:             newData.Models,
		ModelAgents:        newData.ModelAgents,
		IsNeverDisable:     newData.IsNeverDisable,
		UsedQuota:          newData.UsedQuota,
		Status:             newData.Status,
		IsAutoDisabled:     newData.IsAutoDisabled,
		AutoDisabledReason: newData.AutoDisabledReason,
	}

	// 用于处理oldData时判断作用
	newModelAgentMap := make(map[string]string)

	for _, id := range newData.ModelAgents {

		newModelAgentMap[id] = id

		modelAgentKeys, err := s.GetCacheKeys(ctx, id)
		if err != nil {
			logger.Error(ctx, err)
		}

		if len(modelAgentKeys) == 0 {
			if modelAgentKeys, err = s.GetKeys(ctx, id); err != nil {
				logger.Error(ctx, err)
				continue
			}
		}

		newModelAgentKeys := make([]*model.Key, 0)
		// 用于处理新添加了模型时判断作用
		modelAgentKeyMap := make(map[string]*model.Key)

		for _, k := range modelAgentKeys {

			if k.Id != newData.Id {
				newModelAgentKeys = append(newModelAgentKeys, k)
				modelAgentKeyMap[k.Id] = k
			} else {
				newModelAgentKeys = append(newModelAgentKeys, key)
				modelAgentKeyMap[newData.Id] = key
			}
		}

		if modelAgentKeyMap[newData.Id] == nil {
			newModelAgentKeys = append(newModelAgentKeys, key)
		}

		if err = s.SaveCacheKeys(ctx, id, newModelAgentKeys); err != nil {
			logger.Error(ctx, err)
		}
	}

	// 将变更后被移除模型的模型密钥移除
	if oldData != nil {

		for _, id := range oldData.ModelAgents {

			if newModelAgentMap[id] == "" {

				modelAgentKeys, err := s.GetCacheKeys(ctx, id)
				if err != nil {
					logger.Error(ctx, err)
					if modelAgentKeys, err = s.GetKeys(ctx, id); err != nil {
						logger.Error(ctx, err)
						continue
					}
				} else if len(modelAgentKeys) == 0 {
					if modelAgentKeys, err = s.GetKeys(ctx, id); err != nil {
						logger.Error(ctx, err)
						continue
					}
				}

				if len(modelAgentKeys) > 0 && s.modelAgentKeysCache.ContainsKey(ctx, id) {

					newKeys := make([]*model.Key, 0)
					for _, k := range modelAgentKeys {
						if k.Id != oldData.Id {
							newKeys = append(newKeys, k)
						}
					}

					if err = s.modelAgentKeysCache.Set(ctx, id, newKeys, 0); err != nil {
						logger.Error(ctx, err)
					}
				}
			}
		}
	}
}

// 移除缓存中的模型代理密钥
func (s *sModelAgent) RemoveCacheKey(ctx context.Context, key *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent RemoveCacheKey time: %d", gtime.TimestampMilli()-now)
	}()

	for _, id := range key.ModelAgents {

		if modelAgentKeysValue := s.modelAgentKeysCache.GetVal(ctx, id); modelAgentKeysValue != nil {

			if modelAgentKeys := modelAgentKeysValue.([]*model.Key); len(modelAgentKeys) > 0 {

				newModelAgentKeys := make([]*model.Key, 0)
				for _, k := range modelAgentKeys {
					if k.Id != key.Id {
						newModelAgentKeys = append(newModelAgentKeys, k)
					}
				}

				if err := s.modelAgentKeysCache.Set(ctx, id, newModelAgentKeys, 0); err != nil {
					logger.Error(ctx, err)
				}
			}
		}
	}
}

// 获取缓存中的模型代理信息
func (s *sModelAgent) GetCache(ctx context.Context, id string) (*model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetCache time: %d", gtime.TimestampMilli()-now)
	}()

	modelAgents, err := s.GetCacheList(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(modelAgents) == 0 {
		return nil, errors.New("modelAgent is nil")
	}

	return modelAgents[0], nil
}

// 根据模型代理ID获取模型代理信息并保存到缓存
func (s *sModelAgent) GetAndSaveCache(ctx context.Context, id string) (*model.ModelAgent, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetAndSaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	modelAgent, err := s.GetById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if modelAgent != nil {
		if err = s.SaveCache(ctx, modelAgent); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	return modelAgent, nil
}

// 保存模型代理到缓存
func (s *sModelAgent) SaveCache(ctx context.Context, modelAgent *model.ModelAgent) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	return s.SaveCacheList(ctx, []*model.ModelAgent{modelAgent})
}

// 获取后备模型代理
func (s *sModelAgent) GetFallback(ctx context.Context, model *model.Model) (fallbackModelAgent *model.ModelAgent, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent GetFallback time: %d", gtime.TimestampMilli()-now)
	}()

	if fallbackModelAgent, err = s.GetCache(ctx, model.FallbackConfig.ModelAgent); err != nil || fallbackModelAgent == nil {
		if fallbackModelAgent, err = s.GetAndSaveCache(ctx, model.FallbackConfig.ModelAgent); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if fallbackModelAgent.Status != 1 {
		err = errors.ERR_MODEL_AGENT_HAS_BEEN_DISABLED
		logger.Error(ctx, err)
		return nil, err
	}

	return fallbackModelAgent, nil
}

// 保存分组模型代理列表到缓存
func (s *sModelAgent) SaveGroupCache(ctx context.Context, group *model.Group) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent SaveGroupCache time: %d", gtime.TimestampMilli()-now)
	}()

	if len(group.ModelAgents) == 0 {
		return nil
	}

	modelAgents, err := s.GetCacheList(ctx, group.ModelAgents...)
	if err != nil || len(modelAgents) != len(group.ModelAgents) {

		if modelAgents, err = s.List(ctx, group.ModelAgents); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = s.SaveCacheList(ctx, modelAgents); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if err = s.groupModelAgentsCache.Set(ctx, group.Id, modelAgents, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 变更订阅
func (s *sModelAgent) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModelAgent Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sModelAgent Subscribe: %s", gjson.MustEncodeString(message))

	var modelAgent *model.ModelAgent
	switch message.Action {
	case consts.ACTION_CREATE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.AddCache(ctx, modelAgent)

	case consts.ACTION_UPDATE:

		var oldData *entity.ModelAgent
		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &oldData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCache(ctx, oldData, modelAgent)

	case consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCache(ctx, nil, modelAgent)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCache(ctx, modelAgent)
	}

	return nil
}
