package common

import (
	"context"
	"math"

	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
)

func SpendHandler(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model, completion string, response sdkm.ChatCompletionResponse) {

	var (
		textTokens  int
		imageTokens int
		audioTokens int
		totalTokens int
	)

	if mak.ReqModel.Type == 100 { // 多模态

		if response.Usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

			textTokens, imageTokens, totalTokens = MultimodalTokens(ctx, mak, usage, params, model, response)

		} else {
			totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))
		}

		if params.Tools != nil {
			if tools := gconv.String(params.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
				totalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
				response.Usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
			}
		}

		if params.WebSearchOptions != nil {
			searchTokens := GetMultimodalSearchTokens(ctx, params.WebSearchOptions, mak.ReqModel)
			totalTokens += searchTokens
			response.Usage.SearchTokens = searchTokens
		}

		if response.Usage.CacheCreationInputTokens != 0 {
			totalTokens += int(math.Ceil(float64(response.Usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
		}

		if response.Usage.CacheReadInputTokens != 0 {
			totalTokens += int(math.Ceil(float64(response.Usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
		}

		if response.Usage.PromptTokensDetails.CachedTokens != 0 {
			totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
		}

		if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
			totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
		}

	} else if mak.ReqModel.Type == 102 { // 多模态语音

		textTokens, audioTokens, totalTokens = MultimodalAudioTokens(ctx, mak, usage, params, model, response)

	} else if response.Usage == nil || response.Usage.TotalTokens == 0 {

		response.Usage = new(sdkm.Usage)

		response.Usage.PromptTokens = GetPromptTokens(ctx, model, params.Messages)

		if len(response.Choices) > 0 && response.Choices[0].Message != nil {
			for _, choice := range response.Choices {
				response.Usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
			}
		}

		response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
	}

	if mak.ReqModel != nil && response.Usage != nil {
		if mak.ReqModel.Type == 102 {

			if response.Usage.PromptTokensDetails.TextTokens > 0 {
				textTokens = int(math.Ceil(float64(response.Usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
			}

			if response.Usage.PromptTokensDetails.AudioTokens > 0 {
				audioTokens = int(math.Ceil(float64(response.Usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
			} else {
				audioTokens = int(math.Ceil(float64(response.Usage.PromptTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
			}

			if response.Usage.CompletionTokensDetails.TextTokens > 0 {
				textTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
			}

			if response.Usage.CompletionTokensDetails.AudioTokens > 0 {
				audioTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
			} else {
				audioTokens += int(math.Ceil(float64(response.Usage.CompletionTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
			}

			totalTokens = textTokens + audioTokens

			if response.Usage.PromptTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
			}

			if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
			}

		} else if mak.ReqModel.Type != 100 {
			if mak.ReqModel.TextQuota.BillingMethod == 1 {

				totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(response.Usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

				if response.Usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

				if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

			} else {
				totalTokens = mak.ReqModel.TextQuota.FixedQuota
			}
		}
	}

	////////////////////////

	if completion != "" && mak.ReqModel != nil && (usage == nil || usage.PromptTokens == 0 || usage.CompletionTokens == 0 || (mak.ReqModel.Type == 100 && mak.ReqModel.MultimodalQuota.BillingRule == 2)) {

		if usage == nil {
			usage = new(sdkm.Usage)
		}

		if mak.ReqModel.Type == 102 { // 多模态语音
			textTokens, audioTokens = GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
			usage.PromptTokens = textTokens + audioTokens
		} else {
			if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
				textTokens, imageTokens = GetMultimodalTokens(ctx, model, content, mak.ReqModel)
				usage.PromptTokens = textTokens + imageTokens
			} else {
				if usage.PromptTokens == 0 || mak.ReqModel.MultimodalQuota.BillingRule == 2 {
					usage.PromptTokens = GetPromptTokens(ctx, model, params.Messages)
				}
				textTokens = usage.PromptTokens
			}
		}

		if usage.CompletionTokens == 0 {
			usage.CompletionTokens = GetCompletionTokens(ctx, model, completion)
			if mak.ReqModel.Type == 102 { // 多模态语音
				usage.CompletionTokens += 388
			}
		}

		if mak.ReqModel.Type == 100 { // 多模态

			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

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

		} else if mak.ReqModel.Type == 102 { // 多模态语音

			totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

			if usage.PromptTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
			}

			if usage.CompletionTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
			}

		} else {
			if mak.ReqModel.TextQuota.BillingMethod == 1 {

				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

				if usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

				if usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

			} else {
				usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
				totalTokens = mak.ReqModel.TextQuota.FixedQuota
			}
		}

	} else if usage != nil && mak.ReqModel != nil {

		if mak.ReqModel.Type == 100 { // 多模态

			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

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

		} else if mak.ReqModel.Type == 102 { // 多模态语音

			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

			if usage.PromptTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
			}

			if usage.CompletionTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
			}

		} else {
			if mak.ReqModel.TextQuota.BillingMethod == 1 {

				usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

				if usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

				if usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

			} else {
				usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
				totalTokens = mak.ReqModel.TextQuota.FixedQuota
			}
		}
	}
}

func MultimodalTokens(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model string, response sdkm.ChatCompletionResponse) (textTokens, imageTokens, totalTokens int) {

	response.Usage = new(sdkm.Usage)

	if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
		textTokens, imageTokens = GetMultimodalTokens(ctx, model, content, mak.ReqModel)
		response.Usage.PromptTokens = textTokens + imageTokens
	} else {
		if response.Usage.PromptTokens == 0 {
			textTokens = GetPromptTokens(ctx, model, params.Messages)
			response.Usage.PromptTokens = textTokens
		}
	}

	if response.Usage.CompletionTokens == 0 && len(response.Choices) > 0 && response.Choices[0].Message != nil {
		for _, choice := range response.Choices {
			response.Usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
		}
	}

	response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
	totalTokens = imageTokens + int(math.Ceil(float64(textTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

	return
}

func MultimodalAudioTokens(ctx context.Context, mak *MAK, usage *sdkm.Usage, params sdkm.ChatCompletionRequest, model string, response sdkm.ChatCompletionResponse) (textTokens, audioTokens, totalTokens int) {

	if response.Usage == nil {

		response.Usage = new(sdkm.Usage)

		textTokens, audioTokens = GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
		response.Usage.PromptTokens = textTokens + audioTokens

		if len(response.Choices) > 0 && response.Choices[0].Message != nil && response.Choices[0].Message.Audio != nil {
			for _, choice := range response.Choices {
				response.Usage.CompletionTokens += GetCompletionTokens(ctx, model, choice.Message.Audio.Transcript) + 388
			}
		}
	}

	response.Usage.TotalTokens = response.Usage.PromptTokens + response.Usage.CompletionTokens
	totalTokens = int(math.Ceil(float64(response.Usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(response.Usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

	if response.Usage.PromptTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(response.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
	}

	if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
		totalTokens += int(math.Ceil(float64(response.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
	}

	return
}
