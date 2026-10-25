package common

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	sutil "github.com/iimeta/fastapi-sdk/v2/util"
	"github.com/iimeta/fastapi/v2/internal/consts"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
)

// 单次上游尝试的日志快照. 递归重试会覆盖 named return 和 session 中的 RequestIds / 耗时,
// 必须在发起下一次尝试前把本次的请求ID和耗时定住.
type AttemptLog struct {
	RequestIds   map[string]string
	ConnTime     int64
	Duration     int64
	TotalTime    int64
	EnterTime    int64
	InternalTime int64
}

func CaptureAttempt(ctx context.Context, now, totalTime int64, extra ...int64) *AttemptLog {

	var connTime, duration int64

	if len(extra) > 0 {
		connTime = extra[0]
	}

	if len(extra) > 1 {
		duration = extra[1]
	}

	enterTime, internalTime, totalTime := AttemptTimes(ctx, now, totalTime)

	return &AttemptLog{
		RequestIds:   cloneRequestIds(sutil.UpstreamRequestIds(ctx)),
		ConnTime:     connTime,
		Duration:     duration,
		TotalTime:    totalTime,
		EnterTime:    enterTime,
		InternalTime: internalTime,
	}
}

func AttemptTimes(ctx context.Context, now, totalTime int64) (int64, int64, int64) {

	enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
	start := now
	first := true

	if r := g.RequestFromCtx(ctx); r != nil {
		first = r.GetCtxVar(consts.ATTEMPT_INDEX_KEY).Int() <= 1
		if !first {
			if attemptStart := r.GetCtxVar(consts.ATTEMPT_START_KEY).Int64(); attemptStart > 0 {
				start = attemptStart
			}
		} else {
			start = enterTime
		}
	}

	elapsed := gtime.TimestampMilli() - start

	if totalTime <= 0 {

		totalTime = elapsed

		if first {
			receiveTime := g.RequestFromCtx(ctx).GetCtxVar(consts.RECEIVE_TIME_KEY).Int64()
			if totalTime > receiveTime {
				totalTime -= receiveTime
			}
		}

		if totalTime < 0 {
			totalTime = 0
		}

		return enterTime, 0, totalTime
	}

	internalTime := elapsed - totalTime

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

	return enterTime, internalTime, totalTime
}

func FinishAttempt(ctx context.Context, now int64, attempt *AttemptLog, after *mcommon.AfterHandler) {

	if after == nil {
		return
	}

	if attempt != nil {
		after.RequestIds = cloneRequestIds(attempt.RequestIds)
		after.EnterTime = attempt.EnterTime
		after.InternalTime = attempt.InternalTime
		after.TotalTime = attempt.TotalTime
		after.ConnTime = attempt.ConnTime
		after.Duration = attempt.Duration
		return
	}

	enterTime, internalTime, totalTime := AttemptTimes(ctx, now, after.TotalTime)

	after.EnterTime = enterTime
	after.InternalTime = internalTime

	if after.TotalTime == 0 {
		after.TotalTime = totalTime
	}

	if after.RequestIds == nil {
		after.RequestIds = sutil.UpstreamRequestIds(ctx)
	}
}

func cloneRequestIds(ids map[string]string) map[string]string {

	if ids == nil {
		return map[string]string{}
	}

	out := make(map[string]string, len(ids))
	for key, value := range ids {
		out[key] = value
	}

	return out
}
