package common

import (
	"context"
	"math"
	"slices"

	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/tiktoken-go"
)

func ChatUsageSpend(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	for _, billingItem := range mak.ReqModel.Pricing.BillingItems {
		switch billingItem {
		case "text": // 文本
			usageSpendTokens = textSpendTokens(ctx, usageSpend, mak)
		case "text_cache": // 文本缓存
			usageSpendTokens = textCacheSpendTokens(ctx, usageSpend, mak)
		case "tiered_text": // 阶梯文本
		case "tiered_text_cache": // 阶梯文本缓存
		case "image": // 图像
		case "image_generation": // 图像生成
			usageSpendTokens = imageGenerationSpendTokens(ctx, usageSpend, mak)
		case "image_cache": // 图像缓存
		case "vision": // 识图
			usageSpendTokens = visionSpendTokens(ctx, usageSpend, mak)
		case "audio": // 音频
		case "audio_cache": // 音频缓存
		case "search": // 搜索
			usageSpendTokens = searchSpendTokens(ctx, usageSpend, mak)
		case "midjourney": // Midjourney
		case "once": // 一次
			usageSpendTokens = onceSpendTokens(ctx, usageSpend, mak)
		}
	}

	if !slices.Contains(mak.ReqModel.Pricing.BillingMethods, 2) {
		usageSpendTokens.TotalTokens = usageSpendTokens.TextTokens + usageSpendTokens.TextCacheTokens + usageSpendTokens.TieredTextTokens + usageSpendTokens.TieredTextCacheTokens +
			usageSpendTokens.ImageTokens + usageSpendTokens.ImageGenerationTokens + usageSpendTokens.ImageCacheTokens + usageSpendTokens.VisionTokens +
			usageSpendTokens.AudioTokens + usageSpendTokens.AudioCacheTokens + usageSpendTokens.SearchTokens + usageSpendTokens.MidjourneyTokens
	} else {
		usageSpendTokens.TotalTokens = usageSpendTokens.OnceTokens
	}

	return
}

// 多模态
func MultimodalTokens(ctx context.Context, request smodel.ChatCompletionRequest, completion string, usage *smodel.Usage, mak *MAK) (usageSpend common.UsageSpendTokens) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil || mak.ReqModel.MultimodalQuota.BillingRule == 2 {

		usage = new(smodel.Usage)

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

	return
}

// 多模态语音
func MultimodalAudioTokens(ctx context.Context, request smodel.ChatCompletionRequest, completion string, usage *smodel.Usage, mak *MAK) (usageSpend common.UsageSpendTokens) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil {

		usage = new(smodel.Usage)

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

	return
}

// 文生文
func TextTokens(ctx context.Context, request smodel.ChatCompletionRequest, completion string, usage *smodel.Usage, mak *MAK) (usageSpend common.UsageSpendTokens) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil || usage.TotalTokens == 0 {

		usage = new(smodel.Usage)

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

	return
}

// 文本
func textSpendTokens(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usageSpend.Usage == nil || usageSpend.Usage.TotalTokens == 0 || mak.ReqModel.Pricing.BillingRule == 2 {

		if usageSpend.Usage == nil {
			usageSpend.Usage = new(smodel.Usage)
		}

		usageSpend.Usage.PromptTokens = GetPromptTokens(ctx, model, usageSpend.ChatCompletionRequest.Messages)

		if usageSpend.Completion != "" {
			usageSpend.Usage.CompletionTokens = GetCompletionTokens(ctx, model, usageSpend.Completion)
		}
	}

	usageSpendTokens.TextTokens += int(math.Ceil(float64(usageSpend.Usage.PromptTokens)*mak.ReqModel.Pricing.Text.InputRatio + float64(usageSpend.Usage.CompletionTokens)*mak.ReqModel.Pricing.Text.OutputRatio))

	return
}

// 文本缓存
func textCacheSpendTokens(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	if usageSpend.Usage == nil {
		return
	}

	if usageSpend.Usage.PromptTokensDetails.CachedTokens > 0 {
		usageSpendTokens.TextCacheTokens += int(math.Ceil(float64(usageSpend.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	if usageSpend.Usage.CompletionTokensDetails.CachedTokens > 0 {
		usageSpendTokens.TextCacheTokens += int(math.Ceil(float64(usageSpend.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if usageSpend.Usage.CacheReadInputTokens > 0 {
		usageSpendTokens.TextCacheTokens += int(math.Ceil(float64(usageSpend.Usage.CacheReadInputTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if usageSpend.Usage.CacheCreationInputTokens > 0 {
		usageSpendTokens.TextCacheTokens += int(math.Ceil(float64(usageSpend.Usage.CacheCreationInputTokens) * mak.ReqModel.Pricing.TextCache.WriteRatio))
	}

	return
}

// 图像生成
func imageGenerationSpendTokens(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	return
}

// 识图
func visionSpendTokens(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if multiContent, ok := usageSpend.ChatCompletionRequest.Messages[len(usageSpend.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {
		for _, value := range multiContent {

			if content, ok := value.(map[string]interface{}); ok && content["type"] == "image_url" {

				if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

					detail := imageUrl["detail"]

					var visionPricing common.VisionPricing
					for _, vision := range mak.ReqModel.Pricing.Vision {

						if vision.Mode == detail {
							visionPricing = vision
							break
						}

						if vision.IsDefault {
							visionPricing = vision
						}
					}

					usageSpendTokens.VisionTokens += int(math.Ceil(consts.QUOTA_USD_UNIT * visionPricing.OnceRatio))
				}
			}
		}
	}

	return
}

// 搜索
func searchSpendTokens(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	if usageSpend.ChatCompletionRequest.WebSearchOptions == nil && (usageSpend.ChatCompletionRequest.Tools == nil || (!gstr.Contains(gconv.String(usageSpend.ChatCompletionRequest.Tools), "google_search") && !gstr.Contains(gconv.String(usageSpend.ChatCompletionRequest.Tools), "googleSearch"))) {
		return
	}

	var searchContextSize string
	if usageSpend.ChatCompletionRequest.WebSearchOptions != nil {
		if content, ok := usageSpend.ChatCompletionRequest.WebSearchOptions.(map[string]interface{}); ok {
			searchContextSize = gconv.String(content["search_context_size"])
		}
	}

	var searchPricing common.SearchPricing
	for _, search := range mak.ReqModel.Pricing.Search {

		if search.ContextSize == searchContextSize {
			searchPricing = search
			break
		}

		if search.IsDefault {
			searchPricing = search
		}
	}

	usageSpendTokens.SearchTokens += int(math.Ceil(consts.QUOTA_USD_UNIT * searchPricing.OnceRatio))

	return
}

// 一次
func onceSpendTokens(ctx context.Context, usageSpend *common.UsageSpend, mak *MAK) (usageSpendTokens common.UsageSpendTokens) {

	usageSpendTokens.TotalTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * mak.ReqModel.Pricing.Once.OnceRatio))

	return
}
