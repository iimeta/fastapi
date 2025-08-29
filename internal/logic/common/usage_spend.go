package common

import (
	"context"
	"math"

	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/tiktoken-go"
)

func ChatUsageSpend(ctx context.Context, request sdkm.ChatCompletionRequest, completion string, usage *sdkm.Usage, mak *MAK) (usageSpend common.UsageSpend) {

	switch mak.ReqModel.Type {
	case 100: // 多模态
		usageSpend = MultimodalTokens(ctx, request, completion, usage, mak)
	case 102: // 多模态语音
		usageSpend = MultimodalAudioTokens(ctx, request, completion, usage, mak)
	default: // 文生文
		usageSpend = TextTokens(ctx, request, completion, usage, mak)
	}

	return
}

// 多模态
func MultimodalTokens(ctx context.Context, request sdkm.ChatCompletionRequest, completion string, usage *sdkm.Usage, mak *MAK) (usageSpend common.UsageSpend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

		usage = new(sdkm.Usage)

		if content, ok := request.Messages[len(request.Messages)-1].Content.([]interface{}); ok {
			usageSpend.TextTokens, usageSpend.ImageTokens = GetMultimodalTokens(ctx, model, content, mak.ReqModel)
			usage.PromptTokens = usageSpend.TextTokens + usageSpend.ImageTokens
		} else {
			usage.PromptTokens = GetPromptTokens(ctx, model, request.Messages)
			usageSpend.TextTokens = usage.PromptTokens
		}

		if completion != "" {
			usage.CompletionTokens = GetCompletionTokens(ctx, model, completion)
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens

		usageSpend.TotalTokens = usageSpend.ImageTokens + int(math.Ceil(float64(usageSpend.TextTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))

	} else {
		usageSpend.TotalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio)) + int(math.Ceil(float64(usage.CompletionTokens)*mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio))
	}

	if request.Tools != nil {
		if tools := gconv.String(request.Tools); gstr.Contains(tools, "google_search") || gstr.Contains(tools, "googleSearch") {
			usageSpend.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
			usageSpend.TotalTokens += mak.ReqModel.MultimodalQuota.SearchQuota
			usage.SearchTokens = mak.ReqModel.MultimodalQuota.SearchQuota
		}
	}

	if request.WebSearchOptions != nil {
		searchTokens := GetMultimodalSearchTokens(ctx, request.WebSearchOptions, mak.ReqModel)
		usageSpend.SearchTokens = searchTokens
		usageSpend.TotalTokens += searchTokens
		usage.SearchTokens = searchTokens
	}

	if usage.PromptTokensDetails.CachedTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CachedRatio))
	}

	// Claude
	if usage.CacheCreationInputTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.CacheCreationInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.PromptRatio * 1.25))
	}

	// Claude
	if usage.CacheReadInputTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.CacheReadInputTokens) * mak.ReqModel.MultimodalQuota.TextQuota.CompletionRatio * 0.1))
	}

	usageSpend.Usage = usage

	return
}

// 多模态语音
func MultimodalAudioTokens(ctx context.Context, request sdkm.ChatCompletionRequest, completion string, usage *sdkm.Usage, mak *MAK) (usageSpend common.UsageSpend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil {

		usage = new(sdkm.Usage)

		usageSpend.TextTokens, usageSpend.AudioTokens = GetMultimodalAudioTokens(ctx, model, request.Messages, mak.ReqModel)

		usage.PromptTokens = usageSpend.TextTokens + usageSpend.AudioTokens

		if completion != "" {
			usage.CompletionTokens = GetCompletionTokens(ctx, model, completion) + 388
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	if usage.PromptTokensDetails.TextTokens > 0 {
		usageSpend.TextTokens = int(math.Ceil(float64(usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
	}

	if usage.PromptTokensDetails.AudioTokens > 0 {
		usageSpend.AudioTokens = int(math.Ceil(float64(usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
	}

	if usage.CompletionTokensDetails.TextTokens > 0 {
		usageSpend.TextTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
	}

	if usage.CompletionTokensDetails.AudioTokens > 0 {
		usageSpend.AudioTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
	}

	usageSpend.TotalTokens = usageSpend.TextTokens + usageSpend.AudioTokens

	if usage.PromptTokensDetails.CachedTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
	}

	usageSpend.Usage = usage

	return
}

// 文生文
func TextTokens(ctx context.Context, request sdkm.ChatCompletionRequest, completion string, usage *sdkm.Usage, mak *MAK) (usageSpend common.UsageSpend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil || usage.TotalTokens == 0 {

		usage = new(sdkm.Usage)

		usage.PromptTokens = GetPromptTokens(ctx, model, request.Messages)

		if completion != "" {
			usage.CompletionTokens = GetCompletionTokens(ctx, model, completion)
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	if mak.ReqModel.TextQuota.BillingMethod == 2 {
		usage.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
		usageSpend.TotalTokens = mak.ReqModel.TextQuota.FixedQuota
		return
	}

	usageSpend.TotalTokens = int(math.Ceil(float64(usage.PromptTokens)*mak.ReqModel.TextQuota.PromptRatio + float64(usage.CompletionTokens)*mak.ReqModel.TextQuota.CompletionRatio))

	if usage.PromptTokensDetails.CachedTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens > 0 {
		usageSpend.TotalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.TextQuota.CachedRatio))
	}

	usageSpend.Usage = usage

	return
}
