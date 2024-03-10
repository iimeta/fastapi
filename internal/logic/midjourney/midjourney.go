package midjourney

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sMidjourney struct{}

func init() {
	service.RegisterMidjourney(New())
}

func New() service.IMidjourney {
	return &sMidjourney{}
}

func (s *sMidjourney) Imagine(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyImagineReq *model.MidjourneyProxyImagineReq) (*model.MidjourneyProxyImagineRes, error) {

	header := make(map[string]string)
	header[midjourneyProxy.ApiSecretHeader] = midjourneyProxy.ApiSecret

	midjourneyProxyImagineRes := new(model.MidjourneyProxyImagineRes)
	if err := util.HttpPostJson(ctx, midjourneyProxy.ImagineUrl, header, midjourneyProxyImagineReq, &midjourneyProxyImagineRes); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return midjourneyProxyImagineRes, nil
}

func (s *sMidjourney) Change(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyChangeReq *model.MidjourneyProxyChangeReq) (*model.MidjourneyProxyChangeRes, error) {

	header := make(map[string]string)
	header[midjourneyProxy.ApiSecretHeader] = midjourneyProxy.ApiSecret

	midjourneyProxyChangeRes := new(model.MidjourneyProxyChangeRes)
	if err := util.HttpPostJson(ctx, midjourneyProxy.ChangeUrl, header, midjourneyProxyChangeReq, &midjourneyProxyChangeRes); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return midjourneyProxyChangeRes, nil
}

func (s *sMidjourney) Describe(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyDescribeReq *model.MidjourneyProxyDescribeReq) (*model.MidjourneyProxyDescribeRes, error) {

	header := make(map[string]string)
	header[midjourneyProxy.ApiSecretHeader] = midjourneyProxy.ApiSecret

	midjourneyProxyDescribeRes := new(model.MidjourneyProxyDescribeRes)
	if err := util.HttpPostJson(ctx, midjourneyProxy.DescribeUrl, header, midjourneyProxyDescribeReq, &midjourneyProxyDescribeRes); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return midjourneyProxyDescribeRes, nil
}

func (s *sMidjourney) Blend(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, midjourneyProxyBlendReq *model.MidjourneyProxyBlendReq) (*model.MidjourneyProxyBlendRes, error) {

	header := make(map[string]string)
	header[midjourneyProxy.ApiSecretHeader] = midjourneyProxy.ApiSecret

	midjourneyProxyBlendRes := new(model.MidjourneyProxyBlendRes)
	if err := util.HttpPostJson(ctx, midjourneyProxy.BlendUrl, header, midjourneyProxyBlendReq, &midjourneyProxyBlendRes); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return midjourneyProxyBlendRes, nil
}

func (s *sMidjourney) Fetch(ctx context.Context, midjourneyProxy *model.MidjourneyProxy, taskId string) (*model.MidjourneyProxyFetchRes, error) {

	header := make(map[string]string)
	header[midjourneyProxy.ApiSecretHeader] = midjourneyProxy.ApiSecret

	fetchUrl := gstr.Replace(midjourneyProxy.FetchUrl, "${task_id}", taskId, -1)

	midjourneyProxyFetchRes := new(model.MidjourneyProxyFetchRes)
	if err := util.HttpGet(ctx, fetchUrl, header, nil, &midjourneyProxyFetchRes); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	logger.Infof(ctx, "midjourneyProxyFetchRes: %s", gjson.MustEncodeString(midjourneyProxyFetchRes))

	return midjourneyProxyFetchRes, nil
}
