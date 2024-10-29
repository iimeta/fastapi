package embedding

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"time"
)

type sFile struct{}

func init() {
	service.RegisterFile(New())
}

func New() service.IFile {
	return &sFile{}
}

// Files
func (s *sFile) Files(ctx context.Context, params model.FileFilesReq) ([]byte, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sFile Files time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model: params.Model,
		}
	)

	if err := mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	bytes, err := uploadFile(ctx, params.FilePath, fmt.Sprintf("https://generativelanguage.googleapis.com/upload/v1beta/files?key=%s", mak.RealKey))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	logger.Infof(ctx, "Files response: %s", string(bytes))

	return bytes, nil
}

func uploadFile(ctx context.Context, filename string, targetUrl string) ([]byte, error) {

	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			logger.Error(ctx, err)
		}
		// 删除文件
		if err := os.Remove(filename); err != nil {
			logger.Error(ctx, err)
		}
	}()

	// 创建一个缓冲区
	var buffer bytes.Buffer
	// 创建一个multipart/form-data的Writer
	writer := multipart.NewWriter(&buffer)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "file", gfile.Basename(filename)))

	// 添加文件字段
	formFile, err := writer.CreatePart(h)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	// 从文件中读取内容并写入到formFile中
	if _, err := io.Copy(formFile, file); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	// 关闭multipart writer
	if err = writer.Close(); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", targetUrl, &buffer)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := http.Client{Timeout: time.Second * 60}

	if config.Cfg.Http.ProxyUrl != "" {

		proxyUrl, err := url.Parse(config.Cfg.Http.ProxyUrl)
		if err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		}
	}

	// 发送请求
	resp, err := client.Do(req)
	if resp != nil {
		defer func() {
			if err := resp.Body.Close(); err != nil {
				logger.Error(ctx, err)
			}
		}()
	}

	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return body, nil
}
