package common

import (
	"context"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"math"
	"net"
	"strings"
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

	secretKey = strings.TrimPrefix(secretKey, "sk-FastAPI")

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

	if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

		if model.IsEnableModelAgent {
			service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
			service.ModelAgent().RecordErrorModelAgent(ctx, model, modelAgent)
		} else {
			service.Key().RecordErrorModelKey(ctx, model, key)
		}

	}, nil); err != nil {
		logger.Error(ctx, err)
	}
}

func IsAborted(err error) bool {
	return errors.Is(err, context.Canceled) ||
		gstr.Contains(err.Error(), "broken pipe") ||
		gstr.Contains(err.Error(), "aborted")
}

func IsNeedRetry(err error) (isRetry bool, isDisabled bool) {

	apiError := &sdkerr.ApiError{}
	if errors.As(err, &apiError) {

		switch apiError.HttpStatusCode {
		case 400:
			if errors.Is(err, sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED) {
				return false, false
			}
		case 401, 429:
			if errors.Is(err, sdkerr.ERR_INVALID_API_KEY) || errors.Is(err, sdkerr.ERR_INSUFFICIENT_QUOTA) {
				return true, true
			}
		}

		return true, false
	}

	reqError := &sdkerr.RequestError{}
	if errors.As(err, &reqError) {
		return true, false
	}

	opError := &net.OpError{}
	if errors.As(err, &opError) {
		return true, false
	}

	// todo
	return true, false
}

func IsMaxRetry(isEnableModelAgent bool, agentTotal, keyTotal, retry int) bool {

	if config.Cfg.Api.Retry > 0 && retry == config.Cfg.Api.Retry {
		return true
	} else if config.Cfg.Api.Retry < 0 {
		if isEnableModelAgent {
			if retry == agentTotal {
				return true
			}
		} else if retry == keyTotal {
			return true
		}
	} else if config.Cfg.Api.Retry == 0 {
		return true
	}

	return false
}

func HandleMessages(messages []sdkm.ChatCompletionMessage) []sdkm.ChatCompletionMessage {

	var (
		newMessages = make([]sdkm.ChatCompletionMessage, 0)
	)

	for _, message := range messages {
		if message.Content != "" {
			newMessages = append(newMessages, message)
		}
	}

	return newMessages
}

func GetImageQuota(model *model.Model, size string) (imageQuota mcommon.ImageQuota) {

	var (
		width  int
		height int
	)

	if size != "" {

		widthHeight := gstr.Split(size, `×`)

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `x`)
		}

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `X`)
		}

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `*`)
		}

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `:`)
		}

		if len(widthHeight) == 2 {
			width = gconv.Int(widthHeight[0])
			height = gconv.Int(widthHeight[1])
		}
	}

	for _, quota := range model.ImageQuotas {

		if quota.Width == width && quota.Height == height {
			return quota
		}

		if quota.IsDefault {
			imageQuota = quota
		}
	}

	return imageQuota
}

func GetMultimodalTokens(ctx context.Context, model string, multiContent []interface{}, multimodalQuota mcommon.MultimodalQuota) (tokens int) {

	for _, value := range multiContent {

		content := value.(map[string]interface{})

		if content["type"] == "image_url" {

			imageUrl := content["image_url"].(map[string]interface{})
			detail := imageUrl["detail"]

			var imageQuota mcommon.ImageQuota
			for _, quota := range multimodalQuota.ImageQuotas {

				if quota.Mode == detail {
					imageQuota = quota
					break
				}

				if quota.IsDefault {
					imageQuota = quota
				}
			}

			tokens += imageQuota.FixedQuota

		} else {
			// 实际消费额度
			if textTokens, err := tiktoken.NumTokensFromString(model, gconv.String(content["text"])); err != nil {
				logger.Errorf(ctx, "GetMultimodalQuota model: %s, text: %s, NumTokensFromString error: %v", model, gconv.String(content["text"]), err)
			} else {
				tokens += int(math.Ceil(float64(textTokens) * multimodalQuota.TextQuota.PromptRatio))
			}
		}
	}

	return tokens
}

func GetMidjourneyQuota(model *model.Model, request *ghttp.Request, path string) (mcommon.MidjourneyQuota, error) {

	for _, quota := range model.MidjourneyQuotas {
		if quota.Path == path {
			return quota, nil
		}
	}

	return mcommon.MidjourneyQuota{}, errors.ERR_PATH_NOT_FOUND
}
