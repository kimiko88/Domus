import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AutomationFlowchartModal from './AutomationFlowchartModal.vue'

// Mock i18n
vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        'flowchart.modalTitle': 'Flowchart Visuale Automazioni',
        'flowchart.desc': 'Trascina i blocchi, configura le condizioni e testa il flusso',
        'flowchart.btnSimulate': 'Simula Sequenza',
        'flowchart.btnPause': 'In Esecuzione...',
        'flowchart.addTrigger': '+ Trigger',
        'flowchart.addCondition': '+ Condizione',
        'flowchart.addAction': '+ Azione',
        'flowchart.btnSave': 'Salva Automazione'
      }
      return translations[key] || key
    }
  })
}))

describe('AutomationFlowchartModal.vue', () => {
  it('does not render when isOpen is false', () => {
    const wrapper = mount(AutomationFlowchartModal, {
      props: { isOpen: false }
    })
    expect(wrapper.find('.flowchart-modal').exists()).toBe(false)
  })

  it('renders default flow nodes and wires when isOpen is true', () => {
    const wrapper = mount(AutomationFlowchartModal, {
      props: { isOpen: true }
    })
    expect(wrapper.find('.flowchart-modal').exists()).toBe(true)

    // Initial 4 nodes: trigger, condition, delay, action
    const nodes = wrapper.findAll('.flow-node')
    expect(nodes.length).toBe(4)

    // Check SVG wire paths
    const wires = wrapper.findAll('.wire-path')
    expect(wires.length).toBe(3)
  })

  it('allows dragging nodes and updates their coordinates', async () => {
    const wrapper = mount(AutomationFlowchartModal, {
      props: { isOpen: true }
    })

    const conditionNode = wrapper.findAll('.flow-node')[1]
    expect(conditionNode.attributes('style')).toContain('left: 320px')
    expect(conditionNode.attributes('style')).toContain('top: 120px')

    // Start dragging
    await conditionNode.trigger('pointerdown', { clientX: 320, clientY: 120 })

    // Simulate pointermove on window
    const moveEvent = new PointerEvent('pointermove', { clientX: 420, clientY: 200 })
    window.dispatchEvent(moveEvent)
    await wrapper.vm.$nextTick()

    // Node style should have changed
    expect(conditionNode.attributes('style')).toContain('left: 420px')
    expect(conditionNode.attributes('style')).toContain('top: 200px')

    // End drag
    window.dispatchEvent(new PointerEvent('pointerup'))
    await wrapper.vm.$nextTick()
  })

  it('allows changing condition and editing fields via Inspector panel', async () => {
    const wrapper = mount(AutomationFlowchartModal, {
      props: { isOpen: true }
    })

    // Click condition node to select it
    const conditionNode = wrapper.findAll('.flow-node')[1]
    await conditionNode.trigger('click')
    await wrapper.vm.$nextTick()

    // Inspector should be visible
    expect(wrapper.find('.inspector-panel').exists()).toBe(true)

    // Select preset
    const presetSelect = wrapper.find('#preset-condition')
    expect(presetSelect.exists()).toBe(true)
    await presetSelect.setValue('presence_home')
    await presetSelect.trigger('change')
    await wrapper.vm.$nextTick()

    // Title and subtitle in node should be updated
    expect(conditionNode.text()).toContain('Presenza Utente')
    expect(conditionNode.text()).toContain('person.utente == home')

    // Custom edit
    const titleInput = wrapper.find('#node-edit-title')
    await titleInput.setValue('Condizione Sensore Speciale')
    await wrapper.vm.$nextTick()

    expect(conditionNode.text()).toContain('Condizione Sensore Speciale')
  })

  it('automatically selects newly added nodes so the condition can be changed immediately', async () => {
    const wrapper = mount(AutomationFlowchartModal, {
      props: { isOpen: true }
    })

    // Click "+ Condizione" toolbar button
    const addCondBtn = wrapper.find('.btn-tool.type-cond')
    await addCondBtn.trigger('click')
    await wrapper.vm.$nextTick()

    // Should now have 5 nodes
    const nodes = wrapper.findAll('.flow-node')
    expect(nodes.length).toBe(5)

    // Inspector should be open and focused on the newly added node
    expect(wrapper.find('.inspector-panel').exists()).toBe(true)
    expect(wrapper.find('.badge-condition').exists()).toBe(true)

    // Change condition preset for new node
    const presetSelect = wrapper.find('#preset-condition')
    await presetSelect.setValue('temp_high')
    await presetSelect.trigger('change')
    await wrapper.vm.$nextTick()

    const latestNode = wrapper.findAll('.flow-node')[4]
    expect(latestNode.text()).toContain('Temperatura Alta')
  })

  it('emits save event with nodes and connections', async () => {
    const wrapper = mount(AutomationFlowchartModal, {
      props: { isOpen: true }
    })

    await wrapper.find('.btn-save').trigger('click')
    expect(wrapper.emitted('save')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
