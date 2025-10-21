package model

import "github.com/iimeta/fastapi/internal/model/common"

type Model struct {
	Id                   string                 `json:"id,omitempty"`                      // ID
	ProviderId           string                 `json:"provider_id,omitempty"`             // 提供商ID
	Name                 string                 `json:"name,omitempty"`                    // 模型名称
	Model                string                 `json:"model,omitempty"`                   // 模型
	Type                 int                    `json:"type,omitempty"`                    // 模型类型[1:文生文, 2:文生图, 3:图生文, 4:图生图, 5:文生语音, 6:语音生文, 7:文本向量化, 8:文生视频, 100:多模态, 101:多模态实时, 102:多模态语音, 103:多模态向量化]
	BaseUrl              string                 `json:"base_url,omitempty"`                // 模型地址
	Path                 string                 `json:"path,omitempty"`                    // 模型路径
	IsEnablePresetConfig bool                   `json:"is_enable_preset_config,omitempty"` // 是否启用预设配置
	PresetConfig         common.PresetConfig    `json:"preset_config,omitempty"`           // 预设配置
	Pricing              common.Pricing         `json:"pricing,omitempty"`                 // 定价
	RequestDataFormat    int                    `json:"request_data_format,omitempty"`     // 请求数据格式[1:统一格式, 2:官方格式]
	ResponseDataFormat   int                    `json:"response_data_format,omitempty"`    // 响应数据格式[1:统一格式, 2:官方格式]
	IsPublic             bool                   `json:"is_public,omitempty"`               // 是否公开
	IsEnableModelAgent   bool                   `json:"is_enable_model_agent,omitempty"`   // 是否启用模型代理
	LbStrategy           int                    `json:"lb_strategy,omitempty"`             // 代理负载均衡策略[1:轮询, 2:权重]
	ModelAgents          []string               `json:"model_agents,omitempty"`            // 模型代理
	IsEnableForward      bool                   `json:"is_enable_forward,omitempty"`       // 是否启用模型转发
	ForwardConfig        *common.ForwardConfig  `json:"forward_config,omitempty"`          // 模型转发配置
	IsEnableFallback     bool                   `json:"is_enable_fallback,omitempty"`      // 是否启用后备
	FallbackConfig       *common.FallbackConfig `json:"fallback_config,omitempty"`         // 后备配置
	Remark               string                 `json:"remark,omitempty"`                  // 备注
	Status               int                    `json:"status,omitempty"`                  // 状态[1:正常, 2:禁用, -1:删除]
	Creator              string                 `json:"creator,omitempty"`                 // 创建人
	Updater              string                 `json:"updater,omitempty"`                 // 更新人
	CreatedAt            int64                  `json:"created_at,omitempty"`              // 创建时间
	UpdatedAt            int64                  `json:"updated_at,omitempty"`              // 更新时间
}
