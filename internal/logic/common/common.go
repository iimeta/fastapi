package common

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
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

	if IsAborted(err) {
		return false, false
	}

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

func GetPromptTokens(ctx context.Context, model string, messages []sdkm.ChatCompletionMessage) int {

	promptTime := gtime.TimestampMilli()

	promptTokens, err := tiktoken.NumTokensFromMessages(model, messages)
	if err != nil {
		logger.Errorf(ctx, "GetPromptTokens NumTokensFromMessages model: %s, messages: %s, error: %v", model, gjson.MustEncodeString(messages), err)
		if promptTokens, err = tiktoken.NumTokensFromMessages(consts.DEFAULT_MODEL, messages); err != nil {
			logger.Errorf(ctx, "GetPromptTokens NumTokensFromMessages model: %s, messages: %s, error: %v", consts.DEFAULT_MODEL, gjson.MustEncodeString(messages), err)
		}
	}
	logger.Debugf(ctx, "GetPromptTokens NumTokensFromMessages model: %s, len(messages): %d, promptTokens: %d, time: %d", model, len(gjson.MustEncodeString(messages)), promptTokens, gtime.TimestampMilli()-promptTime)

	return promptTokens
}

func GetCompletionTokens(ctx context.Context, model, completion string) int {

	completionTime := gtime.TimestampMilli()
	completionTokens, err := tiktoken.NumTokensFromString(model, completion)
	if err != nil {
		logger.Errorf(ctx, "GetCompletionTokens NumTokensFromString model: %s, completion: %s, error: %v", model, completion, err)
		if completionTokens, err = tiktoken.NumTokensFromString(consts.DEFAULT_MODEL, completion); err != nil {
			logger.Errorf(ctx, "GetCompletionTokens NumTokensFromString model: %s, completion: %s, error: %v", consts.DEFAULT_MODEL, completion, err)
		}
	}
	logger.Debugf(ctx, "GetCompletionTokens NumTokensFromString model: %s, len(completion): %d, completionTokens: %d, time: %d", model, len(completion), completionTokens, gtime.TimestampMilli()-completionTime)

	return completionTokens
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

func GetMultimodalTokens(ctx context.Context, model string, multiContent []interface{}, reqModel *model.Model) (textTokens, imageTokens int) {

	for _, value := range multiContent {

		content := value.(map[string]interface{})

		if content["type"] == "image_url" {

			imageUrl := content["image_url"].(map[string]interface{})
			detail := imageUrl["detail"]

			var imageQuota mcommon.ImageQuota
			for _, quota := range reqModel.MultimodalQuota.ImageQuotas {

				if quota.Mode == detail {
					imageQuota = quota
					break
				}

				if quota.IsDefault {
					imageQuota = quota
				}
			}

			imageTokens += imageQuota.FixedQuota

		} else {
			contentTime := gtime.TimestampMilli()
			tokens, err := tiktoken.NumTokensFromString(model, gconv.String(content))
			if err != nil {
				logger.Errorf(ctx, "GetMultimodalQuota NumTokensFromString model: %s, content: %s, error: %v", model, gconv.String(content), err)
				if tokens, err = tiktoken.NumTokensFromString(consts.DEFAULT_MODEL, gconv.String(content)); err != nil {
					logger.Errorf(ctx, "GetMultimodalQuota NumTokensFromString model: %s, content: %s, error: %v", consts.DEFAULT_MODEL, gconv.String(content), err)
				}
			}
			textTokens += tokens
			logger.Debugf(ctx, "GetMultimodalQuota NumTokensFromString model: %s, len(content): %d, tokens: %d, time: %d", model, len(gconv.String(content)), tokens, gtime.TimestampMilli()-contentTime)
		}
	}

	return textTokens, imageTokens
}

func GetMidjourneyQuota(model *model.Model, request *ghttp.Request, path string) (mcommon.MidjourneyQuota, error) {

	for _, quota := range model.MidjourneyQuotas {
		if quota.Path == path {
			return quota, nil
		}
	}

	return mcommon.MidjourneyQuota{}, errors.ERR_PATH_NOT_FOUND
}
