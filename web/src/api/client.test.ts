import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { apiClient, ApiError } from './client'

describe('ApiClient', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('performs GET request and parses JSON response successfully', async () => {
    const mockData = { id: 'usr-1', username: 'admin', role: 'admin' }
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockData
    })

    const data = await apiClient.get<typeof mockData>('/api/v1/auth/me')
    expect(data).toEqual(mockData)
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/auth/me',
      expect.objectContaining({
        method: 'GET',
        credentials: 'include',
        headers: expect.objectContaining({
          Accept: 'application/json'
        })
      })
    )
  })

  it('performs POST request with body and headers', async () => {
    const requestBody = { username: 'admin', password: 'password123' }
    const mockResponse = { user: { id: 'usr-1', username: 'admin' } }

    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => mockResponse
    })

    const data = await apiClient.post('/api/v1/auth/login', requestBody)
    expect(data).toEqual(mockResponse)
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/auth/login',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify(requestBody),
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
          Accept: 'application/json'
        })
      })
    )
  })

  it('throws ApiError with server message when response is not ok', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      headers: new Headers({ 'content-type': 'application/json' }),
      json: async () => ({ error: 'Invalid credentials' })
    })

    await expect(apiClient.post('/api/v1/auth/login', {})).rejects.toThrow(ApiError)
    await expect(apiClient.post('/api/v1/auth/login', {})).rejects.toMatchObject({
      status: 401,
      message: 'Invalid credentials'
    })
  })

  it('handles DELETE request correctly', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 204,
      headers: new Headers(),
      text: async () => ''
    })

    await apiClient.delete('/api/v1/entities/light.kitchen')
    expect(global.fetch).toHaveBeenCalledWith(
      '/api/v1/entities/light.kitchen',
      expect.objectContaining({
        method: 'DELETE',
        credentials: 'include'
      })
    )
  })
})
