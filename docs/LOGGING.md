# 日志系统配置指南

K8s Wizard 使用结构化日志系统，支持控制台输出和文件落盘，并自动进行日志轮转。

## 快速开始

日志系统默认启用，无需额外配置。日志文件位置：

```
~/.k8s-wizard/logs/k8s-wizard.log
```

查看日志：

```bash
# 实时查看日志
tail -f ~/.k8s-wizard/logs/k8s-wizard.log

# 查看最近 100 行
tail -100 ~/.k8s-wizard/logs/k8s-wizard.log

# 搜索特定操作
grep "create" ~/.k8s-wizard/logs/k8s-wizard.log
```

## 配置选项

在配置文件 `~/.config/k8s-wizard/config.json` 中添加 `log` 字段：

```json
{
  "meta": {"version": "1.0.0"},
  "log": {
    "enableFile": true,
    "filePath": "~/.k8s-wizard/logs/k8s-wizard.log",
    "maxSize": 100,
    "maxBackups": 3,
    "maxAge": 30,
    "compress": true,
    "level": "info",
    "format": "json",
    "console": true
  }
}
```

### 配置项说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enableFile` | bool | `true` | 是否启用文件日志 |
| `filePath` | string | `~/.k8s-wizard/logs/k8s-wizard.log` | 日志文件路径 |
| `maxSize` | int | `100` | 单个日志文件最大大小（MB） |
| `maxBackups` | int | `3` | 保留的旧日志文件数量 |
| `maxAge` | int | `30` | 旧日志文件保留天数 |
| `compress` | bool | `true` | 是否压缩旧日志文件 |
| `level` | string | `info` | 日志级别：`debug`, `info`, `warn`, `error` |
| `format` | string | `json` | 日志格式：`json` 或 `text` |
| `console` | bool | `true` | 是否同时输出到控制台 |

## 日志级别

| 级别 | 说明 |
|------|------|
| `debug` | 调试信息，包含详细的执行流程 |
| `info` | 常规操作信息（默认） |
| `warn` | 警告信息，非致命错误 |
| `error` | 错误信息，操作失败 |

## 日志格式

### JSON 格式（推荐）

```json
{
  "time": "2025-03-01T17:30:00.123456+08:00",
  "level": "INFO",
  "msg": "HTTP 请求",
  "method": "POST",
  "path": "/api/chat",
  "status": 200,
  "latency": "1.234s",
  "clientIP": "127.0.0.1"
}
```

### Text 格式

```
time=2025-03-01T17:30:00.123+08:00 level=INFO msg="HTTP 请求" method=POST path=/api/chat status=200 latency=1.234s clientIP=127.0.0.1
```

## 日志轮转

日志系统使用 [lumberjack](https://github.com/natefinch/lumberjack) 实现自动轮转：

### 轮转触发条件

1. **大小触发**: 当前日志文件超过 `maxSize` MB
2. **时间触发**: 每次启动服务时检查旧文件是否超过 `maxAge` 天

### 轮转行为

```
k8s-wizard.log          # 当前日志文件
k8s-wizard-2025-03-01T00:00:00.000.log.gz  # 轮转后的压缩文件
k8s-wizard-2025-02-28T00:00:00.000.log.gz
k8s-wizard-2025-02-27T00:00:00.000.log.gz
```

### 清理策略

- 保留最近 `maxBackups` 个备份文件
- 删除超过 `maxAge` 天的旧文件

## 使用示例

### 在代码中使用

```go
import "k8s-wizard/pkg/logger"

// 初始化（通常在 main.go）
log, err := logger.Init(&logger.Config{
    EnableFile: true,
    Level:      "info",
    Format:     "json",
    Console:    true,
})
if err != nil {
    log.Fatal("初始化日志失败", "error", err)
}
defer log.Close()

// 记录日志
logger.Info("服务启动", "port", 8080)
logger.Debug("调试信息", "action", "create", "resource", "deployment")
logger.Warn("警告", "reason", "资源不足")
logger.Error("错误", "error", err)

// 带上下文
logger.Info("操作完成",
    "action", "create",
    "resource", "deployment/nginx",
    "namespace", "default",
    "duration", "1.5s",
)
```

### 日志查询示例

```bash
# 查看所有错误日志
grep '"level":"ERROR"' ~/.k8s-wizard/logs/k8s-wizard.log

# 查看特定资源的操作
grep 'deployment/nginx' ~/.k8s-wizard/logs/k8s-wizard.log

# 查看今天的创建操作
grep "$(date +%Y-%m-%d)" ~/.k8s-wizard/logs/k8s-wizard.log | grep 'create'

# 统计错误数量
grep -c '"level":"ERROR"' ~/.k8s-wizard/logs/k8s-wizard.log

# 分析响应时间
grep 'latency' ~/.k8s-wizard/logs/k8s-wizard.log | tail -100
```

## 日志分析

### 使用 jq 分析 JSON 日志

```bash
# 格式化日志
cat ~/.k8s-wizard/logs/k8s-wizard.log | jq .

# 筛选错误日志
cat ~/.k8s-wizard/logs/k8s-wizard.log | jq 'select(.level=="ERROR")'

# 提取特定字段
cat ~/.k8s-wizard/logs/k8s-wizard.log | jq '{time, level, msg, action}'

# 统计各级别日志数量
cat ~/.k8s-wizard/logs/k8s-wizard.log | jq -r '.level' | sort | uniq -c
```

### 日志聚合（生产环境）

对于生产环境，建议将日志发送到日志聚合系统：

1. **Loki + Grafana**: 轻量级日志聚合
2. **Elasticsearch + Kibana**: 功能完整的日志分析
3. **Fluentd/Filebeat**: 日志收集代理

## 性能考虑

- 日志写入使用缓冲，对性能影响极小
- 日志轮转在后台进行，不阻塞主流程
- 压缩旧日志使用 gzip，CPU 开销可控
- JSON 格式略慢于 Text，但更易于解析

## 故障排查

### 日志文件无法创建

检查目录权限：

```bash
mkdir -p ~/.k8s-wizard/logs
chmod 755 ~/.k8s-wizard/logs
```

### 日志文件过大

调整配置：

```json
{
  "log": {
    "maxSize": 50,      // 减小单文件大小
    "maxBackups": 5,    // 增加备份数
    "compress": true    // 确保压缩
  }
}
```

### 日志丢失

检查磁盘空间：

```bash
df -h ~/.k8s-wizard/logs
```

## 最佳实践

1. **生产环境**: 设置 `level: info`，避免 debug 日志过多
2. **开发环境**: 设置 `level: debug`，方便调试
3. **磁盘空间**: 定期检查日志目录大小
4. **日志轮转**: 保持默认配置即可自动管理
5. **敏感信息**: 日志会自动脱敏 API Key 等敏感信息
