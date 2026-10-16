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
// RemoteIp 是 TCP 对端(末级反代/直连), 与反代是否同机无关.
// X-Forwarded-For 左侧 hop 由客户端可控(伪造如 9.9.9.9), 代理追加在右侧;
// 从右往左取第一个非本机、非 RemoteIp 的合法 IP, 避免取到伪造值或反代自身.
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

	if ip := clientIPFromForwarded(xForwardedFor, remoteIp); ip != "" {
		return ip
	}

	if ip := formatIP(parseForwardedIP(xRealIP)); ip != "" && !isForwardedHop(ip, remoteIp) {
		return ip
	}

	return remoteIp
}

func clientIPFromForwarded(xForwardedFor, remoteIp string) string {

	xForwardedFor = strings.TrimSpace(xForwardedFor)
	if xForwardedFor == "" || strings.EqualFold(xForwardedFor, "unknown") {
		return ""
	}

	parts := strings.Split(xForwardedFor, ",")
	for i := len(parts) - 1; i >= 0; i-- {

		ip := formatIP(parseForwardedIP(parts[i]))
		if ip == "" || isForwardedHop(ip, remoteIp) {
			continue
		}

		return ip
	}

	return ""
}

// 本机反代、回环、以及 TCP 对端自身, 都不是客户端.
func isForwardedHop(ip, remoteIp string) bool {

	if ip == "" {
		return true
	}

	if remoteIp != "" && ip == remoteIp {
		return true
	}

	parsed := parseForwardedIP(ip)
	if parsed == nil {
		return true
	}

	if parsed.IsLoopback() || parsed.IsUnspecified() || parsed.IsLinkLocalUnicast() {
		return true
	}

	return IsLocalIp(ip)
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
