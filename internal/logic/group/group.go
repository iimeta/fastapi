package group

import (
	"context"
	"slices"

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

type sGroup struct {
	groupCache      *cache.Cache // [分组ID]Group
	groupQuotaCache *cache.Cache // [分组ID]Quota
}

func init() {
	service.RegisterGroup(New())
}

func New() service.IGroup {
	return &sGroup{
		groupCache:      cache.New(),
		groupQuotaCache: cache.New(),
	}
}

// 根据分组ID获取分组信息
func (s *sGroup) GetById(ctx context.Context, id string) (*model.Group, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetById time: %d", gtime.TimestampMilli()-now)
	}()

	group, err := dao.Group.FindById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Group{
		Id:                 group.Id,
		Name:               group.Name,
		Discount:           group.Discount,
		Models:             group.Models,
		IsEnableModelAgent: group.IsEnableModelAgent,
		LbStrategy:         group.LbStrategy,
		ModelAgents:        group.ModelAgents,
		IsDefault:          group.IsDefault,
		IsLimitQuota:       group.IsLimitQuota,
		Quota:              group.Quota,
		UsedQuota:          group.UsedQuota,
		IsEnableForward:    group.IsEnableForward,
		ForwardConfig:      group.ForwardConfig,
		IsPublic:           group.IsPublic,
		Weight:             group.Weight,
		ExpiresAt:          group.ExpiresAt,
		Status:             group.Status,
	}, nil
}

// 分组列表
func (s *sGroup) List(ctx context.Context) ([]*model.Group, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.Group.Find(ctx, filter)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Group, 0)
	for _, result := range results {
		items = append(items, &model.Group{
			Id:                 result.Id,
			Name:               result.Name,
			Discount:           result.Discount,
			Models:             result.Models,
			IsEnableModelAgent: result.IsEnableModelAgent,
			LbStrategy:         result.LbStrategy,
			ModelAgents:        result.ModelAgents,
			IsDefault:          result.IsDefault,
			IsLimitQuota:       result.IsLimitQuota,
			Quota:              result.Quota,
			UsedQuota:          result.UsedQuota,
			IsEnableForward:    result.IsEnableForward,
			ForwardConfig:      result.ForwardConfig,
			IsPublic:           result.IsPublic,
			Weight:             result.Weight,
			ExpiresAt:          result.ExpiresAt,
			Status:             result.Status,
		})
	}

	return items, nil
}

// 根据分组ID获取分组信息并保存到缓存
func (s *sGroup) GetAndSaveCache(ctx context.Context, id string) (*model.Group, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetAndSaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	group, err := s.GetById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if group != nil {
		if err = s.SaveCache(ctx, group); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	return group, nil
}

// 保存分组到缓存
func (s *sGroup) SaveCache(ctx context.Context, group *model.Group) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	return s.SaveCacheList(ctx, []*model.Group{group})
}

// 保存分组列表到缓存
func (s *sGroup) SaveCacheList(ctx context.Context, groups []*model.Group) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	for _, group := range groups {

		if err := s.groupCache.Set(ctx, group.Id, group, 0); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := service.ModelAgent().SaveGroupCache(ctx, group); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的分组信息
func (s *sGroup) GetCache(ctx context.Context, id string) (*model.Group, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetCache time: %d", gtime.TimestampMilli()-now)
	}()

	groups, err := s.GetCacheList(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(groups) == 0 {
		return nil, errors.New("group is nil")
	}

	return groups[0], nil
}

// 获取缓存中的分组列表
func (s *sGroup) GetCacheList(ctx context.Context, ids ...string) ([]*model.Group, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.Group, 0)

	for _, id := range ids {
		if groupCacheValue := s.groupCache.GetVal(ctx, id); groupCacheValue != nil {
			items = append(items, groupCacheValue.(*model.Group))
		}
	}

	if len(items) == 0 {
		return nil, errors.New("groups is nil")
	}

	return items, nil
}

// 更新缓存中的分组列表
func (s *sGroup) UpdateCache(ctx context.Context, newData *entity.Group) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup UpdateCache time: %d", gtime.TimestampMilli()-now)
	}()

	group := &model.Group{
		Id:                 newData.Id,
		Name:               newData.Name,
		Discount:           newData.Discount,
		Models:             newData.Models,
		IsEnableModelAgent: newData.IsEnableModelAgent,
		LbStrategy:         newData.LbStrategy,
		ModelAgents:        newData.ModelAgents,
		IsDefault:          newData.IsDefault,
		IsLimitQuota:       newData.IsLimitQuota,
		Quota:              newData.Quota,
		UsedQuota:          newData.UsedQuota,
		IsEnableForward:    newData.IsEnableForward,
		ForwardConfig:      newData.ForwardConfig,
		IsPublic:           newData.IsPublic,
		Weight:             newData.Weight,
		ExpiresAt:          newData.ExpiresAt,
		Status:             newData.Status,
	}

	if err := s.SaveCache(ctx, group); err != nil {
		logger.Error(ctx, err)
	}

	if err := s.SaveCacheQuota(ctx, group.Id, group.Quota); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的分组列表
func (s *sGroup) RemoveCache(ctx context.Context, id string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup RemoveCache time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.groupCache.Remove(ctx, id); err != nil {
		logger.Error(ctx, err)
	}
}

// 根据分组Ids获取模型Ids
func (s *sGroup) GetModelIds(ctx context.Context, ids ...string) ([]string, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetModelIds time: %d", gtime.TimestampMilli()-now)
	}()

	groups, err := s.GetCacheList(ctx, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	modelIds := make([]string, 0)
	for _, group := range groups {
		modelIds = append(modelIds, group.Models...)
	}

	return modelIds, nil
}

// 根据分组Ids获取默认分组
func (s *sGroup) GetDefault(ctx context.Context, ids ...string) (*model.Group, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetDefault time: %d", gtime.TimestampMilli()-now)
	}()

	groups, err := s.GetCacheList(ctx, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	for _, group := range groups {
		if group.IsDefault && group.Status == 1 && (group.ExpiresAt == 0 || group.ExpiresAt > gtime.TimestampMilli()) {
			return group, nil
		}
	}

	return nil, nil
}

// 根据model挑选分组和模型
func (s *sGroup) PickGroupAndModel(ctx context.Context, m string, ids ...string) (reqModel *model.Model, group *model.Group, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup PickGroupAndModel time: %d", gtime.TimestampMilli()-now)
	}()

	groups, err := s.GetCacheList(ctx, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return
	}

	if len(groups) == 1 && groups[0].IsDefault {
		group = groups[0]
	} else {
		slices.SortFunc(groups, func(a, b *model.Group) int {

			if a.IsDefault {
				group = a
			}

			if b.IsDefault {
				group = b
			}

			if a.Weight > b.Weight {
				return -1
			} else if a.Weight < b.Weight {
				return 1
			}

			return 0
		})
	}

	if group != nil && group.Status == 1 {
		if reqModel, err = service.Model().GetModelByGroup(ctx, m, group); err != nil && !errors.Is(err, errors.ERR_MODEL_NOT_FOUND) {
			return
		}
	}

	if reqModel != nil {
		return
	}

	for _, group = range groups {

		if group.IsDefault || group.Status != 1 {
			continue
		}

		if reqModel, err = service.Model().GetModelByGroup(ctx, m, group); err != nil && !errors.Is(err, errors.ERR_MODEL_NOT_FOUND) {
			return
		}

		if reqModel != nil {
			return
		}
	}

	return nil, nil, nil
}

// 分组花费额度
func (s *sGroup) SpendQuota(ctx context.Context, group string, spendQuota, currentQuota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup SpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Group.UpdateById(ctx, group, bson.M{
		"$inc": bson.M{
			"quota":      -spendQuota,
			"used_quota": spendQuota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.SaveCacheQuota(ctx, group, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 分组已用额度
func (s *sGroup) UsedQuota(ctx context.Context, group string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup UsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Group.UpdateById(ctx, group, bson.M{
		"$inc": bson.M{
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存分组额度到缓存
func (s *sGroup) SaveCacheQuota(ctx context.Context, group string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup SaveCacheQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.groupQuotaCache.Set(ctx, group, quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的分组额度
func (s *sGroup) GetCacheQuota(ctx context.Context, group string) int {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup GetCacheQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if groupQuotaValue := s.groupQuotaCache.GetVal(ctx, group); groupQuotaValue != nil {
		return groupQuotaValue.(int)
	}

	return 0
}

// 变更订阅
func (s *sGroup) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sGroup Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sGroup Subscribe: %s", gjson.MustEncodeString(message))

	var group *entity.Group
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &group); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCache(ctx, group)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &group); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCache(ctx, group.Id)
	}

	return nil
}
