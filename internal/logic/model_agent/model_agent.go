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
	modelAgentsMap              *gmap.StrAnyMap // map[模型ID][]模型代理列表
	modelAgentsRoundRobinMap    *gmap.StrAnyMap // 模型ID->模型代理下标索引
	modelAgentKeysMap           *gmap.StrAnyMap // map[模型代理ID]模型代理密钥列表
	modelAgentKeysRoundRobinMap *gmap.StrAnyMap // 模型代理ID->模型代理密钥下标索引
	modelAgentCacheMap          *gmap.StrAnyMap // map[模型代理ID]模型代理
	modelAgentKeysCacheMap      *gmap.StrAnyMap // map[模型代理ID]模型代理密钥列表
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

	modelAgent, err := dao.ModelAgent.FindById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	modelList, err := dao.Model.Find(ctx, bson.M{"model_agents": bson.M{"$in": []string{id}}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	models := make([]string, 0)
	modelNames := make([]string, 0)

	for _, model := range modelList {
		models = append(models, model.Id)
		modelNames = append(modelNames, model.Name)
	}

	return &model.ModelAgent{
		Id:         modelAgent.Id,
		Name:       modelAgent.Name,
		BaseUrl:    modelAgent.BaseUrl,
		Path:       modelAgent.Path,
		Weight:     modelAgent.Weight,
		Models:     models,
		ModelNames: modelNames,
		Status:     modelAgent.Status,
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

	modelList, err := dao.Model.Find(ctx, bson.M{"model_agents": bson.M{"$in": ids}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	modelMap := make(map[string][]string)
	modelNameMap := make(map[string][]string)

	for _, model := range modelList {
		for _, id := range model.ModelAgents {
			modelMap[id] = append(modelMap[id], model.Id)
			modelNameMap[id] = append(modelNameMap[id], model.Name)
		}
	}

	items := make([]*model.ModelAgent, 0)
	for _, result := range results {
		items = append(items, &model.ModelAgent{
			Id:         result.Id,
			Name:       result.Name,
			BaseUrl:    result.BaseUrl,
			Path:       result.Path,
			Weight:     result.Weight,
			Models:     modelMap[result.Id],
			ModelNames: modelNameMap[result.Id],
			Status:     result.Status,
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
			UserId:       result.UserId,
			AppId:        result.AppId,
			Corp:         result.Corp,
			Key:          result.Key,
			Type:         result.Type,
			Models:       result.Models,
			ModelAgents:  result.ModelAgents,
			IsLimitQuota: result.IsLimitQuota,
			Quota:        result.Quota,
			RPM:          result.RPM,
			RPD:          result.RPD,
			IpWhitelist:  result.IpWhitelist,
			IpBlacklist:  result.IpBlacklist,
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

	if modelAgentsValue := s.modelAgentsMap.Get(m.Id); modelAgentsValue != nil {
		modelAgents = modelAgentsValue.([]*model.ModelAgent)
	}

	if len(modelAgents) == 0 {

		modelAgents, err = s.GetCacheList(ctx, m.ModelAgents...)
		if err != nil || len(modelAgents) != len(m.ModelAgents) {

			if modelAgents, err = s.List(ctx, m.ModelAgents); err != nil {
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

	modelAgentList := make([]*model.ModelAgent, 0)
	for _, modelAgent := range modelAgents {
		// 过滤被禁用的模型代理
		if modelAgent.Status == 1 {
			modelAgentList = append(modelAgentList, modelAgent)
		}
	}

	if len(modelAgentList) == 0 {
		return nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT
	}

	if roundRobinValue := s.modelAgentsRoundRobinMap.Get(m.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.modelAgentsRoundRobinMap.Set(m.Id, roundRobin)
	}

	return modelAgentList[roundRobin.Index(len(modelAgentList))], nil
}

// 移除模型代理
func (s *sModelAgent) RemoveModelAgent(ctx context.Context, m *model.Model, modelAgent *model.ModelAgent) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RemoveModelAgent time: %d", gtime.TimestampMilli()-now)
	}()

	if modelAgentsValue := s.modelAgentsMap.Get(m.Id); modelAgentsValue != nil {

		if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

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

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT, m.Model), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {

		modelAgent.Status = 2
		s.UpdateCacheModelAgent(ctx, nil, modelAgent)

		if err = dao.ModelAgent.UpdateById(ctx, modelAgent.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
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

	if keysValue := s.modelAgentKeysMap.Get(modelAgent.Id); keysValue != nil {
		keys = keysValue.([]*model.Key)
	}

	if len(keys) == 0 {

		if keys, err = s.GetCacheModelAgentKeys(ctx, modelAgent.Id); err != nil {

			if keys, err = s.GetModelAgentKeys(ctx, modelAgent.Id); err != nil {
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

	keyList := make([]*model.Key, 0)
	for _, key := range keys {
		// 过滤被禁用的模型代理密钥
		if key.Status == 1 {
			keyList = append(keyList, key)
		}
	}

	if len(keyList) == 0 {
		return nil, errors.ERR_NO_AVAILABLE_MODEL_AGENT_KEY
	}

	if roundRobinValue := s.modelAgentKeysRoundRobinMap.Get(modelAgent.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.modelAgentKeysRoundRobinMap.Set(modelAgent.Id, roundRobin)
	}

	return keyList[roundRobin.Index(len(keyList))], nil
}

// 移除模型代理密钥
func (s *sModelAgent) RemoveModelAgentKey(ctx context.Context, modelAgent *model.ModelAgent, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RemoveModelAgentKey time: %d", gtime.TimestampMilli()-now)
	}()

	if keysValue := s.modelAgentKeysMap.Get(modelAgent.Id); keysValue != nil {

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

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_AGENT_KEY, modelAgent.Id), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {

		s.UpdateCacheModelAgentKey(ctx, nil, &entity.Key{
			Id:           key.Id,
			UserId:       key.UserId,
			AppId:        key.AppId,
			Corp:         key.Corp,
			Key:          key.Key,
			Type:         key.Type,
			Models:       key.Models,
			ModelAgents:  key.ModelAgents,
			IsLimitQuota: key.IsLimitQuota,
			Quota:        key.Quota,
			RPM:          key.RPM,
			RPD:          key.RPD,
			IpWhitelist:  key.IpWhitelist,
			IpBlacklist:  key.IpBlacklist,
			Status:       2,
		})

		if err = dao.Key.UpdateById(ctx, key.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
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
		if _, err := redis.HSet(ctx, consts.API_MODEL_AGENTS_KEY, fields); err != nil {
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
		if modelAgentCacheValue := s.modelAgentCacheMap.Get(id); modelAgentCacheValue != nil {
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
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
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

// 新增模型代理到缓存列表中
func (s *sModelAgent) CreateCacheModelAgent(ctx context.Context, newData *model.ModelAgent) {

	if err := s.SaveCacheList(ctx, []*model.ModelAgent{{
		Id:      newData.Id,
		Name:    newData.Name,
		BaseUrl: newData.BaseUrl,
		Path:    newData.Path,
		Weight:  newData.Weight,
		Models:  newData.Models,
		Status:  newData.Status,
	}}); err != nil {
		logger.Error(ctx, err)
	}

	// 将新增的模型代理添加到对应模型的模型代理列表缓存中(有点绕, 哈哈哈...)
	for _, id := range newData.Models {
		if modelAgentsValue := s.modelAgentsMap.Get(id); modelAgentsValue != nil {
			s.modelAgentsMap.Set(id, append(modelAgentsValue.([]*model.ModelAgent), newData))
		}
	}
}

// 更新缓存中的模型代理列表
func (s *sModelAgent) UpdateCacheModelAgent(ctx context.Context, oldData *model.ModelAgent, newData *model.ModelAgent) {

	if err := s.SaveCacheList(ctx, []*model.ModelAgent{{
		Id:      newData.Id,
		Name:    newData.Name,
		BaseUrl: newData.BaseUrl,
		Path:    newData.Path,
		Weight:  newData.Weight,
		Models:  newData.Models,
		Status:  newData.Status,
	}}); err != nil {
		logger.Error(ctx, err)
	}

	// 用于处理oldData时判断作用
	newModelAgentModelMap := make(map[string]string)

	// 将变更后的模型代理替换(或添加)到对应模型的模型代理列表缓存中(有点绕, 哈哈哈...)
	for _, id := range newData.Models {

		newModelAgentModelMap[id] = id

		if modelAgentsValue := s.modelAgentsMap.Get(id); modelAgentsValue != nil {

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

			s.modelAgentsMap.Set(id, newModelAgents)
		}
	}

	// 将变更后被移除模型的模型代理移除
	if oldData != nil {
		for _, id := range oldData.Models {

			if newModelAgentModelMap[id] == "" {

				if modelAgentsValue := s.modelAgentsMap.Get(id); modelAgentsValue != nil {

					if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

						newModelAgents := make([]*model.ModelAgent, 0)
						for _, agent := range modelAgents {
							if agent.Id != oldData.Id {
								newModelAgents = append(newModelAgents, agent)
							}
						}

						s.modelAgentsMap.Set(id, newModelAgents)
					}
				}
			}
		}
	}
}

// 移除缓存中的模型代理
func (s *sModelAgent) RemoveCacheModelAgent(ctx context.Context, modelAgent *model.ModelAgent) {

	for _, id := range modelAgent.Models {

		if modelAgentsValue := s.modelAgentsMap.Get(id); modelAgentsValue != nil {

			if modelAgents := modelAgentsValue.([]*model.ModelAgent); len(modelAgents) > 0 {

				newModelAgents := make([]*model.ModelAgent, 0)
				for _, agent := range modelAgents {
					if agent.Id != modelAgent.Id {
						newModelAgents = append(newModelAgents, agent)
					}
				}

				s.modelAgentsMap.Set(id, newModelAgents)
			}
		}
	}

	if _, err := redis.HDel(ctx, consts.API_MODEL_AGENTS_KEY, modelAgent.Id); err != nil {
		logger.Error(ctx, err)
	}

	s.modelAgentCacheMap.Remove(modelAgent.Id)
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

		if _, err := redis.HSet(ctx, fmt.Sprintf(consts.API_MODEL_AGENT_KEYS_KEY, id), fields); err != nil {
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

	if modelAgentKeysCacheValue := s.modelAgentKeysCacheMap.Get(id); modelAgentKeysCacheValue != nil {
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
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
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

// 新增模型代理密钥到缓存列表中
func (s *sModelAgent) CreateCacheModelAgentKey(ctx context.Context, key *entity.Key) {

	k := &model.Key{
		Id:           key.Id,
		UserId:       key.UserId,
		AppId:        key.AppId,
		Corp:         key.Corp,
		Key:          key.Key,
		Type:         key.Type,
		Models:       key.Models,
		ModelAgents:  key.ModelAgents,
		IsLimitQuota: key.IsLimitQuota,
		Quota:        key.Quota,
		RPM:          key.RPM,
		RPD:          key.RPD,
		IpWhitelist:  key.IpWhitelist,
		IpBlacklist:  key.IpBlacklist,
		Status:       key.Status,
	}

	for _, id := range k.ModelAgents {

		if modelAgentKeysValue := s.modelAgentKeysCacheMap.Get(id); modelAgentKeysValue != nil {
			if err := s.SaveCacheModelAgentKeys(ctx, id, append(modelAgentKeysValue.([]*model.Key), k)); err != nil {
				logger.Error(ctx, err)
			}
		} else {
			if err := s.SaveCacheModelAgentKeys(ctx, id, []*model.Key{k}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if modelAgentKeysValue := s.modelAgentKeysMap.Get(id); modelAgentKeysValue != nil {
			s.modelAgentKeysMap.Set(id, append(modelAgentKeysValue.([]*model.Key), k))
		}
	}
}

// 更新缓存中的模型代理密钥
func (s *sModelAgent) UpdateCacheModelAgentKey(ctx context.Context, oldData *entity.Key, newData *entity.Key) {

	key := &model.Key{
		Id:           newData.Id,
		UserId:       newData.UserId,
		AppId:        newData.AppId,
		Corp:         newData.Corp,
		Key:          newData.Key,
		Type:         newData.Type,
		Models:       newData.Models,
		ModelAgents:  newData.ModelAgents,
		IsLimitQuota: newData.IsLimitQuota,
		Quota:        newData.Quota,
		RPM:          newData.RPM,
		RPD:          newData.RPD,
		IpWhitelist:  newData.IpWhitelist,
		IpBlacklist:  newData.IpBlacklist,
		Status:       newData.Status,
	}

	// 用于处理oldData时判断作用
	newModelAgentMap := make(map[string]string)

	for _, id := range newData.ModelAgents {

		newModelAgentMap[id] = id

		if modelAgentKeysValue := s.modelAgentKeysMap.Get(id); modelAgentKeysValue != nil {

			modelAgentKeys := modelAgentKeysValue.([]*model.Key)
			newModelAgentKeys := make([]*model.Key, 0)
			// 用于处理新添加了模型时判断作用
			modelAgentKeyMap := make(map[string]*model.Key)

			for _, k := range modelAgentKeys {

				if k.Id != newData.Id {
					newModelAgentKeys = append(newModelAgentKeys, k)
					modelAgentKeyMap[key.Id] = k
				} else {
					newModelAgentKeys = append(newModelAgentKeys, key)
					modelAgentKeyMap[newData.Id] = key
				}
			}

			if modelAgentKeyMap[newData.Id] == nil {
				newModelAgentKeys = append(newModelAgentKeys, key)
			}

			if err := s.SaveCacheModelAgentKeys(ctx, id, newModelAgentKeys); err != nil {
				logger.Error(ctx, err)
			}

			s.modelAgentKeysMap.Set(id, newModelAgentKeys)
		}
	}

	// 将变更后被移除模型的模型密钥移除
	if oldData != nil {
		for _, id := range oldData.ModelAgents {

			if newModelAgentMap[id] == "" {

				if keysValue := s.modelAgentKeysMap.Get(id); keysValue != nil {

					if keys := keysValue.([]*model.Key); len(keys) > 0 {

						newKeys := make([]*model.Key, 0)
						for _, k := range keys {

							if k.Id != oldData.Id {
								newKeys = append(newKeys, k)
							} else {
								if _, err := redis.HDel(ctx, fmt.Sprintf(consts.API_MODEL_AGENT_KEYS_KEY, id), oldData.Id); err != nil {
									logger.Error(ctx, err)
								}
							}
						}

						s.modelAgentKeysMap.Set(id, newKeys)
					}
				}
			}
		}
	}
}

// 移除缓存中的模型代理密钥
func (s *sModelAgent) RemoveCacheModelAgentKey(ctx context.Context, key *entity.Key) {

	for _, id := range key.ModelAgents {

		if _, err := redis.HDel(ctx, fmt.Sprintf(consts.API_MODEL_AGENT_KEYS_KEY, id), key.Id); err != nil {
			logger.Error(ctx, err)
		}

		if modelAgentKeysValue := s.modelAgentKeysCacheMap.Get(id); modelAgentKeysValue != nil {

			if modelAgentKeys := modelAgentKeysValue.([]*model.Key); len(modelAgentKeys) > 0 {

				newModelAgentKeys := make([]*model.Key, 0)
				for _, k := range modelAgentKeys {
					if k.Id != key.Id {
						newModelAgentKeys = append(newModelAgentKeys, k)
					}
				}

				s.modelAgentKeysCacheMap.Set(id, newModelAgentKeys)

				if s.modelAgentKeysMap.Get(id) != nil {
					s.modelAgentKeysMap.Set(id, newModelAgentKeys)
				}
			}
		}
	}
}

// 变更订阅
func (s *sModelAgent) Subscribe(ctx context.Context, msg string) error {

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

		s.CreateCacheModelAgent(ctx, modelAgent)

	case consts.ACTION_UPDATE:

		var oldData *model.ModelAgent
		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &oldData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheModelAgent(ctx, oldData, modelAgent)

	case consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheModelAgent(ctx, nil, modelAgent)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &modelAgent); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheModelAgent(ctx, modelAgent)
	}

	return nil
}
