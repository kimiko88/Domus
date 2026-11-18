import { i18n } from '../locales'
import type { AutomationAction, AutomationCondition } from '../api/types'
import { parseAdvancedAutomationAction } from './automationActionValidation'

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
    case 'state': return i18n.global.t('automations.graphErrors.conditionTitles.state')
    case 'numeric_state': return i18n.global.t('automations.graphErrors.conditionTitles.numeric')
    case 'time': return i18n.global.t('automations.graphErrors.conditionTitles.time')
    case 'trigger': return i18n.global.t('automations.graphErrors.conditionTitles.trigger')
    case 'and': return i18n.global.t('automations.graphErrors.conditionTitles.and')
    case 'or': return i18n.global.t('automations.graphErrors.conditionTitles.or')
    case 'not': return i18n.global.t('automations.graphErrors.conditionTitles.not')
    default: return i18n.global.t('automations.graphErrors.conditionTitles.nested')
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
      ? i18n.global.t('automations.graphErrors.actionTitles.delay')
      : action.type === 'scene'
        ? i18n.global.t('automations.graphErrors.actionTitles.scene')
        : action.type === 'service'
          ? i18n.global.t('automations.graphErrors.actionTitles.service')
          : i18n.global.t('automations.graphErrors.actionTitles.advanced', { type: action.type }),
    subtitle: isDelay
      ? i18n.global.t('automations.graphErrors.delaySubtitle', { seconds: (action.delay_ms ?? 1000) / 1000 })
      : i18n.global.t('automations.graphErrors.actionSubtitle', { service: action.service || action.type, target: action.target_entity_key || '' }),
    x,
    y,
    data: {
      originalAction: action,
      target_entity_key: action.target_entity_key,
      service: action.service,
      delay_seconds: (action.delay_ms ?? 1000) / 1000,
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
          : condition.after || condition.before || i18n.global.t('automations.graphErrors.conditionTitles.nested'),
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
          logicCondition: isLogic ? condition : undefined,
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

      let branchCursorY = node.y + 120

      if (action.type === 'if' || action.type === 'if_then_else') {
        if (action.if?.length) {
          branchCursorY = addConditionSequence(
            action.if,
            node,
            'if:conditions',
            `${nodePath}_if_conditions`,
            branchCursorY
          ) + 30
        }
        branchCursorY = appendSequence(action.then || [], node, 'then', depth + 1, branchCursorY, `${nodePath}_then`) + 30
        branchCursorY = appendSequence(action.else || [], node, 'else', depth + 1, branchCursorY, `${nodePath}_else`) + 30
      } else if (action.type === 'choose') {
        ;(action.choose || []).forEach((branch, branchNumber) => {
          if (branch.conditions?.length) {
            branchCursorY = addConditionSequence(
              branch.conditions,
              node,
              `choose:${branchNumber}:conditions`,
              `${nodePath}_choose_${branchNumber}_conditions`,
              branchCursorY
            ) + 30
          }
          branchCursorY = appendSequence(
            branch.sequence || [],
            node,
            `choose:${branchNumber}`,
            depth + 1,
            branchCursorY,
            `${nodePath}_choose_${branchNumber}`
          ) + 30
        })
        branchCursorY = appendSequence(action.default || [], node, 'default', depth + 1, branchCursorY, `${nodePath}_default`) + 30
      } else if (action.type === 'repeat') {
        const repeat = action.repeat
        if (repeat?.while?.length) {
          branchCursorY = addConditionSequence(
            repeat.while, node, 'repeat:while', `${nodePath}_repeat_while`, branchCursorY
          ) + 30
        }
        if (repeat?.until?.length) {
          branchCursorY = addConditionSequence(
            repeat.until, node, 'repeat:until', `${nodePath}_repeat_until`, branchCursorY
          ) + 30
        }
        branchCursorY = appendSequence(
          repeat?.sequence || [], node, 'sequence', depth + 1, branchCursorY, `${nodePath}_repeat_sequence`
        ) + 30
      } else if (action.type === 'parallel') {
        ;(action.parallel || []).forEach((parallelAction, parallelIndex) => {
          branchCursorY = appendSequence(
            [parallelAction],
            node,
            `parallel:${parallelIndex}`,
            depth + 1,
            branchCursorY,
            `${nodePath}_parallel_${parallelIndex}`
          ) + 30
        })
      }

      cursorY = Math.max(cursorY, branchCursorY + 10)
      previousId = node.id
    })

    return cursorY
  }

  appendSequence(actions, null, undefined, 0, startY, 'root')
  return graph
}


function firstBranchWire(parentId: string, branchKey: string, wires: FlowGraphWire[]): FlowGraphWire | undefined {
  const matches = wires.filter((wire) => wire.fromNodeId === parentId && wire.branchKey === branchKey)
  if (matches.length > 1) {
    throw new Error(i18n.global.t('automations.graphErrors.duplicatePort', { port: branchKey }))
  }
  return matches[0]
}

function compileConditionSequence(
  parentId: string,
  branchKey: string,
  nodes: FlowGraphNode[],
  wires: FlowGraphWire[],
  visited: Set<string>
): AutomationCondition[] {
  const first = firstBranchWire(parentId, branchKey, wires)
  if (!first) return []
  const result: AutomationCondition[] = []
  let currentId: string | undefined = first.toNodeId

  while (currentId) {
    if (visited.has(currentId)) throw new Error(i18n.global.t('automations.graphErrors.conditionCycle'))
    visited.add(currentId)
    const node = nodes.find((item) => item.id === currentId)
    if (!node || node.type !== 'condition') {
      throw new Error(i18n.global.t('automations.graphErrors.conditionTarget'))
    }

    if (node.subType === 'logic') {
      let parsed: unknown = node.data.logicCondition
      if (parsed === undefined) {
        try {
          parsed = JSON.parse(node.data.advancedConditionJson || '')
        } catch {
          throw new Error(i18n.global.t('automations.graphErrors.logicJson'))
        }
      }
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        throw new Error(i18n.global.t('automations.graphErrors.logicObject'))
      }
      result.push(parsed as AutomationCondition)
    } else {
      const condition: AutomationCondition = {
        ...(node.data.originalCondition || {}),
        condition: node.subType
      }
      if (node.subType === 'state') {
        condition.entity_key = node.data.entity_key
        condition.state = node.data.state || 'on'
      } else if (node.subType === 'numeric_state') {
        condition.entity_key = node.data.entity_key
        if (node.data.above !== undefined && node.data.above !== '') condition.above = Number(node.data.above)
        if (node.data.below !== undefined && node.data.below !== '') condition.below = Number(node.data.below)
      } else if (node.subType === 'time') {
        condition.after = node.data.after
        condition.before = node.data.before
      } else if (node.subType === 'trigger') {
        condition.trigger_id = node.data.trigger_id
      } else {
        throw new Error(i18n.global.t('automations.graphErrors.unsupportedCondition', { type: node.subType }))
      }
      result.push(condition)
    }

    const next = wires.filter((wire) => wire.fromNodeId === currentId && !wire.branchKey)
    if (next.length > 1) throw new Error(i18n.global.t('automations.graphErrors.conditionSingleOutput'))
    currentId = next[0]?.toNodeId
  }

  return result
}

function compileActionSequence(
  parentId: string,
  branchKey: string,
  nodes: FlowGraphNode[],
  wires: FlowGraphWire[],
  visited: Set<string>
): AutomationAction[] {
  const first = firstBranchWire(parentId, branchKey, wires)
  if (!first) return []
  const result: AutomationAction[] = []
  let currentId: string | undefined = first.toNodeId

  while (currentId) {
    if (visited.has(currentId)) throw new Error(i18n.global.t('automations.graphErrors.actionCycle'))
    visited.add(currentId)
    const node = nodes.find((item) => item.id === currentId)
    if (!node) throw new Error(i18n.global.t('automations.graphErrors.missingNode'))

    if (node.type === 'action' && ADVANCED_ACTIONS.has(node.subType)) {
      const nested = parseAdvancedAutomationAction(node.data.advancedJson, node.subType) as AutomationAction & Record<string, any>
      applyAutomationActionBranches(node, nested, nodes, wires, visited)
      result.push(nested)
    } else if (node.type === 'action') {
      if (node.subType === 'unsupported') throw new Error(i18n.global.t('automations.graphErrors.unsupportedAction'))
      if (node.subType === 'notify') {
        result.push({
          ...(node.data.originalAction || {}),
          type: 'event',
          event_type: 'notification.create',
          event_payload: { message: node.data.message }
        })
      } else {
        const action: AutomationAction = {
          ...(node.data.originalAction || {}),
          type: node.subType === 'scene' ? 'scene' : 'service'
        }
        if (node.subType === 'service') {
          action.service = node.data.service || 'turn_on'
          action.target_entity_key = node.data.target_entity_key
        } else if (node.subType === 'scene') {
          action.service = 'apply_scene'
          action.params = { scene_id: node.data.scene_id }
        }
        result.push(action)
      }
    } else if (node.type === 'delay') {
      result.push({ ...(node.data.originalAction || {}), type: 'delay', delay_ms: Math.max(0, Math.round((Number.isFinite(Number(node.data.delay_seconds)) ? Number(node.data.delay_seconds) : 5) * 1000)) })
    } else {
      throw new Error(i18n.global.t('automations.graphErrors.notExecutable'))
    }

    const next = wires.filter((wire) => wire.fromNodeId === currentId && !wire.branchKey)
    if (next.length > 1) throw new Error(i18n.global.t('automations.graphErrors.actionSingleOutput'))
    currentId = next[0]?.toNodeId
  }

  return result
}

/**
 * Replaces only connected condition/action branches in a composite action.
 * Unconnected branches retain their original JSON, including unknown metadata.
 */
export function applyAutomationActionBranches(
  node: FlowGraphNode,
  action: AutomationAction & Record<string, any>,
  nodes: FlowGraphNode[],
  wires: FlowGraphWire[],
  visited = new Set<string>()
): void {
  const hasBranch = (branchKey: string) => !!firstBranchWire(node.id, branchKey, wires)

  if (action.type === 'if' || action.type === 'if_then_else') {
    if (hasBranch('if:conditions')) {
      action.if = compileConditionSequence(node.id, 'if:conditions', nodes, wires, visited)
    }
    if (hasBranch('then')) action.then = compileActionSequence(node.id, 'then', nodes, wires, visited)
    if (hasBranch('else')) action.else = compileActionSequence(node.id, 'else', nodes, wires, visited)
  } else if (action.type === 'choose') {
    action.choose = (action.choose || []).map((branch, index) => {
      const updated = { ...branch }
      const conditionKey = `choose:${index}:conditions`
      const sequenceKey = `choose:${index}`
      if (hasBranch(conditionKey)) {
        updated.conditions = compileConditionSequence(node.id, conditionKey, nodes, wires, visited)
      }
      if (hasBranch(sequenceKey)) {
        updated.sequence = compileActionSequence(node.id, sequenceKey, nodes, wires, visited)
      }
      return updated
    })
    if (hasBranch('default')) {
      action.default = compileActionSequence(node.id, 'default', nodes, wires, visited)
    }
  } else if (action.type === 'repeat') {
    const repeat: NonNullable<AutomationAction['repeat']> = action.repeat
      ? { ...action.repeat }
      : { sequence: [] }
    if (hasBranch('repeat:while')) {
      repeat.while = compileConditionSequence(node.id, 'repeat:while', nodes, wires, visited)
    }
    if (hasBranch('repeat:until')) {
      repeat.until = compileConditionSequence(node.id, 'repeat:until', nodes, wires, visited)
    }
    if (hasBranch('sequence')) {
      repeat.sequence = compileActionSequence(node.id, 'sequence', nodes, wires, visited)
    }
    action.repeat = repeat
  } else if (action.type === 'parallel') {
    action.parallel = (action.parallel || []).map((original: AutomationAction, index: number) => {
      const branchKey = `parallel:${index}`
      if (!hasBranch(branchKey)) return original
      const sequence = compileActionSequence(node.id, branchKey, nodes, wires, visited)
      if (sequence.length > 1) {
        throw new Error(i18n.global.t('automations.graphErrors.parallelSingle'))
      }
      return sequence[0] || original
    })
  }
}
