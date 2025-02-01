package common

import "time"

type Core struct {
	SecretKeyPrefix string `bson:"secret_key_prefix" json:"secret_key_prefix"` // 密钥前缀
	ErrorPrefix     string `bson:"error_prefix"      json:"error_prefix"`      // 错误码前缀
	ChannelPrefix   string `bson:"channel_prefix"    json:"channel_prefix"`    // 通道前缀
}

type Http struct {
	Timeout  time.Duration `bson:"timeout"   json:"timeout"`   // 超时时间
	ProxyUrl string        `bson:"proxy_url" json:"proxy_url"` // 代理地址
}

type Email struct {
	Open     bool   `bson:"open"      json:"open"`      // 开关
	Host     string `bson:"host"      json:"host"`      // smtp.xxx.com
	Port     int    `bson:"port"      json:"port"`      // 端口号
	UserName string `bson:"user_name" json:"user_name"` // 登录账号
	Password string `bson:"password"  json:"password"`  // 登录密码
	FromName string `bson:"from_name" json:"from_name"` // 发送人名称
}

type Statistics struct {
	Open        bool          `bson:"open"         json:"open"`         // 开关
	Cron        string        `bson:"cron"         json:"cron"`         // CRON表达式
	Limit       int64         `bson:"limit"        json:"limit"`        // 查询条数
	LockMinutes time.Duration `bson:"lock_minutes" json:"lock_minutes"` // 锁定时长
}

type Base struct {
	ErrRetry                int   `bson:"err_retry"                   json:"err_retry"`                   // 错误重试次数
	ModelKeyErrDisable      int64 `bson:"model_key_err_disable"       json:"model_key_err_disable"`       // 模型密钥禁用次数
	ModelAgentErrDisable    int64 `bson:"model_agent_err_disable"     json:"model_agent_err_disable"`     // 模型代理禁用次数
	ModelAgentKeyErrDisable int64 `bson:"model_agent_key_err_disable" json:"model_agent_key_err_disable"` // 模型代理密钥禁用次数
}

type Midjourney struct {
	Open            bool   `bson:"open"              json:"open"` // 开关
	CdnUrl          string `bson:"cdn_url"           json:"cdn_url"`
	ApiBaseUrl      string `bson:"api_base_url"      json:"api_base_url"`
	ApiSecret       string `bson:"api_secret"        json:"api_secret"`
	ApiSecretHeader string `bson:"api_secret_header" json:"api_secret_header"`
	CdnOriginalUrl  string `bson:"cdn_original_url"  json:"cdn_original_url"`
}

type Log struct {
	Open    bool     `bson:"open"    json:"open"`    // 开关
	Records []string `bson:"records" json:"records"` // 日志记录
}

type UserShieldError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type AutoDisabledError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type NotRetryError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type NotShieldError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type Debug struct {
	Open bool `bson:"open" json:"open"` // 开关
}
