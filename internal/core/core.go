package core

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
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

	users, err := service.User().List(ctx)
	if err != nil {
		panic(err)
	}

	userMap := util.ToMap(users, func(t *model.User) int {
		return t.UserId
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
		keyMap[key.AppId] = append(keyMap[key.AppId], key)
	}

	for _, app := range apps {

		user := userMap[app.UserId]
		if user != nil {
			fields := g.Map{
				consts.USER_TOTAL_TOKENS_FIELD:                          user.Quota,
				fmt.Sprintf(consts.APP_TOTAL_TOKENS_FIELD, app.AppId):   app.Quota,
				fmt.Sprintf(consts.APP_IS_LIMIT_QUOTA_FIELD, app.AppId): app.IsLimitQuota,
			}

			keys := keyMap[app.AppId]
			for _, key := range keys {
				fields[fmt.Sprintf(consts.KEY_TOTAL_TOKENS_FIELD, key.AppId, key.Key)] = key.Quota
				fields[fmt.Sprintf(consts.KEY_IS_LIMIT_QUOTA_FIELD, key.AppId, key.Key)] = key.IsLimitQuota
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
				_ = service.User().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_APP:
				_ = service.App().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_APP_KEY:
				_ = service.App().SubscribeKey(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_MODEL:
				_ = service.Model().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_KEY:
				_ = service.Key().Subscribe(ctx, msg.Payload)
			case consts.CHANGE_CHANNEL_AGENT:
				_ = service.ModelAgent().Subscribe(ctx, msg.Payload)
			}
		}
	}, nil); err != nil {
		panic(err)
	}
}
