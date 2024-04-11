package chat

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/iimeta/fastapi/utility/util"
	"time"
)

var baiduCache = cache.New() // [key]AccessToken

func getAccessToken(ctx context.Context, key, baseURL, proxyURL string) string {

	result := gstr.Split(key, "|")

	if accessTokenCacheValue := baiduCache.GetVal(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, result[0])); accessTokenCacheValue != nil {
		return accessTokenCacheValue.(string)
	}

	reply, err := redis.GetStr(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, result[0]))
	if err == nil && reply != "" {

		if expiresIn, err := redis.TTL(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, result[0])); err != nil {
			logger.Errorf(ctx, "getAccessToken Baidu key: %s, error: %v", key, err)
		} else {
			if err = baiduCache.Set(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, result[0]), reply, time.Second*time.Duration(expiresIn-60)); err != nil {
				logger.Errorf(ctx, "getAccessToken Baidu key: %s, error: %v", key, err)
			}
		}

		return reply
	}

	data := g.Map{
		"grant_type":    "client_credentials",
		"client_id":     result[1],
		"client_secret": result[2],
	}

	getAccessTokenRes := new(model.GetAccessTokenRes)
	if err = util.HttpPost(ctx, baseURL+"/oauth/2.0/token", nil, data, &getAccessTokenRes, proxyURL); err != nil {
		logger.Errorf(ctx, "getAccessToken Baidu key: %s, error: %v", key, err)
		return ""
	}

	if getAccessTokenRes.Error != "" {
		logger.Errorf(ctx, "getAccessToken Baidu key: %s, getAccessTokenRes.Error: %s", key, getAccessTokenRes.Error)
		return ""
	}

	if err = baiduCache.Set(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, result[0]), getAccessTokenRes.AccessToken, time.Second*time.Duration(getAccessTokenRes.ExpiresIn-60)); err != nil {
		logger.Errorf(ctx, "getAccessToken Baidu key: %s, error: %v", key, err)
	}

	if err = redis.SetEX(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, result[0]), getAccessTokenRes.AccessToken, getAccessTokenRes.ExpiresIn-60); err != nil {
		logger.Errorf(ctx, "getAccessToken Baidu key: %s, error: %v", key, err)
	}

	return getAccessTokenRes.AccessToken
}
