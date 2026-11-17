import { describe, expect, it } from 'vitest'
import { buildAutomationActionGraph } from './automationFlowGraph'

describe('buildAutomationActionGraph', () => {
  it('expands IF conditions, true/false sequences and nested composite actions', () => {
    const source = [{
      type: 'if',
      if: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }],
      then: [{
        type: 'choose',
        choose: [{
          conditions: [{ condition: 'time', after: '20:00:00' }],
          sequence: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hall' }]
        }]
      }],
      else: [{ type: 'delay', delay_ms: 5000 }]
    }]

    const graph = buildAutomationActionGraph(source as any)
    const first = graph.nodes.find((node) => node.id === 'act_root_0')

    expect(first?.subType).toBe('if')
    expect(graph.wires).toEqual(expect.arrayContaining([
      expect.objectContaining({ fromNodeId: 'act_root_0', branchKey: 'if:conditions' }),
      expect.objectContaining({ fromNodeId: 'act_root_0', branchKey: 'then' }),
      expect.objectContaining({ fromNodeId: 'act_root_0', branchKey: 'else' })
    ]))
    expect(graph.nodes.some((node) => node.subType === 'logic')).toBe(false)
    expect(graph.nodes.filter((node) => node.type === 'condition')).toHaveLength(2)
    expect(graph.nodes.some((node) => node.subType === 'choose')).toBe(true)
    expect(graph.wires.some((wire) => wire.branchKey === 'choose:0:conditions')).toBe(true)
    expect(graph.wires.some((wire) => wire.branchKey === 'choose:0')).toBe(true)
  })

  it('expands repeat while/until conditions and its action sequence', () => {
    const source = [{
      type: 'repeat',
      repeat: {
        while: [{ condition: 'state', entity_key: 'binary_sensor.window', state: 'on' }],
        sequence: [{ type: 'delay', delay_ms: 1000 }],
        until: [{ condition: 'numeric_state', entity_key: 'sensor.temperature', below: 19 }]
      }
    }]

    const graph = buildAutomationActionGraph(source as any)

    expect(graph.wires).toEqual(expect.arrayContaining([
      expect.objectContaining({ branchKey: 'repeat:while' }),
      expect.objectContaining({ branchKey: 'repeat:until' }),
      expect.objectContaining({ branchKey: 'sequence' })
    ]))
    expect(graph.nodes.filter((node) => node.type === 'condition')).toHaveLength(2)
    expect(graph.nodes.some((node) => node.type === 'delay')).toBe(true)
  })

  it('expands parallel entries as independent single-action branches', () => {
    const source = [{
      type: 'parallel',
      parallel: [
        { type: 'service', service: 'turn_on', target_entity_key: 'light.one' },
        { type: 'delay', delay_ms: 250 },
        { type: 'if', if: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }], then: [] }
      ]
    }]

    const graph = buildAutomationActionGraph(source as any)

    expect(graph.wires.filter((wire) => wire.branchKey?.startsWith('parallel:'))).toHaveLength(3)
    expect(graph.nodes.filter((node) => node.id.startsWith('act_root_0_parallel_')).length).toBe(3)
  })

  it('retains original action and nested condition objects for lossless editing', () => {
    const condition = {
      condition: 'and',
      conditions: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }]
    }
    const action = {
      type: 'if',
      if: [condition],
      then: [{ type: 'event', event_type: 'custom.event', event_payload: { source: 'test' } }]
    }

    const graph = buildAutomationActionGraph([action] as any)
    const logicNode = graph.nodes.find((node) => node.subType === 'logic')
    const actionNode = graph.nodes.find((node) => node.subType === 'if')

    expect(logicNode?.data.originalCondition).toEqual(condition)
    expect(JSON.parse(logicNode?.data.advancedConditionJson)).toEqual(condition)
    expect(JSON.parse(actionNode?.data.advancedJson)).toEqual(action)
  })

  it('keeps separate branches and stable unique node identifiers', () => {
    const graph = buildAutomationActionGraph([
      { type: 'parallel', parallel: [
        { type: 'service', service: 'turn_on', target_entity_key: 'light.a' },
        { type: 'service', service: 'turn_off', target_entity_key: 'light.b' }
      ] },
      { type: 'service', service: 'turn_on', target_entity_key: 'light.c' }
    ] as any)

    const ids = graph.nodes.map((node) => node.id)
    expect(new Set(ids).size).toBe(ids.length)
    expect(graph.wires.some((wire) => wire.branchKey === 'parallel:0')).toBe(true)
    expect(graph.wires.some((wire) => wire.branchKey === 'parallel:1')).toBe(true)
    expect(graph.wires.some((wire) => wire.fromNodeId === 'act_root_0' && wire.toNodeId === 'act_root_1' && !wire.branchKey)).toBe(true)
    expect(graph.nodes.find((node) => node.id === 'act_root_1')).toBeDefined()
  })
})
