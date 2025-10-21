package common

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func Before(ctx context.Context, before *mcommon.BeforeHandler) {
}

func After(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {
	chat(ctx, mak, after)
}

func chat(ctx context.Context, mak *MAK, after *mcommon.AfterHandler) {

	if after.RetryInfo == nil && (after.Error == nil || IsAborted(after.Error)) {

		billingData := &mcommon.BillingData{
			ChatCompletionRequest: after.ChatCompletionReq,
			EmbeddingRequest:      after.EmbeddingReq,
			ModerationRequest:     after.ModerationReq,
			Completion:            after.Completion,
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
		after.Usage = billingData.Usage

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

	service.Logger().Chat(ctx, model.ChatLog{
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
