import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import FlowEditorView from './FlowEditorView.vue'
import { apiClient } from '../api/client'

// Mock vue-router
vi.mock('vue-router', () => ({
  useRoute: () => ({
    params: { id: '' }
  }),
  useRouter: () => ({
    push: vi.fn()
  })
}))

describe('FlowEditorView.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.spyOn(apiClient, 'get').mockResolvedValue([])
    vi.spyOn(apiClient, 'post').mockResolvedValue({ id: 'auto_created_1', name: 'Test Flow' })
    vi.spyOn(apiClient, 'put').mockResolvedValue({ id: 'auto_created_1', name: 'Test Flow' })
  })

  it('renders node palette, canvas, and header tools', async () => {
    const wrapper = mount(FlowEditorView)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Libreria Nodi')
    expect(wrapper.text()).toContain('Triggers')
    expect(wrapper.text()).toContain('Conditions')
    expect(wrapper.text()).toContain('Actions & Flow')
    expect(wrapper.text()).toContain('Test Run')
    expect(wrapper.text()).toContain('Salva Flusso')
    expect(wrapper.text()).toContain('AST JSON')
  })

  it('creates a new node when clicking on a palette item', async () => {
    const wrapper = mount(FlowEditorView)
    await wrapper.vm.$nextTick()

    const initialNodesCount = wrapper.findAll('.flow-node').length

    // Click on "Orario Programmato" in palette
    const timePaletteItem = wrapper.findAll('.palette-item').find(el => el.text().includes('Orario Programmato'))
    expect(timePaletteItem).toBeDefined()

    await timePaletteItem?.trigger('click')
    await wrapper.vm.$nextTick()

    const newNodesCount = wrapper.findAll('.flow-node').length
    expect(newNodesCount).toBe(initialNodesCount + 1)
  })

  it('selects a node and opens inspector drawer with editable properties', async () => {
    const wrapper = mount(FlowEditorView)
    await wrapper.vm.$nextTick()

    const nodes = wrapper.findAll('.flow-node')
    expect(nodes.length).toBeGreaterThan(0)

    // Click first node to select
    await nodes[0].trigger('mousedown')
    await wrapper.vm.$nextTick()

    const inspector = wrapper.find('.node-inspector')
    expect(inspector.exists()).toBe(true)
    expect(wrapper.text()).toContain('Titolo Visuale')
  })

  it('opens AST JSON modal when clicking AST JSON button', async () => {
    const wrapper = mount(FlowEditorView)
    await wrapper.vm.$nextTick()

    const astJsonBtn = wrapper.findAll('.btn-tool').find(b => b.text().includes('AST JSON'))
    expect(astJsonBtn?.exists()).toBe(true)

    await astJsonBtn?.trigger('click')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('.modal-backdrop').exists()).toBe(true)
    expect(wrapper.text()).toContain('AST JSON Compilato')
  })
})
