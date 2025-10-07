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

// 花费
func Spend(ctx context.Context, mak *MAK, billingData *common.BillingData, billingItems ...string) (spend common.Spend) {

	if billingItems == nil || len(billingItems) == 0 {
		billingItems = mak.ReqModel.Pricing.BillingItems
	}

	spend.BillingRule = mak.ReqModel.Pricing.BillingRule
	spend.BillingMethods = mak.ReqModel.Pricing.BillingMethods
	spend.BillingItems = billingItems

	for _, billingItem := range billingItems {
		switch billingItem {
		case "text":
			text(ctx, mak, billingData, &spend)
		case "text_cache":
			textCache(ctx, mak, billingData, &spend)
		case "tiered_text":
			tieredText(ctx, mak, billingData, &spend)
		case "tiered_text_cache":
			tieredTextCache(ctx, mak, billingData, &spend)
		case "image":
			image(ctx, mak, billingData, &spend)
		case "image_generation":
			imageGeneration(ctx, mak, billingData, &spend)
		case "image_cache":
			imageCache(ctx, mak, billingData, &spend)
		case "vision":
			vision(ctx, mak, billingData, &spend)
		case "audio":
			audio(ctx, mak, billingData, &spend)
		case "audio_cache":
			audioCache(ctx, mak, billingData, &spend)
		case "search":
			search(ctx, mak, billingData, &spend)
		case "midjourney":
			midjourney(ctx, mak, billingData, &spend)
		case "once":
			once(ctx, mak, &spend)
		}
	}

	if !slices.Contains(mak.ReqModel.Pricing.BillingMethods, 2) {
		spend.TotalSpendTokens = spend.Text.SpendTokens + spend.TextCache.SpendTokens + spend.TieredText.SpendTokens + spend.TieredTextCache.SpendTokens +
			spend.Image.SpendTokens + spend.ImageGeneration.SpendTokens + spend.ImageCache.SpendTokens + spend.Vision.SpendTokens +
			spend.Audio.SpendTokens + spend.AudioCache.SpendTokens + spend.Search.SpendTokens + spend.Midjourney.SpendTokens
	} else {
		spend.TotalSpendTokens = spend.Once.SpendTokens
	}

	// 分组折扣
	if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
		spend.GroupId = mak.Group.Id
		spend.GroupName = mak.Group.Name
		spend.GroupDiscount = mak.Group.Discount
		spend.TotalSpendTokens = int(math.Ceil(float64(spend.TotalSpendTokens) * mak.Group.Discount))
	}

	return
}

// 文本
func text(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
	}

	if mak.ReqModel.Type == 2 || mak.ReqModel.Type == 3 || mak.ReqModel.Type == 4 {

		if billingData.Usage.InputTokensDetails.TextTokens > 0 {
			spend.Text.Pricing = mak.ReqModel.Pricing.Text
			spend.Text.SpendTokens = int(math.Ceil(float64(billingData.Usage.InputTokensDetails.TextTokens) * mak.ReqModel.Pricing.Text.InputRatio))
		}

		return
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

	spend.Text.Pricing = mak.ReqModel.Pricing.Text
	spend.Text.SpendTokens = int(math.Ceil(float64(billingData.Usage.PromptTokens)*mak.ReqModel.Pricing.Text.InputRatio + float64(billingData.Usage.CompletionTokens)*mak.ReqModel.Pricing.Text.OutputRatio))
}

// 文本缓存
func textCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil {
		return
	}

	spend.TextCache.Pricing = mak.ReqModel.Pricing.TextCache

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		spend.TextCache.SpendTokens += int(math.Ceil(float64(billingData.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		spend.TextCache.SpendTokens += int(math.Ceil(float64(billingData.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if billingData.Usage.CacheReadInputTokens > 0 {
		spend.TextCache.SpendTokens += int(math.Ceil(float64(billingData.Usage.CacheReadInputTokens) * mak.ReqModel.Pricing.TextCache.ReadRatio))
	}

	// Claude
	if billingData.Usage.CacheCreationInputTokens > 0 {
		spend.TextCache.SpendTokens += int(math.Ceil(float64(billingData.Usage.CacheCreationInputTokens) * mak.ReqModel.Pricing.TextCache.WriteRatio))
	}
}

// 阶梯文本
func tieredText(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage.PromptTokens+billingData.Usage.CompletionTokens == 0 {
		return
	}

	for i, tieredText := range mak.ReqModel.Pricing.TieredText {
		if (billingData.Usage.PromptTokens > tieredText.Gt && billingData.Usage.PromptTokens <= tieredText.Lte) || (i == len(mak.ReqModel.Pricing.TieredText)-1) {
			spend.TieredText.Pricing = tieredText
			spend.TieredText.SpendTokens = int(math.Ceil(float64(billingData.Usage.PromptTokens)*tieredText.InputRatio)) + int(math.Ceil(float64(billingData.Usage.CompletionTokens)*tieredText.OutputRatio))
			return
		}
	}
}

// 阶梯文本缓存
func tieredTextCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage.CacheReadInputTokens+billingData.Usage.CacheCreationInputTokens == 0 {
		return
	}

	for i, tieredTextCache := range mak.ReqModel.Pricing.TieredTextCache {
		if (billingData.Usage.PromptTokens > tieredTextCache.Gt && billingData.Usage.PromptTokens <= tieredTextCache.Lte) || (i == len(mak.ReqModel.Pricing.TieredTextCache)-1) {
			spend.TieredTextCache.Pricing = tieredTextCache
			spend.TieredTextCache.SpendTokens = int(math.Ceil(float64(billingData.Usage.CacheReadInputTokens)*tieredTextCache.ReadRatio)) + int(math.Ceil(float64(billingData.Usage.CacheCreationInputTokens)*tieredTextCache.WriteRatio))
			return
		}
	}
}

// 图像
func image(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	spend.Image.Pricing = mak.ReqModel.Pricing.Image

	if billingData.Usage.InputTokens > 0 {
		spend.Image.SpendTokens += int(math.Ceil(float64(billingData.Usage.InputTokensDetails.ImageTokens) * mak.ReqModel.Pricing.Image.InputRatio))
	}

	if billingData.Usage.OutputTokens > 0 {
		spend.Image.SpendTokens += int(math.Ceil(float64(billingData.Usage.OutputTokens) * mak.ReqModel.Pricing.Image.OutputRatio))
	}
}

// 图像生成
func imageGeneration(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

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
			spend.ImageGeneration.Pricing = pricing
			break
		}

		if pricing.IsDefault {
			spend.ImageGeneration.Pricing = pricing
		}
	}

	spend.ImageGeneration.SpendTokens = int(math.Ceil(consts.QUOTA_USD_UNIT*spend.ImageGeneration.Pricing.OnceRatio)) * billingData.ImageEditRequest.N
}

// 图像缓存
func imageCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {
	if billingData.Usage.InputTokensDetails.CachedTokens > 0 {
		spend.ImageCache.Pricing = mak.ReqModel.Pricing.ImageCache
		spend.ImageCache.SpendTokens = int(math.Ceil(float64(billingData.Usage.InputTokensDetails.CachedTokens) * mak.ReqModel.Pricing.ImageCache.ReadRatio))
	}
}

// 识图
func vision(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {
		for _, value := range multiContent {

			if content, ok := value.(map[string]interface{}); ok && content["type"] == "image_url" {

				if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

					detail := imageUrl["detail"]

					for _, vision := range mak.ReqModel.Pricing.Vision {

						if vision.Mode == detail {
							spend.Vision.Pricing = vision
							break
						}

						if vision.IsDefault {
							spend.Vision.Pricing = vision
						}
					}

					spend.Vision.SpendTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * spend.Vision.Pricing.OnceRatio))
				}
			}
		}
	}
}

// 音频
func audio(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	spend.Audio.Pricing = mak.ReqModel.Pricing.Audio

	if audioInputLen := len(billingData.AudioInput); audioInputLen > 0 {
		spend.Audio.SpendTokens += int(math.Ceil(float64(audioInputLen) * mak.ReqModel.Pricing.Audio.InputRatio))
	}

	if billingData.AudioMinute > 0 {
		spend.Audio.SpendTokens += int(math.Ceil(billingData.AudioMinute * 1000 * mak.ReqModel.Pricing.Audio.OutputRatio))
	}

	if billingData.Usage.PromptTokensDetails.AudioTokens > 0 {
		spend.Audio.SpendTokens += int(math.Ceil(float64(billingData.Usage.PromptTokensDetails.AudioTokens) * mak.ReqModel.Pricing.Audio.InputRatio))
	}

	if billingData.Usage.CompletionTokensDetails.AudioTokens > 0 {
		spend.Audio.SpendTokens += int(math.Ceil(float64(billingData.Usage.CompletionTokensDetails.AudioTokens) * mak.ReqModel.Pricing.Audio.OutputRatio))
	}
}

// 音频缓存
func audioCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	spend.AudioCache.Pricing = mak.ReqModel.Pricing.AudioCache

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		spend.AudioCache.SpendTokens += int(math.Ceil(float64(billingData.Usage.PromptTokensDetails.CachedTokens) * mak.ReqModel.Pricing.AudioCache.ReadRatio))
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		spend.AudioCache.SpendTokens += int(math.Ceil(float64(billingData.Usage.CompletionTokensDetails.CachedTokens) * mak.ReqModel.Pricing.AudioCache.ReadRatio))
	}
}

// 搜索
func search(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.ChatCompletionRequest.WebSearchOptions == nil && (billingData.ChatCompletionRequest.Tools == nil || (!gstr.Contains(gconv.String(billingData.ChatCompletionRequest.Tools), "google_search") && !gstr.Contains(gconv.String(billingData.ChatCompletionRequest.Tools), "googleSearch"))) {
		return
	}

	var searchContextSize string
	if billingData.ChatCompletionRequest.WebSearchOptions != nil {
		if content, ok := billingData.ChatCompletionRequest.WebSearchOptions.(map[string]interface{}); ok {
			searchContextSize = gconv.String(content["search_context_size"])
		}
	}

	for _, search := range mak.ReqModel.Pricing.Search {

		if search.ContextSize == searchContextSize {
			spend.Search.Pricing = search
			break
		}

		if search.IsDefault {
			spend.Search.Pricing = search
		}
	}

	spend.Search.SpendTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * spend.Search.Pricing.OnceRatio))
}

// Midjourney
func midjourney(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {
	for _, midjourney := range mak.ReqModel.Pricing.Midjourney {
		if billingData.Path == midjourney.Path {
			spend.Midjourney.Pricing = midjourney
			spend.Midjourney.SpendTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * spend.Midjourney.Pricing.OnceRatio))
			return
		}
	}
}

// 一次
func once(ctx context.Context, mak *MAK, spend *common.Spend) {
	spend.Once.Pricing = mak.ReqModel.Pricing.Once
	spend.Once.SpendTokens = int(math.Ceil(consts.QUOTA_USD_UNIT * mak.ReqModel.Pricing.Once.OnceRatio))
}
