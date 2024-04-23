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
)

func init() {

	ctx := gctx.New()

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

	conn, _, err := redis.Subscribe(ctx, consts.CHANGE_CHANNEL_USER, consts.CHANGE_CHANNEL_APP, consts.CHANGE_CHANNEL_MODEL, consts.CHANGE_CHANNEL_KEY, consts.CHANGE_CHANNEL_AGENT, consts.CHANGE_CHANNEL_APP_KEY)
	if err != nil {
		panic(err)
	}

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {
		for {

			msg, err := conn.ReceiveMessage(ctx)
			if err != nil {
				logger.Error(ctx, err)
				continue
			}

			switch msg.Channel {
			case consts.CHANGE_CHANNEL_USER:
				err = service.User().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_APP:
				err = service.App().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_APP_KEY:
				err = service.App().SubscribeKey(ctx, msg.Payload)
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
