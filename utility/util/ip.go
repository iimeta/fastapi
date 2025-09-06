package util

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gipv4"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/utility/logger"
)

var localIp = "127.0.0.1"

func init() {

	ctx := gctx.New()

	if len(config.Cfg.Local.PublicIp) > 0 {

		for _, url := range config.Cfg.Local.PublicIp {

			response, _ := g.Client().Timeout(30*time.Second).Get(ctx, url)
			if response != nil {

				result := gstr.Trim(response.ReadAllString())
				if result != "" && gipv4.Validate(result) {
					localIp = result
					_ = response.Close()
					break
				}

				_ = response.Close()
			}
		}

	} else {
		if ip, err := gipv4.GetIntranetIp(); err != nil {
			logger.Error(ctx, err)
		} else {
			localIp = ip
		}
	}

	logger.Infof(ctx, "LOCAL_IP: %s", localIp)
}

func GetLocalIp() string {
	return localIp
}
