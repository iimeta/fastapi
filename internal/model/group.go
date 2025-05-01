package model

import "github.com/iimeta/fastapi/internal/model/common"

type Group struct {
	Id                 string                `json:"id,omitempty"`                    // ID
	Name               string                `json:"name,omitempty"`                  // 分组名称
	Discount           float64               `json:"discount,omitempty"`              // 分组折扣
	Models             []string              `json:"models,omitempty"`                // 模型权限
	IsEnableModelAgent bool                  `json:"is_enable_model_agent,omitempty"` // 是否启用模型代理
	LbStrategy         int                   `json:"lb_strategy,omitempty"`           // 代理负载均衡策略[1:轮询, 2:权重]
	ModelAgents        []string              `json:"model_agents,omitempty"`          // 模型代理
	IsDefault          bool                  `json:"is_default,omitempty"`            // 是否默认分组
	IsLimitQuota       bool                  `json:"is_limit_quota,omitempty"`        // 是否限制额度
	Quota              int                   `json:"quota,omitempty"`                 // 剩余额度
	UsedQuota          int                   `json:"used_quota,omitempty"`            // 已用额度
	IsEnableForward    bool                  `json:"is_enable_forward,omitempty"`     // 是否启用模型转发
	ForwardConfig      *common.ForwardConfig `json:"forward_config,omitempty"`        // 模型转发配置
	IsPublic           bool                  `json:"is_public,omitempty"`             // 是否公开
	Weight             int                   `json:"weight,omitempty"`                // 权重
	ExpiresAt          int64                 `json:"expires_at,omitempty"`            // 过期时间
	Remark             string                `json:"remark,omitempty"`                // 备注
	Status             int                   `json:"status,omitempty"`                // 状态[1:正常, 2:禁用, -1:删除]
	Creator            string                `json:"creator,omitempty"`               // 创建人
	Updater            string                `json:"updater,omitempty"`               // 更新人
	CreatedAt          string                `json:"created_at,omitempty"`            // 创建时间
	UpdatedAt          string                `json:"updated_at,omitempty"`            // 更新时间
}
