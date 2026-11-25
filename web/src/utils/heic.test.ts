import { beforeEach, describe, expect, it, vi } from 'vitest'

const { isHeicMock, heicToMock } = vi.hoisted(() => ({
  isHeicMock: vi.fn(),
  heicToMock: vi.fn()
}))

vi.mock('heic-to/csp', () => ({
  isHeic: isHeicMock,
  heicTo: heicToMock
}))

import { decodeHeicBitmap } from './heic'

describe('decodeHeicBitmap', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not invoke the decoder for a non-HEIC file', async () => {
    const file = new File(['jpeg'], 'receipt.jpg', { type: 'image/jpeg' })
    isHeicMock.mockResolvedValue(false)

    await expect(decodeHeicBitmap(file)).resolves.toBeNull()
    expect(isHeicMock).toHaveBeenCalledWith(file)
    expect(heicToMock).not.toHaveBeenCalled()
  })

  it('decodes HEIC to an orientation-aware bitmap for canvas compression', async () => {
    const file = new File(['heic'], 'IMG_1024.HEIC', { type: 'image/heic' })
    const bitmap = { width: 4032, height: 3024 } as ImageBitmap
    isHeicMock.mockResolvedValue(true)
    heicToMock.mockResolvedValue(bitmap)

    await expect(decodeHeicBitmap(file)).resolves.toBe(bitmap)
    expect(heicToMock).toHaveBeenCalledWith({
      blob: file,
      type: 'bitmap',
      options: { imageOrientation: 'from-image' }
    })
  })
})
