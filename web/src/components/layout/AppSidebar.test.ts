import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import AppSidebar from './AppSidebar.vue'

describe('AppSidebar.vue', () => {
  it('renders branding and main navigation items', () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/'
      },
      global: {
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
    expect(wrapper.text()).toContain('Impostazioni')
  })

  it('marks current route as active', () => {
    const wrapper = mount(AppSidebar, {
      props: {
        currentPath: '/areas'
      },
      global: {
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
})
