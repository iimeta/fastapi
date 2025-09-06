package config

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gfsnotify"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/utility/logger"
)

var Cfg *Config

func init() {

	file, _ := gcfg.NewAdapterFile()
	path, _ := file.GetFilePath()

	if err := gjson.Unmarshal(gjson.MustEncode(gcfg.Instance().MustData(gctx.New())), &Cfg); err != nil {
		panic(fmt.Sprintf("解析配置文件 %s 错误: %v", path, err))
	}

	// 监听配置文件变化, 热加载
	_, _ = gfsnotify.Add(path, func(event *gfsnotify.Event) {
		ctx := gctx.New()
		if data, err := gcfg.Instance().Data(ctx); err != nil {
			logger.Errorf(ctx, "热加载 获取配置文件 %s 数据错误: %v", path, err)
		} else {
			if err = gjson.Unmarshal(gjson.MustEncode(data), &Cfg); err != nil {
				logger.Errorf(ctx, "热加载 解析配置文件 %s 错误: %v", path, err)
			} else {
				logger.Infof(ctx, "热加载 配置文件 %s 成功 当前配置信息: %s", path, gjson.MustEncodeString(Cfg))
			}
		}
	})
}

// 配置信息
type Config struct {
	ApiServerAddress string `json:"api_server_address"`
	Local            Local  `json:"local"`
	*entity.SysConfig
}

type Local struct {
	PublicIp []string `json:"public_ip"`
}

func Reload(ctx context.Context, sysConfig *entity.SysConfig) {

	if sysConfig.Core.ChannelPrefix == "" && Cfg.SysConfig != nil && Cfg.SysConfig.Core != nil {
		sysConfig.Core.ChannelPrefix = Cfg.SysConfig.Core.ChannelPrefix
	}

	Cfg.SysConfig = sysConfig

	// 重新加载配置文件的配置进行覆盖, 配置文件的配置优先级最高
	if data, err := gcfg.Instance().Data(ctx); err != nil {
		logger.Error(ctx, err)
	} else {
		if err = gjson.Unmarshal(gjson.MustEncode(data), &Cfg); err != nil {
			logger.Error(ctx, err)
		}
	}

	logger.Infof(ctx, "加载配置成功, 当前配置信息: %s", gjson.MustEncodeString(Cfg))
}

func Get(ctx context.Context, pattern string, def ...interface{}) (*gvar.Var, error) {

	value, err := g.Cfg().Get(ctx, pattern, def...)
	if err != nil {
		return nil, err
	}

	return value, nil
}

func GetString(ctx context.Context, pattern string, def ...interface{}) string {

	value, err := Get(ctx, pattern, def...)
	if err != nil {
		logger.Error(ctx, err)
	}

	return value.String()
}

func GetInt(ctx context.Context, pattern string, def ...interface{}) int {

	value, err := Get(ctx, pattern, def...)
	if err != nil {
		logger.Error(ctx, err)
	}

	return value.Int()
}

func GetBool(ctx context.Context, pattern string, def ...interface{}) (bool, error) {

	value, err := Get(ctx, pattern, def...)
	if err != nil {
		return false, err
	}

	return value.Bool(), nil
}

func GetMapStrStr(ctx context.Context, pattern string, def ...interface{}) map[string]string {

	value, err := Get(ctx, pattern, def...)
	if err != nil {
		logger.Error(ctx, err)
	}

	return value.MapStrStr()
}
