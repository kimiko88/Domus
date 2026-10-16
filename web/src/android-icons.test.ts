import { describe, it, expect } from 'vitest'
import fs from 'node:fs'
import path from 'node:path'

describe('Android App Icons & Adaptive Icon Configuration', () => {
  const projectRoot = path.resolve(__dirname, '../../')
  const androidResDir = path.join(projectRoot, 'mobile/android/app/src/main/res')
  const playStoreIconPath = path.join(projectRoot, 'mobile/android/app/src/main/ic_launcher-web.png')

  it('contains Google Play Store 512x512 asset', () => {
    expect(fs.existsSync(playStoreIconPath)).toBe(true)
    const stats = fs.statSync(playStoreIconPath)
    expect(stats.size).toBeGreaterThan(10000)
  })

  it('contains adaptive icon definitions in mipmap-anydpi-v26', () => {
    const anydpiDir = path.join(androidResDir, 'mipmap-anydpi-v26')
    const icLauncherXml = path.join(anydpiDir, 'ic_launcher.xml')
    const icLauncherRoundXml = path.join(anydpiDir, 'ic_launcher_round.xml')

    expect(fs.existsSync(icLauncherXml)).toBe(true)
    expect(fs.existsSync(icLauncherRoundXml)).toBe(true)

    const launcherContent = fs.readFileSync(icLauncherXml, 'utf-8')
    expect(launcherContent).toContain('android:drawable="@drawable/ic_launcher_background"')
    expect(launcherContent).toContain('android:drawable="@mipmap/ic_launcher_foreground"')
    expect(launcherContent).toContain('android:drawable="@drawable/ic_launcher_monochrome"')

    const roundContent = fs.readFileSync(icLauncherRoundXml, 'utf-8')
    expect(roundContent).toContain('android:drawable="@drawable/ic_launcher_background"')
    expect(roundContent).toContain('android:drawable="@mipmap/ic_launcher_foreground"')
    expect(roundContent).toContain('android:drawable="@drawable/ic_launcher_monochrome"')
  })

  it('contains background, monochrome, and notification vector drawables', () => {
    const drawableDir = path.join(androidResDir, 'drawable')
    const bgXml = path.join(drawableDir, 'ic_launcher_background.xml')
    const monoXml = path.join(drawableDir, 'ic_launcher_monochrome.xml')
    const notifXml = path.join(drawableDir, 'ic_stat_domus.xml')

    expect(fs.existsSync(bgXml)).toBe(true)
    expect(fs.existsSync(monoXml)).toBe(true)
    expect(fs.existsSync(notifXml)).toBe(true)

    const bgContent = fs.readFileSync(bgXml, 'utf-8')
    expect(bgContent).toContain('#0B1120')

    const monoContent = fs.readFileSync(monoXml, 'utf-8')
    expect(monoContent).toContain('#FFFFFF')
  })

  it('contains raster mipmaps for all standard Android densities without webp duplicates', () => {
    const densities = ['mdpi', 'hdpi', 'xhdpi', 'xxhdpi', 'xxxhdpi']
    const iconNames = ['ic_launcher.png', 'ic_launcher_round.png', 'ic_launcher_foreground.png']

    for (const density of densities) {
      const folder = path.join(androidResDir, `mipmap-${density}`)
      expect(fs.existsSync(folder), `Missing directory for mipmap-${density}`).toBe(true)

      // Ensure no legacy webp template files remain
      const filesInFolder = fs.readdirSync(folder)
      const webpFiles = filesInFolder.filter((f) => f.endsWith('.webp'))
      expect(webpFiles.length, `Found unexpected webp files in mipmap-${density}: ${webpFiles}`).toBe(0)

      for (const icon of iconNames) {
        const iconPath = path.join(folder, icon)
        expect(fs.existsSync(iconPath), `Missing ${icon} in mipmap-${density}`).toBe(true)
        const stats = fs.statSync(iconPath)
        expect(stats.size).toBeGreaterThan(1000)
      }
    }
  })
})
