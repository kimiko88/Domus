import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'

describe('PWA Icons & Manifest Configuration', () => {
  const publicDir = path.resolve(__dirname, '../public')
  const iconsDir = path.join(publicDir, 'icons')
  const manifestPath = path.join(publicDir, 'manifest.json')

  it('contains valid manifest.json with all standard and maskable icons', () => {
    expect(fs.existsSync(manifestPath)).toBe(true)
    const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf-8'))

    expect(manifest.name).toBe('Domus Smart Home')
    expect(manifest.short_name).toBe('Domus')
    expect(manifest.display).toBe('standalone')
    expect(Array.isArray(manifest.icons)).toBe(true)

    // Check essential sizes
    const sizes = manifest.icons.map((i: any) => i.sizes)
    expect(sizes).toContain('192x192')
    expect(sizes).toContain('512x512')

    // Check maskable icons
    const maskable = manifest.icons.filter((i: any) => i.purpose === 'maskable')
    expect(maskable.length).toBeGreaterThanOrEqual(2)
  })

  it('contains all required icon files in public/icons/', () => {
    const requiredIcons = [
      'icon-72x72.png',
      'icon-96x96.png',
      'icon-128x128.png',
      'icon-144x144.png',
      'icon-152x152.png',
      'icon-192x192.png',
      'icon-384x384.png',
      'icon-512x512.png',
      'icon-maskable-192x192.png',
      'icon-maskable-512x512.png'
    ]

    for (const icon of requiredIcons) {
      const p = path.join(iconsDir, icon)
      expect(fs.existsSync(p), `Missing icon: ${icon}`).toBe(true)
      const stats = fs.statSync(p)
      expect(stats.size).toBeGreaterThan(1000) // Ensure not empty
    }
  })

  it('contains all Apple Touch icons, favicons, and root PWA fallbacks in public/', () => {
    const requiredFiles = [
      'favicon.svg',
      'favicon.ico',
      'favicon-16x16.png',
      'favicon-32x32.png',
      'apple-touch-icon.png',
      'apple-touch-icon-180x180.png',
      'pwa-192x192.png',
      'pwa-512x512.png'
    ]

    for (const file of requiredFiles) {
      const p = path.join(publicDir, file)
      expect(fs.existsSync(p), `Missing file: ${file}`).toBe(true)
      const stats = fs.statSync(p)
      expect(stats.size).toBeGreaterThan(100)
    }
  })
})
