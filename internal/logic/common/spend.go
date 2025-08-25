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

		if usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

			usage = new(sdkm.Usage)

			if content, ok := params.Messages[len(params.Messages)-1].Content.([]interface{}); ok {
				textTokens, imageTokens = GetMultimodalTokens(ctx, model, content, mak.ReqModel)
				usage.PromptTokens = textTokens + imageTokens
			} else {
				if usage.PromptTokens == 0 {
					textTokens = GetPromptTokens(ctx, model, params.Messages)
					usage.PromptTokens = textTokens
				}
			}

			if usage.CompletionTokens == 0 && len(response.Choices) > 0 && response.Choices[0].Message != nil {
				for _, choice := range response.Choices {
					usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
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

	} else if mak.ReqModel.Type == 102 { // 多模态语音

		if usage == nil {

			usage = new(sdkm.Usage)

			textTokens, audioTokens = GetMultimodalAudioTokens(ctx, model, params.Messages, mak.ReqModel)
			usage.PromptTokens = textTokens + audioTokens

			if len(response.Choices) > 0 && response.Choices[0].Message != nil && response.Choices[0].Message.Audio != nil {
				for _, choice := range response.Choices {
					usage.CompletionTokens += GetCompletionTokens(ctx, model, choice.Message.Audio.Transcript) + 388
				}
			}
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
		totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))

		if usage.PromptTokensDetails.CachedTokens != 0 {
			totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
		}

		if usage.CompletionTokensDetails.CachedTokens != 0 {
			totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CachedRatio))
		}

	} else if usage == nil || usage.TotalTokens == 0 {

		usage = new(sdkm.Usage)

		usage.PromptTokens = GetPromptTokens(ctx, model, params.Messages)

		if len(response.Choices) > 0 && response.Choices[0].Message != nil {
			for _, choice := range response.Choices {
				usage.CompletionTokens += GetCompletionTokens(ctx, model, gconv.String(choice.Message.Content))
			}
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	if mak.ReqModel != nil && usage != nil {
		if mak.ReqModel.Type == 102 {

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

			totalTokens = textTokens + audioTokens

			if usage.PromptTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
			}

			if usage.CompletionTokensDetails.CachedTokens != 0 {
				totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
			}

		} else if mak.ReqModel.Type != 100 {
			if mak.ReqModel.TextQuota.BillingMethod == 1 {

				totalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

				if usage.PromptTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

				if usage.CompletionTokensDetails.CachedTokens != 0 {
					totalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
				}

			} else {
				totalTokens = mak.ReqModel.TextQuota.FixedQuota
			}
		}
	}

	////////////////////////

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

	if usage != nil && mak.ReqModel != nil {

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
