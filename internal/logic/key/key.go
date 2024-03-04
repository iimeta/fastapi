package key

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

type sKey struct {
	keysMap           *gmap.StrAnyMap // map[模型ID][]密钥列表
	roundRobinMap     *gmap.StrAnyMap // 模型ID->密钥下标索引
	keyCacheMap       *gmap.StrAnyMap // map[密钥ID]密钥
	modelKeysCacheMap *gmap.StrAnyMap // map[模型ID][]密钥列表
}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{
		keysMap:           gmap.NewStrAnyMap(true),
		roundRobinMap:     gmap.NewStrAnyMap(true),
		keyCacheMap:       gmap.NewStrAnyMap(true),
		modelKeysCacheMap: gmap.NewStrAnyMap(true),
	}
}

// 根据secretKey获取密钥信息
func (s *sKey) GetKey(ctx context.Context, secretKey string) (*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetKey time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := dao.Key.FindOne(ctx, bson.M{"key": secretKey, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Key{
		Id:           key.Id,
		AppId:        key.AppId,
		Corp:         key.Corp,
		Key:          key.Key,
		Type:         key.Type,
		Models:       key.Models,
		IsLimitQuota: key.IsLimitQuota,
		Quota:        key.Quota,
		IpWhitelist:  key.IpWhitelist,
		IpBlacklist:  key.IpBlacklist,
		Remark:       key.Remark,
		Status:       key.Status,
	}, nil
}

// 根据模型ID获取密钥列表
func (s *sKey) GetModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "is_agents_only": false, "status": 1, "models": bson.M{"$in": []string{id}}})
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

// 密钥列表
func (s *sKey) List(ctx context.Context, typ int) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{
		"type":   typ,
		"status": 1,
	}

	results, err := dao.Key.Find(ctx, filter, "-updated_at")
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

// 挑选模型密钥
func (s *sKey) PickModelKey(ctx context.Context, m *model.Model) (key *model.Key, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "PickModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	var keys []*model.Key
	var roundRobin *util.RoundRobin

	if keysValue := s.keysMap.Get(m.Id); keysValue != nil {
		keys = keysValue.([]*model.Key)
	}

	if len(keys) == 0 {

		if keys, err = s.GetCacheModelKeys(ctx, m.Id); err != nil {

			if keys, err = s.GetModelKeys(ctx, m.Id); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			if err = s.SaveCacheModelKeys(ctx, m.Id, keys); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

		if len(keys) == 0 {
			return nil, errors.ERR_NO_AVAILABLE_KEY
		}

		s.keysMap.Set(m.Id, keys)
	}

	if roundRobinValue := s.roundRobinMap.Get(m.Id); roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.roundRobinMap.Set(m.Id, roundRobin)
	}

	return keys[roundRobin.Index(len(keys))], nil
}

// 移除模型密钥
func (s *sKey) RemoveModelKey(ctx context.Context, m *model.Model, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RemoveModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	keysValue := s.keysMap.Get(m.Id)
	if keysValue != nil {

		if keys := keysValue.([]*model.Key); len(keys) > 0 {

			newKeys := make([]*model.Key, 0)
			for _, k := range keys {
				if k.Id != key.Id {
					newKeys = append(newKeys, k)
				}
			}

			s.keysMap.Set(m.Id, newKeys)
		}

		if err := dao.Key.UpdateById(ctx, key.Id, bson.M{"status": 2}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 记录错误模型密钥
func (s *sKey) RecordErrorModelKey(ctx context.Context, m *model.Model, key *model.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RecordErrorModelKey time: %d", gtime.TimestampMilli()-now)
	}()

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), key.Key, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	if _, err = redis.ExpireAt(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), gtime.Now().EndOfDay().Time); err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {
		s.RemoveModelKey(ctx, m, key)
	}
}

// 更改密钥额度
func (s *sKey) ChangeQuota(ctx context.Context, secretKey string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey ChangeQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Key.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存密钥列表到缓存
func (s *sKey) SaveCacheList(ctx context.Context, keys []*model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, key := range keys {
		fields[key.Id] = key
		s.keyCacheMap.Set(key.Id, key)
	}

	if len(fields) > 0 {
		if _, err := redis.HSet(ctx, consts.API_KEYS_KEY, fields); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的密钥列表
func (s *sKey) GetCacheList(ctx context.Context, ids ...string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.Key, 0)
	for _, id := range ids {
		keyCacheValue := s.keyCacheMap.Get(id)
		if keyCacheValue != nil {
			items = append(items, keyCacheValue.(*model.Key))
		}
	}

	if len(items) == len(ids) {
		return items, nil
	}

	reply, err := redis.HMGet(ctx, consts.API_KEYS_KEY, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		if len(items) != 0 {
			return items, nil
		}
		return nil, errors.New("keys is nil")
	}

	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.Key)
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if s.keyCacheMap.Get(result.Id) != nil {
			continue
		}

		if result.Status == 1 {
			items = append(items, result)
			s.keyCacheMap.Set(result.Id, result)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("keys is nil")
	}

	slices.SortFunc(items, func(k1, k2 *model.Key) int {
		return cmp.Compare(k1.Id, k2.Id)
	})

	return items, nil
}

// 保存模型密钥列表到缓存
func (s *sKey) SaveCacheModelKeys(ctx context.Context, id string, keys []*model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey SaveCacheModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, key := range keys {
		fields[key.Id] = key
	}

	if len(fields) > 0 {

		if _, err := redis.HSet(ctx, fmt.Sprintf(consts.API_MODEL_KEYS_KEY, id), fields); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.modelKeysCacheMap.Set(id, keys)
	}

	return nil
}

// 获取缓存中的模型密钥列表
func (s *sKey) GetCacheModelKeys(ctx context.Context, id string) ([]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sKey GetCacheModelKeys time: %d", gtime.TimestampMilli()-now)
	}()

	if modelKeysCacheValue := s.modelKeysCacheMap.Get(id); modelKeysCacheValue != nil {
		return modelKeysCacheValue.([]*model.Key), nil
	}

	reply, err := redis.HVals(ctx, fmt.Sprintf(consts.API_MODEL_KEYS_KEY, id))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		return nil, errors.New("modelKeys is nil")
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
		return nil, errors.New("modelKeys is nil")
	}

	slices.SortFunc(items, func(k1, k2 *model.Key) int {
		return cmp.Compare(k1.Id, k2.Id)
	})

	s.modelKeysCacheMap.Set(id, items)

	return items, nil
}

// 新增模型密钥到缓存列表中
func (s *sKey) CreateCacheModelKey(ctx context.Context, key *entity.Key) {

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

	if err := s.SaveCacheList(ctx, []*model.Key{k}); err != nil {
		logger.Error(ctx, err)
	}

	for _, id := range key.Models {
		if keysValue := s.keysMap.Get(id); keysValue != nil {
			s.keysMap.Set(id, append(keysValue.([]*model.Key), k))
		}
	}
}

// 更新缓存中的模型密钥
func (s *sKey) UpdateCacheModelKey(ctx context.Context, oldData *entity.Key, newData *entity.Key) {

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

	if err := s.SaveCacheList(ctx, []*model.Key{key}); err != nil {
		logger.Error(ctx, err)
	}

	// 用于处理oldData时判断作用
	newModelMap := make(map[string]string)

	for _, id := range newData.Models {

		newModelMap[id] = id

		if keysValue := s.keysMap.Get(id); keysValue != nil {

			keys := keysValue.([]*model.Key)
			newKeys := make([]*model.Key, 0)
			// 用于处理新添加了模型时判断作用
			keyMap := make(map[string]*model.Key)

			for _, k := range keys {

				if k.Id != newData.Id {
					newKeys = append(newKeys, k)
					keyMap[key.Id] = k
				} else {
					newKeys = append(newKeys, key)
					keyMap[newData.Id] = key
				}
			}

			if keyMap[newData.Id] == nil {
				newKeys = append(newKeys, key)
			}

			s.keysMap.Set(id, newKeys)
		}
	}

	// 将变更后被移除模型的模型密钥移除
	if oldData != nil {
		for _, id := range oldData.Models {

			if newModelMap[id] == "" {

				if keysValue := s.keysMap.Get(id); keysValue != nil {

					if keys := keysValue.([]*model.Key); len(keys) > 0 {

						newKeys := make([]*model.Key, 0)
						for _, k := range keys {
							if k.Id != oldData.Id {
								newKeys = append(newKeys, k)
							}
						}

						s.keysMap.Set(id, newKeys)
					}
				}
			}
		}
	}
}

// 移除缓存中的模型密钥
func (s *sKey) RemoveCacheModelKey(ctx context.Context, key *entity.Key) {

	for _, id := range key.Models {

		if keysValue := s.keysMap.Get(id); keysValue != nil {

			if keys := keysValue.([]*model.Key); len(keys) > 0 {

				newKeys := make([]*model.Key, 0)
				for _, k := range keys {
					if k.Id != key.Id {
						newKeys = append(newKeys, k)
					}
				}

				s.keysMap.Set(id, newKeys)
			}
		}
	}

	if _, err := redis.HDel(ctx, consts.API_KEYS_KEY, key.Id); err != nil {
		logger.Error(ctx, err)
	}

	s.keyCacheMap.Remove(key.Id)
}

// 变更订阅
func (s *sKey) Subscribe(ctx context.Context, msg string) error {

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sKey Subscribe: %s", gjson.MustEncodeString(message))

	var key *entity.Key
	switch message.Action {
	case consts.ACTION_CREATE:
		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}
		s.CreateCacheModelKey(ctx, key)
	case consts.ACTION_UPDATE:
		var oldData *entity.Key
		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &oldData); err != nil {
			logger.Error(ctx, err)
			return err
		}
		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}
		s.UpdateCacheModelKey(ctx, oldData, key)
	case consts.ACTION_STATUS:
		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}
		if key.Type == 1 {
			service.Common().UpdateCacheKey(ctx, key)
		} else {
			s.UpdateCacheModelKey(ctx, nil, key)
		}
	case consts.ACTION_DELETE:
		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}
		if key.Type == 1 {
			service.Common().RemoveCacheKey(ctx, key.Key)
		} else {
			s.RemoveCacheModelKey(ctx, key)
		}
	}

	return nil
}
