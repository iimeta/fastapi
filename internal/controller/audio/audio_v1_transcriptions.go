package audio

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/api/audio/v1"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/tcolgate/mp3"
	"os"
	"time"
)

func (c *ControllerV1) Transcriptions(ctx context.Context, req *v1.TranscriptionsReq) (res *v1.TranscriptionsRes, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "Controller Transcriptions time: %d", gtime.TimestampMilli()-now)
	}()

	fileName, err := req.File.Save("./resource/audio/", true)
	if err != nil {
		return nil, err
	}

	req.AudioRequest.FilePath = "./resource/audio/" + fileName

	response, err := service.Audio().Transcriptions(ctx, req.AudioRequest, nil)
	if err != nil {
		return nil, err
	}

	// 打开 MP3 文件
	file, err := os.Open(req.AudioRequest.FilePath)
	if err != nil {
		return res, err
	}
	defer file.Close()

	// 创建解码器
	d := mp3.NewDecoder(file)

	// 计算时长
	var totalDuration time.Duration
	skipped := 0
	for {
		// 解码一帧
		frame := mp3.Frame{}
		if err := d.Decode(&frame, &skipped); err != nil {
			// 到达文件结尾
			break
		}

		// 累加帧时长
		totalDuration += frame.Duration()
	}
	durationSec := totalDuration.Seconds()

	fmt.Printf("duration of %f seconds\n", durationSec)

	if req.AudioRequest.Format == "" || req.AudioRequest.Format == "json" || req.AudioRequest.Format == "verbose_json" {
		g.RequestFromCtx(ctx).Response.WriteJson(response)
	} else {
		g.RequestFromCtx(ctx).Response.Write(response.Text)
	}

	return
}
