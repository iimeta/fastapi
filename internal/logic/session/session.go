package session

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type sSession struct{}

func init() {
	service.RegisterSession(New())
}

func New() service.ISession {
	return &sSession{}
}

// 保存密钥
func (s *sSession) SaveKey(ctx context.Context, key string) error {

	r := g.RequestFromCtx(ctx)

	r.SetCtxVar("key", key)

	return nil
}

// 获取密钥
func (s *sSession) GetKey(ctx context.Context) string {

	key := ctx.Value("key")
	if key == nil {
		logger.Error(ctx, "key is nil")
		return ""
	}

	return key.(string)
}
