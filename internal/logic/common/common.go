package common

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

type sCommon struct{}

func init() {
	service.RegisterCommon(New())
}

func New() service.ICommon {
	return &sCommon{}
}

// 解析密钥
func (s *sCommon) ParseSecretKey(ctx context.Context, secretKey string) (int, int, error) {

	if !gstr.HasPrefix(secretKey, config.Cfg.Core.SecretKeyPrefix) {
		return 0, 0, errors.ERR_INVALID_API_KEY
	}

	secretKey = strings.TrimPrefix(secretKey, config.Cfg.Core.SecretKeyPrefix)

	userId, err := gregex.ReplaceString("[a-zA-Z-]*", "", secretKey[:len(secretKey)/2])
	if err != nil {
		logger.Error(ctx, err)
		return 0, 0, err
	}

	appId, err := gregex.ReplaceString("[a-zA-Z-]*", "", secretKey[len(secretKey)/2:])
	if err != nil {
		logger.Error(ctx, err)
		return 0, 0, err
	}

	return gconv.Int(userId), gconv.Int(appId), nil
}

// 记录错误次数和禁用
func (s *sCommon) RecordError(ctx context.Context, model *model.Model, key *model.Key, modelAgent *model.ModelAgent) {

	sessionKeepHit := service.Session().GetSessionKeepHit(ctx)

	if modelAgent != nil && !sessionKeepHit {
		service.Session().RecordErrorModelAgent(ctx, modelAgent.Id)
	}

	if key != nil {
		service.Session().RecordErrorKey(ctx, key.Id)
	}

	if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

		service.ModelAgent().RecordErrorKey(ctx, modelAgent, key)

		if !sessionKeepHit {
			service.ModelAgent().RecordError(ctx, model, modelAgent)
		}

	}, nil); err != nil {
		logger.Error(ctx, err)
	}
}

func IsAborted(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) ||
		gstr.Contains(err.Error(), "context deadline exceeded") ||
		gstr.Contains(err.Error(), "broken pipe") ||
		gstr.Contains(err.Error(), "aborted")
}

func IsNeedRetry(err error) (isRetry bool, isDisabled bool) {

	if IsAborted(err) {
		return false, false
	}

	defer func() {
		// 自动重试错误
		if isRetry && len(config.Cfg.AutoRetryError.Errors) > 0 {
			isRetry = false
			for _, autoRetryError := range config.Cfg.AutoRetryError.Errors {
				if gstr.Contains(err.Error(), autoRetryError) {
					isRetry = true
					return
				}
			}
		}
	}()

	// 自动禁用错误
	if config.Cfg.AutoDisabledError.Open && len(config.Cfg.AutoDisabledError.Errors) > 0 {
		for _, autoDisabledError := range config.Cfg.AutoDisabledError.Errors {
			if gstr.Contains(err.Error(), autoDisabledError) {
				return config.Cfg.AutoRetryError.Open, true
			}
		}
	}

	// 不重试错误
	if config.Cfg.NotRetryError.Open && len(config.Cfg.NotRetryError.Errors) > 0 {
		for _, notRetryError := range config.Cfg.NotRetryError.Errors {
			if gstr.Contains(err.Error(), notRetryError) {
				return false, false
			}
		}
	}

	return config.Cfg.AutoRetryError.Open, false
}

func IsMaxRetry(agentTotal, retry int) bool {

	if config.Cfg.Base.ErrRetry > 0 && retry == config.Cfg.Base.ErrRetry {
		return true
	} else if config.Cfg.Base.ErrRetry < 0 && retry == agentTotal {
		return true
	} else if config.Cfg.Base.ErrRetry == 0 {
		return true
	}

	return false
}

func CheckIp(ctx context.Context, ipWhitelist, ipBlacklist []string) error {

	clientIp := g.RequestFromCtx(ctx).GetClientIp()
	remoteIp := g.RequestFromCtx(ctx).GetRemoteIp()

	err := checkIp(clientIp, ipWhitelist, ipBlacklist)
	if err == nil {
		return nil
	}

	if remoteIp != "" && remoteIp != clientIp {
		if checkIp(remoteIp, ipWhitelist, ipBlacklist) == nil {
			return nil
		}
	}

	return err
}

func checkIp(clientIp string, ipWhitelist, ipBlacklist []string) error {

	if clientIp == "127.0.0.1" || clientIp == "::1" {
		return nil
	}

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil && clientIp == ipnet.IP.String() {
					return nil
				}
			}
		}
	}

	if (len(ipBlacklist) > 0 && ipBlacklist[0] != "") || len(ipBlacklist) > 1 {

		for _, blacklist := range ipBlacklist {

			if blacklist == "" {
				continue
			}

			if blacklist == clientIp {
				return errors.ERR_FORBIDDEN
			}

			if gstr.Contains(blacklist, "/") {

				_, ipNet, err := net.ParseCIDR(blacklist)
				if err != nil {
					return err
				}

				if ipNet.Contains(net.ParseIP(clientIp)) {
					return errors.ERR_FORBIDDEN
				}
			}

			if gstr.Contains(blacklist, "-") {

				ipRange := gstr.Split(blacklist, "-")

				ipStart := net.ParseIP(ipRange[0])
				ipEnd := net.ParseIP(ipRange[1])
				ip := net.ParseIP(clientIp)

				if bytes.Compare(ip, ipStart) >= 0 && bytes.Compare(ip, ipEnd) <= 0 {
					return errors.ERR_FORBIDDEN
				}
			}
		}
	}

	if (len(ipWhitelist) > 0 && ipWhitelist[0] != "") || len(ipWhitelist) > 1 {

		if slices.Contains(ipWhitelist, "0.0.0.0") {
			return nil
		}

		for _, whitelist := range ipWhitelist {

			if whitelist == "" {
				continue
			}

			if whitelist == clientIp {
				return nil
			}

			if gstr.Contains(whitelist, "/") {

				_, ipNet, err := net.ParseCIDR(whitelist)
				if err != nil {
					return err
				}

				if ipNet.Contains(net.ParseIP(clientIp)) {
					return nil
				}
			}

			if gstr.Contains(whitelist, "-") {

				ipRange := gstr.Split(whitelist, "-")

				ipStart := net.ParseIP(ipRange[0])
				ipEnd := net.ParseIP(ipRange[1])
				ip := net.ParseIP(clientIp)

				if bytes.Compare(ip, ipStart) >= 0 && bytes.Compare(ip, ipEnd) <= 0 {
					return nil
				}
			}
		}

		return errors.NewError(403, "fastapi_error", fmt.Sprintf("IP: %s Forbidden.", clientIp), "fastapi_error", nil)
	}

	return nil
}

// 记录命中的图像生成定价, 空模式按尺寸写入日志
func recordImageGenerationPricing(spend *common.Spend, pricing *common.ImageGenerationPricing) {

	if spend.ImageGeneration == nil {
		spend.ImageGeneration = new(common.ImageGenerationSpend)
	}

	if pricing == nil {
		return
	}

	p := *pricing
	if p.Mode == "" {
		p.Mode = "size"
	}

	spend.ImageGeneration.Pricing = &p
}

// 记录命中的图层拆分定价, 空模式按尺寸写入日志
func recordLayerDecompPricing(spend *common.Spend, pricing *common.LayerDecompPricing) {

	if spend.LayerDecomp == nil {
		spend.LayerDecomp = new(common.LayerDecompSpend)
	}

	if pricing == nil {
		return
	}

	p := *pricing
	if p.Mode == "" {
		p.Mode = "size"
	}

	spend.LayerDecomp.Pricing = &p
}

// 按像素区间匹配图像生成价格, 区间为 [gte, lte], 0 表示该侧不限制
func matchImageGenerationPixel(pixels int, pricing *common.ImageGenerationPricing) bool {

	if pricing == nil {
		return false
	}

	return matchPixelRange(pixels, pricing.PixelGte, pricing.PixelLte)
}

// 按像素区间匹配图层拆分价格, 区间为 [gte, lte], 0 表示该侧不限制
func matchLayerDecompPixel(pixels int, pricing *common.LayerDecompPricing) bool {

	if pricing == nil {
		return false
	}

	return matchPixelRange(pixels, pricing.PixelGte, pricing.PixelLte)
}

func matchPixelRange(pixels int, pixelGte, pixelLte string) bool {

	if pixels <= 0 {
		return false
	}

	gte := parsePixelSize(pixelGte)
	lte := parsePixelSize(pixelLte)

	if gte > 0 && pixels < gte {
		return false
	}

	if lte > 0 && pixels > lte {
		return false
	}

	return true
}

// 解析像素配置, 支持 1024x1024 / 1024×1024 / 1024*1024 或直接填像素个数
func parsePixelSize(size string) int {

	size = gstr.Trim(size)
	if size == "" {
		return 0
	}

	size = gstr.ReplaceByMap(size, map[string]string{
		"×": "x",
		"X": "x",
		"*": "x",
	})

	parts := gstr.Split(size, "x")

	if len(parts) >= 2 {
		w := gconv.Int(gstr.Trim(parts[0]))
		h := gconv.Int(gstr.Trim(parts[1]))
		if w > 0 && h > 0 {
			return w * h
		}
	}

	return gconv.Int(size)
}
