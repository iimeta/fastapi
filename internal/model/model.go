package model

import "github.com/iimeta/fastapi/internal/model/common"

type Model struct {
	Id                   string                      `json:"id,omitempty"`                      // ID
	Corp                 string                      `json:"corp,omitempty"`                    // 公司
	Name                 string                      `json:"name,omitempty"`                    // 模型名称
	Model                string                      `json:"model,omitempty"`                   // 模型
	Type                 int                         `json:"type,omitempty"`                    // 模型类型[1:文生文, 2:文生图, 3:图生文, 4:图生图, 5:文生语音, 6:语音生文, 100:多模态, 101:多模态实时, 102:多模态语音]
	BaseUrl              string                      `json:"base_url,omitempty"`                // 模型地址
	Path                 string                      `json:"path,omitempty"`                    // 模型路径
	IsEnablePresetConfig bool                        `json:"is_enable_preset_config,omitempty"` // 是否启用预设配置
	PresetConfig         common.PresetConfig         `json:"preset_config,omitempty"`           // 预设配置
	TextQuota            common.TextQuota            `json:"text_quota,omitempty"`              // 文本额度
	ImageQuotas          []common.ImageQuota         `json:"image_quotas,omitempty"`            // 图像额度
	AudioQuota           common.AudioQuota           `json:"audio_quota,omitempty"`             // 音频额度
	MultimodalQuota      common.MultimodalQuota      `json:"multimodal_quota,omitempty"`        // 多模态额度
	RealtimeQuota        common.RealtimeQuota        `json:"realtime_quota,omitempty"`          // 多模态实时额度
	MultimodalAudioQuota common.MultimodalAudioQuota `json:"multimodal_audio_quota,omitempty"`  // 多模态语音额度
	MidjourneyQuotas     []common.MidjourneyQuota    `json:"midjourney_quotas,omitempty"`       // Midjourney额度
	DataFormat           int                         `json:"data_format,omitempty"`             // 数据格式[1:统一格式, 2:官方格式]
	IsPublic             bool                        `json:"is_public,omitempty"`               // 是否公开
	IsEnableModelAgent   bool                        `json:"is_enable_model_agent,omitempty"`   // 是否启用模型代理
	LbStrategy           int                         `json:"lb_strategy,omitempty"`             // 代理负载均衡策略[1:轮询, 2:权重]
	ModelAgents          []string                    `json:"model_agents,omitempty"`            // 模型代理
	IsEnableForward      bool                        `json:"is_enable_forward,omitempty"`       // 是否启用模型转发
	ForwardConfig        *common.ForwardConfig       `json:"forward_config,omitempty"`          // 模型转发配置
	IsEnableFallback     bool                        `json:"is_enable_fallback,omitempty"`      // 是否启用后备
	FallbackConfig       *common.FallbackConfig      `json:"fallback_config,omitempty"`         // 后备配置
	Remark               string                      `json:"remark,omitempty"`                  // 备注
	Status               int                         `json:"status,omitempty"`                  // 状态[1:正常, 2:禁用, -1:删除]
	Creator              string                      `json:"creator,omitempty"`                 // 创建人
	Updater              string                      `json:"updater,omitempty"`                 // 更新人
	CreatedAt            int64                       `json:"created_at,omitempty"`              // 创建时间
	UpdatedAt            int64                       `json:"updated_at,omitempty"`              // 更新时间
}
