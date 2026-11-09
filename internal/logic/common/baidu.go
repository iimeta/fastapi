package common

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	sutil "github.com/iimeta/fastapi-sdk/v2/util"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/utility/cache"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
)

var baiduCache = cache.New() // [key]AccessToken

func getBaiduToken(ctx context.Context, key, baseUrl, proxyUrl string) string {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "getBaiduToken time: %d", gtime.TimestampMilli()-now)
	}()

	if accessTokenCacheValue := baiduCache.GetVal(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, key)); accessTokenCacheValue != nil {
		return accessTokenCacheValue.(string)
	}

	reply, err := redis.GetStr(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, key))
	if err == nil && reply != "" {

		if expiresIn, err := redis.TTL(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, key)); err != nil {
			logger.Errorf(ctx, "getBaiduToken key: %s, error: %v", key, err)
		} else {
			if err = baiduCache.Set(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, key), reply, time.Second*time.Duration(expiresIn-60)); err != nil {
				logger.Errorf(ctx, "getBaiduToken key: %s, error: %v", key, err)
			}
		}

		return reply
	}

	result := gstr.Split(key, "|")

	// 与原 gf 客户端行为保持一致, 以表单形式提交
	data := url.Values{
		"client_id":     {result[0]},
		"client_secret": {result[1]},
		"grant_type":    {"client_credentials"},
	}

	header := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	parse, err := url.Parse(baseUrl)
	if err != nil {
		logger.Errorf(ctx, "getBaiduToken url.Parse baseUrl: %s, error: %s", baseUrl, err)
		return ""
	}

	tokenUrl := fmt.Sprintf("%s://%s/oauth/2.0/token", parse.Scheme, parse.Host)

	getBaiduTokenRes := new(model.GetBaiduTokenRes)
	if _, _, err = sutil.HttpPost(ctx, tokenUrl, header, []byte(data.Encode()), getBaiduTokenRes, config.Cfg.Http.Timeout*time.Second, proxyUrl, nil); err != nil {
		logger.Errorf(ctx, "getBaiduToken key: %s, error: %v", key, err)
		return ""
	}

	if getBaiduTokenRes.Error != "" {
		logger.Errorf(ctx, "getBaiduToken key: %s, getBaiduTokenRes.Error: %s", key, getBaiduTokenRes.Error)
		return ""
	}

	if err = baiduCache.Set(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, key), getBaiduTokenRes.AccessToken, time.Second*time.Duration(getBaiduTokenRes.ExpiresIn-60)); err != nil {
		logger.Errorf(ctx, "getBaiduToken key: %s, error: %v", key, err)
	}

	if err = redis.SetEX(ctx, fmt.Sprintf(consts.ACCESS_TOKEN_KEY, key), getBaiduTokenRes.AccessToken, getBaiduTokenRes.ExpiresIn-60); err != nil {
		logger.Errorf(ctx, "getBaiduToken key: %s, error: %v", key, err)
	}

	return getBaiduTokenRes.AccessToken
}
