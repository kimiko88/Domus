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
  parent_area_id?: string
  sort_order?: number
  floor?: number
  order_index?: number
  created_at?: string
  updated_at?: string
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

export interface StateHistory {
  id?: number
  entity_id: string
  state: any
  recorded_at: string
}

export interface SystemMetrics {
  status: string
  uptime_s: number
  entity_count: number
  device_count: number
  mqtt_connected: boolean
  ws_clients: number
  db_size_bytes: number
}

export interface AuditEntry {
  id: number
  actor_type: string
  actor_id?: string | null
  action: string
  target_type?: string | null
  target_id?: string | null
  payload?: Record<string, any>
  created_at: string
}

export interface AuditListResponse {
  entries: AuditEntry[]
  total: number
}

export interface FamilyMember {
  id: string
  username: string
  email?: string
  role: UserRole
  enabled: boolean
  created_at: string
}

export interface FamilyInvite {
  id: string
  token: string
  role: UserRole
  created_by?: string
  expires_at: string
  used_at?: string | null
  used_by?: string | null
  created_at?: string
}

export interface CreateInviteRequest {
  role: UserRole
  expires_in_hours?: number
}

export interface CreateInviteResponse {
  id: string
  token: string
  role: UserRole
  expires_at: string
  invite_url: string
}

export interface InviteValidationResponse {
  valid: boolean
  role: UserRole
  expires_at: string
}

export interface RegisterWithInviteRequest {
  token: string
  username: string
  password: string
  email?: string
}

export interface OSDiagnostics {
  soc_temp_c: number
  cpu_load_1m: number
  cpu_load_5m: number
  cpu_load_15m: number
  cpu_cores: number
  memory_total_bytes: number
  memory_available_bytes: number
  memory_used_bytes: number
  memory_used_percent: number
  disk_total_bytes: number
  disk_free_bytes: number
  disk_used_bytes: number
  disk_used_percent: number
  hostname: string
  os_version: string
  kernel_version: string
  architecture: string
  uptime_seconds: number
  active_slot: string
  booted_slot: string
  slot_confirmed: boolean
  rauc_status: string
  core_healthy: boolean
  running_containers: number
  total_containers: number
  installed_apps: number
  maintenance_mode: boolean
  timestamp: string
}

export interface MaintenanceStatus {
  active: boolean
  expires_at?: string
  remaining_seconds: number
  ssh_port: number
  key_fingerprint?: string
}

export interface EnableMaintenancePayload {
  duration_minutes: number
  authorized_key?: string
}

export interface Floor {
  id: string
  name: string
  level: number
  plan_svg?: string
  plan_image?: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface FloorMarker {
  id: string
  floor_id: string
  entity_key: string
  x_percent: number
  y_percent: number
  icon_override?: string
  show_badge: boolean
  created_at: string
  updated_at: string
}

export type ChorePriority = 'low' | 'medium' | 'high' | 'urgent'
export type ChoreRecurrence = 'none' | 'daily' | 'weekly' | 'monthly' | 'telemetry'
export type ChoreStatus = 'pending' | 'completed' | 'overdue'

export interface Chore {
  id: string
  title: string
  description?: string
  area_id?: string
  assigned_user_id?: string
  priority: ChorePriority
  due_date?: string
  recurrence_type: ChoreRecurrence
  recurrence_rule?: string
  status: ChoreStatus
  last_completed_at?: string
  last_completed_by?: string
  created_at: string
  updated_at: string
}
