import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTourStore, STORAGE_TOUR_COMPLETED } from './tour'

describe('useTourStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
  })

  it('initializes with default values', () => {
    const store = useTourStore()
    expect(store.isOpen).toBe(false)
    expect(store.currentStep).toBe(0)
    expect(store.totalSteps).toBe(9)
    expect(store.isCompleted).toBe(false)
  })

  it('starts the tour from step 0', () => {
    const store = useTourStore()
    store.currentStep = 4
    store.startTour()
    expect(store.isOpen).toBe(true)
    expect(store.currentStep).toBe(0)
  })

  it('steps forward and backward through tour', () => {
    const store = useTourStore()
    store.startTour()
    expect(store.currentStep).toBe(0)

    store.nextStep()
    expect(store.currentStep).toBe(1)

    store.prevStep()
    expect(store.currentStep).toBe(0)

    // Cannot step below 0
    store.prevStep()
    expect(store.currentStep).toBe(0)
  })

  it('completes tour on last step', () => {
    const store = useTourStore()
    store.startTour()
    store.goToStep(8)
    expect(store.currentStep).toBe(8)

    store.nextStep()
    expect(store.isOpen).toBe(false)
    expect(store.isCompleted).toBe(true)
    expect(localStorage.getItem(STORAGE_TOUR_COMPLETED)).toBe('true')
  })

  it('skips tour and persists completion flag', () => {
    const store = useTourStore()
    store.startTour()
    store.skipTour()
    expect(store.isOpen).toBe(false)
    expect(store.isCompleted).toBe(true)
    expect(localStorage.getItem(STORAGE_TOUR_COMPLETED)).toBe('true')
  })

  it('resets tour and clears completion flag', () => {
    localStorage.setItem(STORAGE_TOUR_COMPLETED, 'true')
    const store = useTourStore()
    expect(store.isCompleted).toBe(true)

    store.resetTour()
    expect(store.isOpen).toBe(true)
    expect(store.currentStep).toBe(0)
    expect(store.isCompleted).toBe(false)
    expect(localStorage.getItem(STORAGE_TOUR_COMPLETED)).toBeNull()
  })

  it('auto-starts tour if not previously completed', () => {
    const store = useTourStore()
    store.checkAutoStart()
    expect(store.isOpen).toBe(true)
  })

  it('does not auto-start if already completed in localStorage', () => {
    localStorage.setItem(STORAGE_TOUR_COMPLETED, 'true')
    const store = useTourStore()
    store.checkAutoStart()
    expect(store.isOpen).toBe(false)
  })
})
