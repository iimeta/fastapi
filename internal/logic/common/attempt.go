package common

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	sutil "github.com/iimeta/fastapi-sdk/v2/util"
	"github.com/iimeta/fastapi/v2/internal/consts"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
)

// 单次上游尝试的日志快照. 递归重试会覆盖 named return 和 session 中的 RequestIds,
// 必须在发起下一次尝试前把本次的耗时和请求ID定住.
type AttemptLog struct {
	RequestIds   map[string]string
	TotalTime    int64
	EnterTime    int64
	InternalTime int64
}

func CaptureAttempt(ctx context.Context, now, totalTime int64) *AttemptLog {

	enterTime, internalTime := AttemptTimes(ctx, now, totalTime)

	return &AttemptLog{
		RequestIds:   sutil.UpstreamRequestIds(ctx),
		TotalTime:    totalTime,
		EnterTime:    enterTime,
		InternalTime: internalTime,
	}
}

func AttemptTimes(ctx context.Context, now, totalTime int64) (int64, int64) {

	enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
	start := now
	first := true

	if r := g.RequestFromCtx(ctx); r != nil {
		if attemptStart := r.GetCtxVar(consts.ATTEMPT_START_KEY).Int64(); attemptStart > 0 {
			start = attemptStart
		}
		first = r.GetCtxVar(consts.ATTEMPT_INDEX_KEY).Int() <= 1
	}

	internalTime := gtime.TimestampMilli() - start - totalTime

	if first {
		receiveTime := g.RequestFromCtx(ctx).GetCtxVar(consts.RECEIVE_TIME_KEY).Int64()
		if internalTime > receiveTime {
			internalTime -= receiveTime
		} else {
			internalTime = 0
		}
	}

	if internalTime < 0 {
		internalTime = 0
	}

	return enterTime, internalTime
}

func FinishAttempt(ctx context.Context, now int64, attempt *AttemptLog, after *mcommon.AfterHandler) {

	if after == nil {
		return
	}

	if attempt != nil {
		after.RequestIds = attempt.RequestIds
		after.EnterTime = attempt.EnterTime
		after.InternalTime = attempt.InternalTime
		if after.TotalTime == 0 {
			after.TotalTime = attempt.TotalTime
		}
		return
	}

	after.EnterTime, after.InternalTime = AttemptTimes(ctx, now, after.TotalTime)

	if after.RequestIds == nil {
		after.RequestIds = sutil.UpstreamRequestIds(ctx)
	}
}
