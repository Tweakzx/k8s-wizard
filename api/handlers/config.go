package handlers

import (
	"fmt"
	"net/http"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/agent"
	"k8s-wizard/pkg/config"

	"github.com/gin-gonic/gin"
)

// ConfigHandler 处理配置相关请求
type ConfigHandler struct {
	agent agent.AgentInterface
}

func NewConfigHandler(a agent.AgentInterface) *ConfigHandler {
	return &ConfigHandler{agent: a}
}

// GetModelInfo 获取当前模型信息
func (h *ConfigHandler) GetModelInfo(c *gin.Context) {
	cfg, err := config.LoadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	currentModel := h.agent.GetModelName()

	// 收集所有可用的模型（从 provider API 动态获取）
	var modelInfos []models.ModelInfo
	for providerName, provider := range cfg.Models.Providers {
		// 检查该 provider 是否有配置 API Key
		apiKey, err := config.GetAPIKey(providerName)
		if err != nil {
			continue // 跳过未配置 API Key 的 provider
		}

		// 从 provider API 获取可用模型
		availableModels, err := config.FetchAvailableModels(providerName, provider, apiKey)
		if err != nil {
			// 如果获取失败，使用配置中的模型作为回退
			fmt.Printf("⚠️ 从 %s 获取模型列表失败: %v，使用配置中的模型\n", providerName, err)
			availableModels = provider.Models
		}

		for _, model := range availableModels {
			modelInfos = append(modelInfos, models.ModelInfo{
				Provider: providerName,
				ID:       model.ID,
				Name:     model.Name,
			})
		}
	}

	// 获取配置摘要
	configSummary := map[string]string{
		"provider":       cfg.Agents.Defaults.Model.Primary,
		"configPath":     config.GetConfigPath(),
		"credentialPath": config.GetCredentialPath(),
	}

	c.JSON(http.StatusOK, models.ModelInfoResponse{
		Current: currentModel,
		Models:  modelInfos,
		Config:  configSummary,
	})
}

// GetConfig 获取完整配置
func (h *ConfigHandler) GetConfig(c *gin.Context) {
	cfg, err := config.LoadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := models.ConfigResponse{
		Version: cfg.Meta.Version,
	}
	response.Models.Primary = cfg.Agents.Defaults.Model.Primary
	response.Providers = make(map[string]models.ProviderInfo)

	for name, provider := range cfg.Models.Providers {
		var modelIDs []string
		for _, model := range provider.Models {
			modelIDs = append(modelIDs, model.ID)
		}
		response.Providers[name] = models.ProviderInfo{
			Name:    name,
			Models:  modelIDs,
			BaseURL: provider.BaseURL,
		}
	}

	c.JSON(http.StatusOK, response)
}

// SetModel 切换模型
func (h *ConfigHandler) SetModel(c *gin.Context) {
	var req models.SetModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.SetModelResponse{
			Success: false,
			Error:   "无效的请求格式: " + err.Error(),
		})
		return
	}

	// 默认持久化模型切换结果
	persist := true
	if req.Persist != nil {
		persist = *req.Persist
	}

	// 切换运行时模型（包含 client/graph 重建）
	if err := h.agent.SetModel(req.Model); err != nil {
		c.JSON(http.StatusInternalServerError, models.SetModelResponse{
			Success: false,
			Error:   "模型切换失败: " + err.Error(),
		})
		return
	}

	if persist {
		cfg, err := config.LoadConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.SetModelResponse{
				Success: false,
				Model:   h.agent.GetModelName(),
				Error:   "模型已切换，但加载配置失败: " + err.Error(),
			})
			return
		}

		cfg.Agents.Defaults.Model.Primary = req.Model
		if err := cfg.Save(); err != nil {
			c.JSON(http.StatusInternalServerError, models.SetModelResponse{
				Success: false,
				Model:   h.agent.GetModelName(),
				Error:   "模型已切换，但保存配置失败: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, models.SetModelResponse{
		Success: true,
		Model:   h.agent.GetModelName(),
		Message: "模型切换成功",
	})
}
