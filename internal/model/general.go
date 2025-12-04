package model

type GeneralReq struct {
	RequestData map[string]any // 请求数据
	Stream      bool
}

type GeneralRes struct {
	ResponseData map[string]any // 响应数据
	Completion   string         // 补全(回答)
	Error        error          // 错误信息
	ConnTime     int64          // 连接时间
	Duration     int64          // 持续时间
	TotalTime    int64          // 总时间
	InternalTime int64          // 内耗时间
	EnterTime    int64          // 进入时间
}
