package util

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi-admin/internal/config"
	"github.com/iimeta/fastapi-admin/internal/errors"
	"github.com/iimeta/fastapi-admin/utility/logger"
	"github.com/iimeta/fastapi-admin/utility/redis"
	"strings"
)

type IpAddressResponse struct {
	Code   string `json:"resultcode"`
	Reason string `json:"reason"`
	Result struct {
		Country  string `json:"Country"`
		Province string `json:"Province"`
		City     string `json:"City"`
		Isp      string `json:"Isp"`
	} `json:"result"`
	ErrorCode int `json:"error_code"`
}

func FindAddress(ctx context.Context, ip string) (string, error) {

	if config.Cfg.App.JuheKey == "" {
		return "", nil
	}

	if val, err := getCache(ctx, ip); err == nil && val != "" {
		return val, nil
	}

	params := g.Map{
		"key": config.Cfg.App.JuheKey,
		"ip":  ip,
	}

	url := config.Cfg.App.JuheUrl
	if url == "" {
		url = "https://apis.juhe.cn/ip/ipNew"
	}

	data := &IpAddressResponse{}
	err := HttpGet(ctx, url, nil, params, &data)
	if err != nil {
		logger.Error(ctx, err)
		return "", err
	}

	if data.Code != "200" {
		logger.Error(ctx, data.Reason)
		return "", errors.New(data.Reason)
	}

	arr := []string{data.Result.Country, data.Result.Province, data.Result.City, data.Result.Isp}
	val := strings.Join(Unique(arr), " ")
	val = strings.TrimSpace(val)

	_ = setCache(ctx, ip, val)

	return val, nil
}

func getCache(ctx context.Context, ip string) (string, error) {
	return redis.HGetStr(ctx, "rds:hash:ip-address", ip)
}

func setCache(ctx context.Context, ip string, value string) error {
	_, err := redis.HSetStr(ctx, "rds:hash:ip-address", ip, value)
	return err
}
