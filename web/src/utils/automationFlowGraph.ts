import type { AutomationAction, AutomationCondition } from '../api/types'

export type FlowGraphNodeType = 'trigger' | 'condition' | 'action' | 'delay' | 'logic'

export interface FlowGraphNode {
  id: string
  type: FlowGraphNodeType
  subType: string
  title: string
  subtitle: string
  x: number
  y: number
  data: Record<string, any>
}

export interface FlowGraphWire {
  id: string
  fromNodeId: string
  toNodeId: string
  branchKey?: string
}

export interface AutomationActionGraph {
  nodes: FlowGraphNode[]
  wires: FlowGraphWire[]
}

const COMPOSITE_ACTIONS = new Set(['choose', 'if', 'if_then_else', 'repeat', 'parallel'])
const ADVANCED_ACTIONS = new Set([...COMPOSITE_ACTIONS, 'climate_profile', 'event'])

function safeSegment(value: string): string {
  return value.replace(/[^a-zA-Z0-9_-]/g, '-')
}

function conditionTitle(condition: AutomationCondition): string {
  switch (condition.condition) {
    case 'state': return 'Condizione: stato'
    case 'numeric_state': return 'Condizione: soglia numerica'
    case 'time': return 'Condizione: fascia oraria'
    case 'trigger': return 'Condizione: trigger'
    case 'and': return 'Gruppo AND'
    case 'or': return 'Gruppo OR'
    case 'not': return 'Gruppo NOT'
    default: return `Condizione: ${condition.condition}`
  }
}

function actionNode(action: AutomationAction, id: string, x: number, y: number): FlowGraphNode {
  const isDelay = action.type === 'delay'
  const isAdvanced = ADVANCED_ACTIONS.has(action.type)
  const supportedBasic = action.type === 'service' || action.type === 'scene'
  const subType = isDelay
    ? 'delay'
    : supportedBasic
      ? action.type
      : isAdvanced
        ? action.type
        : 'unsupported'

  return {
    id,
    type: isDelay ? 'delay' : 'action',
    subType,
    title: isDelay
      ? 'Pausa / Delay'
      : action.type === 'scene'
        ? 'Applica Scena'
        : action.type === 'service'
          ? 'Comando'
          : `Azione avanzata (${action.type})`,
    subtitle: isDelay
      ? `Attesa ${(action.delay_ms || 1000) / 1000}s`
      : `${action.service || action.type} -> ${action.target_entity_key || ''}`,
    x,
    y,
    data: {
      originalAction: action,
      target_entity_key: action.target_entity_key,
      service: action.service,
      delay_seconds: (action.delay_ms || 1000) / 1000,
      scene_id: action.params?.scene_id,
      advancedJson: isAdvanced ? JSON.stringify(action, null, 2) : undefined
    }
  }
}

/**
 * Expands all existing composite action sequences into graph nodes and named
 * branch wires. The source JSON remains attached to each node so that fields
 * not represented by the visual editor survive a round trip unchanged.
 */
export function buildAutomationActionGraph(
  actions: AutomationAction[],
  startX = 660,
  startY = 80
): AutomationActionGraph {
  const graph: AutomationActionGraph = { nodes: [], wires: [] }

  const addConditionSequence = (
    conditions: AutomationCondition[] | undefined,
    parent: FlowGraphNode,
    branchKey: string,
    path: string,
    branchY: number
  ): number => {
    if (!Array.isArray(conditions) || conditions.length === 0) return branchY
    let previousId: string | null = null
    conditions.forEach((condition, index) => {
      const id = `cond_${safeSegment(path)}_${index}`
      const isLogic = ['and', 'or', 'not'].includes(condition.condition)
      const node: FlowGraphNode = {
        id,
        type: 'condition',
        subType: isLogic ? 'logic' : condition.condition,
        title: conditionTitle(condition),
        subtitle: condition.entity_key
          ? `${condition.entity_key} ${condition.state ? `= ${condition.state}` : ''}`.trim()
          : condition.after || condition.before || 'Condizione annidata',
        x: parent.x + 300,
        y: branchY + index * 118,
        data: {
          originalCondition: condition,
          entity_key: condition.entity_key,
          state: condition.state,
          above: condition.above,
          below: condition.below,
          after: condition.after,
          before: condition.before,
          trigger_id: condition.trigger_id,
          advancedConditionJson: isLogic ? JSON.stringify(condition, null, 2) : undefined
        }
      }
      graph.nodes.push(node)
      graph.wires.push({
        id: `wire_${safeSegment(path)}_${index}`,
        fromNodeId: previousId || parent.id,
        toNodeId: id,
        ...(!previousId ? { branchKey } : {})
      })
      previousId = id
    })
    return branchY + conditions.length * 118
  }

  const appendSequence = (
    sequence: AutomationAction[],
    parent: FlowGraphNode | null,
    branchKey: string | undefined,
    depth: number,
    initialY: number,
    path: string
  ): number => {
    if (!Array.isArray(sequence) || sequence.length === 0) return initialY
    let cursorY = initialY
    let previousId = parent?.id || null

    sequence.forEach((action, index) => {
      const nodePath = `${path}_${index}`
      const node = actionNode(action, `act_${safeSegment(nodePath)}`, startX + depth * 320, cursorY)
      graph.nodes.push(node)
      if (previousId) {
        graph.wires.push({
          id: `wire_${safeSegment(nodePath)}`,
          fromNodeId: previousId,
          toNodeId: node.id,
          ...(!index && branchKey ? { branchKey } : {})
        })
      }

      cursorY += 130
      if (!COMPOSITE_ACTIONS.has(action.type)) return

      let branchIndex = 0
      let deepestY = cursorY
      const branchStart = (offset: number) => node.y + 120 + offset * 150

      if (action.type === 'if' || action.type === 'if_then_else') {
        const conditionsEnd = addConditionSequence(
          action.if,
          node,
          'if:conditions',
          `${nodePath}_if_conditions`,
          branchStart(branchIndex++)
        )
        deepestY = Math.max(deepestY, conditionsEnd)
        deepestY = Math.max(deepestY, appendSequence(action.then || [], node, 'then', depth + 1, branchStart(branchIndex++), `${nodePath}_then`))
        deepestY = Math.max(deepestY, appendSequence(action.else || [], node, 'else', depth + 1, branchStart(branchIndex++), `${nodePath}_else`))
      } else if (action.type === 'choose') {
        ;(action.choose || []).forEach((branch, branchNumber) => {
          const conditionsEnd = addConditionSequence(
            branch.conditions,
            node,
            `choose:${branchNumber}:conditions`,
            `${nodePath}_choose_${branchNumber}_conditions`,
            branchStart(branchIndex++)
          )
          deepestY = Math.max(deepestY, conditionsEnd)
          deepestY = Math.max(deepestY, appendSequence(
            branch.sequence || [],
            node,
            `choose:${branchNumber}`,
            depth + 1,
            branchStart(branchIndex++),
            `${nodePath}_choose_${branchNumber}`
          ))
        })
        deepestY = Math.max(deepestY, appendSequence(action.default || [], node, 'default', depth + 1, branchStart(branchIndex++), `${nodePath}_default`))
      } else if (action.type === 'repeat') {
        const repeat = action.repeat
        if (repeat?.while?.length) {
          deepestY = Math.max(deepestY, addConditionSequence(
            repeat.while, node, 'repeat:while', `${nodePath}_repeat_while`, branchStart(branchIndex++)
          ))
        }
        if (repeat?.until?.length) {
          deepestY = Math.max(deepestY, addConditionSequence(
            repeat.until, node, 'repeat:until', `${nodePath}_repeat_until`, branchStart(branchIndex++)
          ))
        }
        deepestY = Math.max(deepestY, appendSequence(
          repeat?.sequence || [], node, 'sequence', depth + 1, branchStart(branchIndex++), `${nodePath}_repeat_sequence`
        ))
      } else if (action.type === 'parallel') {
        ;(action.parallel || []).forEach((parallelAction, parallelIndex) => {
          deepestY = Math.max(deepestY, appendSequence(
            [parallelAction],
            node,
            `parallel:${parallelIndex}`,
            depth + 1,
            branchStart(branchIndex++),
            `${nodePath}_parallel_${parallelIndex}`
          ))
        })
      }

      cursorY = Math.max(cursorY, deepestY + 40)
      previousId = node.id
    })

    return cursorY
  }

  appendSequence(actions, null, undefined, 0, startY, 'root')
  return graph
}
