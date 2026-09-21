package util

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/gipv4"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/utility/logger"
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

// 从请求上下文获取真实客户端 IP.
func GetClientIpFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	return GetClientIp(g.RequestFromCtx(ctx))
}

// 获取真实客户端 IP.
// nginx 与服务同机时 RemoteIp 为本机, 此时才信任 X-Forwarded-For / X-Real-IP.
// X-Forwarded-For 左侧 hop 由客户端可控(代理/内网/伪造), nginx $proxy_add_x_forwarded_for 追加在最右侧, 只取最后一个合法 IP.
func GetClientIp(r *ghttp.Request) string {
	if r == nil {
		return ""
	}
	return resolveClientIp(r.GetRemoteIp(), r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP"))
}

// 判断是否为本机地址(回环或本机网卡 IP).
func IsLocalIp(ipStr string) bool {

	ip := parseForwardedIP(ipStr)
	if ip == nil {
		return false
	}

	if ip.IsLoopback() {
		return true
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ipnet.IP.Equal(ip) {
			return true
		}
	}

	return false
}

func resolveClientIp(remoteIp, xForwardedFor, xRealIP string) string {

	remoteIp = formatIP(parseForwardedIP(remoteIp))

	// 直连(未经过本机反代)时不信任任何转发头, 避免伪造 X-Forwarded-For / X-Real-IP.
	if remoteIp == "" || !IsLocalIp(remoteIp) {
		return remoteIp
	}

	if ip := lastForwardedIP(xForwardedFor); ip != "" {
		return ip
	}

	if ip := formatIP(parseForwardedIP(xRealIP)); ip != "" {
		return ip
	}

	return remoteIp
}

func lastForwardedIP(xForwardedFor string) string {

	xForwardedFor = strings.TrimSpace(xForwardedFor)
	if xForwardedFor == "" || strings.EqualFold(xForwardedFor, "unknown") {
		return ""
	}

	parts := strings.Split(xForwardedFor, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if ip := formatIP(parseForwardedIP(parts[i])); ip != "" {
			return ip
		}
	}

	return ""
}

func parseForwardedIP(s string) net.IP {

	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "unknown") {
		return nil
	}

	if strings.HasPrefix(s, "[") {
		end := strings.LastIndex(s, "]")
		if end > 1 {
			s = s[1:end]
		}
	} else if ip := net.ParseIP(s); ip != nil {
		return ip
	} else if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}

	return net.ParseIP(s)
}

func formatIP(ip net.IP) string {

	if ip == nil {
		return ""
	}

	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}

	return ip.String()
}
