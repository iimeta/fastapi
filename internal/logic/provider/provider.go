package provider

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/cache"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sProvider struct {
	providerCache *cache.Cache // [提供商ID]ProviderId
}

func init() {
	service.RegisterProvider(New())
}

func New() service.IProvider {
	return &sProvider{
		providerCache: cache.New(),
	}
}

// 根据提供商ID获取提供商信息
func (s *sProvider) GetById(ctx context.Context, id string) (*model.Provider, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider GetById time: %d", gtime.TimestampMilli()-now)
	}()

	provider, err := dao.Provider.FindById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Provider{
		Id:     provider.Id,
		Name:   provider.Name,
		Code:   provider.Code,
		Sort:   provider.Sort,
		Status: provider.Status,
	}, nil
}

// 提供商列表
func (s *sProvider) List(ctx context.Context) ([]*model.Provider, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.Provider.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Provider, 0)
	for _, result := range results {
		items = append(items, &model.Provider{
			Id:     result.Id,
			Name:   result.Name,
			Code:   result.Code,
			Sort:   result.Sort,
			Status: result.Status,
		})
	}

	return items, nil
}

// 根据提供商ID获取提供商信息并保存到缓存
func (s *sProvider) GetAndSaveCache(ctx context.Context, id string) (*model.Provider, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider GetAndSaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	provider, err := s.GetById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if provider != nil {
		if err = s.SaveCache(ctx, provider); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	return provider, nil
}

// 保存提供商到缓存
func (s *sProvider) SaveCache(ctx context.Context, provider *model.Provider) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	return s.SaveCacheList(ctx, []*model.Provider{provider})
}

// 保存提供商列表到缓存
func (s *sProvider) SaveCacheList(ctx context.Context, providers []*model.Provider) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	for _, provider := range providers {
		if err := s.providerCache.Set(ctx, provider.Id, provider, 0); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的提供商信息
func (s *sProvider) GetCache(ctx context.Context, id string) (*model.Provider, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider GetCache time: %d", gtime.TimestampMilli()-now)
	}()

	providers, err := s.GetCacheList(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(providers) == 0 {
		return nil, errors.New("provider is nil")
	}

	return providers[0], nil
}

// 获取缓存中的提供商列表
func (s *sProvider) GetCacheList(ctx context.Context, ids ...string) ([]*model.Provider, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.Provider, 0)
	for _, id := range ids {
		if providerCacheValue := s.providerCache.GetVal(ctx, id); providerCacheValue != nil {
			items = append(items, providerCacheValue.(*model.Provider))
		}
	}

	if len(items) == 0 {
		return nil, errors.New("providers is nil")
	}

	return items, nil
}

// 更新缓存中的提供商列表
func (s *sProvider) UpdateCache(ctx context.Context, newData *entity.Provider) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider UpdateCache time: %d", gtime.TimestampMilli()-now)
	}()

	provider := &model.Provider{
		Id:     newData.Id,
		Name:   newData.Name,
		Code:   newData.Code,
		Sort:   newData.Sort,
		Status: newData.Status,
	}

	if err := s.SaveCache(ctx, provider); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的提供商列表
func (s *sProvider) RemoveCache(ctx context.Context, id string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider RemoveCache time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.providerCache.Remove(ctx, id); err != nil {
		logger.Error(ctx, err)
	}
}

// 变更订阅
func (s *sProvider) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sProvider Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sProvider Subscribe: %s", gjson.MustEncodeString(message))

	var provider *entity.Provider
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &provider); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCache(ctx, provider)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &provider); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCache(ctx, provider.Id)
	}

	return nil
}
