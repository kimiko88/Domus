import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TwoFactorModal from './TwoFactorModal.vue'
import { apiClient } from '@/api/client'

describe('TwoFactorModal.vue', () => {
  it('renders QR code svg, secret key and 6-digit input', () => {
    const wrapper = mount(TwoFactorModal, {
      props: {
        secret: 'JBSWY3DPEHPK3PXP',
        qrUrl: 'otpauth://totp/Domus:admin?secret=JBSWY3DPEHPK3PXP&issuer=Domus',
      },
    })

    expect(wrapper.text()).toContain('Autenticazione a Due Fattori (2FA / TOTP)')
    expect(wrapper.text()).toContain('JBSWY3DPEHPK3PXP')
    expect(wrapper.find('svg.qr-svg').exists()).toBe(true)
    expect(wrapper.find('input#totp-code').exists()).toBe(true)
  })

  it('submits code and transitions to backup codes success step', async () => {
    vi.spyOn(apiClient, 'post').mockResolvedValue({
      status: 'enabled',
      backup_codes: ['1234-5678', '8765-4321'],
    })

    const wrapper = mount(TwoFactorModal, {
      props: {
        secret: 'JBSWY3DPEHPK3PXP',
      },
    })

    const input = wrapper.find('input#totp-code')
    await input.setValue('123456')

    const form = wrapper.find('form.verify-form')
    await form.trigger('submit')

    await wrapper.vm.$nextTick()
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('2FA Attivata con Successo!')
    expect(wrapper.text()).toContain('1234-5678')
    expect(wrapper.text()).toContain('8765-4321')
    expect(wrapper.emitted('enabled')).toBeTruthy()
  })
})
