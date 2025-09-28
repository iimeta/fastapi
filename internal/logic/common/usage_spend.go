package common

import (
	"context"
	"math"
	"slices"

	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/tiktoken"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/utility/logger"
)

func SpendTokens(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (spendTokens common.SpendTokens) {

	for _, billingItem := range mak.ReqModel.Pricing.BillingItems {
		switch billingItem {
		case "text":
			spendTokens.TextTokens = text(ctx, spendContent, mak)
		case "text_cache":
			spendTokens.TextCacheTokens = textCache(ctx, spendContent, mak)
		case "tiered_text":
		case "tiered_text_cache":
		case "image":
			spendTokens.ImageTokens = image(ctx, spendContent, mak)
		case "image_generation":
			spendTokens.ImageGenerationPricing = imageGeneration(ctx, spendContent, mak)
			spendTokens.ImageGenerationTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * spendTokens.ImageGenerationPricing.OnceRatio))
		case "image_cache":
			spendTokens.ImageCacheTokens = imageCache(ctx, spendContent, mak)
		case "vision":
			spendTokens.VisionTokens = vision(ctx, spendContent, mak)
		case "audio":
		case "audio_cache":
		case "search":
			spendTokens.SearchTokens = search(ctx, spendContent, mak)
		case "midjourney":
		case "once":
			spendTokens.OnceTokens = once(ctx, spendContent, mak)
		}
	}

	if !slices.Contains(mak.ReqModel.Pricing.BillingMethods, 2) {
		spendTokens.TotalTokens = spendTokens.TextTokens + spendTokens.TextCacheTokens + spendTokens.TieredTextTokens + spendTokens.TieredTextCacheTokens +
			spendTokens.ImageTokens + spendTokens.ImageGenerationTokens + spendTokens.ImageCacheTokens + spendTokens.VisionTokens +
			spendTokens.AudioTokens + spendTokens.AudioCacheTokens + spendTokens.SearchTokens + spendTokens.MidjourneyTokens
	} else {
		spendTokens.TotalTokens = spendTokens.OnceTokens
	}

	return
}

// 多模态语音
func MultimodalAudioTokens(ctx context.Context, request smodel.ChatCompletionRequest, completion string, usage *smodel.Usage, mak *MAK) (spendTokens common.SpendTokens) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if usage == nil {

		usage = new(smodel.Usage)

		spendTokens.TextTokens, spendTokens.AudioTokens = GetMultimodalAudioTokens(ctx, model, request.Messages, mak.ReqModel)

		usage.PromptTokens = spendTokens.TextTokens + spendTokens.AudioTokens

		if completion != "" {
			usage.CompletionTokens = GetCompletionTokens(ctx, model, completion) + 388
		}

		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	if usage.PromptTokensDetails.TextTokens > 0 {
		spendTokens.TextTokens = int(math.Ceil(float64(usage.PromptTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.PromptRatio))
	}

	if usage.PromptTokensDetails.AudioTokens > 0 {
		spendTokens.AudioTokens = int(math.Ceil(float64(usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio))
	}

	if usage.CompletionTokensDetails.TextTokens > 0 {
		spendTokens.TextTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.TextTokens) * mak.ReqModel.MultimodalAudioQuota.TextQuota.CompletionRatio))
	}

	if usage.CompletionTokensDetails.AudioTokens > 0 {
		spendTokens.AudioTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio))
	}

	spendTokens.TotalTokens = spendTokens.TextTokens + spendTokens.AudioTokens

	if usage.PromptTokensDetails.CachedTokens > 0 {
		spendTokens.TotalTokens += int(math.Ceil(float64(usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
	}

	if usage.CompletionTokensDetails.CachedTokens > 0 {
		spendTokens.TotalTokens += int(math.Ceil(float64(usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.MultimodalAudioQuota.AudioQuota.CachedRatio))
	}

	return
}

// 文本
func text(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (textTokens int) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if spendContent.Usage == nil {
		spendContent.Usage = new(smodel.Usage)
	}

	if mak.ReqModel.Type == 2 || mak.ReqModel.Type == 3 || mak.ReqModel.Type == 4 {

		if spendContent.Usage.InputTokensDetails.TextTokens > 0 {
			textTokens = int(math.Ceil(float64(spendContent.Usage.InputTokensDetails.TextTokens) * mak.ReqModel.Pricing.Text.InputRatio))
		}

		return textTokens
	}

	if spendContent.Usage.TotalTokens == 0 || mak.ReqModel.Pricing.BillingRule == 2 {

		spendContent.Usage = new(smodel.Usage)

		if mak.ReqModel.Type == 100 {
			if multiContent, ok := spendContent.ChatCompletionRequest.Messages[len(spendContent.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {

				for _, value := range multiContent {

					if content, ok := value.(map[string]interface{}); ok {

						if content["type"] != "image_url" {
							if tokens, err := tiktoken.NumTokensFromString(model, gconv.String(content)); err != nil {
								logger.Errorf(ctx, "SpendTokens NumTokensFromString model: %s, content: %s, error: %v", model, gconv.String(content), err)
								if tokens, err = tiktoken.NumTokensFromString(consts.DEFAULT_MODEL, gconv.String(content)); err != nil {
									logger.Errorf(ctx, "SpendTokens NumTokensFromString model: %s, content: %s, error: %v", consts.DEFAULT_MODEL, gconv.String(content), err)
								}
							} else {
								spendContent.Usage.PromptTokens += tokens
							}
						}

					} else {
						if tokens, err := tiktoken.NumTokensFromString(model, gconv.String(value)); err != nil {
							logger.Errorf(ctx, "SpendTokens NumTokensFromString model: %s, value: %s, error: %v", model, gconv.String(value), err)
							if tokens, err = tiktoken.NumTokensFromString(consts.DEFAULT_MODEL, gconv.String(value)); err != nil {
								logger.Errorf(ctx, "SpendTokens NumTokensFromString model: %s, value: %s, error: %v", consts.DEFAULT_MODEL, gconv.String(value), err)
							}
						} else {
							spendContent.Usage.PromptTokens += tokens
						}
					}
				}

			} else {
				spendContent.Usage.PromptTokens = GetPromptTokens(ctx, model, spendContent.ChatCompletionRequest.Messages)
			}

		} else {
			spendContent.Usage.PromptTokens = GetPromptTokens(ctx, model, spendContent.ChatCompletionRequest.Messages)
		}

		if spendContent.Completion != "" {
			spendContent.Usage.CompletionTokens = GetCompletionTokens(ctx, model, spendContent.Completion)
		}
	}

	textTokens = int(math.Ceil(float64(spendContent.Usage.PromptTokens)*mak.ReqModel.Pricing.Text.InputRatio + float64(spendContent.Usage.CompletionTokens)*mak.ReqModel.Pricing.Text.OutputRatio))

	return textTokens
}

// 文本缓存
func textCache(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (textCacheTokens int) {

	if spendContent.Usage == nil {
		return
	}

	if spendContent.Usage.PromptTokensDetails.CachedTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(spendContent.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	if spendContent.Usage.CompletionTokensDetails.CachedTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(spendContent.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if spendContent.Usage.CacheReadInputTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(spendContent.Usage.CacheReadInputTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if spendContent.Usage.CacheCreationInputTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(spendContent.Usage.CacheCreationInputTokens) * mak.ReqModel.Pricing.TextCache.WriteRatio))
	}

	return textCacheTokens
}

// 图像
func image(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (imageTokens int) {

	if spendContent.Usage.InputTokens > 0 {
		imageTokens += int(math.Ceil(float64(spendContent.Usage.InputTokensDetails.ImageTokens) * mak.ReqModel.Pricing.Image.InputRatio))
	}

	if spendContent.Usage.OutputTokens > 0 {
		imageTokens += int(math.Ceil(float64(spendContent.Usage.OutputTokens) * mak.ReqModel.Pricing.Image.OutputRatio))
	}

	return imageTokens
}

// 图像生成
func imageGeneration(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (imageGenerationPricing common.ImageGenerationPricing) {

	var (
		quality = spendContent.ImageGenerationRequest.Quality
		size    = spendContent.ImageGenerationRequest.Size
		width   int
		height  int
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

	for _, pricing := range mak.ReqModel.Pricing.ImageGeneration {

		if pricing.Quality == quality && pricing.Width == width && pricing.Height == height {
			return pricing
		}

		if pricing.IsDefault {
			imageGenerationPricing = pricing
		}
	}

	return imageGenerationPricing
}

// 图像缓存
func imageCache(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (imageCacheTokens int) {

	if spendContent.Usage.InputTokensDetails.CachedTokens > 0 {
		imageCacheTokens = int(math.Ceil(float64(spendContent.Usage.InputTokensDetails.CachedTokens) * mak.ReqModel.Pricing.ImageCache.ReadRatio))
	}

	return imageCacheTokens
}

// 识图
func vision(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (visionTokens int) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if multiContent, ok := spendContent.ChatCompletionRequest.Messages[len(spendContent.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {
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

					visionTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * visionPricing.OnceRatio))
				}
			}
		}
	}

	return visionTokens
}

// 搜索
func search(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (searchTokens int) {

	if spendContent.ChatCompletionRequest.WebSearchOptions == nil && (spendContent.ChatCompletionRequest.Tools == nil || (!gstr.Contains(gconv.String(spendContent.ChatCompletionRequest.Tools), "google_search") && !gstr.Contains(gconv.String(spendContent.ChatCompletionRequest.Tools), "googleSearch"))) {
		return
	}

	var searchContextSize string
	if spendContent.ChatCompletionRequest.WebSearchOptions != nil {
		if content, ok := spendContent.ChatCompletionRequest.WebSearchOptions.(map[string]interface{}); ok {
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

	searchTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * searchPricing.OnceRatio))

	return searchTokens
}

// 一次
func once(ctx context.Context, spendContent *common.SpendContent, mak *MAK) (onceTokens int) {

	onceTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * mak.ReqModel.Pricing.Once.OnceRatio))

	return onceTokens
}
