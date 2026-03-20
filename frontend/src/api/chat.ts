export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface ChatRequest {
  model: string
  messages: ChatMessage[]
  stream?: boolean
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

/**
 * Fetch available chat models
 */
export async function getModels(): Promise<{ models: string[] }> {
  const token = localStorage.getItem('auth_token')
  const res = await fetch(`${API_BASE_URL}/chat/models`, {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  })
  if (!res.ok) {
    throw new Error(`Failed to fetch models: ${res.status}`)
  }
  const json = await res.json()
  // Unwrap standard API response if present
  if (json && typeof json === 'object' && 'code' in json) {
    if (json.code === 0) return json.data
    throw new Error(json.message || 'Failed to fetch models')
  }
  return json
}

/**
 * Stream a chat completion using SSE via fetch + ReadableStream.
 */
export function streamChat(
  request: ChatRequest,
  onChunk: (text: string) => void,
  onDone: () => void,
  onError: (err: Error) => void
): AbortController {
  const controller = new AbortController()
  const token = localStorage.getItem('auth_token')

  const body: ChatRequest = {
    ...request,
    stream: true
  }

  fetch(`${API_BASE_URL}/chat/completions`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify(body),
    signal: controller.signal
  })
    .then(async (res) => {
      if (!res.ok) {
        let msg = `HTTP ${res.status}`
        try {
          const errJson = await res.json()
          msg = errJson?.message || errJson?.detail || msg
        } catch {
          // ignore parse errors
        }
        throw new Error(msg)
      }

      const reader = res.body?.getReader()
      if (!reader) {
        throw new Error('No response body')
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        // Keep the last (possibly incomplete) line in the buffer
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed || trimmed.startsWith(':')) continue // skip empty lines and comments
          if (trimmed.startsWith('data: ')) {
            const data = trimmed.slice(6)
            if (data === '[DONE]') {
              onDone()
              return
            }
            try {
              const parsed = JSON.parse(data)
              const content = parsed.choices?.[0]?.delta?.content
              if (content) {
                onChunk(content)
              }
            } catch {
              // skip unparseable lines
            }
          }
        }
      }

      // Process any remaining buffer
      if (buffer.trim()) {
        const trimmed = buffer.trim()
        if (trimmed.startsWith('data: ')) {
          const data = trimmed.slice(6)
          if (data !== '[DONE]') {
            try {
              const parsed = JSON.parse(data)
              const content = parsed.choices?.[0]?.delta?.content
              if (content) {
                onChunk(content)
              }
            } catch {
              // ignore
            }
          }
        }
      }

      onDone()
    })
    .catch((err) => {
      if (err.name === 'AbortError') return
      onError(err instanceof Error ? err : new Error(String(err)))
    })

  return controller
}
