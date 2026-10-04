export type UserRole = 'admin' | 'operator' | 'viewer'

export interface User {
  id: string
  username: string
  role: UserRole
  created_at: string
  updated_at: string
}

export interface SetupStatus {
  setup_required: boolean
}

export type EntityDomain = 'light' | 'switch' | 'sensor' | 'binary_sensor' | 'climate' | 'cover' | 'lock'

export interface EntityAttributes {
  brightness?: number
  color_temp?: number
  rgb?: [number, number, number]
  unit_of_measurement?: string
  device_class?: string
  friendly_name?: string
  icon?: string
  current_position?: number
  temperature?: number
  humidity?: number
  battery?: number
  power?: number
  [key: string]: any
}

export interface Entity {
  id: string
  device_id?: string
  domain: EntityDomain
  name: string
  state: string
  attributes: EntityAttributes
  last_changed: string
  last_updated: string
}

export interface Device {
  id: string
  name: string
  area_id?: string | null
  integration_id: string
  manufacturer?: string
  model?: string
  sw_version?: string
  status: 'online' | 'offline' | 'unknown'
  created_at: string
  updated_at: string
}

export interface Area {
  id: string
  name: string
  icon?: string
  floor?: number
  order_index?: number
  created_at: string
  updated_at: string
}

export interface AutomationTrigger {
  id?: string
  platform: string // 'state' | 'numeric_state' | 'time' | 'event'
  entity_key?: string
  from?: string
  to?: string
  above?: number
  below?: number
  at?: string
  event_type?: string
  for_seconds?: number
}

export interface AutomationCondition {
  id?: string
  condition: string // 'state' | 'numeric_state' | 'time' | 'and' | 'or' | 'not'
  entity_key?: string
  state?: string
  above?: number
  below?: number
  after?: string
  before?: string
  conditions?: AutomationCondition[]
}

export interface AutomationAction {
  id?: string
  type: string // 'service' | 'delay' | 'event'
  service?: string
  target_entity_key?: string
  params?: Record<string, any>
  delay_ms?: number
  event_type?: string
  event_payload?: Record<string, any>
}

export interface Automation {
  id: string
  name: string
  description?: string
  enabled: boolean
  execution_mode: 'single' | 'restart' | 'queued' | 'parallel'
  cooldown_seconds: number
  triggers: AutomationTrigger[]
  conditions?: AutomationCondition[]
  actions: AutomationAction[]
  last_run_at?: string | null
  last_status?: string | null
  created_at: string
  updated_at: string
}

export interface AutomationRun {
  id: string
  automation_id: string
  started_at: string
  finished_at?: string | null
  status: 'running' | 'completed' | 'failed' | 'skipped'
  trigger_context: string
  log: string
  duration_ms: number
}

export interface AutomationListResponse {
  automations: Automation[]
  total: number
}

export interface AutomationRunsResponse {
  runs: AutomationRun[]
  total: number
}

export interface SceneAction {
  entity_key: string
  service: string
  params?: Record<string, any>
}

export interface Scene {
  id: string
  name: string
  icon?: string
  description?: string
  actions: SceneAction[]
  created_at: string
  updated_at: string
}

export interface SceneListResponse {
  scenes: Scene[]
  total: number
}

export type WsMessageType =
  | 'auth'
  | 'auth_ok'
  | 'auth_required'
  | 'subscribe'
  | 'unsubscribe'
  | 'call_service'
  | 'event'
  | 'ping'
  | 'pong'
  | 'error'

export interface WsMessage<T = any> {
  type: WsMessageType
  event_type?: string
  payload?: T
  domain?: string
  service?: string
  entity_id?: string
  data?: Record<string, any>
  message?: string
  code?: number
  timestamp?: string
}

export interface EntityStateEventPayload {
  entity_id: string
  old_state: string
  new_state: string
  old_attributes: EntityAttributes
  new_attributes: EntityAttributes
  timestamp: string
}
