package common

import (
	"context"
	"math"

	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

func SpendHandler(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model, completion string, response sdkm.ChatCompletionResponse) (textTokens, imageTokens, audioTokens, totalTokens int) {

	switch mak.ReqModel.Type {
	case 100: // 多模态
		textTokens, imageTokens, totalTokens = MultimodalTokens(ctx, mak, usage, params, model, completion, response)
	case 102: // 多模态语音
		textTokens, audioTokens, totalTokens = MultimodalAudioTokens(ctx, mak, usage, params, model, completion, response)
	default:
		totalTokens = TextTokens(ctx, mak, usage, params, model, completion, response)
	}

	return
}

func MultimodalTokens(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model, completion string, response sdkm.ChatCompletionResponse) (textTokens, imageTokens, totalTokens int) {

	if usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

		usage = new(sdkm.Usage)

		if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
			textTokens, imageTokens = GetMultimodalTokens(ctx, model, content, mak.ReqModel)
			usage.PromptTokens = textTokens + imageTokens
		} else {
			if usage.PromptTokens == 0 || mak.ReqModel.MultimodalQuota.BillingRule == 2 {
				usage.PromptTokens = GetPromptTokens(ctx, model, params.Messages)
			}
			textTokens = usage.PromptTokens
		}

		if usage.CompletionTokens == 0 {

			usage.CompletionTokens = GetCompletionTokens(ctx, model, completion)

			if len(response.Choices) > 0 && response.Choices[0].Message != nil {
				for _, choice := range response.Choices {
					usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
				}
			}
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

	} else {
		totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))
	}

	if params.Tools != nil {
		if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
			totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
			usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
		}
	}

	if params.WebSearchOptions != nil {
		searchTokens := GetMultimodalSearchTokens(ctx, params.WebSearchOptions, mak.ReqModel)
		totalTokens += searchTokens
		usage.SearchTokens = searchTokens
	}

	if usage.CacheCreationInputTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
	}

	if usage.CacheReadInputTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
	}

	if usage.PromptTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
	}

	return
}

func MultimodalAudioTokens(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model, completion string, response sdkm.ChatCompletionResponse) (textTokens, audioTokens, totalTokens int) {

	if usage == nil {

		usage = new(sdkm.Usage)

		textTokens, audioTokens = GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
		usage.PromptTokens = textTokens + audioTokens

		usage.CompletionTokens = GetCompletionTokens(ctx, model, completion) + 388

		if len(response.Choices) > 0 && response.Choices[0].Message != nil {
			for _, choice := range response.Choices {
				usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
				usage.CompletionTokens += GetCompletionTokens(ctx, model, choice.Message.Audio.Transcript) + 388
			}
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

	if usage.PromptTokensDetails.TextTokens > 0 {
		textTokens = int(math.Ceil(float64(usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
	}

	if usage.PromptTokensDetails.AudioTokens > 0 {
		audioTokens = int(math.Ceil(float64(usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
	} else {
		audioTokens = int(math.Ceil(float64(usage.PromptTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
	}

	if usage.CompletionTokensDetails.TextTokens > 0 {
		textTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
	}

	if usage.CompletionTokensDetails.AudioTokens > 0 {
		audioTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
	} else {
		audioTokens += int(math.Ceil(float64(usage.CompletionTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
	}

	if usage.PromptTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
	}

	return
}

func TextTokens(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model, completion string, response sdkm.ChatCompletionResponse) (totalTokens int) {

	if usage == nil || usage.TotalTokens == 0 {

		usage = new(sdkm.Usage)

		usage.PromptTokens = GetPromptTokens(ctx, model, params.Messages)
		usage.CompletionTokens = GetCompletionTokens(ctx, model, completion)

		if len(response.Choices) > 0 && response.Choices[0].Message != nil {
			for _, choice := range response.Choices {
				usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
			}
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	if mak.ReqModel.TextQuota.BillingMethod == 2 {
		usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
		totalTokens = mak.ReqModel.TextQuota.FixedQuota
		return
	}

	totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

	if usage.PromptTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
	}

	return
}
