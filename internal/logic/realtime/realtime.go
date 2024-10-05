package audio

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gorilla/websocket"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type sRealtime struct {
	upgrader websocket.Upgrader
}

func init() {
	service.RegisterRealtime(New())
}

func New() service.IRealtime {
	return &sRealtime{}
}

// Realtime
func (s *sRealtime) Realtime(r *ghttp.Request) error {

	ctx := r.Context()
	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sRealtime Realtime time: %d", gtime.TimestampMilli()-now)
	}()

	conn, err := s.upgrader.Upgrade(r.Response.Writer, r.Request, nil)
	if conn != nil {
		defer func() {
			if err := conn.Close(); err != nil {
				logger.Error(ctx, err)
			}
		}()
	}

	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	for {

		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			logger.Error(ctx, err)
			return err
		}

		if err = conn.WriteMessage(msgType, msg); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}
}
