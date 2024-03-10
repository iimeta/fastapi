package model

type MidjourneyProxyImagineReq struct {
	Prompt string `json:"prompt"`
	Base64 string `json:"base64"`
}
type MidjourneyProxyImagineRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
}

type MidjourneyProxyChangeReq struct {
	Action string `json:"action"`
	Index  int    `json:"index"`
	TaskId string `json:"taskId"`
}
type MidjourneyProxyChangeRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
}

type MidjourneyProxyDescribeReq struct {
	Base64 string `json:"base64"`
}
type MidjourneyProxyDescribeRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
}

type MidjourneyProxyBlendReq struct {
	Base64Array []string `json:"base64Array"`
}
type MidjourneyProxyBlendRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
}

type MidjourneyProxyFetchRes struct {
	Action      string      `json:"action"`
	Id          string      `json:"id"`
	Prompt      string      `json:"prompt"`
	PromptEn    string      `json:"promptEn"`
	Description string      `json:"description"`
	State       interface{} `json:"state"`
	SubmitTime  int64       `json:"submitTime"`
	StartTime   int64       `json:"startTime"`
	FinishTime  int64       `json:"finishTime"`
	ImageUrl    string      `json:"imageUrl"`
	Status      string      `json:"status"`
	Progress    string      `json:"progress"`
	FailReason  string      `json:"failReason"`
}
