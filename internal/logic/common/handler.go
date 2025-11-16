package common

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

// 前置处理器
func BeforeHandler(ctx context.Context, before *mcommon.BeforeHandler) {
}

// 后置处理器
func AfterHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {
	switch mak.ReqModel.Type {
	case 1, 100, 101, 102, 7, 103:
		chatHandler(ctx, mak, after)
	case 2, 3, 4:
		if mak.ReqModel.Model == "midjourney" || gstr.HasPrefix(mak.ReqModel.Model, "midjourney") {
			midjourneyHandler(ctx, mak, after)
		} else {
			imageHandler(ctx, mak, after)
		}
	case 5, 6:
		audioHandler(ctx, mak, after)
	default:
		chatHandler(ctx, mak, after)
	}
}

func chatHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

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
	})
}

func imageHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			ImageGenerationRequest: after.ImageGenerationRequest,
			Usage:                  after.Usage,
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
		Created:      after.ImageResponse.Created,
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

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {
		audioRes.TotalTokens = after.Spend.TotalSpendTokens
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

func midjourneyHandler(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			Path: after.MidjourneyPath,
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
