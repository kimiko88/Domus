import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import TemplateEntitiesView from './TemplateEntitiesView.vue'
import { useTemplatesStore } from '@/stores/templates'

describe('TemplateEntitiesView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('renders template view header and playground live sandbox', async () => {
    const store = useTemplatesStore()
    vi.spyOn(store, 'fetchTemplates').mockResolvedValue([])

    store.templates = [
      {
        id: 'tmpl-1',
        name: 'Potenza Totale Solare',
        entity_id: 'sensor.solare_totale',
        domain: 'sensor',
        state_template: "{{ states('sensor.inverter_1') | float + states('sensor.inverter_2') | float }}",
        unit_of_measurement: 'W',
        device_class: 'power',
        source_entity_ids: ['sensor.inverter_1', 'sensor.inverter_2'],
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ]

    const wrapper = mount(TemplateEntitiesView)

    expect(wrapper.text()).toContain('Entità Template & Sensori Virtuali')
    expect(wrapper.find('textarea.template-editor').exists()).toBe(true)
    expect(wrapper.text()).toContain('Potenza Totale Solare')
    expect(wrapper.text()).toContain('sensor.solare_totale')
    expect(wrapper.text()).toContain('Unità: W')
  })

  it('evaluates template preview when button is clicked', async () => {
    const store = useTemplatesStore()
    vi.spyOn(store, 'fetchTemplates').mockResolvedValue([])
    vi.spyOn(store, 'previewTemplate').mockResolvedValue('3450.5 W')

    const wrapper = mount(TemplateEntitiesView)

    const btn = wrapper.find('.playground-header button')
    await btn.trigger('click')

    expect(store.previewTemplate).toHaveBeenCalled()
    await wrapper.vm.$nextTick()
    expect(wrapper.text()).toContain('3450.5 W')
  })
})
