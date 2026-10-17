/**
 * Procedural Web Audio API sound synthesizer for tactile feedback, alarm chimes, and confirmations.
 * Zero external MP3/WAV assets required.
 */
class DomusAudioSynthesizer {
  private ctx: AudioContext | null = null
  private enabled: boolean = true

  constructor() {
    // Check if user prefers reduced motion or muted sounds
    if (typeof window !== 'undefined') {
      const prefersReduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      this.enabled = !prefersReduced
    }
  }

  private initContext(): AudioContext | null {
    if (typeof window === 'undefined') return null
    if (!this.ctx) {
      const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
      if (AudioContextClass) {
        this.ctx = new AudioContextClass()
      }
    }
    if (this.ctx && this.ctx.state === 'suspended') {
      this.ctx.resume()
    }
    return this.ctx
  }

  public setEnabled(enabled: boolean) {
    this.enabled = enabled
  }

  public isEnabled(): boolean {
    return this.enabled
  }

  /**
   * Subtle tactile click sound when toggling a switch or button.
   */
  public playClick(pitch: number = 800) {
    if (!this.enabled) return
    const ctx = this.initContext()
    if (!ctx) return

    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'sine'
    osc.frequency.setValueAtTime(pitch, ctx.currentTime)
    osc.frequency.exponentialRampToValueAtTime(pitch * 0.4, ctx.currentTime + 0.04)

    gain.gain.setValueAtTime(0.08, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.04)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start()
    osc.stop(ctx.currentTime + 0.04)
  }

  /**
   * Harmonic pleasant chime on scene activation or successful action.
   */
  public playSuccessChime() {
    if (!this.enabled) return
    const ctx = this.initContext()
    if (!ctx) return

    const notes = [523.25, 659.25, 783.99] // C5, E5, G5 triad
    notes.forEach((freq, idx) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()

      osc.type = 'triangle'
      osc.frequency.setValueAtTime(freq, ctx.currentTime + idx * 0.06)

      gain.gain.setValueAtTime(0.1, ctx.currentTime + idx * 0.06)
      gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + idx * 0.06 + 0.25)

      osc.connect(gain)
      gain.connect(ctx.destination)

      osc.start(ctx.currentTime + idx * 0.06)
      osc.stop(ctx.currentTime + idx * 0.06 + 0.25)
    })
  }

  /**
   * Warning / countdown beep for entry/exit delay.
   */
  public playAlarmBeep(isFinalWarning: boolean = false) {
    if (!this.enabled) return
    const ctx = this.initContext()
    if (!ctx) return

    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    const freq = isFinalWarning ? 1200 : 880
    const duration = isFinalWarning ? 0.08 : 0.12

    osc.type = 'square'
    osc.frequency.setValueAtTime(freq, ctx.currentTime)

    gain.gain.setValueAtTime(0.12, ctx.currentTime)
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + duration)

    osc.connect(gain)
    gain.connect(ctx.destination)

    osc.start()
    osc.stop(ctx.currentTime + duration)
  }
}

export const audioSynthesizer = new DomusAudioSynthesizer()
export const soundEffects = audioSynthesizer

