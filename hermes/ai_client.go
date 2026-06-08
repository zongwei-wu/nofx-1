package hermes

import (
	"fmt"
	"log"

	"nofx/config"
	"nofx/mcp"
)

// NewAIClientFromModel 根据用户模型配置创建 AI 客户端
func NewAIClientFromModel(model *config.AIModelConfig) (mcp.AIClient, error) {
	if model == nil {
		return nil, fmt.Errorf("AI 模型配置为空")
	}
	if !model.Enabled || model.APIKey == "" {
		return nil, fmt.Errorf("AI 模型 %s 未启用或未配置 API Key", model.ID)
	}

	switch model.Provider {
	case "qwen":
		client := mcp.NewQwenClient()
		client.SetAPIKey(model.APIKey, model.CustomAPIURL, model.CustomModelName)
		return client, nil
	case "deepseek":
		client := mcp.NewDeepSeekClient()
		client.SetAPIKey(model.APIKey, model.CustomAPIURL, model.CustomModelName)
		return client, nil
	case "custom":
		client := mcp.New()
		client.SetAPIKey(model.APIKey, model.CustomAPIURL, model.CustomModelName)
		return client, nil
	default:
		client := mcp.NewDeepSeekClient()
		client.SetAPIKey(model.APIKey, model.CustomAPIURL, model.CustomModelName)
		log.Printf("⚠️ Hermes: 未知 provider %s，回退 DeepSeek", model.Provider)
		return client, nil
	}
}
