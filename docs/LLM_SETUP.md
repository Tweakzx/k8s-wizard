# LLM 配置指南

K8s Wizard 支持多个 LLM 提供商，采用灵活的配置系统。本文档介绍如何配置 API Key 和切换模型。

## 目录

- [支持的 LLM 提供商](#支持的-llm-提供商)
- [配置方式](#配置方式)
- [配置文件详解](#配置文件详解)
- [环境变量参考](#环境变量参考)
- [动态切换模型](#动态切换模型)
- [常见问题](#常见问题)

---

## 支持的 LLM 提供商

### 1. GLM (智谱) ⭐ 推荐

| 属性 | 值 |
|------|-----|
| 提供商 ID | `glm` |
| API 端点 | `https://open.bigmodel.cn/api/coding/paas/v4` |
| 认证方式 | Bearer Token |
| 环境变量 | `GLM_API_KEY` |

**特点**：
- 国内 API，无需科学上网
- 响应速度快
- 性价比高
- Coding 端点支持更多模型

**可用模型**（通过 API 动态获取）：
- `glm/glm-4-flash` - 快速响应，性价比高（推荐）
- `glm/glm-4-air` - 成本更低
- `glm/glm-4.5` - 新一代模型
- `glm/glm-4.6` - 增强版本
- `glm/glm-4.7` - 最新版本
- `glm/glm-5` - 最新旗舰

**获取 API Key**：
1. 访问 https://open.bigmodel.cn/
2. 注册并登录
3. 进入「API 密钥」页面
4. 创建 API Key
5. 复制 API Key

---

### 2. DeepSeek

| 属性 | 值 |
|------|-----|
| 提供商 ID | `deepseek` |
| API 端点 | `https://api.deepseek.com/v1` |
| 认证方式 | Bearer Token |
| 环境变量 | `DEEPSEEK_API_KEY` |

**特点**：
- 国内 API，无需科学上网
- 性价比极高
- 支持长文本和推理

**可用模型**：
- `deepseek/deepseek-chat` - 通用对话模型
- `deepseek/deepseek-coder` - 代码优化模型
- `deepseek/deepseek-reasoner` - 推理增强模型

**获取 API Key**：
1. 访问 https://platform.deepseek.com/
2. 注册并登录
3. 进入「API Keys」页面
4. 创建 API Key

---

### 3. Claude (Anthropic)

| 属性 | 值 |
|------|-----|
| 提供商 ID | `claude` |
| API 端点 | `https://api.anthropic.com/v1` |
| 认证方式 | x-api-key Header |
| 环境变量 | `ANTHROPIC_API_KEY` |

**特点**：
- 强大的推理能力
- 支持长上下文（200K tokens）
- 需要科学上网

**可用模型**：
- `claude/claude-sonnet-4-20250514` - Claude Sonnet 4

**获取 API Key**：
1. 访问 https://console.anthropic.com/
2. 注册并登录
3. 创建 API Key

---

## 配置方式

K8s Wizard 支持三种配置 API Key 的方式，按优先级排序：

### 方式 1：环境变量（推荐）

最简单的方式，适合快速测试和 CI/CD 环境。

```bash
# GLM
export GLM_API_KEY=your-glm-api-key

# DeepSeek
export DEEPSEEK_API_KEY=your-deepseek-api-key

# Claude
export ANTHROPIC_API_KEY=your-anthropic-api-key
```

### 方式 2：凭证文件

适合多用户、多环境场景，API Key 与配置分离。

**位置**（按优先级）：
1. `~/.config/k8s-wizard/credentials.json` (XDG 标准位置)
2. `~/.k8s-wizard/credentials.json` (传统位置)

**格式**：
```json
{
  "profiles": {
    "glm:default": {
      "apiKey": "your-glm-api-key"
    },
    "deepseek:default": {
      "apiKey": "your-deepseek-api-key"
    },
    "claude:default": {
      "apiKey": "your-anthropic-api-key"
    }
  }
}
```

### 方式 3：同时配置多个提供商

可以同时配置多个提供商的 API Key，然后在运行时切换模型：

```bash
# 设置多个 API Key
export GLM_API_KEY=your-glm-api-key
export DEEPSEEK_API_KEY=your-deepseek-api-key

# 启动服务
make dev
```

---

## 配置文件详解

### 主配置文件 (`config.json`)

**位置**（按优先级）：
1. `~/.config/k8s-wizard/config.json`
2. `~/.k8s-wizard/config.json`

**完整示例**：
```json
{
  "meta": {
    "version": "1.0.0",
    "lastTouchedAt": "",
    "lastTouchedBy": ""
  },
  "models": {
    "mode": "merge",
    "providers": {
      "glm": {
        "baseUrl": "https://open.bigmodel.cn/api/coding/paas/v4",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [
          {
            "id": "glm-4-flash",
            "name": "GLM-4 Flash",
            "reasoning": true,
            "contextWindow": 128000,
            "maxTokens": 131072
          }
        ]
      },
      "deepseek": {
        "baseUrl": "https://api.deepseek.com/v1",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [
          {
            "id": "deepseek-chat",
            "name": "DeepSeek Chat",
            "reasoning": true,
            "contextWindow": 64000,
            "maxTokens": 8192
          }
        ]
      },
      "claude": {
        "baseUrl": "https://api.anthropic.com/v1",
        "auth": "api-key",
        "api": "anthropic",
        "models": [
          {
            "id": "claude-sonnet-4-20250514",
            "name": "Claude Sonnet 4",
            "reasoning": true,
            "contextWindow": 200000,
            "maxTokens": 8192
          }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "model": {
        "primary": "glm/glm-4-flash"
      }
    }
  },
  "api": {
    "port": 8080,
    "host": "0.0.0.0"
  }
}
```

### 配置字段说明

| 字段 | 说明 |
|------|------|
| `meta.version` | 配置文件版本 |
| `models.mode` | 模型合并模式（`merge` 或 `replace`） |
| `models.providers` | 提供商配置映射 |
| `providers[].baseUrl` | API 端点 URL |
| `providers[].auth` | 认证方式（`api-key`） |
| `providers[].api` | API 格式（`openai-completions` 或 `anthropic`） |
| `providers[].models` | 可用模型列表 |
| `agents.defaults.model.primary` | 默认使用的模型 |
| `api.port` | API 服务端口 |
| `api.host` | API 服务监听地址 |

---

## 环境变量参考

| 变量 | 说明 | 示例 |
|------|------|------|
| `GLM_API_KEY` | GLM API Key | `abc123...` |
| `DEEPSEEK_API_KEY` | DeepSeek API Key | `sk-xxx...` |
| `ANTHROPIC_API_KEY` | Claude API Key | `sk-ant-xxx...` |
| `K8S_WIZARD_MODEL` | 覆盖默认模型 | `deepseek/deepseek-chat` |
| `PORT` | API 服务端口 | `8080` |
| `KUBECONFIG` | K8s 配置文件路径 | `~/.kube/config` |

**优先级**：环境变量 > 凭证文件 > 配置文件默认值

---

## 动态切换模型

### 前端切换

在 Web UI 中，点击顶部的模型选择器即可切换模型。只有配置了 API Key 的提供商会显示可用模型。

### 后端 API 切换

```bash
# 获取当前模型
curl http://localhost:8080/api/config/model

# 切换模型
curl -X PUT http://localhost:8080/api/config/model \
  -H "Content-Type: application/json" \
  -d '{"model": "deepseek/deepseek-chat"}'
```

### 配置文件切换

编辑 `~/.k8s-wizard/config.json`：

```json
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "deepseek/deepseek-chat"
      }
    }
  }
}
```

### 环境变量切换

```bash
export K8S_WIZARD_MODEL="deepseek/deepseek-chat"
make dev:api
```

---

## 模型字符串格式

K8s Wizard 使用 `provider/model-id` 格式标识模型：

```
glm/glm-4-flash          # GLM-4 Flash
glm/glm-4-air            # GLM-4 Air
deepseek/deepseek-chat   # DeepSeek Chat
claude/claude-sonnet-4-20250514  # Claude Sonnet 4
```

---

## 常见问题

### Q: 应该选择哪个 LLM？

**推荐顺序**：

1. **GLM** - 国内用户首选，速度快、性价比高
2. **DeepSeek** - 成本更低，适合大量调用
3. **Claude** - 需要科学上网，推理能力最强

### Q: 如何测试 API Key 是否有效？

启动服务后，访问 API 检查：

```bash
# 健康检查
curl http://localhost:8080/health

# 查看当前模型配置
curl http://localhost:8080/api/config/model

# 测试聊天
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"content": "你好"}'
```

### Q: 提示 "API key not found" 怎么办？

检查以下项：

1. 环境变量是否设置正确
2. 凭证文件路径是否正确
3. 凭证文件 JSON 格式是否有效
4. API Key 对应的提供商是否正确

```bash
# 检查环境变量
echo $GLM_API_KEY

# 检查凭证文件
cat ~/.k8s-wizard/credentials.json
```

### Q: 多个 API Key 都设置了会使用哪个？

系统会根据当前选择的模型自动使用对应的 API Key：

- 选择 `glm/*` 模型 → 使用 `GLM_API_KEY`
- 选择 `deepseek/*` 模型 → 使用 `DEEPSEEK_API_KEY`
- 选择 `claude/*` 模型 → 使用 `ANTHROPIC_API_KEY`

### Q: 前端看不到某些模型？

可能原因：

1. **未配置 API Key** - 只有配置了 API Key 的提供商会显示
2. **API 获取失败** - 检查网络连接和 API Key 有效性
3. **模型不存在** - 系统会从 API 动态获取真实可用的模型列表

### Q: 如何添加新的 LLM 提供商？

编辑配置文件，在 `models.providers` 中添加新的提供商：

```json
{
  "models": {
    "providers": {
      "new-provider": {
        "baseUrl": "https://api.example.com/v1",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [
          {
            "id": "model-1",
            "name": "Model 1"
          }
        ]
      }
    }
  }
}
```

然后在凭证文件中添加 API Key：

```json
{
  "profiles": {
    "new-provider:default": {
      "apiKey": "your-api-key"
    }
  }
}
```

### Q: 配置文件可以备份吗？

可以。配置文件是 JSON 格式，可以直接复制备份。

**注意**：
- `config.json` 可以提交到版本控制
- `credentials.json` 包含敏感信息，**不要提交到版本控制**

建议在 `.gitignore` 中添加：
```
credentials.json
**/credentials.json
```

---

## API Key 获取链接汇总

| 提供商 | 获取地址 | 文档 |
|--------|----------|------|
| GLM (智谱) | https://open.bigmodel.cn/ | https://open.bigmodel.cn/dev/api |
| DeepSeek | https://platform.deepseek.com/ | https://platform.deepseek.com/docs |
| Claude | https://console.anthropic.com/ | https://docs.anthropic.com/ |

---

## 快速开始示例

### 使用 GLM

```bash
# 1. 设置 API Key
export GLM_API_KEY=your-glm-api-key

# 2. 启动服务
make dev

# 3. 访问 Web UI
open http://localhost:5173
```

### 使用凭证文件

```bash
# 1. 创建配置目录
mkdir -p ~/.k8s-wizard

# 2. 创建凭证文件
cat > ~/.k8s-wizard/credentials.json << 'EOF'
{
  "profiles": {
    "glm:default": {
      "apiKey": "your-glm-api-key"
    }
  }
}
EOF

# 3. 启动服务
make dev
```

---

## 安全建议

1. **不要将 API Key 提交到版本控制**
2. **使用环境变量** 在生产环境中配置敏感信息
3. **定期轮换 API Key** 提高安全性
4. **限制 API Key 权限** 如果提供商支持
5. **监控 API 使用量** 避免意外超支
