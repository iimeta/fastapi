package model

type GetGcpTokenRes struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int64  `json:"expires_in"`
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"`
	IdToken          string `json:"id_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}
