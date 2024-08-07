package chat

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"time"
)

var gcpCache = cache.New() // [key]Token

func getGcpToken(ctx context.Context, key, proxyURL string) string {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "getGcpToken time: %d", gtime.TimestampMilli()-now)
	}()

	if gcpTokenCacheValue := gcpCache.GetVal(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key)); gcpTokenCacheValue != nil {
		return gcpTokenCacheValue.(string)
	}

	reply, err := redis.GetStr(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key))
	if err == nil && reply != "" {

		if expiresIn, err := redis.TTL(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key)); err != nil {
			logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key, err)
		} else {
			if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key), reply, time.Second*time.Duration(expiresIn-60)); err != nil {
				logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key, err)
			}
		}

		return reply
	}

	result := gstr.Split(key, "|")

	data := g.Map{
		"client_id":     result[1],
		"client_secret": result[2],
		"refresh_token": result[3],
		"grant_type":    "refresh_token",
	}

	getGcpTokenRes := new(model.GetGcpTokenRes)
	if err = util.HttpPost(ctx, config.Cfg.Gcp.GetTokenUrl, nil, data, &getGcpTokenRes, proxyURL); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key, err)
		return ""
	}

	if getGcpTokenRes.Error != "" {
		logger.Errorf(ctx, "getGcpToken key: %s, getGcpTokenRes.Error: %s", key, getGcpTokenRes.Error)
		return ""
	}

	if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key), getGcpTokenRes.AccessToken, time.Second*time.Duration(getGcpTokenRes.ExpiresIn-60)); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key, err)
	}

	if err = redis.SetEX(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, key), getGcpTokenRes.AccessToken, getGcpTokenRes.ExpiresIn-60); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key, err)
	}

	return getGcpTokenRes.AccessToken
}
