import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { ref, nextTick } from 'vue'
import { useModalA11y } from './useModalA11y'

describe('useModalA11y composable', () => {
  let container: HTMLDivElement
  let triggerBtn: HTMLButtonElement
  let modal: HTMLDivElement
  let firstBtn: HTMLButtonElement
  let secondBtn: HTMLButtonElement

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)

    triggerBtn = document.createElement('button')
    triggerBtn.textContent = 'Open'
    container.appendChild(triggerBtn)

    modal = document.createElement('div')
    modal.tabIndex = -1
    firstBtn = document.createElement('button')
    firstBtn.textContent = 'First'
    secondBtn = document.createElement('button')
    secondBtn.textContent = 'Second'

    modal.appendChild(firstBtn)
    modal.appendChild(secondBtn)
    container.appendChild(modal)
  })

  afterEach(() => {
    document.body.removeChild(container)
  })

  it('calls onClose when Escape key is pressed', async () => {
    const isOpen = ref(false)
    const modalRef = ref<HTMLElement | null>(modal)
    const onClose = vi.fn()

    useModalA11y({
      isOpen,
      modalRef,
      onClose
    })

    isOpen.value = true
    await nextTick()

    const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true })
    document.dispatchEvent(escapeEvent)

    expect(onClose).toHaveBeenCalled()
  })

  it('restores focus to previous active element on close', async () => {
    triggerBtn.focus()
    expect(document.activeElement).toBe(triggerBtn)

    const isOpen = ref(false)
    const modalRef = ref<HTMLElement | null>(modal)

    useModalA11y({
      isOpen,
      modalRef
    })

    isOpen.value = true
    await nextTick()

    // simulate close
    isOpen.value = false
    await nextTick()

    expect(document.activeElement).toBe(triggerBtn)
  })

  it('traps focus inside the modal with Tab and Shift+Tab', async () => {
    const isOpen = ref(false)
    const modalRef = ref<HTMLElement | null>(modal)

    useModalA11y({
      isOpen,
      modalRef
    })

    isOpen.value = true
    await nextTick()

    secondBtn.focus()
    expect(document.activeElement).toBe(secondBtn)

    // Tab from last element loops to first
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    document.dispatchEvent(tabEvent)
    expect(document.activeElement).toBe(firstBtn)

    // Shift+Tab from first element loops to last
    const shiftTabEvent = new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true })
    document.dispatchEvent(shiftTabEvent)
    expect(document.activeElement).toBe(secondBtn)
  })
})
