package model

import "github.com/iimeta/fastapi/v2/internal/model/common"

type ModelAgent struct {
	Id                    string                        `json:"id,omitempty"`
	ProviderId            string                        `json:"provider_id,omitempty"`
	Name                  string                        `json:"name,omitempty"`
	BaseUrl               string                        `json:"base_url,omitempty"`
	Path                  string                        `json:"path,omitempty"`
	Weight                int                           `json:"weight,omitempty"`
	CurrentWeight         int                           `json:"current_weight,omitempty"`
	BillingMethods        []int                         `json:"billing_methods,omitempty"`
	Models                []string                      `json:"models,omitempty"`
	IsEnableModelReplace  bool                          `json:"is_enable_model_replace,omitempty"`
	ReplaceModels         []string                      `json:"replace_models,omitempty"`
	TargetModels          []string                      `json:"target_models,omitempty"`
	IsEnableHealthCheck   bool                          `json:"is_enable_health_check,omitempty"`
	IsEnableSessionKeep   bool                          `json:"is_enable_session_keep,omitempty"`
	SessionKeepConfig     *common.ModelAgentSessionKeep `json:"session_keep_config,omitempty"`
	IsRemoveAbnormalModel bool                          `json:"is_remove_abnormal_model,omitempty"`
	AbnormalModels        []string                      `json:"abnormal_models,omitempty"`
	IsNeverDisable        bool                          `json:"is_never_disable,omitempty"`
	LbStrategy            int                           `json:"lb_strategy,omitempty"`
	Remark                string                        `json:"remark,omitempty"`
	Status                int                           `json:"status,omitempty"`
	IsAutoDisabled        bool                          `json:"is_auto_disabled,omitempty"`
	AutoDisabledReason    string                        `json:"auto_disabled_reason,omitempty"`
	Creator               string                        `json:"creator,omitempty"`
	Updater               string                        `json:"updater,omitempty"`
	CreatedAt             string                        `json:"created_at,omitempty"`
	UpdatedAt             string                        `json:"updated_at,omitempty"`
}
