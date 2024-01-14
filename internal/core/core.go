package core

import (
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
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

		fields := g.Map{
			consts.USER_TOTAL_TOKENS_FIELD:                        userMap[app.UserId].Quota,
			fmt.Sprintf(consts.APP_TOTAL_TOKENS_FIELD, app.AppId): app.Quota,
		}

		keys := keyMap[app.AppId]
		for _, key := range keys {
			fields[fmt.Sprintf(consts.KEY_TOTAL_TOKENS_FIELD, key.AppId, key.Key)] = key.Quota
		}

		_, err = redis.HSet(ctx, fmt.Sprintf(consts.API_USAGE_KEY, app.UserId), fields)
		if err != nil {
			panic(err)
		}
	}
}
