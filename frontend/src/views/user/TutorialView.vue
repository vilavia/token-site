<template>
  <AppLayout>
    <div class="mx-auto max-w-3xl">
      <div class="card">
        <div class="p-6 sm:p-8">
          <div
            class="prose prose-gray dark:prose-invert max-w-none
                   prose-headings:font-semibold
                   prose-h1:text-2xl prose-h1:mb-4
                   prose-h2:text-xl prose-h2:mt-8 prose-h2:mb-3
                   prose-h3:text-base prose-h3:mt-5 prose-h3:mb-2
                   prose-p:text-gray-700 dark:prose-p:text-dark-200
                   prose-code:bg-gray-100 dark:prose-code:bg-dark-800
                   prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded
                   prose-code:text-sm prose-code:font-mono prose-code:before:content-none prose-code:after:content-none
                   prose-pre:bg-gray-900 dark:prose-pre:bg-dark-900
                   prose-pre:rounded-xl prose-pre:p-4
                   prose-pre:overflow-x-auto
                   prose-a:text-primary-600 dark:prose-a:text-primary-400
                   prose-li:text-gray-700 dark:prose-li:text-dark-200
                   prose-strong:text-gray-900 dark:prose-strong:text-white
                   prose-blockquote:border-primary-400 prose-blockquote:bg-primary-50 dark:prose-blockquote:bg-primary-900/20"
            v-html="renderedContent"
          ></div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AppLayout from '@/components/layout/AppLayout.vue'

const { locale } = useI18n()

const baseUrl = window.location.origin + '/v1'

const markdownEn = computed(() => `
# OpenClaw Usage Tutorial

## 1. What is OpenClaw?

**OpenClaw** is an open-source command-line AI assistant that connects to any OpenAI-compatible API endpoint. It lets you have conversations with AI models directly from your terminal, run one-shot questions, and pipe output into other tools — all without opening a browser.

It is built for developers and power users who want fast, scriptable access to large language models.

---

## 2. Installation

Install OpenClaw via **npm** (Node.js) or **pip** (Python):

\`\`\`bash
# npm (Node.js >= 18)
npm install -g openclaw

# pip (Python >= 3.9)
pip install openclaw
\`\`\`

Verify the installation:

\`\`\`bash
openclaw --version
\`\`\`

---

## 3. Configuration

OpenClaw reads its settings from environment variables or a config file (\`~/.openclaw/config.json\`).

### Environment Variables

\`\`\`bash
export OPENCLAW_API_KEY="your-api-key-here"
export OPENCLAW_BASE_URL="${baseUrl}"
\`\`\`

Add these lines to your \`~/.bashrc\`, \`~/.zshrc\`, or equivalent shell profile so they persist across sessions.

### Config File

Alternatively, run the interactive setup:

\`\`\`bash
openclaw config
\`\`\`

This will prompt you for your API key and base URL, then write:

\`\`\`json
{
  "api_key": "your-api-key-here",
  "base_url": "${baseUrl}",
  "model": "gpt-4o"
}
\`\`\`

> **Tip:** Your API key can be found on the **API Keys** page of this platform.

---

## 4. Usage

### Interactive Chat

Start a multi-turn conversation session:

\`\`\`bash
openclaw chat
\`\`\`

Type your messages and press Enter. Use \`/exit\` or \`Ctrl+C\` to quit.

**Useful flags:**

\`\`\`bash
openclaw chat --model gpt-4o-mini    # choose a specific model
openclaw chat --system "You are a helpful DevOps expert"
openclaw chat --temperature 0.2      # lower = more deterministic
\`\`\`

### One-Shot Questions

Ask a single question and get an answer — perfect for scripts:

\`\`\`bash
openclaw ask "Explain the difference between TCP and UDP in one sentence"

# Pipe input from another command
git diff HEAD~1 | openclaw ask "Summarize these changes"

# Save output to a file
openclaw ask "Write a Python hello-world script" > hello.py
\`\`\`

---

## 5. API Integration (Python)

Use the standard **openai** Python SDK pointed at this platform's base URL:

\`\`\`python
from openai import OpenAI

client = OpenAI(
    api_key="your-api-key-here",
    base_url="${baseUrl}",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user",   "content": "Hello! What can you do?"},
    ],
)

print(response.choices[0].message.content)
\`\`\`

Install the SDK if you haven't already:

\`\`\`bash
pip install openai
\`\`\`

---

## 6. cURL Example

Test the API directly from your terminal:

\`\`\`bash
curl ${baseUrl}/chat/completions \\
  -H "Authorization: Bearer your-api-key-here" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Say hello in three languages."}
    ]
  }'
\`\`\`

---

## 7. Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| **401 Unauthorized** | Invalid or missing API key | Check your key on the API Keys page |
| **429 Too Many Requests** | Rate limit exceeded | Wait a moment and retry; consider reducing request frequency |
| **Insufficient balance** | Account balance is zero | Top up your account on the Top Up page |
| **Connection refused** | Wrong base URL | Ensure \`base_url\` is set to \`${baseUrl}\` |
| **Model not found** | Invalid model name | Visit the Models page to see available models |

If issues persist, please contact the platform administrator.
`)

const markdownZh = computed(() => `
# OpenClaw 使用教程

## 1. 什么是 OpenClaw？

**OpenClaw** 是一款开源的命令行 AI 助手，可连接任何兼容 OpenAI 的 API 接口。它让你直接在终端中与 AI 模型对话、快速提问，并将输出传递给其他工具 —— 无需打开浏览器。

OpenClaw 专为需要快速、可脚本化访问大语言模型的开发者和高级用户设计。

---

## 2. 安装

通过 **npm**（Node.js）或 **pip**（Python）安装 OpenClaw：

\`\`\`bash
# npm（需要 Node.js >= 18）
npm install -g openclaw

# pip（需要 Python >= 3.9）
pip install openclaw
\`\`\`

验证安装：

\`\`\`bash
openclaw --version
\`\`\`

---

## 3. 配置

OpenClaw 从环境变量或配置文件（\`~/.openclaw/config.json\`）读取设置。

### 环境变量

\`\`\`bash
export OPENCLAW_API_KEY="你的API密钥"
export OPENCLAW_BASE_URL="${baseUrl}"
\`\`\`

将以上内容添加到 \`~/.bashrc\`、\`~/.zshrc\` 或对应的 Shell 配置文件中，使其在会话间持久生效。

### 配置文件

也可以运行交互式配置向导：

\`\`\`bash
openclaw config
\`\`\`

系统将提示你输入 API 密钥和 Base URL，然后生成以下配置文件：

\`\`\`json
{
  "api_key": "你的API密钥",
  "base_url": "${baseUrl}",
  "model": "gpt-4o"
}
\`\`\`

> **提示：** 你的 API 密钥可在本平台的 **API 密钥** 页面找到。

---

## 4. 使用方法

### 交互式对话

启动多轮对话会话：

\`\`\`bash
openclaw chat
\`\`\`

输入内容后按回车发送，使用 \`/exit\` 或 \`Ctrl+C\` 退出。

**常用参数：**

\`\`\`bash
openclaw chat --model gpt-4o-mini      # 指定模型
openclaw chat --system "你是一位专业的 DevOps 专家"
openclaw chat --temperature 0.2        # 数值越低结果越确定
\`\`\`

### 单次提问

提交单个问题并获取回答，非常适合在脚本中使用：

\`\`\`bash
openclaw ask "用一句话解释 TCP 和 UDP 的区别"

# 从其他命令获取输入
git diff HEAD~1 | openclaw ask "总结这些改动"

# 将输出保存到文件
openclaw ask "写一个 Python 的 Hello World 脚本" > hello.py
\`\`\`

---

## 5. API 集成（Python）

使用标准 **openai** Python SDK，将 base_url 指向本平台：

\`\`\`python
from openai import OpenAI

client = OpenAI(
    api_key="你的API密钥",
    base_url="${baseUrl}",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "你是一位乐于助人的助手。"},
        {"role": "user",   "content": "你好！你能做什么？"},
    ],
)

print(response.choices[0].message.content)
\`\`\`

安装 SDK（如尚未安装）：

\`\`\`bash
pip install openai
\`\`\`

---

## 6. cURL 示例

直接在终端测试 API：

\`\`\`bash
curl ${baseUrl}/chat/completions \\
  -H "Authorization: Bearer 你的API密钥" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "用三种语言打个招呼。"}
    ]
  }'
\`\`\`

---

## 7. 故障排查

| 错误 | 原因 | 解决方案 |
|------|------|----------|
| **401 Unauthorized** | API 密钥无效或缺失 | 在 API 密钥页面检查你的密钥 |
| **429 Too Many Requests** | 超出频率限制 | 稍等片刻后重试；考虑降低请求频率 |
| **余额不足** | 账户余额为零 | 在充值页面为账户充值 |
| **Connection refused** | Base URL 配置错误 | 确保 \`base_url\` 设置为 \`${baseUrl}\` |
| **Model not found** | 模型名称无效 | 访问模型目录页面查看可用模型列表 |

如果问题仍然存在，请联系平台管理员。
`)

const renderedContent = computed(() => {
  const md = locale.value === 'zh' ? markdownZh.value : markdownEn.value
  const html = marked(md) as string
  return DOMPurify.sanitize(html)
})
</script>
