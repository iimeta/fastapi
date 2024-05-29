package corp

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
)

type sCorp struct {
	corpCache *cache.Cache // [公司ID]Corp
}

func init() {
	service.RegisterCorp(New())
}

func New() service.ICorp {
	return &sCorp{
		corpCache: cache.New(),
	}
}

// 根据公司ID获取公司信息
func (s *sCorp) GetCorp(ctx context.Context, id string) (*model.Corp, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp GetCorp time: %d", gtime.TimestampMilli()-now)
	}()

	corp, err := dao.Corp.FindById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Corp{
		Id:     corp.Id,
		Name:   corp.Name,
		Code:   corp.Code,
		Sort:   corp.Sort,
		Status: corp.Status,
	}, nil
}

// 公司列表
func (s *sCorp) List(ctx context.Context) ([]*model.Corp, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.Corp.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Corp, 0)
	for _, result := range results {
		items = append(items, &model.Corp{
			Id:     result.Id,
			Name:   result.Name,
			Code:   result.Code,
			Sort:   result.Sort,
			Status: result.Status,
		})
	}

	return items, nil
}

// 根据公司ID获取公司信息并保存到缓存
func (s *sCorp) GetCorpAndSaveCache(ctx context.Context, id string) (*model.Corp, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp GetCorpAndSaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	corp, err := s.GetCorp(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if corp != nil {
		if err = s.SaveCache(ctx, corp); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	return corp, nil
}

// 保存公司到缓存
func (s *sCorp) SaveCache(ctx context.Context, m *model.Corp) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	return s.SaveCacheList(ctx, []*model.Corp{m})
}

// 保存公司列表到缓存
func (s *sCorp) SaveCacheList(ctx context.Context, corps []*model.Corp) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, corp := range corps {
		fields[corp.Id] = corp
		if err := s.corpCache.Set(ctx, corp.Id, corp, 0); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if len(fields) > 0 {
		if _, err := redis.HSet(ctx, consts.API_CORPS_KEY, fields); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的公司信息
func (s *sCorp) GetCacheCorp(ctx context.Context, id string) (*model.Corp, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp GetCacheCorp time: %d", gtime.TimestampMilli()-now)
	}()

	corps, err := s.GetCacheList(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(corps) == 0 {
		return nil, errors.New("corp is nil")
	}

	return corps[0], nil
}

// 获取缓存中的公司列表
func (s *sCorp) GetCacheList(ctx context.Context, ids ...string) ([]*model.Corp, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.Corp, 0)

	for _, id := range ids {
		if corpCacheValue := s.corpCache.GetVal(ctx, id); corpCacheValue != nil {
			items = append(items, corpCacheValue.(*model.Corp))
		}
	}

	if len(items) == len(ids) {
		return items, nil
	}

	reply, err := redis.HMGet(ctx, consts.API_CORPS_KEY, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		if len(items) != 0 {
			return items, nil
		}
		return nil, errors.New("corps is nil")
	}

	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.Corp)
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if s.corpCache.ContainsKey(ctx, result.Id) {
			continue
		}

		items = append(items, result)
		if err = s.corpCache.Set(ctx, result.Id, result, 0); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, errors.New("corps is nil")
	}

	return items, nil
}

// 更新缓存中的公司列表
func (s *sCorp) UpdateCacheCorp(ctx context.Context, newData *entity.Corp) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp UpdateCacheCorp time: %d", gtime.TimestampMilli()-now)
	}()

	m := &model.Corp{
		Id:     newData.Id,
		Name:   newData.Name,
		Code:   newData.Code,
		Sort:   newData.Sort,
		Status: newData.Status,
	}

	if err := s.SaveCache(ctx, m); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的公司列表
func (s *sCorp) RemoveCacheCorp(ctx context.Context, id string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp RemoveCacheCorp time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.corpCache.Remove(ctx, id); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HDel(ctx, consts.API_CORPS_KEY, id); err != nil {
		logger.Error(ctx, err)
	}
}

// 变更订阅
func (s *sCorp) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCorp Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sCorp Subscribe: %s", gjson.MustEncodeString(message))

	var corp *entity.Corp
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &corp); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheCorp(ctx, corp)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &corp); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheCorp(ctx, corp.Id)
	}

	return nil
}
