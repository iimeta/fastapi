package key

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/container/gmap"
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

type sKey struct {
	keysMap       *gmap.StrAnyMap
	roundRobinMap *gmap.StrAnyMap
}

func init() {
	service.RegisterKey(New())
}

func New() service.IKey {
	return &sKey{
		keysMap:       gmap.NewStrAnyMap(true),
		roundRobinMap: gmap.NewStrAnyMap(true),
	}
}

// 根据secretKey获取密钥信息
func (s *sKey) GetKey(ctx context.Context, secretKey string) (*model.Key, error) {

	key, err := dao.Key.FindOne(ctx, bson.M{"key": secretKey})
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

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "status": 1, "models": bson.M{"$in": []string{id}}})
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

	filter := bson.M{
		"type": typ,
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

// 根据模型ID挑选密钥
func (s *sKey) PickModelKey(ctx context.Context, id string) (key *model.Key, err error) {

	var keys []*model.Key
	var roundRobin *util.RoundRobin

	keysValue := s.keysMap.Get(id)
	roundRobinValue := s.roundRobinMap.Get(id)

	if keysValue != nil {
		keys = keysValue.([]*model.Key)
	}

	if len(keys) == 0 {

		keys, err = s.GetModelKeys(ctx, id)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if len(keys) == 0 {
			return nil, errors.ERR_NO_AVAILABLE_KEY
		}

		s.keysMap.Set(id, keys)
	}

	if roundRobinValue != nil {
		roundRobin = roundRobinValue.(*util.RoundRobin)
	}

	if roundRobin == nil {
		roundRobin = new(util.RoundRobin)
		s.roundRobinMap.Set(id, roundRobin)
	}

	return keys[roundRobin.Index(len(keys))], nil
}

// 根据模型ID移除密钥
func (s *sKey) RemoveModelKey(ctx context.Context, m *model.Model, key *model.Key) {

	keysValue := s.keysMap.Get(m.Id)
	if keysValue != nil {

		keys := keysValue.([]*model.Key)

		if len(keys) > 0 {

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

// 记录模型错误密钥
func (s *sKey) RecordModelErrorKey(ctx context.Context, m *model.Model, key *model.Key) {

	reply, err := redis.HIncrBy(ctx, fmt.Sprintf(consts.ERROR_MODEL_KEY, m.Model), key.Key, 1)
	if err != nil {
		logger.Error(ctx, err)
	}

	if reply >= 10 {
		s.RemoveModelKey(ctx, m, key)
	}
}

// 根据密钥更新额度
func (s *sKey) UpdateQuota(ctx context.Context, secretKey string, quota int) error {

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
