package common

import (
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
)

func GetImageQuota(model *model.Model, size string) (imageQuota mcommon.ImageQuota) {

	var (
		width  int
		height int
	)

	if size != "" {

		widthHeight := gstr.Split(size, `×`)

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `x`)
		}

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `X`)
		}

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `*`)
		}

		if len(widthHeight) != 2 {
			widthHeight = gstr.Split(size, `:`)
		}

		if len(widthHeight) == 2 {
			width = gconv.Int(widthHeight[0])
			height = gconv.Int(widthHeight[1])
		}
	}

	for _, quota := range model.ImageQuotas {

		if quota.Width == width && quota.Height == height {
			return quota
		}

		if quota.IsDefault {
			imageQuota = quota
		}
	}

	return imageQuota
}

func GetMidjourneyQuota(model *model.Model, request *ghttp.Request, path string) (mcommon.MidjourneyQuota, error) {

	for _, quota := range model.MidjourneyQuotas {
		if quota.Path == path {
			return quota, nil
		}
	}

	return mcommon.MidjourneyQuota{}, errors.ERR_PATH_NOT_FOUND
}
