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

// 计算花费
func Billing(ctx context.Context, mak *MAK, billingData *common.BillingData, billingItems ...string) (spend common.Spend) {

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
		case "video":
			video(ctx, mak, billingData, &spend)
		case "search":
			search(ctx, mak, billingData, &spend)
		case "midjourney":
			midjourney(ctx, mak, billingData, &spend)
		case "once":
			once(ctx, mak, billingData, &spend)
		}
	}

	if spend.Text != nil {
		spend.TotalSpendTokens += spend.Text.SpendTokens
	}

	if spend.TextCache != nil {
		spend.TotalSpendTokens += spend.TextCache.SpendTokens
	}

	if spend.TieredText != nil {
		spend.TotalSpendTokens += spend.TieredText.SpendTokens
	}

	if spend.TieredTextCache != nil {
		spend.TotalSpendTokens += spend.TieredTextCache.SpendTokens
	}

	if spend.Image != nil {
		spend.TotalSpendTokens += spend.Image.SpendTokens
	}

	if spend.ImageGeneration != nil {
		spend.TotalSpendTokens += spend.ImageGeneration.SpendTokens
	}

	if spend.ImageCache != nil {
		spend.TotalSpendTokens += spend.ImageCache.SpendTokens
	}

	if spend.Vision != nil {
		spend.TotalSpendTokens += spend.Vision.SpendTokens
	}

	if spend.Audio != nil {
		spend.TotalSpendTokens += spend.Audio.SpendTokens
	}

	if spend.AudioCache != nil {
		spend.TotalSpendTokens += spend.AudioCache.SpendTokens
	}

	if spend.Video != nil {
		spend.TotalSpendTokens += spend.Video.SpendTokens
	}

	if spend.Search != nil {
		spend.TotalSpendTokens += spend.Search.SpendTokens
	}

	if spend.Midjourney != nil {
		spend.TotalSpendTokens += spend.Midjourney.SpendTokens
	}

	if spend.Once != nil && (spend.TotalSpendTokens == 0 || mak.AppKey == nil || slices.Contains(mak.AppKey.BillingMethods, 2)) {
		spend.TotalSpendTokens = spend.Once.SpendTokens
	}

	// 分组折扣
	if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
		spend.GroupId = mak.Group.Id
		spend.GroupName = mak.Group.Name
		spend.GroupDiscount = mak.Group.Discount
		spend.TotalSpendTokens = int(math.Ceil(float64(spend.TotalSpendTokens) * mak.Group.Discount))
	}

	return spend
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

			if spend.Text == nil {
				spend.Text = new(common.TextSpend)
			}

			serviceTier := "all"
			if billingData.ServiceTier != "" {
				serviceTier = billingData.ServiceTier
			} else if billingData.ChatCompletionRequest.ServiceTier != "" {
				serviceTier = billingData.ChatCompletionRequest.ServiceTier
			}

			for i, text := range mak.ReqModel.Pricing.Text {
				if serviceTier == text.ServiceTier || i == len(mak.ReqModel.Pricing.Text)-1 {
					spend.Text.Pricing = text
					break
				}
			}

			spend.Text.InputTokens = billingData.Usage.InputTokensDetails.TextTokens
			spend.Text.SpendTokens = int(math.Ceil(float64(spend.Text.InputTokens) * spend.Text.Pricing.InputRatio))
		}

		return
	}

	if spend.Text == nil {
		spend.Text = new(common.TextSpend)
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

	serviceTier := "all"
	if billingData.ServiceTier != "" {
		serviceTier = billingData.ServiceTier
	} else if billingData.ChatCompletionRequest.ServiceTier != "" {
		serviceTier = billingData.ChatCompletionRequest.ServiceTier
	}

	for i, text := range mak.ReqModel.Pricing.Text {
		if serviceTier == text.ServiceTier || i == len(mak.ReqModel.Pricing.Text)-1 {
			spend.Text.Pricing = text
			break
		}
	}

	spend.Text.InputTokens = billingData.Usage.PromptTokens
	spend.Text.OutputTokens = billingData.Usage.CompletionTokens
	spend.Text.SpendTokens = int(math.Ceil(float64(spend.Text.InputTokens)*spend.Text.Pricing.InputRatio)) + int(math.Ceil(float64(spend.Text.OutputTokens)*spend.Text.Pricing.OutputRatio))
}

// 文本缓存
func textCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil || billingData.Usage.PromptTokensDetails.CachedTokens+billingData.Usage.CompletionTokensDetails.CachedTokens+billingData.Usage.CacheReadInputTokens+billingData.Usage.CacheCreationInputTokens == 0 {
		return
	}

	if spend.TextCache == nil {
		spend.TextCache = new(common.CacheSpend)
	}

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		spend.TextCache.ReadTokens += billingData.Usage.PromptTokensDetails.CachedTokens
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		spend.TextCache.ReadTokens += billingData.Usage.CompletionTokensDetails.CachedTokens
	}

	// Claude
	if billingData.Usage.CacheReadInputTokens > 0 {
		spend.TextCache.ReadTokens += billingData.Usage.CacheReadInputTokens
	}

	// Claude
	if billingData.Usage.CacheCreationInputTokens > 0 {
		spend.TextCache.WriteTokens += billingData.Usage.CacheCreationInputTokens
	}

	serviceTier := "all"
	if billingData.ServiceTier != "" {
		serviceTier = billingData.ServiceTier
	} else if billingData.ChatCompletionRequest.ServiceTier != "" {
		serviceTier = billingData.ChatCompletionRequest.ServiceTier
	}

	for i, textCache := range mak.ReqModel.Pricing.TextCache {
		if serviceTier == textCache.ServiceTier || i == len(mak.ReqModel.Pricing.TextCache)-1 {
			spend.TextCache.Pricing = textCache
			break
		}
	}

	spend.TextCache.SpendTokens = int(math.Ceil(float64(spend.TextCache.ReadTokens)*spend.TextCache.Pricing.ReadRatio)) + int(math.Ceil(float64(spend.TextCache.WriteTokens)*spend.TextCache.Pricing.WriteRatio))
}

// 阶梯文本
func tieredText(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil || billingData.Usage.PromptTokens+billingData.Usage.CompletionTokens == 0 {
		return
	}

	if spend.TieredText == nil {
		spend.TieredText = new(common.TextSpend)
	}

	mode := "all"
	if billingData.ChatCompletionRequest.EnableThinking != nil {
		if *billingData.ChatCompletionRequest.EnableThinking {
			mode = "thinking"
		} else {
			mode = "non_thinking"
		}
	}

	for i, tieredText := range mak.ReqModel.Pricing.TieredText {
		if mode == tieredText.Mode && ((billingData.Usage.PromptTokens > tieredText.Gt && billingData.Usage.PromptTokens <= tieredText.Lte) || (i == len(mak.ReqModel.Pricing.TieredText)-1)) {
			spend.TieredText.Pricing = tieredText
			spend.TieredText.InputTokens = billingData.Usage.PromptTokens
			spend.TieredText.OutputTokens = billingData.Usage.CompletionTokens
			spend.TieredText.SpendTokens = int(math.Ceil(float64(spend.TieredText.InputTokens)*spend.TieredText.Pricing.InputRatio)) + int(math.Ceil(float64(spend.TieredText.OutputTokens)*spend.TieredText.Pricing.OutputRatio))
			return
		}
	}
}

// 阶梯文本缓存
func tieredTextCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil || billingData.Usage.CacheReadInputTokens+billingData.Usage.CacheCreationInputTokens == 0 {
		return
	}

	if spend.TieredTextCache == nil {
		spend.TieredTextCache = new(common.CacheSpend)
	}

	mode := "all"
	if billingData.ChatCompletionRequest.EnableThinking != nil {
		if *billingData.ChatCompletionRequest.EnableThinking {
			mode = "thinking"
		} else {
			mode = "non_thinking"
		}
	}

	for i, tieredTextCache := range mak.ReqModel.Pricing.TieredTextCache {
		if mode == tieredTextCache.Mode && ((billingData.Usage.PromptTokens > tieredTextCache.Gt && billingData.Usage.PromptTokens <= tieredTextCache.Lte) || (i == len(mak.ReqModel.Pricing.TieredTextCache)-1)) {
			spend.TieredTextCache.Pricing = tieredTextCache
			spend.TieredTextCache.ReadTokens = billingData.Usage.CacheReadInputTokens
			spend.TieredTextCache.WriteTokens = billingData.Usage.CacheCreationInputTokens
			spend.TieredTextCache.SpendTokens = int(math.Ceil(float64(spend.TieredTextCache.ReadTokens)*spend.TieredTextCache.Pricing.ReadRatio)) + int(math.Ceil(float64(spend.TieredTextCache.WriteTokens)*spend.TieredTextCache.Pricing.WriteRatio))
			return
		}
	}
}

// 图像
func image(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil || billingData.Usage.InputTokensDetails.ImageTokens+billingData.Usage.OutputTokens == 0 {
		return
	}

	if spend.Image == nil {
		spend.Image = new(common.ImageSpend)
	}

	if billingData.Usage.InputTokensDetails.ImageTokens > 0 {
		spend.Image.InputTokens += billingData.Usage.InputTokensDetails.ImageTokens
	}

	if billingData.Usage.OutputTokens > 0 {
		spend.Image.OutputTokens += billingData.Usage.OutputTokens
	}

	spend.Image.Pricing = mak.ReqModel.Pricing.Image
	spend.Image.SpendTokens = int(math.Ceil(float64(spend.Image.InputTokens)*spend.Image.Pricing.InputRatio)) + int(math.Ceil(float64(spend.Image.OutputTokens)*spend.Image.Pricing.OutputRatio))
}

// 图像生成
func imageGeneration(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.ImageGeneration == nil {
		spend.ImageGeneration = new(common.ImageGenerationSpend)
	}

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

	spend.ImageGeneration.N = billingData.ImageGenerationRequest.N
	if spend.ImageGeneration.N == 0 {
		spend.ImageGeneration.N = billingData.ImageEditRequest.N
		if spend.ImageGeneration.N == 0 {
			spend.ImageGeneration.N = 1
		}
	}

	spend.ImageGeneration.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT*spend.ImageGeneration.Pricing.OnceRatio)) * spend.ImageGeneration.N
}

// 图像缓存
func imageCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil || billingData.Usage.InputTokensDetails.CachedTokens == 0 {
		return
	}

	if spend.ImageCache == nil {
		spend.ImageCache = new(common.CacheSpend)
	}

	spend.ImageCache.Pricing = mak.ReqModel.Pricing.ImageCache
	spend.ImageCache.ReadTokens = billingData.Usage.InputTokensDetails.CachedTokens
	spend.ImageCache.SpendTokens = int(math.Ceil(float64(spend.ImageCache.ReadTokens) * spend.ImageCache.Pricing.ReadRatio))
}

// 识图
func vision(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]interface{}); ok {

		if spend.Vision == nil {
			spend.Vision = new(common.VisionSpend)
		}

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

					spend.Vision.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT * spend.Vision.Pricing.OnceRatio))
				}
			}
		}
	}
}

// 音频
func audio(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	audioInputLen := len(billingData.AudioInput)

	if audioInputLen+int(math.Ceil(billingData.AudioMinute*1000000)) == 0 && (billingData.Usage == nil || billingData.Usage.PromptTokensDetails.AudioTokens+billingData.Usage.CompletionTokensDetails.AudioTokens == 0) {
		return
	}

	if spend.Audio == nil {
		spend.Audio = new(common.AudioSpend)
	}

	if audioInputLen > 0 {
		spend.Audio.InputTokens += audioInputLen
	}

	if billingData.AudioMinute > 0 {
		spend.Audio.OutputTokens += int(math.Ceil(billingData.AudioMinute * 1000000))
	}

	if billingData.Usage != nil {

		if billingData.Usage.PromptTokensDetails.AudioTokens > 0 {
			spend.Audio.InputTokens += billingData.Usage.PromptTokensDetails.AudioTokens
		}

		if billingData.Usage.CompletionTokensDetails.AudioTokens > 0 {
			spend.Audio.OutputTokens += billingData.Usage.CompletionTokensDetails.AudioTokens
		}
	}

	spend.Audio.Pricing = mak.ReqModel.Pricing.Audio
	spend.Audio.SpendTokens = int(math.Ceil(float64(spend.Audio.InputTokens)*spend.Audio.Pricing.InputRatio)) + int(math.Ceil(float64(spend.Audio.OutputTokens)*spend.Audio.Pricing.OutputRatio))
}

// 音频缓存
func audioCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Usage == nil || billingData.Usage.PromptTokensDetails.CachedTokens+billingData.Usage.CompletionTokensDetails.CachedTokens == 0 {
		return
	}

	if spend.AudioCache == nil {
		spend.AudioCache = new(common.CacheSpend)
	}

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		spend.AudioCache.ReadTokens += billingData.Usage.PromptTokensDetails.CachedTokens
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		spend.AudioCache.ReadTokens += billingData.Usage.CompletionTokensDetails.CachedTokens
	}

	spend.AudioCache.Pricing = mak.ReqModel.Pricing.AudioCache
	spend.AudioCache.SpendTokens = int(math.Ceil(float64(spend.AudioCache.ReadTokens) * spend.AudioCache.Pricing.ReadRatio))
}

// 视频
func video(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.Video == nil {
		spend.Video = new(common.VideoSpend)
	}

	for _, video := range mak.ReqModel.Pricing.Video {
		if video.IsDefault {
			spend.Video.Pricing = video
		}
	}

	spend.Video.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT * spend.Video.Pricing.OnceRatio))
}

// 搜索
func search(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.ChatCompletionRequest.WebSearchOptions == nil && (billingData.ChatCompletionRequest.Tools == nil || (!gstr.Contains(gconv.String(billingData.ChatCompletionRequest.Tools), "google_search") && !gstr.Contains(gconv.String(billingData.ChatCompletionRequest.Tools), "googleSearch"))) {
		return
	}

	if spend.Search == nil {
		spend.Search = new(common.SearchSpend)
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

	spend.Search.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT * spend.Search.Pricing.OnceRatio))
}

// Midjourney
func midjourney(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if billingData.Path == "" {
		return
	}

	for _, midjourney := range mak.ReqModel.Pricing.Midjourney {
		if billingData.Path == midjourney.Path {

			if spend.Midjourney == nil {
				spend.Midjourney = new(common.MidjourneySpend)
			}

			spend.Midjourney.Pricing = midjourney
			spend.Midjourney.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT * spend.Midjourney.Pricing.OnceRatio))
			return
		}
	}
}

// 一次
func once(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.Once == nil {
		spend.Once = new(common.OnceSpend)
	}

	spend.Once.Pricing = mak.ReqModel.Pricing.Once
	spend.Once.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT * spend.Once.Pricing.OnceRatio))

	if billingData.Usage != nil {
		spend.Once.InputTokens = billingData.Usage.PromptTokens
		spend.Once.OutputTokens = billingData.Usage.CompletionTokens
	}
}
