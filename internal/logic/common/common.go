package common

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
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

	if modelAgent != nil {
		service.Session().RecordErrorModelAgent(ctx, modelAgent.Id)
	}

	if key != nil {
		service.Session().RecordErrorKey(ctx, key.Id)
	}

	if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
		if model.IsEnableModelAgent {
			service.ModelAgent().RecordErrorKey(ctx, modelAgent, key)
			service.ModelAgent().RecordError(ctx, model, modelAgent)
		} else {
			service.Key().RecordError(ctx, model, key)
		}
	}, nil); err != nil {
		logger.Error(ctx, err)
	}
}

func IsAborted(err error) bool {
	return errors.Is(err, context.Canceled) ||
		gstr.Contains(err.Error(), "context deadline exceeded") ||
		gstr.Contains(err.Error(), "broken pipe") ||
		gstr.Contains(err.Error(), "aborted")
}

func IsNeedRetry(err error) (isRetry bool, isDisabled bool) {

	if IsAborted(err) {
		return false, false
	}

	// 自动禁用错误
	if config.Cfg.AutoDisabledError.Open && len(config.Cfg.AutoDisabledError.Errors) > 0 {
		for _, autoDisabledError := range config.Cfg.AutoDisabledError.Errors {
			if gstr.Contains(err.Error(), autoDisabledError) {
				return true, true
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

	return true, false
}

func IsMaxRetry(isEnableModelAgent bool, agentTotal, keyTotal, retry int) bool {

	if config.Cfg.Base.ErrRetry > 0 && retry == config.Cfg.Base.ErrRetry {
		return true
	} else if config.Cfg.Base.ErrRetry < 0 {
		if isEnableModelAgent {
			if retry == agentTotal {
				return true
			}
		} else if retry == keyTotal {
			return true
		}
	} else if config.Cfg.Base.ErrRetry == 0 {
		return true
	}

	return false
}

func HandleMessages(messages []smodel.ChatCompletionMessage) []smodel.ChatCompletionMessage {

	var (
		newMessages = make([]smodel.ChatCompletionMessage, 0)
	)

	for _, message := range messages {
		if message.Content != "" || message.Role != sconsts.ROLE_USER {
			newMessages = append(newMessages, message)
		}
	}

	return newMessages
}

func CheckIp(ctx context.Context, ipWhitelist, ipBlacklist []string) error {

	clientIp := g.RequestFromCtx(ctx).GetClientIp()

	if clientIp == "127.0.0.1" || clientIp == "::1" {
		return nil
	}

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			// 检查是否为IP地址, 而不是其他类型的地址(例如MAC地址)
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

		return errors.NewError(403, "fastapi_error", fmt.Sprintf("IP: %s Forbidden.", g.RequestFromCtx(ctx).GetClientIp()), "fastapi_error", nil)
	}

	return nil
}
