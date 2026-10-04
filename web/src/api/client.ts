export class ApiError extends Error {
  status: number
  data?: any

  constructor(status: number, message: string, data?: any) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

export interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>
}

class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl
  }

  private buildUrl(path: string, params?: Record<string, string | number | boolean | undefined>): string {
    const url = new URL(path, window?.location?.origin || 'http://localhost')
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined) {
          url.searchParams.append(key, String(value))
        }
      })
    }
    return url.pathname + url.search
  }

  async request<T>(path: string, options: RequestOptions = {}): Promise<T> {
    const { params, headers, ...customConfig } = options
    const url = this.buildUrl(path, params)

    const config: RequestInit = {
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        ...headers
      },
      ...customConfig
    }

    const response = await fetch(url, config)

    if (!response.ok) {
      let errorMessage = `HTTP ${response.status} ${response.statusText}`
      let errorData: any = null
      try {
        errorData = await response.json()
        if (errorData?.error) {
          errorMessage = errorData.error
        } else if (errorData?.message) {
          errorMessage = errorData.message
        }
      } catch {
        // Fallback to text or status text
      }
      throw new ApiError(response.status, errorMessage, errorData)
    }

    if (response.status === 204) {
      return {} as T
    }

    const contentType = response.headers.get('content-type')
    if (contentType && contentType.includes('application/json')) {
      return (await response.json()) as T
    }

    return (await response.text()) as unknown as T
  }

  get<T>(path: string, params?: Record<string, any>, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, { method: 'GET', params, ...options })
  }

  post<T>(path: string, body?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers
      },
      ...options
    })
  }

  put<T>(path: string, body?: any, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers
      },
      ...options
    })
  }

  delete<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>(path, { method: 'DELETE', ...options })
  }
}

export const apiClient = new ApiClient()
