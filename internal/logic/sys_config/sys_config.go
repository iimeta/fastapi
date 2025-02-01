package sys_config

import (
	"context"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

type sSysConfig struct{}

func init() {

	ctx := gctx.New()
	sSysConfig := New()

	service.RegisterSysConfig(sSysConfig)
	if _, err := sSysConfig.Init(ctx); err != nil {
		panic(err)
	}

	conn, _, err := redis.Subscribe(ctx, consts.CHANGE_CHANNEL_CONFIG)
	if err != nil {
		panic(err)
	}

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {
		for {

			msg, err := conn.ReceiveMessage(ctx)
			if err != nil {
				logger.Errorf(ctx, "sSysConfig Subscribe error: %v", err)
				time.Sleep(5 * time.Second)
				if conn, _, err = redis.Subscribe(ctx, consts.CHANGE_CHANNEL_CONFIG); err != nil {
					logger.Errorf(ctx, "sSysConfig Subscribe Reconnect error: %v", err)
				} else {
					logger.Info(ctx, "sSysConfig Subscribe Reconnect success")
				}
				continue
			}

			switch msg.Channel {
			case config.Cfg.Core.ChannelPrefix + consts.CHANGE_CHANNEL_CONFIG:
				_, err = service.SysConfig().Init(ctx)
			}

			if err != nil {
				logger.Error(ctx, err)
			}
		}
	}, nil); err != nil {
		panic(err)
	}
}

func New() service.ISysConfig {
	return &sSysConfig{}
}

// 初始化配置
func (s *sSysConfig) Init(ctx context.Context) (sysConfig *entity.SysConfig, err error) {

	defer func() {
		if err == nil && sysConfig != nil {
			config.Reload(ctx, sysConfig)
		}
	}()

	if sysConfig, err = dao.SysConfig.FindOne(ctx, bson.M{}); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return sysConfig, nil
}
