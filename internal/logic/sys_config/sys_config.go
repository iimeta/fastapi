package sys_config

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtimer"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type sSysConfig struct{}

func init() {

	ctx := gctx.New()
	sSysConfig := New()

	service.RegisterSysConfig(sSysConfig)
	if _, err := sSysConfig.Init(ctx); err != nil {
		panic(err)
	}

	_, _ = gcron.AddSingleton(gctx.New(), "0 0/30 * * * ?", func(ctx context.Context) {
		_, _ = service.SysConfig().Init(ctx)
	})

	_ = gtimer.AddSingleton(gctx.New(), 30*time.Minute, func(ctx context.Context) {
		_, _ = service.SysConfig().Init(ctx)
	})

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
