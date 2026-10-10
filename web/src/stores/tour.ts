import { defineStore } from 'pinia'
import { ref } from 'vue'

export const STORAGE_TOUR_COMPLETED = 'domus_tour_completed'

export const useTourStore = defineStore('tour', () => {
  const isOpen = ref(false)
  const currentStep = ref(0)
  const totalSteps = ref(9)
  const isCompleted = ref(localStorage.getItem(STORAGE_TOUR_COMPLETED) === 'true')

  function startTour() {
    currentStep.value = 0
    isOpen.value = true
  }

  function nextStep() {
    if (currentStep.value < totalSteps.value - 1) {
      currentStep.value++
    } else {
      completeTour()
    }
  }

  function prevStep() {
    if (currentStep.value > 0) {
      currentStep.value--
    }
  }

  function goToStep(stepIndex: number) {
    if (stepIndex >= 0 && stepIndex < totalSteps.value) {
      currentStep.value = stepIndex
    }
  }

  function skipTour() {
    isOpen.value = false
    isCompleted.value = true
    try {
      localStorage.setItem(STORAGE_TOUR_COMPLETED, 'true')
    } catch {
      // ignore in constrained environments
    }
  }

  function completeTour() {
    isOpen.value = false
    isCompleted.value = true
    try {
      localStorage.setItem(STORAGE_TOUR_COMPLETED, 'true')
    } catch {
      // ignore
    }
  }

  function resetTour() {
    try {
      localStorage.removeItem(STORAGE_TOUR_COMPLETED)
    } catch {
      // ignore
    }
    isCompleted.value = false
    startTour()
  }

  function checkAutoStart() {
    try {
      const done = localStorage.getItem(STORAGE_TOUR_COMPLETED)
      if (done !== 'true') {
        startTour()
      }
    } catch {
      // ignore
    }
  }

  return {
    isOpen,
    currentStep,
    totalSteps,
    isCompleted,
    startTour,
    nextStep,
    prevStep,
    goToStep,
    skipTour,
    completeTour,
    resetTour,
    checkAutoStart
  }
})
