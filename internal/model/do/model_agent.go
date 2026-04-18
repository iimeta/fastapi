package do

import (
	"github.com/gogf/gf/v2/util/gmeta"
	"github.com/iimeta/fastapi/v2/internal/model/common"
)

type ModelAgent struct {
	gmeta.Meta            `collection:"model_agent" bson:"-"`
	ProviderId            string                        `bson:"provider_id,omitempty"`
	Name                  string                        `bson:"name,omitempty"`
	BaseUrl               string                        `bson:"base_url,omitempty"`
	Path                  string                        `bson:"path,omitempty"`
	Weight                int                           `bson:"weight,omitempty"`
	BillingMethods        []int                         `bson:"billing_methods,omitempty"`
	Models                []string                      `bson:"models,omitempty"`
	IsEnableModelReplace  bool                          `bson:"is_enable_model_replace,omitempty"`
	ReplaceModels         []string                      `bson:"replace_models,omitempty"`
	TargetModels          []string                      `bson:"target_models,omitempty"`
	IsEnableHealthCheck   bool                          `bson:"is_enable_health_check,omitempty"`
	IsEnableSessionKeep   bool                          `bson:"is_enable_session_keep,omitempty"`
	SessionKeepConfig     *common.ModelAgentSessionKeep `bson:"session_keep_config,omitempty"`
	IsNeverDisable        bool                          `bson:"is_never_disable,omitempty"`
	LbStrategy            int                           `bson:"lb_strategy,omitempty"`
	Remark                string                        `bson:"remark,omitempty"`
	Status                int                           `bson:"status,omitempty"`
	IsAutoDisabled        bool                          `bson:"is_auto_disabled,omitempty"`
	AutoDisabledReason    string                        `bson:"auto_disabled_reason,omitempty"`
	Creator               string                        `bson:"creator,omitempty"`
	Updater               string                        `bson:"updater,omitempty"`
	CreatedAt             int64                         `bson:"created_at,omitempty"`
	UpdatedAt             int64                         `bson:"updated_at,omitempty"`
}
