import { watch, onUnmounted, nextTick, getCurrentInstance, type Ref } from 'vue'

export interface UseModalA11yOptions {
  isOpen: Ref<boolean>
  modalRef: Ref<HTMLElement | null>
  onClose?: () => void
  initialFocusRef?: Ref<HTMLElement | null>
  restoreFocus?: boolean
}

export function useModalA11y(options: UseModalA11yOptions) {
  const { isOpen, modalRef, onClose, initialFocusRef, restoreFocus = true } = options
  let previousActiveElement: HTMLElement | null = null

  const FOCUSABLE_SELECTOR = [
    'a[href]',
    'button:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])'
  ].join(', ')

  function getFocusableElements(): HTMLElement[] {
    if (!modalRef.value) return []
    return Array.from(modalRef.value.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
      .filter((el) => el.offsetParent !== null || el.offsetWidth > 0 || el.offsetHeight > 0 || el.tagName === 'BUTTON' || el.tagName === 'INPUT')
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!isOpen.value || !modalRef.value) return

    if (event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      onClose?.()
      return
    }

    if (event.key === 'Tab') {
      const focusables = getFocusableElements()
      if (focusables.length === 0) {
        event.preventDefault()
        return
      }

      const firstElement = focusables[0]
      const lastElement = focusables[focusables.length - 1]

      if (event.shiftKey) {
        if (document.activeElement === firstElement || document.activeElement === modalRef.value) {
          event.preventDefault()
          lastElement.focus()
        }
      } else {
        if (document.activeElement === lastElement) {
          event.preventDefault()
          firstElement.focus()
        }
      }
    }
  }

  watch(
    isOpen,
    async (open) => {
      if (typeof window === 'undefined') return

      if (open) {
        previousActiveElement = document.activeElement as HTMLElement | null
        document.addEventListener('keydown', handleKeydown, true)

        await nextTick()
        if (initialFocusRef?.value) {
          initialFocusRef.value.focus()
        } else {
          const focusables = getFocusableElements()
          if (focusables.length > 0) {
            focusables[0].focus()
          } else if (modalRef.value) {
            modalRef.value.focus()
          }
        }
      } else {
        document.removeEventListener('keydown', handleKeydown, true)
        if (restoreFocus && previousActiveElement && typeof previousActiveElement.focus === 'function') {
          previousActiveElement.focus()
        }
        previousActiveElement = null
      }
    },
    { immediate: true }
  )

  if (getCurrentInstance()) {
    onUnmounted(() => {
      if (typeof window !== 'undefined') {
        document.removeEventListener('keydown', handleKeydown, true)
      }
    })
  }
}

