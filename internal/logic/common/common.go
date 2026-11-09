package common

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
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

	if errors.Is(err, context.Canceled) {
		return true
	}

	// 上游请求错误(*url.Error), 如 broken pipe/connection reset 等是上游连接问题, 不属于客户端中止
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return errors.Is(err, context.DeadlineExceeded) || gstr.Contains(err.Error(), "context deadline exceeded")
	}

	return gstr.Contains(err.Error(), "context deadline exceeded") ||
		gstr.Contains(err.Error(), "broken pipe") ||
		gstr.Contains(err.Error(), "aborted")
}

// 是否为客户端原因导致的中止
// 上游原因(连接断开/错误)或请求超时导致的中止不属于客户端中止
func IsClientAborted(err error) bool {

	if err == nil {
		return false
	}

	// 客户端断开导致写响应失败
	if errors.Is(err, errors.ERR_CLIENT_ABORTED) {
		return true
	}

	// 超时(上游请求超时/Client.Timeout)属于系统原因
	if isTimeout(err) {
		return false
	}

	// 请求上下文被取消(客户端断开连接/主动取消请求)
	return errors.Is(err, context.Canceled)
}

// 中止的请求是否需要计费
//  1. 客户端原因中止: 计费
//  2. 超时中止, 但上游已正常响应并返回了usage(如流式响应过久直到长连接超时): 计费
//  3. 超时中止, 但上游已返回了生成结果(如流式生图已返回完成的图像): 计费
//  4. 其它上游原因中止, 或超时但上游未返回任何usage/结果: 不计费
func IsBillableAborted(err error, usage *smodel.Usage, hasOutput ...bool) bool {

	if IsClientAborted(err) {
		return true
	}

	if !isTimeout(err) {
		return false
	}

	return hasUsage(usage) || (len(hasOutput) > 0 && hasOutput[0])
}

// 是否为超时错误
func isTimeout(err error) bool {

	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return gstr.Contains(err.Error(), "deadline exceeded") ||
		gstr.Contains(err.Error(), "Client.Timeout") ||
		gstr.Contains(err.Error(), "i/o timeout")
}

// 上游是否返回了有效usage(任一token计数或官方价格)
func hasUsage(usage *smodel.Usage) bool {
	return usage != nil && (usage.CostInUsdTicks > 0 || usageHasTokenBreakdown(usage))
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
	return checkIp(util.GetClientIpFromCtx(ctx), ipWhitelist, ipBlacklist)
}

func checkIp(clientIp string, ipWhitelist, ipBlacklist []string) error {

	// 仅真实 clientIp 为本机时放行, RemoteIp 为本机只说明经过本机反代, 不能作为放行依据.
	if util.IsLocalIp(clientIp) {
		return nil
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

// 是否为流式生图的中间预览图事件(如 image_generation.partial_image / image_edit.partial_image), 预览图不属于最终生成结果
func IsPartialImageEvent(event string, responseBytes []byte) bool {

	if gstr.Contains(event, "partial_image") {
		return true
	}

	if len(responseBytes) == 0 || !bytes.Contains(responseBytes, []byte("partial_image")) {
		return false
	}

	return gstr.Contains(gjson.New(responseBytes).Get("type").String(), "partial_image")
}

// 统计流式生图本次事件返回的最终图像数量(排除中间预览图和无图像内容的数据, 如失败项)
func CountImageOutputs(event string, responseBytes []byte, data []smodel.ImageResponseData) (n int) {

	if IsPartialImageEvent(event, responseBytes) {
		return 0
	}

	return CountImages(data)
}

// 统计上游实际返回的图像数量(仅统计有图像内容的数据, 排除失败项)
func CountImages(data []smodel.ImageResponseData) (n int) {

	for _, item := range data {
		if item.Url != "" || item.B64Json != "" ||
			(item.FileOutput != nil && (item.FileOutput.PublicUrl != "" || item.FileOutput.FileId != "")) {
			n++
		}
	}

	return n
}
