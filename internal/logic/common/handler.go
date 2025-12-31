package common

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/model/do"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

// 前置处理器
func BeforeHandler(ctx context.Context, before *mcommon.BeforeHandler) {
}

// 后置处理器
func AfterHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.IsFile {
		fileHandler(ctx, mak, after)
		return
	} else if after.IsBatch {
		batchHandler(ctx, mak, after)
		return
	}

	switch mak.ReqModel.Type {
	case 1, 100, 101, 102, 7, 103:
		textHandler(ctx, mak, after)
	case 2, 3, 4:
		if mak.ReqModel.Model == "midjourney" || gstr.HasPrefix(mak.ReqModel.Model, "midjourney") {
			midjourneyHandler(ctx, mak, after)
		} else {
			imageHandler(ctx, mak, after)
		}
	case 5, 6:
		audioHandler(ctx, mak, after)
	case 8:
		videoHandler(ctx, mak, after)
	default:
		generalHandler(ctx, mak, after)
	}
}

func textHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		if after.ServiceTier == "" {
			after.ServiceTier = after.ChatCompletionRes.ServiceTier
			if after.ServiceTier == "" {
				after.ServiceTier = after.ChatCompletionReq.ServiceTier
			}
		}

		billingData := &mcommon.BillingData{
			ChatCompletionRequest: after.ChatCompletionReq,
			EmbeddingRequest:      after.EmbeddingReq,
			ModerationRequest:     after.ModerationReq,
			Completion:            after.Completion,
			ServiceTier:           after.ServiceTier,
			Usage:                 after.Usage,
			IsAborted:             IsAborted(after.Error),
		}

		if billingData.Completion == "" && len(after.ChatCompletionRes.Choices) > 0 && after.ChatCompletionRes.Choices[0].Message != nil {
			if mak.RealModel.Type == 102 && after.ChatCompletionRes.Choices[0].Message.Audio != nil {
				billingData.Completion = after.ChatCompletionRes.Choices[0].Message.Audio.Transcript
			} else {
				for _, choice := range after.ChatCompletionRes.Choices {
					billingData.Completion += gconv.String(choice.Message.Content)
					billingData.Completion += gconv.String(choice.Message.ToolCalls)
				}
			}
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if !after.IsSmartMatch {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				// 记录花费
				if err := RecordSpend(ctx, after.Spend, mak); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	completionsRes := &model.CompletionsRes{
		Completion:   after.Completion,
		Error:        after.Error,
		ConnTime:     after.ConnTime,
		Duration:     after.Duration,
		TotalTime:    after.TotalTime,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if completionsRes.Completion == "" && after.RetryInfo == nil && len(after.ChatCompletionRes.Choices) > 0 && after.ChatCompletionRes.Choices[0].Message != nil {
		if mak.RealModel.Type == 102 && after.ChatCompletionRes.Choices[0].Message.Audio != nil {
			completionsRes.Completion = after.ChatCompletionRes.Choices[0].Message.Audio.Transcript
		} else {
			if len(after.ChatCompletionRes.Choices) > 1 {
				for i, choice := range after.ChatCompletionRes.Choices {

					if choice.Message.Content != nil {
						completionsRes.Completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, gconv.String(choice.Message.Content))
					}

					if choice.Message.ToolCalls != nil {
						completionsRes.Completion += fmt.Sprintf("index: %d\ntool_calls: %s\n\n", i, gconv.String(choice.Message.ToolCalls))
					}
				}
			} else {

				if after.ChatCompletionRes.Choices[0].Message.ReasoningContent != nil {
					completionsRes.Completion = gconv.String(after.ChatCompletionRes.Choices[0].Message.ReasoningContent)
				}

				completionsRes.Completion += gconv.String(after.ChatCompletionRes.Choices[0].Message.Content)

				if after.ChatCompletionRes.Choices[0].Message.ToolCalls != nil {
					completionsRes.Completion += fmt.Sprintf("\ntool_calls: %s", gconv.String(after.ChatCompletionRes.Choices[0].Message.ToolCalls))
				}
			}
		}
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().Text(ctx, model.LogText{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		CompletionsReq:     &after.ChatCompletionReq,
		CompletionsRes:     completionsRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
		IsSmartMatch:       after.IsSmartMatch,
	})
}

func imageHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			ImageGenerationRequest: after.ImageGenerationRequest,
			Usage:                  after.Usage,
			IsAborted:              IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	imageRes := &model.ImageRes{
		Data:         after.ImageResponse.Data,
		TotalTime:    after.ImageResponse.TotalTime,
		Error:        after.Error,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().Image(ctx, model.LogImage{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		ImageReq:           &after.ImageGenerationRequest,
		ImageRes:           imageRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}

func audioHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			AudioInput:  after.AudioInput,
			AudioMinute: after.AudioMinute,
			IsAborted:   IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	audioReq := &model.AudioReq{
		Input: after.AudioInput,
	}

	audioRes := &model.AudioRes{
		Text:         after.AudioText,
		Minute:       after.AudioMinute,
		Error:        after.Error,
		TotalTime:    after.TotalTime,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().Audio(ctx, model.LogAudio{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		AudioReq:           audioReq,
		AudioRes:           audioRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}

func videoHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			Seconds:   after.Seconds,
			Size:      after.Size,
			IsAborted: IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}

		if after.Action == consts.ACTION_CREATE || after.Action == consts.ACTION_REMIX {

			taskVideo := do.TaskVideo{
				TraceId: gtrace.GetTraceID(ctx),
				UserId:  service.Session().GetUserId(ctx),
				AppId:   service.Session().GetAppId(ctx),
				Model:   mak.ReqModel.Name,
				VideoId: after.VideoId,
				Prompt:  after.Prompt,
				Status:  "queued",
				Rid:     service.Session().GetRid(ctx),
			}

			if after.Spend.VideoGeneration != nil {
				taskVideo.Seconds = after.Spend.VideoGeneration.Seconds
				if after.Spend.VideoGeneration.Pricing != nil {
					taskVideo.Width = after.Spend.VideoGeneration.Pricing.Width
					taskVideo.Height = after.Spend.VideoGeneration.Pricing.Height
				}
			}

			if _, err := dao.TaskVideo.Insert(ctx, taskVideo); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	videoReq := &model.VideoReq{
		Action:      after.Action,
		RequestData: after.RequestData,
	}

	videoRes := &model.VideoRes{
		VideoId:      after.VideoId,
		ResponseData: after.ResponseData,
		Error:        after.Error,
		TotalTime:    after.TotalTime,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().Video(ctx, model.LogVideo{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		VideoReq:           videoReq,
		VideoRes:           videoRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}

func fileHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			Usage:     after.Usage,
			IsAborted: IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}

		if after.Action == consts.ACTION_UPLOAD {

			taskFile := do.TaskFile{
				TraceId:   gtrace.GetTraceID(ctx),
				UserId:    service.Session().GetUserId(ctx),
				AppId:     service.Session().GetAppId(ctx),
				Model:     mak.ReqModel.Name,
				Purpose:   after.FileRes.Purpose,
				FileId:    after.FileId,
				FileName:  after.FileRes.Filename,
				Bytes:     after.FileRes.Bytes,
				ExpiresAt: after.FileRes.ExpiresAt,
				Status:    after.FileRes.Status,
				Rid:       service.Session().GetRid(ctx),
			}

			if _, err := dao.TaskFile.Insert(ctx, taskFile); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	fileReq := &model.FileReq{
		Action:      after.Action,
		RequestData: after.RequestData,
	}

	fileRes := &model.FileRes{
		FileId:       after.FileId,
		ResponseData: after.ResponseData,
		Error:        after.Error,
		TotalTime:    after.TotalTime,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().File(ctx, model.LogFile{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		FileReq:            fileReq,
		FileRes:            fileRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}

func batchHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			Usage:     after.Usage,
			IsAborted: IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}

		if after.Action == consts.ACTION_CREATE {

			taskBatch := do.TaskBatch{
				TraceId:      gtrace.GetTraceID(ctx),
				UserId:       service.Session().GetUserId(ctx),
				AppId:        service.Session().GetAppId(ctx),
				Model:        mak.ReqModel.Name,
				BatchId:      after.BatchId,
				InputFileId:  after.FileId,
				Status:       "validating",
				ResponseData: after.ResponseData,
				Rid:          service.Session().GetRid(ctx),
			}

			if _, err := dao.TaskBatch.Insert(ctx, taskBatch); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	batchReq := &model.BatchReq{
		Action:      after.Action,
		RequestData: after.RequestData,
	}

	batchRes := &model.BatchRes{
		BatchId:      after.BatchId,
		ResponseData: after.ResponseData,
		Error:        after.Error,
		TotalTime:    after.TotalTime,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().Batch(ctx, model.LogBatch{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		BatchReq:           batchReq,
		BatchRes:           batchRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}

func midjourneyHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			Path:      after.MidjourneyPath,
			IsAborted: IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	midjourneyResponse := model.MidjourneyResponse{
		ReqUrl:             after.MidjourneyReqUrl,
		TaskId:             after.MidjourneyTaskId,
		Prompt:             after.MidjourneyPrompt,
		MidjourneyResponse: after.MidjourneyResponse,
		TotalTime:          after.TotalTime,
		Error:              after.Error,
		InternalTime:       after.InternalTime,
		EnterTime:          after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().Midjourney(ctx, model.LogMidjourney{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		Response:           midjourneyResponse,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}

func generalHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		if after.ServiceTier == "" {
			after.ServiceTier = after.ChatCompletionRes.ServiceTier
			if after.ServiceTier == "" {
				after.ServiceTier = after.ChatCompletionReq.ServiceTier
			}
		}

		billingData := &mcommon.BillingData{
			ChatCompletionRequest:  after.ChatCompletionReq,
			EmbeddingRequest:       after.EmbeddingReq,
			ModerationRequest:      after.ModerationReq,
			Completion:             after.Completion,
			ServiceTier:            after.ServiceTier,
			ImageGenerationRequest: after.ImageGenerationRequest,
			AudioInput:             after.AudioInput,
			AudioMinute:            after.AudioMinute,
			Seconds:                after.Seconds,
			Size:                   after.Size,
			Usage:                  after.Usage,
			IsAborted:              IsAborted(after.Error),
		}

		// 计算花费
		after.Spend = Billing(ctx, mak, billingData)

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := RecordSpend(ctx, after.Spend, mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	generalReq := &model.GeneralReq{
		RequestData: after.RequestData,
		Stream:      after.ChatCompletionReq.Stream,
	}

	generalRes := &model.GeneralRes{
		ResponseData: after.ResponseData,
		Completion:   after.Completion,
		Error:        after.Error,
		ConnTime:     after.ConnTime,
		Duration:     after.Duration,
		TotalTime:    after.TotalTime,
		InternalTime: after.InternalTime,
		EnterTime:    after.EnterTime,
	}

	if after.Spend.GroupId == "" && mak.Group != nil {
		after.Spend.GroupId = mak.Group.Id
		after.Spend.GroupName = mak.Group.Name
		after.Spend.GroupDiscount = mak.Group.Discount
	}

	service.Log().General(ctx, model.LogGeneral{
		ReqModel:           mak.ReqModel,
		RealModel:          mak.RealModel,
		ModelAgent:         mak.ModelAgent,
		FallbackModelAgent: mak.FallbackModelAgent,
		FallbackModel:      mak.FallbackModel,
		Key:                mak.Key,
		GeneralReq:         generalReq,
		GeneralRes:         generalRes,
		RetryInfo:          after.RetryInfo,
		Spend:              after.Spend,
	})
}
