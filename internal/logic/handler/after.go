package handler

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type ChatHandler struct {
	Params             smodel.ChatCompletionRequest
	FallbackModelAgent *model.ModelAgent
	FallbackModel      *model.Model
	Mak                *common.MAK
	RetryInfo          *mcommon.Retry
	Response           smodel.ChatCompletionResponse
	Spend              mcommon.Spend
	Error              error
	IsSmartMatch       bool
}

func chat(ctx context.Context, c ChatHandler) {

	enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
	internalTime := gtime.TimestampMilli() - enterTime - c.Response.TotalTime

	if c.RetryInfo == nil && (c.Error == nil || common.IsAborted(c.Error)) {

		billingData := &mcommon.BillingData{
			ChatCompletionRequest: c.Params,
			Usage:                 c.Response.Usage,
		}

		if len(c.Response.Choices) > 0 && c.Response.Choices[0].Message != nil {
			if c.Mak.RealModel.Type == 102 && c.Response.Choices[0].Message.Audio != nil {
				billingData.Completion = c.Response.Choices[0].Message.Audio.Transcript
			} else {
				for _, choice := range c.Response.Choices {
					billingData.Completion += gconv.String(choice.Message.Content)
					billingData.Completion += gconv.String(choice.Message.ToolCalls)
				}
			}
		}

		// 计算花费
		c.Spend = common.Billing(ctx, c.Mak, billingData)
		c.Response.Usage = billingData.Usage

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			// 记录花费
			if err := common.RecordSpend(ctx, c.Spend, c.Mak); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	completionsRes := &model.CompletionsRes{
		Error:        c.Error,
		ConnTime:     c.Response.ConnTime,
		Duration:     c.Response.Duration,
		TotalTime:    c.Response.TotalTime,
		InternalTime: internalTime,
		EnterTime:    enterTime,
	}

	if c.RetryInfo == nil && len(c.Response.Choices) > 0 && c.Response.Choices[0].Message != nil {
		if c.Mak.RealModel.Type == 102 && c.Response.Choices[0].Message.Audio != nil {
			completionsRes.Completion = c.Response.Choices[0].Message.Audio.Transcript
		} else {
			if len(c.Response.Choices) > 1 {
				for i, choice := range c.Response.Choices {

					if choice.Message.Content != nil {
						completionsRes.Completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, gconv.String(choice.Message.Content))
					}

					if choice.Message.ToolCalls != nil {
						completionsRes.Completion += fmt.Sprintf("index: %d\ntool_calls: %s\n\n", i, gconv.String(choice.Message.ToolCalls))
					}
				}
			} else {

				if c.Response.Choices[0].Message.ReasoningContent != nil {
					completionsRes.Completion = gconv.String(c.Response.Choices[0].Message.ReasoningContent)
				}

				completionsRes.Completion += gconv.String(c.Response.Choices[0].Message.Content)

				if c.Response.Choices[0].Message.ToolCalls != nil {
					completionsRes.Completion += fmt.Sprintf("\ntool_calls: %s", gconv.String(c.Response.Choices[0].Message.ToolCalls))
				}
			}
		}
	}

	if c.Spend.GroupId == "" && c.Mak.Group != nil {
		c.Spend.GroupId = c.Mak.Group.Id
		c.Spend.GroupName = c.Mak.Group.Name
		c.Spend.GroupDiscount = c.Mak.Group.Discount
	}

	service.Chat().SaveLog(ctx, model.ChatLog{
		ReqModel:           c.Mak.ReqModel,
		RealModel:          c.Mak.RealModel,
		ModelAgent:         c.Mak.ModelAgent,
		FallbackModelAgent: c.FallbackModelAgent,
		FallbackModel:      c.FallbackModel,
		Key:                c.Mak.Key,
		CompletionsReq:     &c.Params,
		CompletionsRes:     completionsRes,
		RetryInfo:          c.RetryInfo,
		Spend:              c.Spend,
	})
}
