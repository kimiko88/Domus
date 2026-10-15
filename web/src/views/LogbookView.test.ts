import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import LogbookView from './LogbookView.vue'
import { useLogbookStore } from '@/stores/logbook'

describe('LogbookView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
    const store = useLogbookStore()
    vi.spyOn(store, 'fetchEntries').mockResolvedValue([])
  })

  it('renders logbook header, search input, and domain chips', async () => {
    const wrapper = mount(LogbookView)

    expect(wrapper.text()).toContain('Registro Eventi (Logbook)')
    expect(wrapper.find('input[type="search"]').exists()).toBe(true)
    expect(wrapper.findAll('.chip').length).toBeGreaterThan(3)
  })

  it('renders timeline entries with entity badges and context attribution', async () => {
    const store = useLogbookStore()
    store.entries = [
      {
        id: 'entry-1',
        entity_id: 'light.living_room',
        entity_name: 'Luce Salotto',
        domain: 'light',
        state: 'on',
        message: 'Luce Salotto accesa da Automazione Rientro',
        context_type: 'automation',
        context_name: 'Rientro Casa',
        timestamp: new Date().toISOString(),
      },
      {
        id: 'entry-2',
        entity_id: 'person.mario',
        entity_name: 'Mario Rossi',
        domain: 'person',
        state: 'home',
        message: 'Mario Rossi è arrivato a Casa',
        context_type: 'user',
        context_name: 'Mario Rossi',
        timestamp: new Date().toISOString(),
      },
    ]

    const wrapper = mount(LogbookView)

    expect(wrapper.findAll('.timeline-item')).toHaveLength(2)
    expect(wrapper.text()).toContain('Luce Salotto')
    expect(wrapper.text()).toContain('Innescato da "Rientro Casa"')
    expect(wrapper.text()).toContain('Mario Rossi')
    expect(wrapper.text()).toContain('Azionato da Mario Rossi')
  })

  it('filters entries when search query is entered', async () => {
    const store = useLogbookStore()
    store.entries = [
      {
        id: 'entry-1',
        entity_id: 'light.kitchen',
        entity_name: 'Luce Cucina',
        domain: 'light',
        state: 'on',
        message: 'Luce Cucina accesa',
        context_type: 'system',
        timestamp: new Date().toISOString(),
      },
      {
        id: 'entry-2',
        entity_id: 'switch.coffee',
        entity_name: 'Macchina Caffè',
        domain: 'switch',
        state: 'on',
        message: 'Macchina Caffè accesa',
        context_type: 'user',
        timestamp: new Date().toISOString(),
      },
    ]

    const wrapper = mount(LogbookView)
    expect(wrapper.findAll('.timeline-item')).toHaveLength(2)

    const search = wrapper.find('input[type="search"]')
    await search.setValue('Caffè')

    expect(wrapper.findAll('.timeline-item')).toHaveLength(1)
    expect(wrapper.text()).toContain('Macchina Caffè')
    expect(wrapper.text()).not.toContain('Luce Cucina')
  })
})
