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
)

// 花费额度
func SpendTokens(ctx context.Context, mak *MAK, billingData *common.BillingData, billingItems ...string) (spendTokens common.SpendTokens) {

	if billingItems == nil || len(billingItems) == 0 {
		billingItems = mak.ReqModel.Pricing.BillingItems
	}

	for _, billingItem := range billingItems {
		switch billingItem {
		case "text":
			spendTokens.TextTokens = text(ctx, mak, billingData)
		case "text_cache":
			spendTokens.TextCacheTokens = textCache(ctx, mak, billingData)
		case "tiered_text":
			spendTokens.TieredTextTokens = tieredText(ctx, mak, billingData)
		case "tiered_text_cache":
			spendTokens.TieredTextCacheTokens = tieredTextCache(ctx, mak, billingData)
		case "image":
			spendTokens.ImageTokens = image(ctx, mak, billingData)
		case "image_generation":
			spendTokens.ImageGenerationPricing = imageGeneration(ctx, mak, billingData)
			spendTokens.ImageGenerationTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * spendTokens.ImageGenerationPricing.OnceRatio))
		case "image_cache":
			spendTokens.ImageCacheTokens = imageCache(ctx, mak, billingData)
		case "vision":
			spendTokens.VisionTokens = vision(ctx, mak, billingData)
		case "audio":
			spendTokens.AudioTokens = audio(ctx, mak, billingData)
		case "audio_cache":
			spendTokens.AudioCacheTokens = audioCache(ctx, mak, billingData)
		case "search":
			spendTokens.SearchTokens = search(ctx, mak, billingData)
		case "midjourney":
			spendTokens.MidjourneyPricing = midjourney(ctx, mak, billingData)
			spendTokens.MidjourneyTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * spendTokens.MidjourneyPricing.OnceRatio))
		case "once":
			spendTokens.OnceTokens = once(ctx, mak)
		}
	}

	if !slices.Contains(mak.ReqModel.Pricing.BillingMethods, 2) {
		spendTokens.TotalTokens = spendTokens.TextTokens + spendTokens.TextCacheTokens + spendTokens.TieredTextTokens + spendTokens.TieredTextCacheTokens +
			spendTokens.ImageTokens + spendTokens.ImageGenerationTokens + spendTokens.ImageCacheTokens + spendTokens.VisionTokens +
			spendTokens.AudioTokens + spendTokens.AudioCacheTokens + spendTokens.SearchTokens + spendTokens.MidjourneyTokens
	} else {
		spendTokens.TotalTokens = spendTokens.OnceTokens
	}

	// 分组折扣
	if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
		spendTokens.TotalTokens = int(math.Ceil(float64(spendTokens.TotalTokens) * mak.Group.Discount))
	}

	return
}

// 文本
func text(ctx context.Context, mak *MAK, billingData *common.BillingData) (textTokens int) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
	}

	if mak.ReqModel.Type == 2 || mak.ReqModel.Type == 3 || mak.ReqModel.Type == 4 {

		if billingData.Usage.InputTokensDetails.TextTokens > 0 {
			textTokens = int(math.Ceil(float64(billingData.Usage.InputTokensDetails.TextTokens) * mak.ReqModel.Pricing.Text.InputRatio))
		}

		return textTokens
	}

	if billingData.Usage.TotalTokens == 0 || mak.ReqModel.Pricing.BillingRule == 2 {

		billingData.Usage = new(smodel.Usage)

		if mak.ReqModel.Type == 100 {
			if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {

				for _, value := range multiContent {
					if content, ok := value.(map[string]interface{}); ok {
						if content["type"] == "text" {
							billingData.Usage.PromptTokens += TokensFromString(ctx, mak.ReqModel.Model, gconv.String(content))
						}
					} else {
						billingData.Usage.PromptTokens += TokensFromString(ctx, mak.ReqModel.Model, gconv.String(value))
					}
				}

			} else {
				billingData.Usage.PromptTokens = TokensFromMessages(ctx, model, billingData.ChatCompletionRequest.Messages)
			}

		} else {
			billingData.Usage.PromptTokens = TokensFromMessages(ctx, model, billingData.ChatCompletionRequest.Messages)
		}

		if billingData.Completion != "" {
			billingData.Usage.CompletionTokens = TokensFromString(ctx, model, billingData.Completion)
		}
	}

	textTokens = int(math.Ceil(float64(billingData.Usage.PromptTokens)*mak.ReqModel.Pricing.Text.InputRatio + float64(billingData.Usage.CompletionTokens)*mak.ReqModel.Pricing.Text.OutputRatio))

	return textTokens
}

// 文本缓存
func textCache(ctx context.Context, mak *MAK, billingData *common.BillingData) (textCacheTokens int) {

	if billingData.Usage == nil {
		return
	}

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(billingData.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(billingData.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if billingData.Usage.CacheReadInputTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(billingData.Usage.CacheReadInputTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if billingData.Usage.CacheCreationInputTokens > 0 {
		textCacheTokens += int(math.Ceil(float64(billingData.Usage.CacheCreationInputTokens) * mak.ReqModel.Pricing.TextCache.WriteRatio))
	}

	return textCacheTokens
}

// 阶梯文本
func tieredText(ctx context.Context, mak *MAK, billingData *common.BillingData) (tieredTextTokens int) {

	if billingData.Usage.PromptTokens+billingData.Usage.CompletionTokens == 0 {
		return
	}

	for i, tieredText := range mak.ReqModel.Pricing.TieredText {

		if billingData.Usage.PromptTokens > tieredText.Gt && billingData.Usage.PromptTokens <= tieredText.Lte {
			tieredTextTokens = int(math.Ceil(float64(billingData.Usage.PromptTokens)*tieredText.InputRatio)) + int(math.Ceil(float64(billingData.Usage.CompletionTokens)*tieredText.OutputRatio))
			break
		}

		if i == len(mak.ReqModel.Pricing.TieredText)-1 {
			tieredTextTokens = int(math.Ceil(float64(billingData.Usage.PromptTokens)*tieredText.InputRatio)) + int(math.Ceil(float64(billingData.Usage.CompletionTokens)*tieredText.OutputRatio))
		}
	}

	return tieredTextTokens
}

// 阶梯文本缓存
func tieredTextCache(ctx context.Context, mak *MAK, billingData *common.BillingData) (tieredTextCacheTokens int) {

	if billingData.Usage.CacheReadInputTokens+billingData.Usage.CacheCreationInputTokens == 0 {
		return
	}

	for i, tieredTextCache := range mak.ReqModel.Pricing.TieredTextCache {

		if billingData.Usage.PromptTokens > tieredTextCache.Gt && billingData.Usage.PromptTokens <= tieredTextCache.Lte {
			tieredTextCacheTokens = int(math.Ceil(float64(billingData.Usage.CacheReadInputTokens)*tieredTextCache.ReadRatio)) + int(math.Ceil(float64(billingData.Usage.CacheCreationInputTokens)*tieredTextCache.WriteRatio))
		}

		if i == len(mak.ReqModel.Pricing.TieredTextCache)-1 {
			tieredTextCacheTokens = int(math.Ceil(float64(billingData.Usage.CacheReadInputTokens)*tieredTextCache.ReadRatio)) + int(math.Ceil(float64(billingData.Usage.CacheCreationInputTokens)*tieredTextCache.WriteRatio))
		}
	}

	return tieredTextCacheTokens
}

// 图像
func image(ctx context.Context, mak *MAK, billingData *common.BillingData) (imageTokens int) {

	if billingData.Usage.InputTokens > 0 {
		imageTokens += int(math.Ceil(float64(billingData.Usage.InputTokensDetails.ImageTokens) * mak.ReqModel.Pricing.Image.InputRatio))
	}

	if billingData.Usage.OutputTokens > 0 {
		imageTokens += int(math.Ceil(float64(billingData.Usage.OutputTokens) * mak.ReqModel.Pricing.Image.OutputRatio))
	}

	return imageTokens
}

// 图像生成
func imageGeneration(ctx context.Context, mak *MAK, billingData *common.BillingData) (imageGenerationPricing common.ImageGenerationPricing) {

	var (
		quality = billingData.ImageGenerationRequest.Quality
		size    = billingData.ImageGenerationRequest.Size
		width   int
		height  int
	)

	if quality == "" {
		quality = billingData.ImageEditRequest.Quality
	}

	if size == "" {
		size = billingData.ImageEditRequest.Size
	}

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
func imageCache(ctx context.Context, mak *MAK, billingData *common.BillingData) (imageCacheTokens int) {

	if billingData.Usage.InputTokensDetails.CachedTokens > 0 {
		imageCacheTokens = int(math.Ceil(float64(billingData.Usage.InputTokensDetails.CachedTokens) * mak.ReqModel.Pricing.ImageCache.ReadRatio))
	}

	return imageCacheTokens
}

// 识图
func vision(ctx context.Context, mak *MAK, billingData *common.BillingData) (visionTokens int) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {
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

// 音频
func audio(ctx context.Context, mak *MAK, billingData *common.BillingData) (audioTokens int) {

	if audioInputLen := len(billingData.AudioInput); audioInputLen > 0 {
		audioTokens += int(math.Ceil(float64(audioInputLen) * mak.ReqModel.Pricing.Audio.InputRatio))
	}

	if billingData.AudioMinute > 0 {
		audioTokens += int(math.Ceil(billingData.AudioMinute * 1000 * mak.ReqModel.Pricing.Audio.OutputRatio))
	}

	if billingData.Usage.PromptTokensDetails.AudioTokens > 0 {
		audioTokens += int(math.Ceil(float64(billingData.Usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.Pricing.Audio.InputRatio))
	}

	if billingData.Usage.CompletionTokensDetails.AudioTokens > 0 {
		audioTokens += int(math.Ceil(float64(billingData.Usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.Pricing.Audio.OutputRatio))
	}

	return audioTokens
}

// 音频缓存
func audioCache(ctx context.Context, mak *MAK, billingData *common.BillingData) (audioCacheTokens int) {

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		audioCacheTokens += int(math.Ceil(float64(billingData.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.Pricing.AudioCache.ReadRatio))
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		audioCacheTokens += int(math.Ceil(float64(billingData.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.Pricing.AudioCache.ReadRatio))
	}

	return audioCacheTokens
}

// 搜索
func search(ctx context.Context, mak *MAK, billingData *common.BillingData) (searchTokens int) {

	if billingData.ChatCompletionRequest.WebSearchOptions == nil && (billingData.ChatCompletionRequest.Tools == nil || (!gstr.Contains(gconv.String(billingData.ChatCompletionRequest.Tools), "google_search") && !gstr.Contains(gconv.String(billingData.ChatCompletionRequest.Tools), "googleSearch"))) {
		return
	}

	var searchContextSize string
	if billingData.ChatCompletionRequest.WebSearchOptions != nil {
		if content, ok := billingData.ChatCompletionRequest.WebSearchOptions.(map[string]interface{}); ok {
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

// Midjourney
func midjourney(ctx context.Context, mak *MAK, billingData *common.BillingData) (midjourneyPricing common.MidjourneyPricing) {

	for _, midjourney := range mak.ReqModel.Pricing.Midjourney {
		if billingData.Path == midjourney.Path {
			return midjourney
		}
	}

	return midjourneyPricing
}

// 一次
func once(ctx context.Context, mak *MAK) (onceTokens int) {

	onceTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * mak.ReqModel.Pricing.Once.OnceRatio))

	return onceTokens
}
