package reseller

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

type sReseller struct {
	resellerCache      *cache.Cache // [userId]Reseller
	resellerQuotaCache *cache.Cache // [userId]Quota
}

func init() {
	service.RegisterReseller(New())
}

func New() service.IReseller {
	return &sReseller{
		resellerCache:      cache.New(),
		resellerQuotaCache: cache.New(),
	}
}

// 根据用户ID获取代理商信息
func (s *sReseller) GetByUserId(ctx context.Context, userId int) (*model.Reseller, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller GetByUserId time: %d", gtime.TimestampMilli()-now)
	}()

	reseller, err := dao.Reseller.FindResellerByUserId(ctx, userId)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Reseller{
		Id:             reseller.Id,
		UserId:         reseller.UserId,
		Name:           reseller.Name,
		Avatar:         reseller.Avatar,
		Email:          reseller.Email,
		Phone:          reseller.Phone,
		Quota:          reseller.Quota,
		UsedQuota:      reseller.UsedQuota,
		QuotaExpiresAt: reseller.QuotaExpiresAt,
		Groups:         reseller.Groups,
		Status:         reseller.Status,
	}, nil
}

// 代理商列表
func (s *sReseller) List(ctx context.Context) ([]*model.Reseller, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.Reseller.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Reseller, 0)
	for _, result := range results {
		items = append(items, &model.Reseller{
			Id:             result.Id,
			UserId:         result.UserId,
			Name:           result.Name,
			Avatar:         result.Avatar,
			Email:          result.Email,
			Phone:          result.Phone,
			Quota:          result.Quota,
			UsedQuota:      result.UsedQuota,
			QuotaExpiresAt: result.QuotaExpiresAt,
			Groups:         result.Groups,
			Status:         result.Status,
		})
	}

	return items, nil
}

// 代理商花费额度
func (s *sReseller) SpendQuota(ctx context.Context, userId, spendQuota, currentQuota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller SpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Reseller.UpdateOne(ctx, bson.M{"user_id": userId}, bson.M{
		"$inc": bson.M{
			"quota":      -spendQuota,
			"used_quota": spendQuota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.SaveCacheQuota(ctx, userId, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 保存代理商信息到缓存
func (s *sReseller) SaveCache(ctx context.Context, reseller *model.Reseller) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	if reseller == nil {
		return errors.New("reseller is nil")
	}

	service.Session().SaveReseller(ctx, reseller)

	if err := s.resellerCache.Set(ctx, reseller.UserId, reseller, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.resellerQuotaCache.Set(ctx, reseller.UserId, reseller.Quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的代理商信息
func (s *sReseller) GetCache(ctx context.Context, userId int) (*model.Reseller, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller GetCache time: %d", gtime.TimestampMilli()-now)
	}()

	if reseller := service.Session().GetReseller(ctx); reseller != nil {
		return reseller, nil
	}

	if resellerCacheValue := s.resellerCache.GetVal(ctx, userId); resellerCacheValue != nil {
		return resellerCacheValue.(*model.Reseller), nil
	}

	return nil, errors.New("reseller is nil")
}

// 更新缓存中的代理商信息
func (s *sReseller) UpdateCache(ctx context.Context, reseller *entity.Reseller) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller UpdateCache time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCache(ctx, &model.Reseller{
		Id:             reseller.Id,
		UserId:         reseller.UserId,
		Name:           reseller.Name,
		Avatar:         reseller.Avatar,
		Email:          reseller.Email,
		Phone:          reseller.Phone,
		Quota:          reseller.Quota,
		UsedQuota:      reseller.UsedQuota,
		QuotaExpiresAt: reseller.QuotaExpiresAt,
		Groups:         reseller.Groups,
		Status:         reseller.Status,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的代理商信息
func (s *sReseller) RemoveCache(ctx context.Context, userId int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller RemoveCache time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.resellerCache.Remove(ctx, userId); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := s.resellerQuotaCache.Remove(ctx, userId); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存代理商额度到缓存
func (s *sReseller) SaveCacheQuota(ctx context.Context, userId, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller SaveCacheQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.resellerQuotaCache.Set(ctx, userId, quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的代理商额度
func (s *sReseller) GetCacheQuota(ctx context.Context, userId int) int {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller GetCacheQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if resellerQuotaValue := s.resellerQuotaCache.GetVal(ctx, userId); resellerQuotaValue != nil {
		return resellerQuotaValue.(int)
	}

	return 0
}

// 变更订阅
func (s *sReseller) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sReseller Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sReseller Subscribe: %s", gjson.MustEncodeString(message))

	var reseller *entity.Reseller
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS, consts.ACTION_MODELS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &reseller); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCache(ctx, reseller)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &reseller); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCache(ctx, reseller.UserId)
	}

	return nil
}
