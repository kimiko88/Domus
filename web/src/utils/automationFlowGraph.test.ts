import { describe, expect, it } from 'vitest'
import { applyAutomationActionBranches, buildAutomationActionGraph } from './automationFlowGraph'
import { parseAdvancedAutomationAction } from './automationActionValidation'

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
    expect(graph.nodes.filter((node) => node.type === 'condition')).toHaveLength(1)
    expect(graph.nodes.some((node) => node.subType === 'choose')).toBe(true)
    expect(graph.wires.some((wire) => wire.branchKey === 'choose:0:conditions')).toBe(true)
    expect(graph.wires.some((wire) => wire.branchKey === 'choose:0')).toBe(true)
  })

  it('expands repeat while/until conditions and its action sequence', () => {
    const source = [{
      type: 'repeat',
      repeat: {
        while: [{ condition: 'state', entity_key: 'binary_sensor.window', state: 'on' }],
        sequence: [{ type: 'delay', delay_ms: 1000 }]
      }
    }]

    const graph = buildAutomationActionGraph(source as any)

    expect(graph.wires).toEqual(expect.arrayContaining([
      expect.objectContaining({ branchKey: 'repeat:while' }),
      expect.objectContaining({ branchKey: 'sequence' })
    ]))
    expect(graph.nodes.filter((node) => node.type === 'condition')).toHaveLength(2)
    expect(graph.nodes.some((node) => node.type === 'delay')).toBe(true)

    const untilGraph = buildAutomationActionGraph([{
      type: 'repeat',
      repeat: {
        until: [{ condition: 'numeric_state', entity_key: 'sensor.temperature', below: 19 }],
        sequence: [{ type: 'delay', delay_ms: 1000 }]
      }
    }] as any)
    expect(untilGraph.wires.some((wire) => wire.branchKey === 'repeat:until')).toBe(true)
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
  it('round-trips nested IF, CHOOSE, PARALLEL and editable branch conditions', () => {
    const source = {
      type: 'if',
      if: [
        {
          condition: 'and',
          conditions: [
            { condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' },
            { condition: 'not', conditions: [{ condition: 'time', before: '06:00:00' }] }
          ]
        },
        { condition: 'numeric_state', entity_key: 'sensor.room', above: 24 }
      ],
      then: [{
        type: 'choose',
        choose: [{
          conditions: [{ condition: 'time', after: '20:00:00' }],
          sequence: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hall' }]
        }],
        default: [{ type: 'delay', delay_ms: 250 }]
      }],
      else: [{
        type: 'parallel',
        parallel: [
          { type: 'service', service: 'turn_off', target_entity_key: 'light.hall' },
          { type: 'climate_profile', target_entity_key: 'climate.room', profile: 'eco', duration_minutes: 30 }
        ]
      }]
    }

    const graph = buildAutomationActionGraph([source] as any)
    const rootNode = graph.nodes.find((node) => node.id === 'act_root_0')!
    const compiled = parseAdvancedAutomationAction(rootNode.data.advancedJson, 'if') as any

    applyAutomationActionBranches(rootNode, compiled, graph.nodes, graph.wires)

    expect(compiled).toEqual(source)
  })

  it('preserves unconnected legacy branches and metadata exactly', () => {
    const source = {
      type: 'if',
      if: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }],
      then: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hall' }],
      else: [{ type: 'event', event_type: 'legacy.fallback', event_payload: { retained: true }, custom_metadata: 'keep-me' }]
    }
    const graph = buildAutomationActionGraph([source] as any)
    const rootNode = graph.nodes.find((node) => node.id === 'act_root_0')!
    const compiled = parseAdvancedAutomationAction(rootNode.data.advancedJson, 'if') as any
    applyAutomationActionBranches(rootNode, compiled, [rootNode], [])

    expect(compiled).toEqual(source)
  })

  it('rejects cycles and multiple destinations in a branch sequence', () => {
    const source = {
      type: 'if',
      if: [{ condition: 'state', entity_key: 'binary_sensor.motion', state: 'on' }],
      then: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hall' }]
    }
    const graph = buildAutomationActionGraph([source] as any)
    const rootNode = graph.nodes.find((node) => node.id === 'act_root_0')!
    const compiled = parseAdvancedAutomationAction(rootNode.data.advancedJson, 'if') as any
    const thenWire = graph.wires.find((wire) => wire.branchKey === 'then')!

    const cyclic = [...graph.wires, {
      id: 'cycle',
      fromNodeId: thenWire.toNodeId,
      toNodeId: thenWire.toNodeId
    }]
    expect(() => applyAutomationActionBranches(rootNode, compiled, graph.nodes, cyclic)).toThrow('ciclo')

    const multiple = [...graph.wires, {
      id: 'second-destination',
      fromNodeId: rootNode.id,
      toNodeId: thenWire.toNodeId,
      branchKey: 'then'
    }]
    expect(() => applyAutomationActionBranches(rootNode, compiled, graph.nodes, multiple)).toThrow('più collegamenti')
  })

})
