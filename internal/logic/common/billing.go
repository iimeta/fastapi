package common

import (
	"context"
	"math"
	"slices"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi-sdk/v2/tiktoken"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/common"
)

// 计算花费
func Billing(ctx context.Context, mak *MAK, billingData *common.BillingData, billingItems ...string) (spend common.Spend) {

	if billingItems == nil || len(billingItems) == 0 {
		billingItems = mak.ReqModel.Pricing.BillingItems
	}

	spend.BillingRule = mak.ReqModel.Pricing.BillingRule
	spend.BillingMethods = mak.ReqModel.Pricing.BillingMethods
	spend.BillingItems = billingItems
	spend.CurrencySymbol = mak.ReqModel.Pricing.CurrencySymbol

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
		case "video_generation":
			videoGeneration(ctx, mak, billingData, &spend)
		case "search":
			search(ctx, mak, billingData, &spend)
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

	if spend.VideoGeneration != nil {
		spend.TotalSpendTokens += spend.VideoGeneration.SpendTokens
	}

	if spend.VideoCache != nil {
		spend.TotalSpendTokens += spend.VideoCache.SpendTokens
	}

	if spend.Search != nil {
		spend.TotalSpendTokens += spend.Search.SpendTokens
	}

	if spend.Once != nil && (spend.TotalSpendTokens == 0 || mak.AppKey == nil || slices.Contains(mak.AppKey.BillingMethods, 2)) {
		spend.TotalSpendTokens = spend.Once.SpendTokens
	}

	// 模型时段折扣
	if mak.ReqModel.TimeRules != nil {
		if modelTimeRule := MatchTimeRule(ctx, mak.ReqModel.TimeRules); modelTimeRule != nil {
			spend.ModelTimeRule = modelTimeRule
			spend.TotalSpendTokens = int(math.Ceil(float64(spend.TotalSpendTokens) * modelTimeRule.Discount))
		}
	}

	// 分组时段折扣
	if mak.Group != nil {
		spend.GroupId = mak.Group.Id
		spend.GroupName = mak.Group.Name
		if groupTimeRule := MatchTimeRule(ctx, mak.Group.TimeRules, mak.ReqModel); groupTimeRule != nil {
			spend.GroupTimeRule = groupTimeRule
			spend.TotalSpendTokens = int(math.Ceil(float64(spend.TotalSpendTokens) * groupTimeRule.Discount))
		}
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
		spend.Text.OutputTokens = billingData.Usage.CompletionTokensDetails.TextTokens
		spend.Text.ReasoningTokens = billingData.Usage.OutputTokensDetails.ReasoningTokens
		spend.Text.SpendTokens = int(math.Ceil(float64(spend.Text.InputTokens)*spend.Text.Pricing.InputRatio)) + int(math.Ceil(float64(spend.Text.OutputTokens)*spend.Text.Pricing.OutputRatio)) + int(math.Ceil(float64(spend.Text.ReasoningTokens)*spend.Text.Pricing.ReasoningRatio))

		return
	}

	if spend.Text == nil {
		spend.Text = new(common.TextSpend)
	}

	if billingData.Usage.PromptTokens == 0 || billingData.Usage.CompletionTokens == 0 || mak.ReqModel.Pricing.BillingRule == 2 || billingData.IsAborted {

		var (
			promptTokens     int
			completionTokens int
		)

		if mak.ReqModel.Type == 100 {

			if len(billingData.ChatCompletionRequest.Messages) > 0 {

				if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]any); ok {

					for _, value := range multiContent {
						if content, ok := value.(map[string]any); ok {
							if content["type"] == "text" {
								promptTokens += TokensFromString(ctx, mak.ReqModel.Model, gconv.String(content))
							}
						} else {
							promptTokens += TokensFromString(ctx, mak.ReqModel.Model, gconv.String(value))
						}
					}

				} else {
					promptTokens = TokensFromMessages(ctx, model, billingData.ChatCompletionRequest.Messages)
				}
			}

		} else {
			promptTokens = TokensFromMessages(ctx, model, billingData.ChatCompletionRequest.Messages)
		}

		if billingData.Completion != "" {
			completionTokens = TokensFromString(ctx, model, billingData.Completion)
		}

		if promptTokens > billingData.Usage.PromptTokens {
			billingData.Usage.PromptTokens = promptTokens
		}

		if completionTokens > billingData.Usage.CompletionTokens {
			billingData.Usage.CompletionTokens = completionTokens
		}

		if promptTokens+completionTokens > billingData.Usage.TotalTokens {
			billingData.Usage.TotalTokens = promptTokens + completionTokens
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

	spend.Text.InputTokens = billingData.Usage.PromptTokens - billingData.Usage.PromptTokensDetails.CachedTokens
	spend.Text.OutputTokens = billingData.Usage.CompletionTokens
	spend.Text.ReasoningTokens = billingData.Usage.OutputTokensDetails.ReasoningTokens
	spend.Text.SpendTokens = int(math.Ceil(float64(spend.Text.InputTokens)*spend.Text.Pricing.InputRatio)) + int(math.Ceil(float64(spend.Text.OutputTokens)*spend.Text.Pricing.OutputRatio)) + int(math.Ceil(float64(spend.Text.ReasoningTokens)*spend.Text.Pricing.ReasoningRatio))
}

// 文本缓存
func textCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.TextCache == nil {
		spend.TextCache = new(common.CacheSpend)
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
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

	// Claude 5分钟缓存写入
	if billingData.Usage.CacheCreation5MInputTokens > 0 {
		spend.TextCache.Write5MTokens += billingData.Usage.CacheCreation5MInputTokens
	}

	// Claude 1小时缓存写入
	if billingData.Usage.CacheCreation1HInputTokens > 0 {
		spend.TextCache.Write1HTokens += billingData.Usage.CacheCreation1HInputTokens
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

	if spend.TextCache.Pricing.Write5MRatio > 0 || spend.TextCache.Pricing.Write1HRatio > 0 {
		spend.TextCache.SpendTokens = int(math.Ceil(float64(spend.TextCache.ReadTokens)*spend.TextCache.Pricing.ReadRatio)) +
			int(math.Ceil(float64(spend.TextCache.Write5MTokens)*spend.TextCache.Pricing.Write5MRatio)) +
			int(math.Ceil(float64(spend.TextCache.Write1HTokens)*spend.TextCache.Pricing.Write1HRatio))
	} else {
		spend.TextCache.SpendTokens = int(math.Ceil(float64(spend.TextCache.ReadTokens)*spend.TextCache.Pricing.ReadRatio)) + int(math.Ceil(float64(spend.TextCache.WriteTokens)*spend.TextCache.Pricing.WriteRatio))
	}
}

// 阶梯文本
func tieredText(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	model := mak.ReqModel.Model
	if !tiktoken.IsEncodingForModel(model) {
		model = consts.DEFAULT_MODEL
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
	}

	if billingData.Usage.PromptTokens == 0 || billingData.Usage.CompletionTokens == 0 || mak.ReqModel.Pricing.BillingRule == 2 || billingData.IsAborted {

		var (
			promptTokens     int
			completionTokens int
		)

		if mak.ReqModel.Type == 100 {

			if len(billingData.ChatCompletionRequest.Messages) > 0 {

				if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]any); ok {

					for _, value := range multiContent {
						if content, ok := value.(map[string]any); ok {
							if content["type"] == "text" {
								promptTokens += TokensFromString(ctx, mak.ReqModel.Model, gconv.String(content))
							}
						} else {
							promptTokens += TokensFromString(ctx, mak.ReqModel.Model, gconv.String(value))
						}
					}

				} else {
					promptTokens = TokensFromMessages(ctx, model, billingData.ChatCompletionRequest.Messages)
				}
			}

		} else {
			promptTokens = TokensFromMessages(ctx, model, billingData.ChatCompletionRequest.Messages)
		}

		if billingData.Completion != "" {
			completionTokens = TokensFromString(ctx, model, billingData.Completion)
		}

		if promptTokens > billingData.Usage.PromptTokens {
			billingData.Usage.PromptTokens = promptTokens
		}

		if completionTokens > billingData.Usage.CompletionTokens {
			billingData.Usage.CompletionTokens = completionTokens
		}

		if promptTokens+completionTokens > billingData.Usage.TotalTokens {
			billingData.Usage.TotalTokens = promptTokens + completionTokens
		}
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

	promptTokens := billingData.Usage.PromptTokens

	if billingData.Usage.CompletionTokens > promptTokens {
		promptTokens = billingData.Usage.CompletionTokens
	}

	if billingData.Usage.OutputTokensDetails.ReasoningTokens > promptTokens {
		promptTokens = billingData.Usage.OutputTokensDetails.ReasoningTokens
	}

	for i, tieredText := range mak.ReqModel.Pricing.TieredText {
		if mode == tieredText.Mode && ((promptTokens > tieredText.Gt && promptTokens <= tieredText.Lte) || (i == len(mak.ReqModel.Pricing.TieredText)-1)) {
			spend.TieredText.Pricing = tieredText
			spend.TieredText.InputTokens = billingData.Usage.PromptTokens - billingData.Usage.PromptTokensDetails.CachedTokens
			spend.TieredText.OutputTokens = billingData.Usage.CompletionTokens
			spend.TieredText.ReasoningTokens = billingData.Usage.OutputTokensDetails.ReasoningTokens
			spend.TieredText.SpendTokens = int(math.Ceil(float64(spend.TieredText.InputTokens)*spend.TieredText.Pricing.InputRatio)) + int(math.Ceil(float64(spend.TieredText.OutputTokens)*spend.TieredText.Pricing.OutputRatio)) + int(math.Ceil(float64(spend.TieredText.ReasoningTokens)*spend.TieredText.Pricing.ReasoningRatio))
			return
		}
	}
}

// 阶梯文本缓存
func tieredTextCache(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.TieredTextCache == nil {
		spend.TieredTextCache = new(common.CacheSpend)
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
	}

	if billingData.Usage.PromptTokensDetails.CachedTokens > 0 {
		spend.TieredTextCache.ReadTokens += billingData.Usage.PromptTokensDetails.CachedTokens
	}

	if billingData.Usage.CompletionTokensDetails.CachedTokens > 0 {
		spend.TieredTextCache.ReadTokens += billingData.Usage.CompletionTokensDetails.CachedTokens
	}

	// Claude
	if billingData.Usage.CacheReadInputTokens > 0 {
		spend.TieredTextCache.ReadTokens += billingData.Usage.CacheReadInputTokens
	}

	// Claude
	if billingData.Usage.CacheCreationInputTokens > 0 {
		spend.TieredTextCache.WriteTokens += billingData.Usage.CacheCreationInputTokens
	}

	// Claude 5分钟缓存写入
	if billingData.Usage.CacheCreation5MInputTokens > 0 {
		spend.TieredTextCache.Write5MTokens += billingData.Usage.CacheCreation5MInputTokens
	}

	// Claude 1小时缓存写入
	if billingData.Usage.CacheCreation1HInputTokens > 0 {
		spend.TieredTextCache.Write1HTokens += billingData.Usage.CacheCreation1HInputTokens
	}

	mode := "all"
	if billingData.ChatCompletionRequest.EnableThinking != nil {
		if *billingData.ChatCompletionRequest.EnableThinking {
			mode = "thinking"
		} else {
			mode = "non_thinking"
		}
	}

	readTokens := spend.TieredTextCache.ReadTokens

	if spend.TieredTextCache.WriteTokens > readTokens {
		readTokens = spend.TieredTextCache.WriteTokens
	}

	if spend.TieredTextCache.Write5MTokens > readTokens {
		readTokens = spend.TieredTextCache.Write5MTokens
	}

	if spend.TieredTextCache.Write1HTokens > readTokens {
		readTokens = spend.TieredTextCache.Write1HTokens
	}

	for i, tieredTextCache := range mak.ReqModel.Pricing.TieredTextCache {
		if mode == tieredTextCache.Mode && ((readTokens > tieredTextCache.Gt && readTokens <= tieredTextCache.Lte) || (i == len(mak.ReqModel.Pricing.TieredTextCache)-1)) {
			spend.TieredTextCache.Pricing = tieredTextCache
			if spend.TieredTextCache.Pricing.Write5MRatio > 0 || spend.TieredTextCache.Pricing.Write1HRatio > 0 {
				spend.TieredTextCache.SpendTokens = int(math.Ceil(float64(spend.TieredTextCache.ReadTokens)*spend.TieredTextCache.Pricing.ReadRatio)) +
					int(math.Ceil(float64(spend.TieredTextCache.Write5MTokens)*spend.TieredTextCache.Pricing.Write5MRatio)) +
					int(math.Ceil(float64(spend.TieredTextCache.Write1HTokens)*spend.TieredTextCache.Pricing.Write1HRatio))
			} else {
				spend.TieredTextCache.SpendTokens = int(math.Ceil(float64(spend.TieredTextCache.ReadTokens)*spend.TieredTextCache.Pricing.ReadRatio)) + int(math.Ceil(float64(spend.TieredTextCache.WriteTokens)*spend.TieredTextCache.Pricing.WriteRatio))
			}
			return
		}
	}
}

// 图像
func image(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.Image == nil {
		spend.Image = new(common.ImageSpend)
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
	}

	if billingData.Usage.InputTokensDetails.ImageTokens > 0 {
		spend.Image.InputTokens += billingData.Usage.InputTokensDetails.ImageTokens
	}

	if billingData.Usage.OutputTokensDetails.ImageTokens > 0 {
		spend.Image.OutputTokens += billingData.Usage.OutputTokensDetails.ImageTokens
	} else if billingData.Usage.CompletionTokensDetails.ImageTokens > 0 {
		spend.Image.OutputTokens += billingData.Usage.CompletionTokensDetails.ImageTokens
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
		quality     = billingData.ImageGenerationRequest.Quality
		size        = billingData.ImageGenerationRequest.Size
		aspectRatio = billingData.ImageGenerationRequest.AspectRatio
		width       int
		height      int
	)

	if quality == "" {
		quality = billingData.ImageEditRequest.Quality
	}

	if size == "" {
		size = billingData.ImageEditRequest.Size
	}

	if aspectRatio == "" {
		aspectRatio = billingData.ImageEditRequest.AspectRatio
	}

	if aspectRatio == "" {
		aspectRatio = "1:1"
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
		} else {

			if gstr.HasSuffix(size, "K") {
				quality = size
			}

			if quality == "" || !gstr.HasSuffix(quality, "K") {
				quality = "1K"
			}

			if size = consts.RESOLUTION_ASPECT_RATIO[quality+aspectRatio]; size != "" {
				widthHeight = gstr.Split(size, `x`)
				width = gconv.Int(widthHeight[0])
				height = gconv.Int(widthHeight[1])
			}
		}

	} else if gstr.HasSuffix(quality, "K") {

		if size = consts.RESOLUTION_ASPECT_RATIO[quality+aspectRatio]; size != "" {
			widthHeight := gstr.Split(size, `x`)
			width = gconv.Int(widthHeight[0])
			height = gconv.Int(widthHeight[1])
		}
	}

	for _, imageGeneration := range mak.ReqModel.Pricing.ImageGeneration {

		if (imageGeneration.Quality == quality || imageGeneration.Quality == "") && imageGeneration.Width == width && imageGeneration.Height == height {
			spend.ImageGeneration.Pricing = imageGeneration
			break
		}

		if imageGeneration.IsDefault {
			spend.ImageGeneration.Pricing = imageGeneration
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

	if spend.ImageCache == nil {
		spend.ImageCache = new(common.CacheSpend)
	}

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
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

	if len(billingData.ChatCompletionRequest.Messages) > 0 {

		if multiContent, ok := billingData.ChatCompletionRequest.Messages[len(billingData.ChatCompletionRequest.Messages)-1].Content.([]any); ok {

			for _, value := range multiContent {

				if content, ok := value.(map[string]any); ok && content["type"] == "image_url" {

					if imageUrl, ok := content["image_url"].(map[string]any); ok {

						if spend.Vision == nil {
							spend.Vision = new(common.VisionSpend)
						}

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

	if billingData.Usage == nil {
		billingData.Usage = new(smodel.Usage)
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

// 视频生成
func videoGeneration(ctx context.Context, mak *MAK, billingData *common.BillingData, spend *common.Spend) {

	if spend.VideoGeneration == nil {
		spend.VideoGeneration = new(common.VideoGenerationSpend)
	}

	var (
		mode   = billingData.VideoMode
		size   = billingData.Size
		width  int
		height int
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

	} else if billingData.IsVolcEngine && billingData.VolcVideoCreateReq != nil {

		resolution := billingData.VolcVideoCreateReq.Resolution
		if resolution == "" {
			resolution = "720p"
		}

		ratio := billingData.VolcVideoCreateReq.Ratio
		if ratio == "" {
			ratio = "16:9"
		}

		if size = consts.VIDEO_RESOLUTION_RATIO[resolution+ratio]; size != "" {
			widthHeight := gstr.Split(size, `x`)
			width = gconv.Int(widthHeight[0])
			height = gconv.Int(widthHeight[1])
		}
	}

	for _, videoGeneration := range mak.ReqModel.Pricing.VideoGeneration {

		if videoGeneration.Width == width && videoGeneration.Height == height {
			if mode == "" || videoGeneration.Mode == "" || videoGeneration.Mode == mode {
				spend.VideoGeneration.Pricing = videoGeneration
				break
			}
		}

		if videoGeneration.IsDefault {
			spend.VideoGeneration.Pricing = videoGeneration
		}
	}

	spend.VideoGeneration.Seconds = billingData.Seconds

	if !billingData.IsVolcEngine {
		spend.VideoGeneration.SpendTokens = int(math.Ceil(consts.QUOTA_DEFAULT_UNIT*spend.VideoGeneration.Pricing.OnceRatio)) * spend.VideoGeneration.Seconds
	}
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
		if content, ok := billingData.ChatCompletionRequest.WebSearchOptions.(map[string]any); ok {
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

func MatchTimeRule(ctx context.Context, rules []*common.TimeRule, model ...*model.Model) *common.TimeRule {

	if len(rules) == 0 {
		return nil
	}

	enterTime := g.RequestFromCtx(ctx).EnterTime
	weekday := int(enterTime.Weekday())
	dayOfMonth := enterTime.Day()

	enterTimeMs := int64(enterTime.Hour()*3600+enterTime.Minute()*60+enterTime.Second()) * 1000

	hasModel := len(model) > 0 && model[0] != nil

	var firstTimeRule, fallbackRule *common.TimeRule

	for _, rule := range rules {

		if rule.TimeType != "all" && len(rule.Days) > 0 {

			matched := false

			if rule.DayMode == "month" {
				for _, d := range rule.Days {
					if d == dayOfMonth {
						matched = true
						break
					}
				}
			} else {
				for _, d := range rule.Days {
					if d == weekday {
						matched = true
						break
					}
				}
			}

			if !matched {
				continue
			}
		}

		if !matchTimeRange(enterTimeMs, rule.StartTime, rule.EndTime) {
			continue
		}

		if hasModel {

			if len(rule.Models) > 0 {
				if slices.Contains(rule.Models, model[0].Id) {
					return rule
				}
				continue
			}

			if fallbackRule == nil {
				fallbackRule = rule
			}

		} else {
			if firstTimeRule == nil {
				firstTimeRule = rule
			}
		}
	}

	if hasModel {
		return fallbackRule
	}

	return firstTimeRule
}

func matchTimeRange(enterTimeMs, startTime, endTime int64) bool {
	if startTime <= endTime {
		return enterTimeMs >= startTime && enterTimeMs <= endTime
	}
	return enterTimeMs >= startTime || enterTimeMs <= endTime
}
