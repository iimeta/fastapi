package vip

import (
	"context"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
	"slices"
)

type sVip struct{}

func init() {
	service.RegisterVip(New())
}

func New() service.IVip {
	return &sVip{}
}

func (s *sVip) CheckUserVipPermissions(ctx context.Context, secretKey, model string) bool {

	user, err := service.User().GetUserById(ctx, service.Auth().GetUid(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return false
	}

	if user.SecretKey != secretKey {
		logger.Errorf(ctx, "invalid user secretKey: %s", secretKey)
		return false
	}

	vip, err := dao.Vip.FindOne(ctx, bson.M{"level": user.VipLevel})
	if err != nil {
		logger.Error(ctx, err)
		return false
	}

	isContains := slices.Contains(vip.Models, model)
	if !isContains {

		for _, m := range vip.Models {
			if gstr.HasPrefix(m, model) {
				isContains = true
				break
			}
		}

		if !isContains {
			logger.Errorf(ctx, "no model: %s permissions", model)
			return false
		}
	}

	return true
}
