package core

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/os/gtimer"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	_ "github.com/iimeta/fastapi/internal/logic/app"
	_ "github.com/iimeta/fastapi/internal/logic/corp"
	_ "github.com/iimeta/fastapi/internal/logic/group"
	_ "github.com/iimeta/fastapi/internal/logic/key"
	_ "github.com/iimeta/fastapi/internal/logic/model"
	_ "github.com/iimeta/fastapi/internal/logic/model_agent"
	_ "github.com/iimeta/fastapi/internal/logic/reseller"
	_ "github.com/iimeta/fastapi/internal/logic/user"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"sync"
	"time"
)

type sCore struct {
	mutex sync.Mutex
}

func init() {

	ctx := gctx.New()

	logger.Info(ctx, "sCore init ing...")

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCore init time: %d", gtime.TimestampMilli()-now)
	}()

	core := New()

	service.RegisterCore(core)

	if err := core.Refresh(ctx); err != nil {
		panic(err)
	}

	_, _ = gcron.AddSingleton(ctx, "0 0/30 * * * ?", func(ctx context.Context) {
		if err := core.Refresh(gctx.New()); err != nil {
			logger.Error(ctx, err)
		}
	})

	_ = gtimer.AddSingleton(ctx, 30*time.Minute, func(ctx context.Context) {
		if err := core.Refresh(gctx.New()); err != nil {
			logger.Error(ctx, err)
		}
	})

	channels := make([]string, 0)
	channels = append(channels, consts.CHANGE_CHANNEL_RESELLER)
	channels = append(channels, consts.CHANGE_CHANNEL_USER)
	channels = append(channels, consts.CHANGE_CHANNEL_APP)
	channels = append(channels, consts.CHANGE_CHANNEL_APP_KEY)
	channels = append(channels, consts.CHANGE_CHANNEL_CORP)
	channels = append(channels, consts.CHANGE_CHANNEL_MODEL)
	channels = append(channels, consts.CHANGE_CHANNEL_KEY)
	channels = append(channels, consts.CHANGE_CHANNEL_AGENT)
	channels = append(channels, consts.CHANGE_CHANNEL_GROUP)
	channels = append(channels, consts.REFRESH_CHANNEL_API)

	conn, _, err := redis.Subscribe(ctx, channels[0], channels[1:]...)
	if err != nil {
		panic(err)
	}

	if err = grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
		for {

			msg, err := conn.ReceiveMessage(ctx)
			if err != nil {
				logger.Errorf(ctx, "sCore Subscribe error: %v", err)
				time.Sleep(5 * time.Second)
				if conn, _, err = redis.Subscribe(ctx, channels[0], channels[1:]...); err != nil {
					logger.Errorf(ctx, "sCore Subscribe Reconnect error: %v", err)
				} else {
					logger.Info(ctx, "sCore Subscribe Reconnect success")
				}
				continue
			}

			switch msg.Channel {
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_RESELLER:
				err = service.Reseller().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_USER:
				err = service.User().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_APP:
				err = service.App().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_APP_KEY:
				err = service.App().SubscribeKey(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_CORP:
				err = service.Corp().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_MODEL:
				err = service.Model().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_KEY:
				err = service.Key().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_AGENT:
				err = service.ModelAgent().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_GROUP:
				err = service.Group().Subscribe(gctx.New(), msg.Payload)
			case config.Cfg.Core.ChannelPrefix + consts.REFRESH_CHANNEL_API:
				err = core.Refresh(gctx.New())
			}

			if err != nil {
				logger.Error(ctx, err)
			}
		}
	}, nil); err != nil {
		panic(err)
	}
}

func New() service.ICore {
	return &sCore{}
}

// 刷新缓存
func (s *sCore) Refresh(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	logger.Info(ctx, "sCore Refresh ing...")

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCore Refresh time: %d", gtime.TimestampMilli()-now)
	}()

	resellers, err := service.Reseller().List(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	for _, reseller := range resellers {

		if err = service.Reseller().SaveCacheReseller(ctx, reseller); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = service.Reseller().SaveCacheResellerQuota(ctx, reseller.UserId, reseller.Quota); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
			if _, err = redis.HSetStrAny(ctx, fmt.Sprintf(consts.API_RESELLER_USAGE_KEY, reseller.UserId), consts.RESELLER_QUOTA_FIELD, reseller.Quota); err != nil {
				logger.Error(ctx, err)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	users, err := service.User().List(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	userMap := make(map[int]*model.User)
	for _, user := range users {

		if err = service.User().SaveCacheUser(ctx, user); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = service.User().SaveCacheUserQuota(ctx, user.UserId, user.Quota); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
			if _, err = redis.HSetStrAny(ctx, fmt.Sprintf(consts.API_USER_USAGE_KEY, user.UserId), consts.USER_QUOTA_FIELD, user.Quota); err != nil {
				logger.Error(ctx, err)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
			return err
		}

		userMap[user.UserId] = user
	}

	appKeys, err := service.Key().List(ctx, 1)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	appKeyMap := make(map[int][]*model.Key)
	for _, key := range appKeys {

		if err = service.App().SaveCacheAppKey(ctx, key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = service.App().SaveCacheAppKeyQuota(ctx, key.Key, key.Quota); err != nil {
			logger.Error(ctx, err)
			return err
		}

		appKeyMap[key.AppId] = append(appKeyMap[key.AppId], key)
	}

	apps, err := service.App().List(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	for _, app := range apps {

		if err = service.App().SaveCacheApp(ctx, app); err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = service.App().SaveCacheAppQuota(ctx, app.AppId, app.Quota); err != nil {
			logger.Error(ctx, err)
			return err
		}

		user := userMap[app.UserId]
		if user != nil {

			fields := g.Map{
				fmt.Sprintf(consts.APP_QUOTA_FIELD, app.AppId): app.Quota,
			}

			keys := appKeyMap[app.AppId]
			for _, key := range keys {
				fields[fmt.Sprintf(consts.KEY_QUOTA_FIELD, key.AppId, key.Key)] = key.Quota
			}

			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if _, err = redis.HSet(ctx, fmt.Sprintf(consts.API_USER_USAGE_KEY, app.UserId), fields); err != nil {
					logger.Error(ctx, err)
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}

	corps, err := service.Corp().List(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if len(corps) > 0 {
		if err = service.Corp().SaveCacheList(ctx, corps); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	models, modelKeyMap, err := service.Model().GetModelsAndKeys(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if len(models) > 0 {

		if err = service.Model().SaveCacheList(ctx, models); err != nil {
			logger.Error(ctx, err)
			return err
		}

		for _, model := range models {
			if err = service.Key().SaveCacheModelKeys(ctx, model.Id, modelKeyMap[model.Id]); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}

	modelAgents, modelAgentKeyMap, err := service.ModelAgent().GetModelAgentsAndKeys(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if len(modelAgents) > 0 {

		if err = service.ModelAgent().SaveCacheList(ctx, modelAgents); err != nil {
			logger.Error(ctx, err)
			return err
		}

		for _, modelAgent := range modelAgents {
			if err = service.ModelAgent().SaveCacheModelAgentKeys(ctx, modelAgent.Id, modelAgentKeyMap[modelAgent.Id]); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}

	groups, err := service.Group().List(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if len(groups) > 0 {

		if err = service.Group().SaveCacheList(ctx, groups); err != nil {
			logger.Error(ctx, err)
			return err
		}

		fields := g.Map{}
		for _, group := range groups {

			if err = service.Group().SaveCacheGroupQuota(ctx, group.Id, group.Quota); err != nil {
				logger.Error(ctx, err)
				return err
			}

			fields[group.Id] = group.Quota
		}

		if _, err = redis.HSet(ctx, consts.API_GROUP_USAGE_KEY, fields); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}
