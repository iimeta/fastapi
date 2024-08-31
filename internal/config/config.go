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
	"github.com/iimeta/fastapi/utility/logger"
	"time"
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
	Core             Core       `json:"core"`
	ApiServerAddress string     `json:"api_server_address"`
	Http             Http       `json:"http"`
	Local            Local      `json:"local"`
	Api              Api        `json:"api"`
	Midjourney       Midjourney `json:"midjourney"`
	Gcp              Gcp        `json:"gcp"`
	RecordLogs       []string   `json:"record_logs"`
	Error            Error      `json:"error"`
	Debug            bool       `json:"debug"`
}

type Core struct {
	ChannelPrefix string `json:"channel_prefix"`
}

type Api struct {
	Retry                   int   `json:"retry"`
	ModelKeyErrDisable      int64 `json:"model_key_err_disable"`
	ModelAgentErrDisable    int64 `json:"model_agent_err_disable"`
	ModelAgentKeyErrDisable int64 `json:"model_agent_key_err_disable"`
}

type Http struct {
	Timeout  time.Duration `json:"timeout"`
	ProxyUrl string        `json:"proxy_url"`
}

type Local struct {
	PublicIp []string `json:"public_ip"`
}

type Midjourney struct {
	CdnUrl          string          `json:"cdn_url"`
	MidjourneyProxy MidjourneyProxy `json:"midjourney_proxy"`
}

type MidjourneyProxy struct {
	ApiBaseUrl      string `json:"api_base_url"`
	ApiSecret       string `json:"api_secret"`
	ApiSecretHeader string `json:"api_secret_header"`
	CdnOriginalUrl  string `json:"cdn_original_url"`
}

type Gcp struct {
	GetTokenUrl string `json:"get_token_url" d:"https://www.googleapis.com/oauth2/v4/token"`
}

type Error struct {
	AutoDisabled []string `json:"auto_disabled"`
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
