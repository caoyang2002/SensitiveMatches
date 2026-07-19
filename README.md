# SensitiveMatches – 内容安全检测工具

基于 TinyForum 敏感词库规范实现的 Go 语言内容审核库，支持：
- YAML 配置化规则（正则 + 精确词）
- 多级风险评分与动作决策（pass / review / block）
- LLM 复判
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
│       └── main.go           # HTTP 服务入口
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
import "sensitive_matches/pkg/sensitive"

func main() {
    checker, _ := sensitive.NewChecker("./sensitive-dicts")
    result := checker.Check("今天天气真好，赌博是不好的")
    fmt.Printf("敏感: %v, 动作: %s, 分数: %d\n", result.Sensitive, result.Action, result.Score)
}
```

### 启动 HTTP 服务

```bash
go run ./cmd/server
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

例如：

```bash
curl -X POST http://localhost:8080/check -H "Content-Type: application/json" -d '{"text":"自从Go 1.18正式引入泛型后，我们团队一直在探索如何在实际项目中合理使用这一特性。本文分享了我们在重构数据结构和算法库时的经验，包括类型约束的设计、性能影响分析，以及常见陷阱的规避。通过三个 真实案例，展示了泛型如何提升代码复用性和类型安全性。套现"}'
```
响应：

```json
{
  "original":"自从Go 1.18正式引入泛型后，我们团队一直在探索如何在实际项目中合理使用这一特性。本文分享了我们在重构数据结构和算法库时的经验，包括类型约束的设计、性能影响分析，以及常见陷阱的规避。通过三个 真实案例，展示了泛型如何提升代码复用性和类型安全性。套现",
  "masked":"自从go 1.18正式引入泛型后,我们团队一直在探索如何在实际项目中合理使用这一特性。本文分享了我们在重构数据结构和算法库时的经验,包括类型约束的设计、性能影响分析,以及常见陷阱的规避。通过三个 真实案例,展示了泛型如何提升代码复用性和类型安全性。套现",
  "sensitive":false,
  "level":"Normal",
  "score":0,
  "action":"pass",
  "matches":[{
    "rule_id":"GAMBLE_002",
    "category":"finance",
    "word":"套现",
    "start":348,
    "end":354,
    "action":"review",
    "score":60,
    "tags":null
  }]
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

## rule 格式

# 规则文件格式说明

## 概述

规则文件是敏感词审核系统的核心，用于定义**识别模式**（关键词、正则表达式）及其**基础处置方式**。规则按类别（如 `gambling`、`politics`）组织，通常每个类别对应一个 YAML 文件，存放在 `sensitive-dicts` 的子目录中。

系统支持通过 **策略文件（`policy.yml`）** 统一覆盖规则自带的动作和分数，实现更灵活的管理。

---

## 规则文件结构

规则文件为 YAML 格式，根节点包含 `version` 和 `rules` 列表。

```yaml
version: 1

rules:
  - id: <字符串>
    category: <字符串>      # 可选，默认从父目录名继承
    name: <字符串>          # 可选，规则名称
    action: <pass|review|block|shadow|replace>   # 可选，默认 review
    score: <整数>           # 风险分值，建议 0~100
    regex: <字符串或列表>    # 正则表达式，支持多行
    tags: <列表>            # 标签，用于策略匹配
    replace: <布尔>         # 是否替换敏感词（暂未广泛使用）
    description: <字符串>   # 规则说明
    examples: <列表>        # 示例文本，仅用于文档
```

> **注意**：`whitelist.yml` 和 `blacklist.yml` 为特殊文件，使用 `word` 字段进行精确词匹配，而不是 `regex`。

---

## 字段详解

| 字段        | 类型           | 必填 | 说明 |
|------------|---------------|------|------|
| `id`       | string        | 是   | 规则唯一标识符，如 `GAMBLE_001`。建议按类别+序号命名。 |
| `category` | string        | 否   | 规则所属类别（如 `gambling`）。若未填写，系统自动从父目录名继承。 |
| `name`     | string        | 否   | 规则名称（人类可读），用于日志和调试。 |
| `action`   | string        | 否   | 处置动作，可选值：`pass`、`review`、`block`、`shadow`、`replace`。若未填写，默认 `review`。 |
| `score`    | int           | 是   | 风险分数（0~100），用于累计风险等级。通常在策略中会被覆盖。 |
| `regex`    | string / list | 否*  | 正则表达式，支持单个字符串或字符串列表。**如果未提供，必须提供 `word`**。 |
| `word`     | string        | 否*  | 精确匹配词（仅限白/黑名单文件）。**如果未提供，必须提供 `regex`**。 |
| `tags`     | list          | 否   | 字符串标签列表，用于策略条件匹配（如 `[porn, adult]`）。 |
| `replace`  | bool          | 否   | 是否执行敏感词替换，默认 `false`。 |
| `description` | string     | 否   | 规则说明，供业务人员参考。 |
| `examples` | list          | 否   | 示例敏感文本，仅用于文档。 |

> *`regex` 和 `word` 二选一，普通规则使用 `regex`，白/黑名单使用 `word`。

---

## 正则表达式写法

- **单个正则**：可直接写字符串，也支持多行格式（`>`）：
  ```yaml
  regex: >
      (赌博|六合彩|赌球)
  ```
  系统会自动去除多行中的空白字符，因此可安全换行和缩进。

- **多个正则**：写为列表，系统会用 `|` 连接：
  ```yaml
  regex:
    - "赌博"
    - "六合彩"
  ```

- **注意**：正则中如需匹配特殊字符（如 `\`、`(`），请按 Go 正则语法转义。

---

## 完整示例

### 1. 普通规则文件 `gambling.yml`
```yaml
version: 1

name: gambling

rules:
  - id: GAMBLE_001
    category: gambling
    name: 赌博博彩
    action: block
    score: 90
    regex: >
      (六合彩|赌球|真人百家乐|赌场|网赌|网络赌博|赌博平台|时时彩|外围|赌博|跑分)
    tags:
      - gambling
      - fraud
    description: 赌博及博彩类内容
    examples:
      - "网络赌博平台"
      - "六合彩开奖"

  - id: GAMBLE_002
    category: fraud
    name: 金融诈骗
    action: review
    score: 80
    regex:
      - "杀猪盘"
      - "高利贷"
      - "套现"
      - "刷单返利"
    tags:
      - fraud
    description: 金融诈骗类内容
```

### 2. 白名单文件 `whitelist.yml`
```yaml
version: 1
rules:
  - word: "微信支付"
    action: pass
    score: 0
  - word: "赌博小说"
    action: pass
    score: 0
```

### 3. 黑名单文件 `blacklist.yml`
```yaml
version: 1
rules:
  - word: "赌博"
    action: block
    score: 100
```

---

## 与策略文件（`policy.yml`）配合

策略文件按 `category` 统一覆盖规则的动作和分数，例如：

```yaml
version: 1
actions:
  block:
    score: 90
    categories:
      - gambling
      - fraud
      - violence
  review:
    score: 60
    categories:
      - politics
      - medical
  shadow:
    score: 40
    categories:
      - spam
      - advertisement
```

当规则命中某类别时，若该类别出现在策略中，则其 `action` 和 `score` 会被策略值覆盖。这实现了 **识别与处置分离**。

---

## 注意事项

1. **文件命名**：建议用类别名命名（如 `gambling.yml`），存放在对应子目录（如 `finance/`）。
2. **编码**：YAML 文件必须使用 **UTF-8** 编码，支持中文。
3. **避免空白**：多行正则中的缩进和换行会被自动移除，因此可放心格式化。
4. **默认值**：若未写 `category`，系统会从父目录名继承；若未写 `action`，默认为 `review`。
5. **测试**：新增或修改规则后，可通过 `/reload` 热更新，无需重启服务。

---

## 快速检查清单

- [ ] 文件扩展名为 `.yml`
- [ ] 包含 `version: 1` 和 `rules:` 列表
- [ ] 每条规则有唯一 `id`
- [ ] `regex` 或 `word` 至少有一个
- [ ] 正则表达式语法正确（可用 Go 的 `regexp` 包验证）

---

如有更多问题，请参考项目 README 或联系开发团队。
## 贡献

欢迎提交 Issue 和 Pull Request。请确保新增功能有对应的测试用例。

---

## License

内部使用，版权归 TinyForum 所有。
