package common

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/utility/logger"
)

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

func GetMultimodalTokens(ctx context.Context, model string, multiContent []interface{}, reqModel *model.Model) (textTokens, imageTokens int) {

	for _, value := range multiContent {

		if content, ok := value.(map[string]interface{}); ok && content["type"] == "image_url" {

			if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

				detail := imageUrl["detail"]

				var visionQuota mcommon.VisionQuota
				for _, quota := range reqModel.MultimodalQuota.VisionQuotas {

					if quota.Mode == detail {
						visionQuota = quota
						break
					}

					if quota.IsDefault {
						visionQuota = quota
					}
				}

				imageTokens += visionQuota.FixedQuota
			}

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

func GetMultimodalAudioTokens(ctx context.Context, model string, messages []sdkm.ChatCompletionMessage, reqModel *model.Model) (textTokens, audioTokens int) {

	var text string

	for _, message := range messages {
		if multiContent, ok := message.Content.([]interface{}); ok {
			for _, value := range multiContent {
				if content, ok := value.(map[string]interface{}); ok {
					if content["type"] == "text" {
						text += gconv.String(content["text"])
					}
				}
			}
		} else {
			text += gconv.String(message.Content)
		}
	}

	contentTime := gtime.TimestampMilli()

	tokens, err := tiktoken.NumTokensFromString(model, text)
	if err != nil {
		logger.Errorf(ctx, "GetMultimodalAudioTokens NumTokensFromString model: %s, content: %s, error: %v", model, text, err)
		if tokens, err = tiktoken.NumTokensFromString(consts.DEFAULT_MODEL, text); err != nil {
			logger.Errorf(ctx, "GetMultimodalAudioTokens NumTokensFromString model: %s, content: %s, error: %v", consts.DEFAULT_MODEL, text, err)
		}
	}

	textTokens += tokens

	logger.Debugf(ctx, "GetMultimodalAudioTokens NumTokensFromString model: %s, len(content): %d, tokens: %d, time: %d", model, len(text), tokens, gtime.TimestampMilli()-contentTime)

	return textTokens, 888
}

func GetMultimodalSearchTokens(ctx context.Context, webSearchOptions any, reqModel *model.Model) (searchTokens int) {

	var searchContextSize string
	if content, ok := webSearchOptions.(map[string]interface{}); ok {
		searchContextSize = gconv.String(content["search_context_size"])
	}

	for _, size := range reqModel.MultimodalQuota.SearchQuotas {

		if size.SearchContextSize == searchContextSize {
			searchTokens = size.FixedQuota
			break
		}

		if size.IsDefault {
			searchTokens = size.FixedQuota
		}
	}

	return searchTokens
}
