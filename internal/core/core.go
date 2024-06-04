package core

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	_ "github.com/iimeta/fastapi/internal/logic"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"time"
)

func init() {

	var (
		ctx = gctx.New()
	)

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "core init time: %d", gtime.TimestampMilli()-now)
	}()

	users, err := service.User().List(ctx)
	if err != nil {
		panic(err)
	}

	userMap := util.ToMap(users, func(user *model.User) int {

		err = service.User().SaveCacheUser(ctx, user)
		if err != nil {
			panic(err)
		}

		if _, err = redis.HSetStrAny(ctx, fmt.Sprintf(consts.API_USAGE_KEY, user.UserId), consts.USER_QUOTA_FIELD, user.Quota); err != nil {
			panic(err)
		}

		return user.UserId
	})

	apps, err := service.App().List(ctx)
	if err != nil {
		panic(err)
	}

	keys, err := service.Key().List(ctx, 1)
	if err != nil {
		panic(err)
	}

	keyMap := make(map[int][]*model.Key)
	for _, key := range keys {

		err = service.App().SaveCacheAppKey(ctx, key)
		if err != nil {
			panic(err)
		}

		keyMap[key.AppId] = append(keyMap[key.AppId], key)
	}

	for _, app := range apps {

		err = service.App().SaveCacheApp(ctx, app)
		if err != nil {
			panic(err)
		}

		user := userMap[app.UserId]
		if user != nil {
			fields := g.Map{
				fmt.Sprintf(consts.APP_QUOTA_FIELD, app.AppId): app.Quota,
			}

			keys := keyMap[app.AppId]
			for _, key := range keys {
				fields[fmt.Sprintf(consts.KEY_QUOTA_FIELD, key.AppId, key.Key)] = key.Quota
			}

			if _, err = redis.HSet(ctx, fmt.Sprintf(consts.API_USAGE_KEY, app.UserId), fields); err != nil {
				panic(err)
			}
		}

	}

	// 获取所有公司
	corps, err := service.Corp().List(ctx)
	if err != nil {
		panic(err)
	}

	if len(corps) > 0 {
		// 初始化公司信息到缓存
		if err = service.Corp().SaveCacheList(ctx, corps); err != nil {
			panic(err)
		}
	}

	// 获取所有模型
	models, err := service.Model().ListAll(ctx)
	if err != nil {
		panic(err)
	}

	if len(models) > 0 {

		// 初始化模型信息到缓存
		if err = service.Model().SaveCacheList(ctx, models); err != nil {
			panic(err)
		}

		// 初始化模型密钥到缓存
		for _, model := range models {

			modelKeys, err := service.Key().GetModelKeys(ctx, model.Id)
			if err != nil {
				panic(err)
			}

			if err = service.Key().SaveCacheModelKeys(ctx, model.Id, modelKeys); err != nil {
				panic(err)
			}
		}
	}

	// 获取所有模型代理
	modelAgents, err := service.ModelAgent().ListAll(ctx)
	if err != nil {
		panic(err)
	}

	if len(modelAgents) > 0 {

		// 初始化模型代理信息到缓存
		if err = service.ModelAgent().SaveCacheList(ctx, modelAgents); err != nil {
			panic(err)
		}

		// 初始化模型代理密钥到缓存
		for _, modelAgent := range modelAgents {

			agentKeys, err := service.ModelAgent().GetModelAgentKeys(ctx, modelAgent.Id)
			if err != nil {
				panic(err)
			}

			if err = service.ModelAgent().SaveCacheModelAgentKeys(ctx, modelAgent.Id, agentKeys); err != nil {
				panic(err)
			}
		}
	}

	channels := make([]string, 0)
	channels = append(channels, consts.CHANGE_CHANNEL_USER)
	channels = append(channels, consts.CHANGE_CHANNEL_APP)
	channels = append(channels, consts.CHANGE_CHANNEL_APP_KEY)
	channels = append(channels, consts.CHANGE_CHANNEL_CORP)
	channels = append(channels, consts.CHANGE_CHANNEL_MODEL)
	channels = append(channels, consts.CHANGE_CHANNEL_KEY)
	channels = append(channels, consts.CHANGE_CHANNEL_AGENT)

	conn, _, err := redis.Subscribe(ctx, channels[0], channels[1:]...)
	if err != nil {
		panic(err)
	}

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {
		for {

			msg, err := conn.ReceiveMessage(ctx)
			if err != nil {
				logger.Error(ctx, err)
				time.Sleep(5 * time.Second)
				conn, _, err = redis.Subscribe(ctx, channels[0], channels[1:]...)
				if err != nil {
					logger.Error(ctx, err)
				}
				continue
			}

			switch msg.Channel {
			case consts.CHANGE_CHANNEL_USER:
				err = service.User().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_APP:
				err = service.App().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_APP_KEY:
				err = service.App().SubscribeKey(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_CORP:
				err = service.Corp().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_MODEL:
				err = service.Model().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_KEY:
				err = service.Key().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_AGENT:
				err = service.ModelAgent().Subscribe(ctx, msg.Payload)
			}

			if err != nil {
				logger.Error(ctx, err)
			}
		}
	}, nil); err != nil {
		panic(err)
	}
}
