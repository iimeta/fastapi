package common

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/utility/logger"
)

func TokensFromMessages(ctx context.Context, model string, messages []smodel.ChatCompletionMessage) int {

	now := gtime.TimestampMilli()

	tokens, err := tiktoken.NumTokensFromMessages(model, messages)
	if err != nil {
		logger.Errorf(ctx, "TokensFromMessages model: %s, messages: %s, error: %v", model, gjson.MustEncodeString(messages), err)
		if tokens, err = tiktoken.NumTokensFromMessages(consts.DEFAULT_MODEL, messages); err != nil {
			logger.Errorf(ctx, "TokensFromMessages model: %s, messages: %s, error: %v", consts.DEFAULT_MODEL, gjson.MustEncodeString(messages), err)
		}
	}

	logger.Debugf(ctx, "TokensFromMessages model: %s, len(messages): %d, tokens: %d, time: %d", model, len(gjson.MustEncodeString(messages)), tokens, gtime.TimestampMilli()-now)

	return tokens
}

func TokensFromString(ctx context.Context, model, text string) int {

	now := gtime.TimestampMilli()

	tokens, err := tiktoken.NumTokensFromString(model, text)
	if err != nil {
		logger.Errorf(ctx, "TokensFromString model: %s, text: %s, error: %v", model, text, err)
		if tokens, err = tiktoken.NumTokensFromString(consts.DEFAULT_MODEL, text); err != nil {
			logger.Errorf(ctx, "TokensFromString model: %s, text: %s, error: %v", consts.DEFAULT_MODEL, text, err)
		}
	}

	logger.Debugf(ctx, "TokensFromString model: %s, len(text): %d, tokens: %d, time: %d", model, len(text), tokens, gtime.TimestampMilli()-now)

	return tokens
}
