import type { AutomationAction } from '../api/types'

type JsonObject = Record<string, unknown>

const supportedActionTypes = new Set([
  'service',
  'delay',
  'event',
  'climate_profile',
  'choose',
  'if',
  'if_then_else',
  'repeat',
  'parallel'
])

const supportedConditionTypes = new Set([
  'state',
  'numeric_state',
  'time',
  'and',
  'or',
  'not',
  'trigger'
])

const MAX_NESTING_DEPTH = 10
const MAX_ACTIONS = 100
const MAX_REPEAT_COUNT = 1000

function isObject(value: unknown): value is JsonObject {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function requireObject(value: unknown, path: string): JsonObject {
  if (!isObject(value)) throw new Error(`${path} deve essere un oggetto JSON.`)
  return value
}

function requireArray(value: unknown, path: string, minLength = 0): unknown[] {
  if (!Array.isArray(value)) throw new Error(`${path} deve essere un array.`)
  if (value.length < minLength) {
    throw new Error(`${path} deve contenere almeno ${minLength} elemento/i.`)
  }
  return value
}

function requireNonEmptyString(value: unknown, path: string): string {
  if (typeof value !== 'string' || value.trim() === '') {
    throw new Error(`${path} è obbligatorio.`)
  }
  return value
}

function requireFiniteNumber(value: unknown, path: string, min: number, max: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < min || value > max) {
    throw new Error(`${path} deve essere un numero JSON compreso tra ${min} e ${max}.`)
  }
  return value
}

function requireInteger(value: unknown, path: string, min: number, max: number): number {
  const parsed = requireFiniteNumber(value, path, min, max)
  if (!Number.isInteger(parsed)) throw new Error(`${path} deve essere un numero intero.`)
  return parsed
}

function validateConditionList(value: unknown, path: string, depth: number): void {
  const conditions = requireArray(value, path, 1)
  if (depth > MAX_NESTING_DEPTH) throw new Error(`${path}: profondità massima superata.`)

  conditions.forEach((entry, index) => {
    const conditionPath = `${path}[${index}]`
    const condition = requireObject(entry, conditionPath)
    const type = requireNonEmptyString(condition.condition, `${conditionPath}.condition`)
    if (!supportedConditionTypes.has(type)) {
      throw new Error(`${conditionPath}: tipo di condizione non supportato "${type}".`)
    }

    switch (type) {
      case 'state':
        requireNonEmptyString(condition.entity_key, `${conditionPath}.entity_key`)
        requireNonEmptyString(condition.state, `${conditionPath}.state`)
        break
      case 'numeric_state': {
        requireNonEmptyString(condition.entity_key, `${conditionPath}.entity_key`)
        const hasAbove = condition.above !== undefined && condition.above !== null
        const hasBelow = condition.below !== undefined && condition.below !== null
        if (!hasAbove && !hasBelow) {
          throw new Error(`${conditionPath}: specifica almeno una soglia above o below.`)
        }
        if (hasAbove) requireFiniteNumber(condition.above, `${conditionPath}.above`, -1e9, 1e9)
        if (hasBelow) requireFiniteNumber(condition.below, `${conditionPath}.below`, -1e9, 1e9)
        break
      }
      case 'time':
        if (
          (typeof condition.after !== 'string' || condition.after.trim() === '') &&
          (typeof condition.before !== 'string' || condition.before.trim() === '')
        ) {
          throw new Error(`${conditionPath}: specifica after, before o entrambi.`)
        }
        break
      case 'trigger':
        requireNonEmptyString(condition.trigger_id, `${conditionPath}.trigger_id`)
        break
      case 'and':
      case 'or':
        validateConditionList(condition.conditions, `${conditionPath}.conditions`, depth + 1)
        break
      case 'not': {
        const children = requireArray(condition.conditions, `${conditionPath}.conditions`)
        if (children.length !== 1) {
          throw new Error(`${conditionPath}.conditions deve contenere esattamente una condizione per NOT.`)
        }
        validateConditionList(children, `${conditionPath}.conditions`, depth + 1)
        break
      }
    }
  })
}

function validateActionList(value: unknown, path: string, depth: number, count: { value: number }): void {
  const actions = requireArray(value, path)
  actions.forEach((action, index) => validateAction(action, `${path}[${index}]`, depth + 1, count))
}

function validateAction(value: unknown, path: string, depth: number, count: { value: number }): void {
  if (depth > MAX_NESTING_DEPTH) throw new Error(`${path}: profondità massima delle azioni superata.`)
  count.value += 1
  if (count.value > MAX_ACTIONS) throw new Error(`La configurazione supera il limite di ${MAX_ACTIONS} azioni annidate.`)

  const action = requireObject(value, path)
  const type = requireNonEmptyString(action.type, `${path}.type`)
  if (!supportedActionTypes.has(type)) {
    throw new Error(`${path}: tipo di azione non supportato "${type}".`)
  }

  switch (type) {
    case 'service':
      requireNonEmptyString(action.service, `${path}.service`)
      requireNonEmptyString(action.target_entity_key, `${path}.target_entity_key`)
      if (action.params !== undefined && !isObject(action.params)) {
        throw new Error(`${path}.params deve essere un oggetto JSON.`)
      }
      break

    case 'delay':
      requireInteger(action.delay_ms, `${path}.delay_ms`, 0, 86_400_000)
      break

    case 'event':
      requireNonEmptyString(action.event_type, `${path}.event_type`)
      if (action.event_payload !== undefined && !isObject(action.event_payload)) {
        throw new Error(`${path}.event_payload deve essere un oggetto JSON.`)
      }
      break

    case 'climate_profile': {
      requireNonEmptyString(action.target_entity_key, `${path}.target_entity_key`)
      const hasTemperature = action.temperature !== undefined && action.temperature !== null
      if (hasTemperature) {
        requireFiniteNumber(action.temperature, `${path}.temperature`, 5, 35)
      } else {
        const profile = requireNonEmptyString(action.profile, `${path}.profile`)
        if (!['comfort', 'eco', 'night', 'away'].includes(profile)) {
          throw new Error(`${path}.profile deve essere comfort, eco, night o away.`)
        }
      }
      if (action.duration_minutes !== undefined) {
        requireInteger(action.duration_minutes, `${path}.duration_minutes`, 0, 1440)
      }
      break
    }

    case 'choose': {
      const branches = requireArray(action.choose, `${path}.choose`)
      const fallback = action.default === undefined
        ? []
        : requireArray(action.default, `${path}.default`)
      if (branches.length === 0 && fallback.length === 0) {
        throw new Error(`${path}: aggiungi almeno un ramo choose o un'azione di fallback default.`)
      }
      branches.forEach((branchValue, index) => {
        const branchPath = `${path}.choose[${index}]`
        const branch = requireObject(branchValue, branchPath)
        validateConditionList(branch.conditions, `${branchPath}.conditions`, depth + 1)
        validateActionList(branch.sequence, `${branchPath}.sequence`, depth, count)
      })
      validateActionList(fallback, `${path}.default`, depth, count)
      break
    }

    case 'if':
    case 'if_then_else': {
      validateConditionList(action.if, `${path}.if`, depth + 1)
      validateActionList(action.then, `${path}.then`, depth, count)
      const fallback = action.else === undefined ? [] : requireArray(action.else, `${path}.else`)
      validateActionList(fallback, `${path}.else`, depth, count)
      break
    }

    case 'repeat': {
      const repeat = requireObject(action.repeat, `${path}.repeat`)
      const countValue = repeat.count
      const hasCount = countValue !== undefined && countValue !== null && Number(countValue) > 0
      const hasWhile = Array.isArray(repeat.while) && repeat.while.length > 0
      const hasUntil = Array.isArray(repeat.until) && repeat.until.length > 0
      if ([hasCount, hasWhile, hasUntil].filter(Boolean).length !== 1) {
        throw new Error(`${path}.repeat: scegli una sola modalità tra count, while e until.`)
      }
      if (hasCount) requireInteger(countValue, `${path}.repeat.count`, 1, MAX_REPEAT_COUNT)
      if (hasWhile) validateConditionList(repeat.while, `${path}.repeat.while`, depth + 1)
      if (hasUntil) validateConditionList(repeat.until, `${path}.repeat.until`, depth + 1)
      validateActionList(repeat.sequence, `${path}.repeat.sequence`, depth, count)
      if ((repeat.sequence as unknown[]).length === 0) {
        throw new Error(`${path}.repeat.sequence non può essere vuota.`)
      }
      break
    }

    case 'parallel': {
      const parallel = requireArray(action.parallel, `${path}.parallel`, 1)
      validateActionList(parallel, `${path}.parallel`, depth, count)
      break
    }
  }
}

/**
 * Parses and validates a composite action before it can be persisted by the
 * visual editor. Unknown action types are rejected rather than silently lost
 * or accepted only to fail later in the runtime.
 */
export function parseAdvancedAutomationAction(raw: unknown, expectedType: string): AutomationAction {
  if (typeof raw !== 'string' || raw.trim() === '') {
    throw new Error('Inserisci la configurazione JSON completa per questa azione.')
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch (error) {
    const reason = error instanceof Error ? error.message : 'JSON non valido'
    throw new Error(`Configurazione JSON non valida: ${reason}`)
  }

  const action = requireObject(parsed, 'Azione')
  if (action.type !== expectedType) {
    throw new Error(`Il campo type deve essere "${expectedType}" per questo nodo.`)
  }

  const count = { value: 0 }
  validateAction(action, 'action', 0, count)
  return action as unknown as AutomationAction
}
