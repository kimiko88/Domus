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
  id: string
  type: string
  entity_id?: string
  from?: string
  to?: string
  above?: number
  below?: number
  event_type?: string
}

export interface AutomationCondition {
  id: string
  type: string
  entity_id?: string
  state?: string
  above?: number
  below?: number
  conditions?: AutomationCondition[]
}

export interface AutomationAction {
  id: string
  type: string
  entity_id?: string
  service: string
  payload?: Record<string, any>
  delay_ms?: number
}

export interface Automation {
  id: string
  name: string
  description?: string
  enabled: boolean
  mode: 'single' | 'restart' | 'queued' | 'parallel'
  cooldown_ms: number
  triggers: AutomationTrigger[]
  conditions?: AutomationCondition[]
  actions: AutomationAction[]
  last_triggered_at?: string | null
  created_at: string
  updated_at: string
}

export interface SceneAction {
  entity_id: string
  service: string
  payload?: Record<string, any>
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
