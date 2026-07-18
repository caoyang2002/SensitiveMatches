# TinyForum Sensitive Dictionary

TinyForum 内容安全敏感词库。

基于 **规则 + 标签 + 风险等级 + 正则表达式** 的审核规则系统，用于论坛、社区、评论、私信等用户生成内容（UGC）的风险检测。

支持：

- YAML 配置化规则
- 正则匹配
- 分类管理
- 风险评分
- 自动拦截
- 人工审核
- 敏感词替换
- 规则热更新

---

# 目录结构

```bash
sensitive-dicts/

├── README.md
├── index.yml                 # 规则加载入口

├── normalize.yml             # 文本归一化
├── whitelist.yml             # 白名单
├── blacklist.yml             # 黑名单


# =========================
# 内容分类规则
# =========================

├── advertisement.yml         # 广告法
├── authority.yml             # 权威背书
├── promotion.yml             # 营销诱导


├── finance/
│   ├── investment.yml        # 投资诈骗
│   ├── stock.yml             # 荐股
│   ├── crypto.yml            # 虚拟货币
│   ├── gambling.yml          # 赌博博彩
│   └── fraud.yml             # 金融诈骗


├── health/
│   ├── medical.yml           # 医疗疾病
│   ├── efficacy.yml          # 医疗功效
│   └── drugs.yml             # 毒品药物


├── adult/
│   ├── pornography.yml       # 色情内容
│   ├── sexual.yml            # 性行为
│   └── abuse.yml             # 虐待内容


├── religion/
│   ├── superstition.yml      # 算命、占卜
│   ├── fengshui.yml          # 风水改运
│   ├── occult.yml            # 巫术、通灵
│   ├── cultivation.yml       # 修仙、炼丹
│   └── religion.yml          # 宗教内容


├── security/
│   ├── violence.yml          # 暴力内容
│   ├── weapons.yml           # 武器
│   ├── dangerous.yml         # 危险行为
│   └── self_harm.yml         # 自伤风险


├── society/
│   ├── politics.yml          # 政治内容
│   ├── illegal.yml           # 违法违规
│   ├── teenager.yml          # 未成年人保护
│   └── contact.yml           # 联系方式


├── spam/
│   ├── spam.yml              # 垃圾信息
│   └── urls.yml              # 非法网址


# =========================
# 审核策略
# =========================

└── policy/

    ├── block.yml             # 直接拦截
    ├── review.yml            # 人工审核
    └── shadow.yml            # 隐藏词
```

---

# 设计原则

## 1. 分类与策略分离

规则文件只负责：

- 内容分类
- 风险识别
- 标签定义

策略文件负责：

- 是否拦截
- 是否审核
- 是否替换

例如：

```text
pornography.yml

发现色情内容


↓

policy/block.yml

决定直接拦截
```

---

# 规则格式

所有规则统一采用 YAML：

```yaml
id:
name:
category:
priority:
action:
score:
regex:
tags:
replace:
description:
examples:
```

示例：

```yaml
version: 1

category: pornography

rules:
  - id: PORN_001

    name: 色情内容

    priority: 900

    action: block

    score: 100

    regex:
      - "色情"

      - "裸聊"

      - "成人视频"

    tags:
      - 色情

      - 成人内容

    replace: true

    description: 色情低俗内容

    examples:
      - 裸聊网站

      - 成人视频
```

---

# Rule 数据结构

Go 定义：

```go
package sensitive


type Rule struct {

	ID string `yaml:"id"`

	Name string `yaml:"name"`

	Category string `yaml:"category"`

	Priority int `yaml:"priority"`


	Action Action `yaml:"action"`

	Score int `yaml:"score"`


	Regex []string `yaml:"regex"`


	Tags []string `yaml:"tags"`


	Replace bool `yaml:"replace"`


	Description string `yaml:"description"`


	Examples []string `yaml:"examples"`
}
```

---

# MatchResult

规则命中结果：

```go
type MatchResult struct {

	RuleID string


	Category string


	Label string


	Word string


	Start int


	End int


	Action Action


	Score int


	Tags []string
}
```

字段说明：

| 字段     | 说明     |
| -------- | -------- |
| RuleID   | 规则编号 |
| Category | 风险分类 |
| Word     | 命中内容 |
| Start    | 开始位置 |
| End      | 结束位置 |
| Score    | 风险分数 |
| Action   | 处理动作 |
| Tags     | 风险标签 |

---

# CheckResult

最终审核结果：

```go
type CheckResult struct {


	Original string


	Masked string


	Sensitive bool


	Level Level


	Score int


	Action Action


	Matches []*MatchResult
}
```

示例：

输入：

```text
添加微信领取福利
```

输出：

```json
{
  "sensitive": true,

  "masked": "添加****领取福利",

  "score": 40,

  "action": "review",

  "matches": [
    {
      "category": "contact",
      "word": "微信",
      "score": 40
    }
  ]
}
```

---

# Action 类型

审核动作：

```text
pass

↓

review

↓

block
```

说明：

| Action | 说明         |
| ------ | ------------ |
| pass   | 正常通过     |
| review | 进入人工审核 |
| block  | 直接拒绝     |

---

# 风险等级

根据 Score 计算：

| Score  | 等级     | 处理     |
| ------ | -------- | -------- |
| 0-20   | Normal   | 通过     |
| 21-50  | Low      | 记录     |
| 51-80  | Medium   | 审核     |
| 81-100 | High     | 拦截     |
| 100+   | Critical | 强制拦截 |

---

# 审核流程

```
                    Text
                      |
                      v

              normalize.go

                      |
                      v

              regexp matcher

                      |
                      v


             +----------------+

             | MatchResult[]  |

             +----------------+

                      |
                      v

             risk calculator

                      |
                      v


        +-------------+-------------+

        |                           |

      block                      review

        |                           |

    reject API              admin queue

```

---

# 文本标准化

normalize.yml 用于处理：

- 大小写转换
- 繁简转换
- 全角半角转换
- 特殊字符清理
- Unicode 归一化

例如：

输入：

```
微 信
微-信
VX
ＶＸ
```

统一：

```
微信
```

---

# 白名单机制

whitelist.yml

用于降低误伤：

例如：

```yaml
rules:
  - word: 微信支付

    action: pass

  - word: 赌博小说

    action: pass
```

---

# 黑名单机制

blacklist.yml

高风险强制规则：

```yaml
rules:
  - word: xxx

    action: block

    score: 100
```

---

# 性能优化

推荐：

## 规则加载

启动时：

```
YAML

↓

Rule Loader

↓

Regex Cache

↓

Matcher
```

## 匹配优化

规则数量较大时：

推荐：

- Trie 前缀树
- Aho-Corasick 自动机
- Regex 编译缓存

流程：

```
Rule

 |

 +-- exact words

 |

 +-- regex

 |

 +-- keywords index

```

---

# 版本管理

建议：

```
sensitive-dicts

v1.0

v1.1

v1.2
```

每次修改：

记录：

- 修改人
- 修改时间
- 新增规则
- 删除规则
- 命中变化

---

# 使用场景

适用于：

- 社区论坛
- 评论系统
- 私信系统
- 博客平台
- 内容管理系统
- AI 对话系统

---

# License

TinyForum Sensitive Dictionary

内部内容安全规则库。
