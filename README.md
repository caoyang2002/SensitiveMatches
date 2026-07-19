# SensitiveMatches – 内容安全敏感词 SDK

基于 TinyForum 敏感词库规范实现的 Go 语言内容审核 SDK，支持：
- YAML 配置化规则（正则 + 精确词）
- 多级风险评分与动作决策（pass / review / block）
- 文本归一化（全角转半角、自定义映射）
- 白名单 / 黑名单机制
- 热更新规则
- 独立 HTTP 服务或 Go 库集成

---

## 项目结构

```bash
SensitiveMatches/
├── go.mod
├── go.sum
├── sensitive-dicts/          # 规则目录（可自定义）
│   ├── normalize.yml         # 归一化映射
│   ├── whitelist.yml         # 白名单
│   ├── blacklist.yml         # 黑名单
│   └── *.yml                 # 分类规则（如 gambling.yml）
├── pkg/
│   └── sensitive/
│       ├── rule.go           # 数据结构
│       ├── loader.go         # 加载 YAML 规则
│       ├── normalizer.go     # 文本归一化
│       ├── matcher.go        # 正则 + 精确词匹配
│       ├── checker.go        # 审核逻辑
│       └── checker_test.go   # 单元测试
├── cmd/
│   └── server/
│       ├── main.go           # HTTP 服务入口
│       └── server_test.go    # API 测试
└── README.md
```

---

## 功能特性

- **规则驱动**：所有规则均通过 YAML 文件配置，支持 `regex` 多行或数组格式。
- **动作决策**：根据命中的规则自动计算最终动作（`pass` / `review` / `block`）。
- **风险评分**：每个规则带 `score`，累加后转换为风险等级（Normal ~ Critical）。
- **白/黑名单**：精确词匹配，黑名单强制 `block`，白名单强制 `pass`。
- **文本归一化**：全角转半角、大小写统一、自定义映射（如“VX”→“微信”）。
- **热更新**：通过 `/reload` 端点或代码调用 `Reload()` 动态加载新规则。
- **高性能**：使用读写锁，支持高并发请求。

---

## 安装

### 1. 获取代码

```bash
git clone https://github.com/caoyan2002/SensitiveMatches.git
cd SensitiveMatches
```

### 2. 安装依赖

```bash
go mod tidy
```

---

## 配置规则目录

- 默认词典目录为项目根下的 `sensitive-dicts`。
- 可通过环境变量 `DICT_DIR` 指定其他路径。

规则文件格式示例：

```yaml
version: 1
rules:
  - id: GAMBLE_001
    category: gambling
    name: 赌博内容
    action: block
    score: 90
    regex:
      - "六合彩"
      - "赌博"
    tags:
      - gambling
```

**特殊文件：**
- `normalize.yml`：归一化映射
- `whitelist.yml`：白名单（只支持 `word` 字段）
- `blacklist.yml`：黑名单（只支持 `word` 字段）

---

## 使用方式

### 作为 Go 库引入

```go
import "SensitiveMatches/pkg/sensitive"

func main() {
    checker, _ := sensitive.NewChecker("./sensitive-dicts")
    result := checker.Check("今天天气真好，赌博是不好的")
    fmt.Printf("敏感: %v, 动作: %s, 分数: %d\n", result.Sensitive, result.Action, result.Score)
}
```

### 启动 HTTP 服务

```bash
go run ./cmd/server/main.go
```

服务监听 `:8080`，提供以下端点：

- `POST /check` – 检查文本敏感度
- `POST /reload` – 热更新规则

---

## API 文档

### `POST /check`

**请求体：**
```json
{
  "text": "待检测的文本"
}
```

**响应示例：**
```json
{
  "original": "今天天气真好，赌博是不好的",
  "masked": "今天天气真好，**是不好的",
  "sensitive": true,
  "level": "High",
  "score": 90,
  "action": "block",
  "matches": [
    {
      "rule_id": "GAMBLE_001",
      "category": "gambling",
      "word": "赌博",
      "start": 15,
      "end": 21,
      "action": "block",
      "score": 90,
      "tags": ["gambling"]
    }
  ]
}
```

### `POST /reload`

无请求体，成功返回 `重载成功`，失败返回错误信息。

---

## 运行测试

```bash
# 单元测试
go test -v ./pkg/sensitive

# API 测试
go test -v ./cmd/server

# 全部测试
go test ./...
```

---

## 自定义扩展

- **添加新规则**：在 `sensitive-dicts` 下创建新的 `.yml` 文件，遵循 `Rule` 结构。
- **修改归一化**：编辑 `normalize.yml` 的 `mappings` 列表。
- **调整评分策略**：修改 `calcLevel` 函数。
- **增加缓存**：在 `Check` 方法中加入 `sync.Map` 缓存结果。

---

## 性能建议

- 规则数量较多（>1000）时，可考虑使用 **Aho-Corasick** 替代精确词遍历。
- 正则表达式尽量简单，避免回溯过长的模式。
- 使用 `sync.RWMutex` 保证热更新时的并发安全。

---

## 常见问题

### Q: 为什么我的规则不生效？
- 检查 YAML 格式是否正确（`version` 和 `rules` 字段）。
- 如果 `regex` 使用了多行字符串（`>`），请确保无多余空白字符（或 SDK 已自动清理）。
- 查看启动日志中是否有“编译规则失败”的警告。

### Q: 如何让白名单优先级最高？
当前逻辑：**黑名单 > 白名单 > 普通规则**。即命中黑名单则强制 `block`，命中白名单且未命中黑名单则 `pass`，否则按普通规则中最高动作（`block` > `review` > `pass`）决定。

### Q: 支持动态添加规则吗？
支持，调用 `/reload` 端点或 `Reload()` 方法即可，无需重启服务。

---

## 贡献

欢迎提交 Issue 和 Pull Request。请确保新增功能有对应的测试用例。

---

## License

内部使用，版权归 TinyForum 所有。
