import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AppSidebar from './AppSidebar.vue'
import { i18n } from '@/locales'

describe('AppSidebar.vue', () => {
  it('renders branding and main navigation items', () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/'
      },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a :href="to" class="router-link"><slot /></a>',
            props: ['to']
          }
        }
      }
    })

    expect(wrapper.text()).toContain('Domus')
    expect(wrapper.text()).toContain('Dashboard')
    expect(wrapper.text()).toContain('Aree')
    expect(wrapper.text()).toContain('Dispositivi')
    expect(wrapper.text()).toContain('Integrazioni')
    expect(wrapper.text()).toContain('Automazioni')
    expect(wrapper.text()).toContain('Scene')
    expect(wrapper.text()).toContain('Storico')
    expect(wrapper.text()).toContain('Impostazioni')
  })

  it('marks current route as active', () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/areas'
      },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a :href="to" class="router-link" :class="{ \'is-active\': $attrs.class?.includes(\'active\') }"><slot /></a>',
            props: ['to']
          }
        }
      }
    })

    const activeLink = wrapper.find('.nav-item.active')
    expect(activeLink.exists()).toBe(true)
    expect(activeLink.text()).toContain('Aree')
  })

  it('emits close event when backdrop is clicked on mobile', async () => {
    const wrapper = mount(AppSidebar, {
      props: {
        isOpen: true,
        currentPath: '/'
      },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>',
            props: ['to']
          }
        }
      }
    })

    const backdrop = wrapper.find('.sidebar-backdrop')
    expect(backdrop.exists()).toBe(true)
    await backdrop.trigger('click')

    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('renders all 5 macro-sections with counts', () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/'
      },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a :href="to"><slot /></a>',
            props: ['to']
          }
        }
      }
    })

    const macroSections = wrapper.findAll('.macro-section')
    expect(macroSections).toHaveLength(5)

    const headers = wrapper.findAll('.macro-section-header')
    expect(headers).toHaveLength(5)
  })

  it('toggles section accordion on header click', async () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/'
      },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a :href="to"><slot /></a>',
            props: ['to']
          }
        }
      }
    })

    // Find the first macro section button
    const firstHeader = wrapper.find('.macro-section-header')
    const initialAria = firstHeader.attributes('aria-expanded')

    await firstHeader.trigger('click')
    expect(firstHeader.attributes('aria-expanded')).not.toBe(initialAria)
  })

  it('auto-expands the macro-section containing the active path', () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/automations'
      },
      global: {
        plugins: [i18n],
        stubs: {
          RouterLink: {
            template: '<a :href="to"><slot /></a>',
            props: ['to']
          }
        }
      }
    })

    // The automations macro-section should have 'has-active' and 'is-expanded' class
    const activeMacro = wrapper.find('.macro-section.has-active')
    expect(activeMacro.exists()).toBe(true)
    expect(activeMacro.classes()).toContain('is-expanded')
  })
})
