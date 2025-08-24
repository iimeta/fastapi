package common

import "time"

type Core struct {
	SecretKeyPrefix string `bson:"secret_key_prefix" json:"secret_key_prefix"` // 密钥前缀
	ErrorPrefix     string `bson:"error_prefix"      json:"error_prefix"`      // 错误码前缀
	ChannelPrefix   string `bson:"channel_prefix"    json:"channel_prefix"`    // 通道前缀
}

type Http struct {
	Timeout  time.Duration `bson:"timeout"   json:"timeout"`   // 超时时间, 单位: 秒
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
	LockMinutes time.Duration `bson:"lock_minutes" json:"lock_minutes"` // 锁定时长, 单位: 分钟
}

type Base struct {
	ErrRetry                int           `bson:"err_retry"                   json:"err_retry"`                   // 错误重试次数
	ModelKeyErrDisable      int64         `bson:"model_key_err_disable"       json:"model_key_err_disable"`       // 模型密钥禁用次数
	ModelAgentErrDisable    int64         `bson:"model_agent_err_disable"     json:"model_agent_err_disable"`     // 模型代理禁用次数
	ModelAgentKeyErrDisable int64         `bson:"model_agent_key_err_disable" json:"model_agent_key_err_disable"` // 模型代理密钥禁用次数
	ShortTimeout            time.Duration `bson:"short_timeout"               json:"short_timeout"`               // 短连接超时时间, 单位: 秒
	LongTimeout             time.Duration `bson:"long_timeout"                json:"long_timeout"`                // 长连接超时时间, 单位: 秒
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
	Open         bool          `bson:"open"          json:"open"`          // 开关
	ChatRecords  []string      `bson:"chat_records"  json:"chat_records"`  // 聊天日志记录内容
	ChatReserve  time.Duration `bson:"chat_reserve"  json:"chat_reserve"`  // 聊天日志保留天数
	ImageReserve time.Duration `bson:"image_reserve" json:"image_reserve"` // 绘图日志保留天数
	AudioReserve time.Duration `bson:"audio_reserve" json:"audio_reserve"` // 音频日志保留天数
	Cron         string        `bson:"cron"          json:"cron"`          // CRON表达式
}

type UserLoginRegister struct {
	AccountLogin  bool `bson:"account_login"  json:"account_login"`  // 账密登录
	EmailLogin    bool `bson:"email_login"    json:"email_login"`    // 邮箱登录
	EmailRegister bool `bson:"email_register" json:"email_register"` // 邮箱注册
	EmailRetrieve bool `bson:"email_retrieve" json:"email_retrieve"` // 找回密码
	SessionExpire int  `bson:"session_expire" json:"session_expire"` // 会话过期, 单位: 秒
}

type UserShieldError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type AdminLogin struct {
	AccountLogin  bool `bson:"account_login"  json:"account_login"`  // 账密登录
	EmailLogin    bool `bson:"email_login"    json:"email_login"`    // 邮箱登录
	EmailRetrieve bool `bson:"email_retrieve" json:"email_retrieve"` // 找回密码
	SessionExpire int  `bson:"session_expire" json:"session_expire"` // 会话过期, 单位: 秒
}

type AutoDisabledError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type AutoEnableError struct {
	Open         bool          `bson:"open"          json:"open"`          // 开关
	EnableErrors []EnableError `bson:"enable_errors" json:"enable_errors"` // 启用错误
}

type EnableError struct {
	Cron       string        `bson:"cron"        json:"cron"`        // CRON表达式
	EnableTime time.Duration `bson:"enable_time" json:"enable_time"` // 启用时间, 单位: 秒
	Error      string        `bson:"error"       json:"error"`       // 错误
}

type NotRetryError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type NotShieldError struct {
	Open   bool     `bson:"open"   json:"open"`   // 开关
	Errors []string `bson:"errors" json:"errors"` // 错误
}

type QuotaWarning struct {
	Open             bool          `bson:"open"              json:"open"`              // 额度预警开关
	Threshold        int           `bson:"threshold"         json:"threshold"`         // 额度预警阈值, 单位: $
	ExhaustionNotice bool          `bson:"exhaustion_notice" json:"exhaustion_notice"` // 额度耗尽通知开关
	ExpireWarning    bool          `bson:"expire_warning"    json:"expire_warning"`    // 额度过期预警开关
	ExpireThreshold  time.Duration `bson:"expire_threshold"  json:"expire_threshold"`  // 额度过期预警阈值, 单位: 天
	ExpireNotice     bool          `bson:"expire_notice"     json:"expire_notice"`     // 额度过期通知开关
}

type Debug struct {
	Open bool `bson:"open" json:"open"` // 开关
}
