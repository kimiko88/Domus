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
  /** Override the per-request timeout in milliseconds. Pass 0 to disable. Default: 30 000. */
  timeoutMs?: number
}

const DEFAULT_TIMEOUT_MS = 30_000

class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl
  }

  private buildUrl(
    path: string,
    params?: Record<string, string | number | boolean | undefined>
  ): string {
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
    const {
      params,
      headers,
      signal: externalSignal,
      timeoutMs = DEFAULT_TIMEOUT_MS,
      ...customConfig
    } = options
    const url = this.buildUrl(path, params)

    const controller = new AbortController()
    let timer: ReturnType<typeof setTimeout> | undefined

    if (timeoutMs > 0) {
      timer = setTimeout(() => controller.abort(), timeoutMs)
    }

    // Propagate an external abort signal (e.g. from a component unmount)
    const onExternalAbort = () => controller.abort()
    if (externalSignal) {
      externalSignal.addEventListener('abort', onExternalAbort)
    }

    const requestHeaders: Record<string, string> = {
      Accept: 'application/json',
      ...((headers as Record<string, string>) || {})
    }

    // Auto-inject CSRF protection header from cookie if present
    if (typeof document !== 'undefined' && !requestHeaders['X-CSRF-Token']) {
      const match = document.cookie.match(/(?:^|;\s*)domus_csrf=([^;]+)/)
      if (match) {
        requestHeaders['X-CSRF-Token'] = decodeURIComponent(match[1])
      }
    }

    const config: RequestInit = {
      credentials: 'include',
      headers: requestHeaders,
      signal: controller.signal,
      ...customConfig
    }

    try {
      const response = await fetch(url, config)

      if (!response.ok) {
        if (response.status === 401 && typeof window !== 'undefined') {
          window.dispatchEvent(new CustomEvent('domus:unauthorized'))
        }
        let errorMessage = `HTTP ${response.status}: ${response.statusText}`
        let errorData: any = null
        try {
          errorData = await response.json()
          if (errorData?.error) {
            errorMessage = errorData.error
          } else if (errorData?.message) {
            errorMessage = errorData.message
          }
        } catch {
          // Fallback to status text
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
    } catch (err: any) {
      if (err?.name === 'AbortError') {
        throw new ApiError(408, 'Request timed out')
      }
      throw err
    } finally {
      if (timer !== undefined) clearTimeout(timer)
      if (externalSignal) {
        externalSignal.removeEventListener('abort', onExternalAbort)
      }
    }
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
