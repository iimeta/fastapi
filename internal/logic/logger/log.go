package logger

import (
	"context"
	"slices"
	"time"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	serrors "github.com/iimeta/fastapi-sdk/errors"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sLogger struct{}

func init() {
	service.RegisterLogger(New())
}

func New() service.ILogger {
	return &sLogger{}
}

// 文本日志
func (s *sLogger) Text(ctx context.Context, chatLog model.LogText, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLogger LogText time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if chatLog.CompletionsRes.Error != nil && checkError(chatLog.CompletionsRes.Error) {
		return
	}

	chat := do.LogText{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: chatLog.IsSmartMatch,
		Stream:       chatLog.CompletionsReq.Stream,
		Spend:        chatLog.Spend,
		ConnTime:     chatLog.CompletionsRes.ConnTime,
		Duration:     chatLog.CompletionsRes.Duration,
		TotalTime:    chatLog.CompletionsRes.TotalTime,
		InternalTime: chatLog.CompletionsRes.InternalTime,
		ReqTime:      chatLog.CompletionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(chatLog.CompletionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {

		if chatLog.CompletionsReq != nil {

			if len(chatLog.CompletionsReq.Messages) > 0 {

				prompt := chatLog.CompletionsReq.Messages[len(chatLog.CompletionsReq.Messages)-1].Content

				if chatLog.ReqModel.Type == 102 {

					if slices.Contains(config.Cfg.Log.ChatRecords, "audio") {
						chat.Prompt = gconv.String(prompt)
					} else {
						if multiContent, ok := prompt.([]interface{}); ok {

							multiContents := make([]interface{}, 0)

							for _, value := range multiContent {

								if content, ok := value.(map[string]interface{}); ok {

									if content["type"] == "input_audio" {

										if inputAudio, ok := content["input_audio"].(map[string]interface{}); ok {

											inputAudio = gmap.NewStrAnyMapFrom(inputAudio).MapCopy()
											inputAudio["data"] = "[BASE64音频数据]"

											content = gmap.NewStrAnyMapFrom(content).MapCopy()
											content["input_audio"] = inputAudio
										}
									}

									value = content
								}

								multiContents = append(multiContents, value)
							}

							chat.Prompt = gconv.String(multiContents)

						} else {
							chat.Prompt = gconv.String(prompt)
						}
					}

				} else {

					if slices.Contains(config.Cfg.Log.ChatRecords, "image") {
						chat.Prompt = gconv.String(prompt)
					} else {
						if multiContent, ok := prompt.([]interface{}); ok {

							multiContents := make([]interface{}, 0)

							for _, value := range multiContent {

								if content, ok := value.(map[string]interface{}); ok {

									if content["type"] == "image_url" {

										if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

											if !gstr.HasPrefix(gconv.String(imageUrl["url"]), "http") {

												imageUrl = gmap.NewStrAnyMapFrom(imageUrl).MapCopy()
												imageUrl["url"] = "[BASE64图像数据]"

												content = gmap.NewStrAnyMapFrom(content).MapCopy()
												content["image_url"] = imageUrl
											}
										}
									}

									if content["type"] == "image" {
										if source, ok := content["source"].(smodel.Source); ok {
											source.Data = "[BASE64图像数据]"
											content = gmap.NewStrAnyMapFrom(content).MapCopy()
											content["source"] = source
										}
									}

									value = content
								}

								multiContents = append(multiContents, value)
							}

							chat.Prompt = gconv.String(multiContents)

						} else if multiContent, ok := prompt.([]smodel.OpenAIResponsesContent); ok {

							multiContents := make([]smodel.OpenAIResponsesContent, 0)

							for _, value := range multiContent {
								if value.Type == "input_image" && !gstr.HasPrefix(value.ImageUrl, "http") {
									value.ImageUrl = "[BASE64图像数据]"
								}
								multiContents = append(multiContents, value)
							}

							chat.Prompt = gconv.String(multiContents)

						} else {
							chat.Prompt = gconv.String(prompt)
						}
					}
				}
			}

		} else if chatLog.EmbeddingReq != nil {
			chat.Prompt = gconv.String(chatLog.EmbeddingReq.Input)
		} else if chatLog.ModerationReq != nil {
			chat.Prompt = gconv.String(chatLog.ModerationReq.Input)
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") && chatLog.CompletionsRes != nil {
		chat.Completion = chatLog.CompletionsRes.Completion
	}

	if chatLog.ReqModel != nil {
		chat.ProviderId = chatLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, chatLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			chat.ProviderName = provider.Name
		}
		chat.ModelId = chatLog.ReqModel.Id
		chat.ModelName = chatLog.ReqModel.Name
		chat.Model = chatLog.ReqModel.Model
		chat.ModelType = chatLog.ReqModel.Type
	}

	if chatLog.RealModel != nil {
		chat.IsEnablePresetConfig = chatLog.RealModel.IsEnablePresetConfig
		chat.PresetConfig = chatLog.RealModel.PresetConfig
		chat.IsEnableForward = chatLog.RealModel.IsEnableForward
		chat.ForwardConfig = chatLog.RealModel.ForwardConfig
		chat.IsEnableModelAgent = chatLog.RealModel.IsEnableModelAgent
		chat.RealModelId = chatLog.RealModel.Id
		chat.RealModelName = chatLog.RealModel.Name
		chat.RealModel = chatLog.RealModel.Model
	}

	if chatLog.ModelAgent != nil {
		chat.IsEnableModelAgent = true
		chat.ModelAgentId = chatLog.ModelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			ProviderId: chatLog.ModelAgent.ProviderId,
			Name:       chatLog.ModelAgent.Name,
			BaseUrl:    chatLog.ModelAgent.BaseUrl,
			Path:       chatLog.ModelAgent.Path,
			Weight:     chatLog.ModelAgent.Weight,
			Remark:     chatLog.ModelAgent.Remark,
		}
	}

	if chatLog.FallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     chatLog.FallbackModelAgent.Id,
			ModelAgentName: chatLog.FallbackModelAgent.Name,
		}
	}

	if chatLog.FallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = chatLog.FallbackModel.Model
		chat.FallbackConfig.ModelName = chatLog.FallbackModel.Name
	}

	if chatLog.Key != nil {
		chat.Key = chatLog.Key.Key
	}

	if chatLog.CompletionsRes.Error != nil {

		chat.ErrMsg = chatLog.CompletionsRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(chatLog.CompletionsRes.Error, &openaiApiError) {
			chat.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(chatLog.CompletionsRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "messages") && chatLog.CompletionsReq != nil {
		for _, message := range chatLog.CompletionsReq.Messages {

			content := message.Content

			if !slices.Contains(config.Cfg.Log.ChatRecords, "image") {

				if multiContent, ok := content.([]interface{}); ok {

					multiContents := make([]interface{}, 0)

					for _, value := range multiContent {

						if content, ok := value.(map[string]interface{}); ok {

							if content["type"] == "image_url" {

								if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

									if !gstr.HasPrefix(gconv.String(imageUrl["url"]), "http") {

										imageUrl = gmap.NewStrAnyMapFrom(imageUrl).MapCopy()
										imageUrl["url"] = "[BASE64图像数据]"

										content = gmap.NewStrAnyMapFrom(content).MapCopy()
										content["image_url"] = imageUrl
									}
								}
							}

							if content["type"] == "image" {
								if source, ok := content["source"].(smodel.Source); ok {
									source.Data = "[BASE64图像数据]"
									content = gmap.NewStrAnyMapFrom(content).MapCopy()
									content["source"] = source
								}
							}

							value = content
						}

						multiContents = append(multiContents, value)
					}

					content = multiContents

				} else if multiContent, ok := content.([]smodel.OpenAIResponsesContent); ok {

					multiContents := make([]smodel.OpenAIResponsesContent, 0)

					for _, value := range multiContent {
						if value.Type == "input_image" && !gstr.HasPrefix(value.ImageUrl, "http") {
							value.ImageUrl = "[BASE64图像数据]"
						}
						multiContents = append(multiContents, value)
					}

					content = multiContents
				}
			}

			if !slices.Contains(config.Cfg.Log.ChatRecords, "audio") {

				if multiContent, ok := content.([]interface{}); ok {

					multiContents := make([]interface{}, 0)

					for _, value := range multiContent {

						if content, ok := value.(map[string]interface{}); ok {

							if content["type"] == "input_audio" {

								if inputAudio, ok := content["input_audio"].(map[string]interface{}); ok {

									inputAudio = gmap.NewStrAnyMapFrom(inputAudio).MapCopy()
									inputAudio["data"] = "[BASE64音频数据]"

									content = gmap.NewStrAnyMapFrom(content).MapCopy()
									content["input_audio"] = inputAudio
								}
							}

							value = content
						}

						multiContents = append(multiContents, value)
					}

					content = multiContents
				}
			}

			chat.Messages = append(chat.Messages, mcommon.Message{
				Role:         message.Role,
				Content:      gconv.String(content),
				Refusal:      message.Refusal,
				Name:         message.Name,
				FunctionCall: message.FunctionCall,
				ToolCalls:    message.ToolCalls,
				ToolCallId:   message.ToolCallId,
				Audio:        message.Audio,
			})
		}
	}

	if chatLog.RetryInfo != nil {

		chat.IsRetry = chatLog.RetryInfo.IsRetry
		chat.Retry = &mcommon.Retry{
			IsRetry:    chatLog.RetryInfo.IsRetry,
			RetryCount: chatLog.RetryInfo.RetryCount,
			ErrMsg:     chatLog.RetryInfo.ErrMsg,
		}

		if chat.IsRetry {
			chat.Status = 3
			chat.ErrMsg = chatLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogText.Insert(ctx, chat); err != nil {
		logger.Errorf(ctx, "sLogger LogText error: %v", err)

		if err.Error() == "an inserted document is too large" {
			if chatLog.CompletionsReq != nil {
				chatLog.CompletionsReq.Messages = []smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: err.Error(),
				}}
			} else if chatLog.EmbeddingReq != nil {
				chatLog.EmbeddingReq.Input = err.Error()
			} else if chatLog.ModerationReq != nil {
				chatLog.ModerationReq.Input = err.Error()
			}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLogger LogText retry: %d", len(retry))

		s.Text(ctx, chatLog, retry...)
	}
}

// 绘图日志
func (s *sLogger) Image(ctx context.Context, imageLog model.LogImage, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLogger LogImage time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if imageLog.ImageRes.Error != nil && checkError(imageLog.ImageRes.Error) {
		return
	}

	image := do.LogImage{
		TraceId:        gctx.CtxId(ctx),
		UserId:         service.Session().GetUserId(ctx),
		AppId:          service.Session().GetAppId(ctx),
		Prompt:         imageLog.ImageReq.Prompt,
		Size:           imageLog.ImageReq.Size,
		N:              imageLog.ImageReq.N,
		Quality:        imageLog.ImageReq.Quality,
		Style:          imageLog.ImageReq.Style,
		ResponseFormat: imageLog.ImageReq.ResponseFormat,
		Spend:          imageLog.Spend,
		TotalTime:      imageLog.ImageRes.TotalTime,
		InternalTime:   imageLog.ImageRes.InternalTime,
		ReqTime:        imageLog.ImageRes.EnterTime,
		ReqDate:        gtime.NewFromTimeStamp(imageLog.ImageRes.EnterTime).Format("Y-m-d"),
		ClientIp:       g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:       g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:        util.GetLocalIp(),
		Status:         1,
		Host:           g.RequestFromCtx(ctx).GetHost(),
		Rid:            service.Session().GetRid(ctx),
	}

	for _, data := range imageLog.ImageRes.Data {
		image.ImageData = append(image.ImageData, mcommon.ImageData{
			Url: data.Url,
			//B64Json:       data.B64Json, // todo 太大了, 不存
			RevisedPrompt: data.RevisedPrompt,
		})
	}

	if imageLog.ReqModel != nil {
		image.ProviderId = imageLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, imageLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			image.ProviderName = provider.Name
		}
		image.ModelId = imageLog.ReqModel.Id
		image.ModelName = imageLog.ReqModel.Name
		image.Model = imageLog.ReqModel.Model
		image.ModelType = imageLog.ReqModel.Type
	}

	if imageLog.RealModel != nil {
		image.IsEnablePresetConfig = imageLog.RealModel.IsEnablePresetConfig
		image.PresetConfig = imageLog.RealModel.PresetConfig
		image.IsEnableForward = imageLog.RealModel.IsEnableForward
		image.ForwardConfig = imageLog.RealModel.ForwardConfig
		image.IsEnableModelAgent = imageLog.RealModel.IsEnableModelAgent
		image.RealModelId = imageLog.RealModel.Id
		image.RealModelName = imageLog.RealModel.Name
		image.RealModel = imageLog.RealModel.Model
	}

	if image.IsEnableModelAgent && imageLog.ModelAgent != nil {
		image.ModelAgentId = imageLog.ModelAgent.Id
		image.ModelAgent = &do.ModelAgent{
			ProviderId: imageLog.ModelAgent.ProviderId,
			Name:       imageLog.ModelAgent.Name,
			BaseUrl:    imageLog.ModelAgent.BaseUrl,
			Path:       imageLog.ModelAgent.Path,
			Weight:     imageLog.ModelAgent.Weight,
			Remark:     imageLog.ModelAgent.Remark,
		}
	}

	if imageLog.FallbackModelAgent != nil {
		image.IsEnableFallback = true
		image.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     imageLog.FallbackModelAgent.Id,
			ModelAgentName: imageLog.FallbackModelAgent.Name,
		}
	}

	if imageLog.FallbackModel != nil {
		image.IsEnableFallback = true
		if image.FallbackConfig == nil {
			image.FallbackConfig = new(mcommon.FallbackConfig)
		}
		image.FallbackConfig.Model = imageLog.FallbackModel.Model
		image.FallbackConfig.ModelName = imageLog.FallbackModel.Name
	}

	if imageLog.Key != nil {
		image.Key = imageLog.Key.Key
	}

	if imageLog.ImageRes.Error != nil {

		image.ErrMsg = imageLog.ImageRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(imageLog.ImageRes.Error, &openaiApiError) {
			image.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(imageLog.ImageRes.Error) {
			image.Status = 2
		} else {
			image.Status = -1
		}
	}

	if imageLog.RetryInfo != nil {

		image.IsRetry = imageLog.RetryInfo.IsRetry
		image.Retry = &mcommon.Retry{
			IsRetry:    imageLog.RetryInfo.IsRetry,
			RetryCount: imageLog.RetryInfo.RetryCount,
			ErrMsg:     imageLog.RetryInfo.ErrMsg,
		}

		if image.IsRetry {
			image.Status = 3
			image.ErrMsg = imageLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogImage.Insert(ctx, image); err != nil {
		logger.Errorf(ctx, "sLogger LogImage error: %v", err)

		if err.Error() == "an inserted document is too large" {
			imageLog.ImageReq.Prompt = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLogger LogImage retry: %d", len(retry))

		s.Image(ctx, imageLog, retry...)
	}
}

// 音频日志
func (s *sLogger) Audio(ctx context.Context, audioLog model.LogAudio, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLogger LogAudio time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if audioLog.AudioRes.Error != nil && checkError(audioLog.AudioRes.Error) {
		return
	}

	audio := do.LogAudio{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Input:        audioLog.AudioReq.Input,
		Text:         audioLog.AudioRes.Text,
		Spend:        audioLog.Spend,
		TotalTime:    audioLog.AudioRes.TotalTime,
		InternalTime: audioLog.AudioRes.InternalTime,
		ReqTime:      audioLog.AudioRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(audioLog.AudioRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if audioLog.ReqModel != nil {
		audio.ProviderId = audioLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, audioLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			audio.ProviderName = provider.Name
		}
		audio.ModelId = audioLog.ReqModel.Id
		audio.ModelName = audioLog.ReqModel.Name
		audio.Model = audioLog.ReqModel.Model
		audio.ModelType = audioLog.ReqModel.Type
	}

	if audioLog.RealModel != nil {
		audio.IsEnablePresetConfig = audioLog.RealModel.IsEnablePresetConfig
		audio.PresetConfig = audioLog.RealModel.PresetConfig
		audio.IsEnableForward = audioLog.RealModel.IsEnableForward
		audio.ForwardConfig = audioLog.RealModel.ForwardConfig
		audio.IsEnableModelAgent = audioLog.RealModel.IsEnableModelAgent
		audio.RealModelId = audioLog.RealModel.Id
		audio.RealModelName = audioLog.RealModel.Name
		audio.RealModel = audioLog.RealModel.Model
	}

	if audio.IsEnableModelAgent && audioLog.ModelAgent != nil {
		audio.ModelAgentId = audioLog.ModelAgent.Id
		audio.ModelAgent = &do.ModelAgent{
			ProviderId: audioLog.ModelAgent.ProviderId,
			Name:       audioLog.ModelAgent.Name,
			BaseUrl:    audioLog.ModelAgent.BaseUrl,
			Path:       audioLog.ModelAgent.Path,
			Weight:     audioLog.ModelAgent.Weight,
			Remark:     audioLog.ModelAgent.Remark,
			Status:     audioLog.ModelAgent.Status,
		}
	}

	if audioLog.FallbackModelAgent != nil {
		audio.IsEnableFallback = true
		audio.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     audioLog.FallbackModelAgent.Id,
			ModelAgentName: audioLog.FallbackModelAgent.Name,
		}
	}

	if audioLog.FallbackModel != nil {
		audio.IsEnableFallback = true
		if audio.FallbackConfig == nil {
			audio.FallbackConfig = new(mcommon.FallbackConfig)
		}
		audio.FallbackConfig.Model = audioLog.FallbackModel.Model
		audio.FallbackConfig.ModelName = audioLog.FallbackModel.Name
	}

	if audioLog.Key != nil {
		audio.Key = audioLog.Key.Key
	}

	if audioLog.AudioRes.Error != nil {

		audio.ErrMsg = audioLog.AudioRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(audioLog.AudioRes.Error, &openaiApiError) {
			audio.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(audioLog.AudioRes.Error) {
			audio.Status = 2
		} else {
			audio.Status = -1
		}
	}

	if audioLog.RetryInfo != nil {

		audio.IsRetry = audioLog.RetryInfo.IsRetry
		audio.Retry = &mcommon.Retry{
			IsRetry:    audioLog.RetryInfo.IsRetry,
			RetryCount: audioLog.RetryInfo.RetryCount,
			ErrMsg:     audioLog.RetryInfo.ErrMsg,
		}

		if audio.IsRetry {
			audio.Status = 3
			audio.ErrMsg = audioLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogAudio.Insert(ctx, audio); err != nil {
		logger.Errorf(ctx, "sLogger LogAudio error: %v", err)

		if err.Error() == "an inserted document is too large" {
			audioLog.AudioReq.Input = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLogger LogAudio retry: %d", len(retry))

		s.Audio(ctx, audioLog, retry...)
	}
}

// Midjourney日志
func (s *sLogger) Midjourney(ctx context.Context, midjourneyLog model.LogMidjourney, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLogger LogMidjourney time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if midjourneyLog.Response.Error != nil && checkError(midjourneyLog.Response.Error) {
		return
	}

	midjourney := do.LogMidjourney{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		ReqUrl:       midjourneyLog.Response.ReqUrl,
		TaskId:       midjourneyLog.Response.TaskId,
		Action:       midjourneyLog.Response.Action,
		Prompt:       midjourneyLog.Response.Prompt,
		PromptEn:     midjourneyLog.Response.PromptEn,
		ImageUrl:     midjourneyLog.Response.ImageUrl,
		Progress:     midjourneyLog.Response.Progress,
		Spend:        midjourneyLog.Spend,
		ConnTime:     midjourneyLog.Response.ConnTime,
		Duration:     midjourneyLog.Response.Duration,
		TotalTime:    midjourneyLog.Response.TotalTime,
		InternalTime: midjourneyLog.Response.InternalTime,
		ReqTime:      midjourneyLog.Response.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(midjourneyLog.Response.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if midjourneyLog.ReqModel != nil {
		midjourney.ProviderId = midjourneyLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, midjourneyLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			midjourney.ProviderName = provider.Name
		}
		midjourney.ModelId = midjourneyLog.ReqModel.Id
		midjourney.ModelName = midjourneyLog.ReqModel.Name
		midjourney.Model = midjourneyLog.ReqModel.Model
		midjourney.ModelType = midjourneyLog.ReqModel.Type
	}

	if midjourneyLog.RealModel != nil {
		midjourney.IsEnablePresetConfig = midjourneyLog.RealModel.IsEnablePresetConfig
		midjourney.PresetConfig = midjourneyLog.RealModel.PresetConfig
		midjourney.IsEnableForward = midjourneyLog.RealModel.IsEnableForward
		midjourney.ForwardConfig = midjourneyLog.RealModel.ForwardConfig
		midjourney.IsEnableModelAgent = midjourneyLog.RealModel.IsEnableModelAgent
		midjourney.RealModelId = midjourneyLog.RealModel.Id
		midjourney.RealModelName = midjourneyLog.RealModel.Name
		midjourney.RealModel = midjourneyLog.RealModel.Model
	}

	if midjourney.IsEnableModelAgent && midjourneyLog.ModelAgent != nil {
		midjourney.ModelAgentId = midjourneyLog.ModelAgent.Id
		midjourney.ModelAgent = &do.ModelAgent{
			ProviderId: midjourneyLog.ModelAgent.ProviderId,
			Name:       midjourneyLog.ModelAgent.Name,
			BaseUrl:    midjourneyLog.ModelAgent.BaseUrl,
			Path:       midjourneyLog.ModelAgent.Path,
			Weight:     midjourneyLog.ModelAgent.Weight,
			Remark:     midjourneyLog.ModelAgent.Remark,
		}
	}

	if midjourneyLog.FallbackModelAgent != nil {
		midjourney.IsEnableFallback = true
		midjourney.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     midjourneyLog.FallbackModelAgent.Id,
			ModelAgentName: midjourneyLog.FallbackModelAgent.Name,
		}
	}

	if midjourneyLog.FallbackModel != nil {
		midjourney.IsEnableFallback = true
		if midjourney.FallbackConfig == nil {
			midjourney.FallbackConfig = new(mcommon.FallbackConfig)
		}
		midjourney.FallbackConfig.Model = midjourneyLog.FallbackModel.Model
		midjourney.FallbackConfig.ModelName = midjourneyLog.FallbackModel.Name
	}

	if midjourneyLog.Key != nil {
		midjourney.Key = midjourneyLog.Key.Key
	}

	if midjourneyLog.Response.Response != nil {
		if err := gjson.Unmarshal(midjourneyLog.Response.Response, &midjourney.Response); err != nil {
			logger.Error(ctx, err)
		}
	}

	if midjourneyLog.Response.Error != nil {
		midjourney.ErrMsg = midjourneyLog.Response.Error.Error()
		if common.IsAborted(midjourneyLog.Response.Error) {
			midjourney.Status = 2
		} else {
			midjourney.Status = -1
		}
	}

	if midjourneyLog.RetryInfo != nil {

		midjourney.IsRetry = midjourneyLog.RetryInfo.IsRetry
		midjourney.Retry = &mcommon.Retry{
			IsRetry:    midjourneyLog.RetryInfo.IsRetry,
			RetryCount: midjourneyLog.RetryInfo.RetryCount,
			ErrMsg:     midjourneyLog.RetryInfo.ErrMsg,
		}

		if midjourney.IsRetry {
			midjourney.Status = 3
			midjourney.ErrMsg = midjourneyLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogMidjourney.Insert(ctx, midjourney); err != nil {
		logger.Errorf(ctx, "sLogger LogMidjourney error: %v", err)

		if err.Error() == "an inserted document is too large" {
			midjourneyLog.Response.Prompt = err.Error()
			midjourneyLog.Response.PromptEn = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLogger LogMidjourney retry: %d", len(retry))

		s.Midjourney(ctx, midjourneyLog, retry...)
	}
}

func checkError(err error) bool {
	return errors.Is(err, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(err, errors.ERR_MODEL_DISABLED) ||
		errors.Is(err, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(err, errors.ERR_GROUP_DISABLED) ||
		errors.Is(err, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(err, errors.ERR_GROUP_INSUFFICIENT_QUOTA)
}
