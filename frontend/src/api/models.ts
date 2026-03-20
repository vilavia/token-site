import { apiClient } from './client'

export interface ModelInfo {
  id: string
  display_name: string
  provider: string
  context_window: number
  input_price_mtok: number
  output_price_mtok: number
  input_formats: string[]
  output_formats: string[]
  description: string
  max_output_tokens: number
}

export const modelsAPI = {
  list() {
    return apiClient.get<{ models: ModelInfo[] }>('/models')
  },
  getById(id: string) {
    return apiClient.get<ModelInfo>(`/models/${encodeURIComponent(id)}`)
  }
}
