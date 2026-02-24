package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	scommon "github.com/iimeta/fastapi-sdk/v2/common"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/cache"
	"github.com/iimeta/fastapi/v2/utility/crypto"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
)

var gcpCache = cache.New() // [key]Token

func getGcpToken(ctx context.Context, key *model.Key, proxyUrl string) (string, string, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "getGcpToken time: %d", gtime.TimestampMilli()-now)
	}()

	adc := scommon.ApplicationDefaultCredentials{}
	if err := json.Unmarshal([]byte(key.Key), &adc); err != nil {
		logger.Errorf(ctx, "getGcpToken json.Unmarshal key: %s, error: %v", key.Key, err)
		return "", "", err
	}

	if gcpTokenCacheValue := gcpCache.GetVal(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key))); gcpTokenCacheValue != nil {
		return adc.ProjectId, gcpTokenCacheValue.(string), nil
	}

	reply, err := redis.GetStr(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)))
	if err == nil && reply != "" {

		if expiresIn, err := redis.TTL(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key))); err != nil {
			logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
		} else {
			if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)), reply, time.Second*time.Duration(expiresIn-60)); err != nil {
				logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
			}
		}

		return adc.ProjectId, reply, nil
	}

	accessToken, err := scommon.GetGcpToken(ctx, key.Key, proxyUrl)
	if err != nil {
		logger.Errorf(ctx, "getGcpToken scommon.GetGcpToken key: %s, error: %v", key.Key, err)
		if config.Cfg.AutoDisabledError.Open && len(config.Cfg.AutoDisabledError.Errors) > 0 {
			for _, autoDisabledError := range config.Cfg.AutoDisabledError.Errors {
				if gstr.Contains(err.Error(), autoDisabledError) {
					if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
						service.Key().Disabled(ctx, key, err.Error())
					}); err != nil {
						logger.Error(ctx, err)
					}
					break
				}
			}
		}
		return "", "", err
	}

	if err = gcpCache.Set(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)), accessToken, time.Minute*50); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
	}

	if err = redis.SetEX(ctx, fmt.Sprintf(consts.GCP_TOKEN_KEY, crypto.SM3(key.Key)), accessToken, 60*50); err != nil {
		logger.Errorf(ctx, "getGcpToken key: %s, error: %v", key.Key, err)
	}

	return adc.ProjectId, accessToken, nil
}
