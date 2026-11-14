export type UserRole = 'admin' | 'operator' | 'viewer' | 'guest' | 'kid'

export interface User {
  id: string
  username: string
  role: UserRole
  email?: string
  is_guest?: boolean
  guest_expires_at?: string
  pin_code?: string
  kiosk_mode?: boolean
  allowed_views?: string[]
  allowed_area_ids?: string[]
  allowed_entity_ids?: string[]
  created_at: string
  updated_at: string
}

export interface NotificationAction {
  id: string
  title: string
  destructive?: boolean
}

export interface ActionableNotification {
  id: string
  title: string
  message: string
  image_url?: string
  critical?: boolean
  volume?: number
  actions: NotificationAction[]
  created_at: string
  resolved_at?: string
}

export interface DiscoveredHost {
  ip: string
  mac: string
  hostname?: string
  vendor?: string
  suggested_plugin?: string
  confidence: number
  open_ports?: number[]
  first_seen_at: string
  last_seen_at: string
  is_iot: boolean
  metadata?: Record<string, string>
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
  external_id?: string
  name: string
  area_id?: string | null
  integration_id: string
  manufacturer?: string
  model?: string
  sw_version?: string
  online?: boolean
  metadata?: string
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
  type: string // 'service' | 'delay' | 'event' | 'choose' | 'if' | 'repeat' | 'parallel'
  service?: string
  target_entity_key?: string
  params?: Record<string, any>
  delay_ms?: number
  event_type?: string
  event_payload?: Record<string, any>
  choose?: { conditions: AutomationCondition[]; sequence: AutomationAction[] }[]
  default?: AutomationAction[]
  if?: AutomationCondition[]
  then?: AutomationAction[]
  else?: AutomationAction[]
  repeat?: {
    count?: number
    while?: AutomationCondition[]
    until?: AutomationCondition[]
    sequence: AutomationAction[]
  }
  parallel?: AutomationAction[]
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
  is_guest?: boolean
  guest_expires_at?: string
  pin_code?: string
  kiosk_mode?: boolean
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

// 1. Alarm Control Panel
export type AlarmState = 'disarmed' | 'armed_home' | 'armed_away' | 'armed_night' | 'pending' | 'triggered'

export interface AlarmControlPanel {
  id: string
  name: string
  state: AlarmState
  entry_delay_seconds: number
  exit_delay_seconds: number
  trigger_delay_seconds: number
  perimeter_sensors: string[]
  interior_sensors: string[]
  siren_entities: string[]
  last_changed_by?: string
  last_triggered_reason?: string
  triggered_at?: string
  updated_at: string
}

// 2. Local Voice & LLM Assistant
export interface AssistantActionExecuted {
  entity_key: string
  service: string
  params?: Record<string, any>
  success: boolean
  message: string
}

export interface AssistantChatResponse {
  user_prompt: string
  response_text: string
  actions_executed: AssistantActionExecuted[]
  engine_used: string
  execution_time_ms: number
}

export interface AssistantConfig {
  ollama_url: string
  model: string
  enabled: boolean
  system_prompt: string
}

// 3. Inbound Webhooks
export interface InboundWebhook {
  id: string
  name: string
  webhook_id: string
  secret_token: string
  allowed_methods: string
  enabled: boolean
  trigger_count: number
  last_triggered_at?: string
  created_at: string
}

// 4. Zigbee Direct Binding
export interface ZigbeeBinding {
  id: string
  source_device_id: string
  source_device_name: string
  target_device_id: string
  target_device_name: string
  clusters: string[]
  bound_at: string
}

// 5. Better Thermostat (Climate Room Compensator)
export interface ClimateCompensatorConfig {
  trv_entity_key: string
  room_temp_sensor_key: string
  window_sensor_key?: string
  target_temperature: number
  delta_factor: number
  current_room_temp?: number
  compensated_setpoint?: number
  window_open?: boolean
}

// 6. Template Entities (Virtual Computed Sensors & Formula Evaluator)
export interface TemplateEntity {
  id: string
  entity_id: string
  name: string
  domain: string
  state_template: string
  attributes_template?: string
  unit_of_measurement?: string
  device_class?: string
  icon?: string
  source_entity_ids: string[]
  created_at: string
  updated_at: string
}

// 7. Long-Term Statistics (LTS Rollup)
export interface StatisticsRecord {
  id: string
  entity_id: string
  period: 'hourly' | 'daily'
  period_start: string
  period_end: string
  mean?: number
  min?: number
  max?: number
  sum?: number
  count: number
  unit?: string
  created_at: string
}

// 8. Multi-Utility Energy Tracking (Electricity, Gas, Water)
export type UtilityType = 'electricity' | 'gas' | 'water'

export interface UtilityMeter {
  id: string
  name: string
  utility_type: UtilityType
  unit: string
  cost_per_unit: number
  source_entity_id: string
  current_reading: number
  created_at: string
  updated_at: string
}

export interface MultiUtilitySummary {
  electricity: {
    total_consumption: number
    unit: string
    cost_eur: number
  }
  gas: {
    total_consumption: number
    unit: string
    cost_eur: number
  }
  water: {
    total_consumption: number
    unit: string
    cost_eur: number
  }
  total_cost_eur: number
  currency: string
}

// 9. Multi-Zone Spatial Geofencing & Bayesian Presence
export interface SpatialZone {
  id: string
  name: string
  latitude: number
  longitude: number
  radius: number
  icon: string
  passive: boolean
  created_at: string
}

export interface TrackedPerson {
  id: string
  name: string
  user_id?: string
  state: string
  latitude?: number
  longitude?: number
  device_trackers: string[]
  updated_at: string
}

// 10. Two-Factor Authentication & Security Defense
export interface TwoFactorSetupResponse {
  secret: string
  qr_url: string
  backup_codes: string[]
}

export interface BannedIP {
  ip: string
  failed_attempts: number
  banned_until?: string
  updated_at: string
}

// 11. Reusable Scripts (script.*)
export interface ScriptField {
  name: string
  description?: string
  required?: boolean
  default?: any
  example?: any
  selector?: Record<string, any>
}

export interface Script {
  id: string
  name: string
  description?: string
  icon?: string
  mode: 'single' | 'restart' | 'queued' | 'parallel'
  max_runs?: number
  fields?: Record<string, ScriptField>
  sequence: AutomationAction[]
  created_at?: string
  updated_at?: string
}

// 12. Human Contextual Activity Logbook
export interface LogbookEntry {
  id: string
  entity_id: string
  entity_name: string
  domain: string
  state: string
  message: string
  context_type: 'user' | 'automation' | 'script' | 'system'
  context_id?: string
  context_name?: string
  timestamp: string
}

export interface LogbookFilter {
  entity_id?: string
  domain?: string
  from?: string
  to?: string
  limit?: number
}

// 13. Weekly Schedule Matrix (7-day x 24h Cronotermostato)
export interface TimeSlot {
  start: string
  end: string
  value?: number
  state?: string
}

export type WeeklyGrid = Record<string, TimeSlot[]>

export interface WeeklySchedule {
  id: string
  name: string
  icon?: string
  target_entity_id: string
  target_service?: string
  grid: WeeklyGrid
  current_slot_state: string
  created_at?: string
  updated_at?: string
}

// 14. Native Push Companion Device Tokens
export interface DevicePushToken {
  id?: string
  user_id?: string
  platform: 'ios' | 'android'
  device_token: string
  device_name?: string
  app_version?: string
  created_at?: string
  updated_at?: string
}

// 15. Intelligent Climate Compensation (Better Thermostat)
export interface ClimateCompensation {
  id?: string
  trv_entity_key: string
  room_temp_sensor_key: string
  window_sensor_key?: string
  target_temperature: number
  delta_factor: number
  compensated_setpoint?: number
  window_open?: boolean
  enabled?: boolean
  updated_at?: string
}

// 16. Long Term Statistics
export interface LongTermStatistic {
  start: string
  end: string
  mean?: number
  min?: number
  max?: number
  sum?: number
  state?: number
}

// 17. Home Finance & Expense Management
export type ExpenseCategory =
  | 'food'
  | 'utilities'
  | 'transport'
  | 'home_maintenance'
  | 'insurance'
  | 'other'

export type ExpenseSubcategory =
  | 'groceries'
  | 'bakery'
  | 'butcher'
  | 'restaurant_takeaway'
  | 'electricity'
  | 'gas_heating'
  | 'water'
  | 'waste_tari'
  | 'telecom_fiber'
  | 'fuel_petrol'
  | 'fuel_diesel'
  | 'ev_charging'
  | 'toll_telepass'
  | 'vehicle_maintenance'
  | 'vehicle_tax_bollo'
  | 'diy_hardware'
  | 'appliances'
  | 'furniture_decor'
  | 'repairs'
  | 'auto_insurance'
  | 'home_insurance'
  | 'health_insurance'
  | 'life_insurance'
  | 'health_pharmacy'
  | 'education'
  | 'leisure_dining'
  | 'clothing'
  | 'miscellaneous'

export interface Expense {
  id: string
  title: string
  amount: number
  currency: string
  category: ExpenseCategory
  subcategory?: ExpenseSubcategory
  expense_date: string
  is_fixed: boolean
  payment_method?: string
  payer_user_id?: string
  notes?: string
  recurring_frequency?: string
  contract_expiry_date?: string
  energy_kwh?: number
  gas_smc?: number
  food_waste_value?: number
  created_at: string
  updated_at: string
}

export interface BudgetTarget {
  id: string
  category: string
  monthly_limit: number
  month_year: string
  created_at: string
  updated_at: string
}

export interface CategoryBreakdown {
  category: ExpenseCategory
  name: string
  total_amount: number
  percentage: number
  count: number
}

export interface MonthlyTrend {
  year_month: string
  month_name: string
  total_amount: number
  utilities_amount: number
  food_amount: number
  transport_amount: number
}

export interface FixedVsVariable {
  fixed_amount: number
  variable_amount: number
  fixed_percentage: number
  variable_percentage: number
}

export interface SavingsInsight {
  id: string
  type: 'energy_tariff' | 'food_waste' | 'contract_expiry' | 'category_spike'
  title: string
  message: string
  potential_saving?: number
  severity: 'info' | 'warning' | 'alert'
  action_url?: string
}

export interface FinancialSummary {
  period: 'day' | 'week' | 'month' | 'year'
  start_date: string
  end_date: string
  total_spent: number
  previous_period_total: number
  percentage_change: number
  daily_average: number
  monthly_budget: number
  remaining_budget: number
  categories: CategoryBreakdown[]
  monthly_trends: MonthlyTrend[]
  fixed_vs_variable: FixedVsVariable
  insights: SavingsInsight[]
}

// ==========================================
// Pantry, Recipes, Meal Planning & Receipt OCR
// ==========================================

export type PantryCategory =
  | 'produce'
  | 'dairy'
  | 'meat_fish'
  | 'bakery'
  | 'pantry'
  | 'frozen'
  | 'drinks'
  | 'condiments'
  | 'snacks'
  | 'household'
  | 'other'

export type PantryUnit = 'pz' | 'g' | 'kg' | 'ml' | 'l' | 'pack'

export type PantryLocation = 'fridge' | 'freezer' | 'pantry' | 'cellar' | 'other'

export interface PantryItem {
  id: string
  name: string
  category: PantryCategory
  quantity: number
  unit: PantryUnit
  min_threshold: number
  expiration_date?: string
  location: PantryLocation
  barcode?: string
  cost?: number
  created_at: string
  updated_at: string
}

export type RecipeCategory =
  | 'first_course'
  | 'main_course'
  | 'side_dish'
  | 'single_dish'
  | 'breakfast'
  | 'dessert'
  | 'snack'

export interface RecipeIngredient {
  id?: string
  recipe_id?: string
  pantry_item_id?: string
  name: string
  quantity: number
  unit: PantryUnit
  is_optional?: boolean
}

export interface Recipe {
  id: string
  title: string
  description?: string
  category: RecipeCategory
  portions_base: number
  prep_time_minutes: number
  cook_time_minutes: number
  instructions: string
  image_url?: string
  ingredients: RecipeIngredient[]
  created_at: string
  updated_at: string
}

export interface CookableRecipeResult {
  recipe: Recipe
  is_cookable: boolean
  available_percent: number
  missing_ingredients?: string[]
}

export type MealType = 'breakfast' | 'lunch' | 'dinner' | 'snack' | 'dessert'
export type MealStatus = 'planned' | 'consumed' | 'skipped'

export interface MealPlan {
  id: string
  plan_date: string // YYYY-MM-DD
  meal_type: MealType
  recipe_id?: string
  recipe?: Recipe
  custom_title?: string
  persons_count: number
  notes?: string
  status: MealStatus
  consumed_at?: string
  created_at: string
  updated_at: string
}

export interface ReceiptItem {
  raw_line?: string
  name: string
  quantity: number
  unit: PantryUnit
  unit_price: number
  total_price: number
  category: PantryCategory
  matched_pantry_id?: string
}

export interface Receipt {
  store_name: string
  date: string // YYYY-MM-DD
  items: ReceiptItem[]
  total_amount: number
  raw_text?: string
  confidence: number
}

export interface ReceiptImportResult {
  items_added: number
  items_updated: number
  expense_id?: string
  expense_amount: number
}

// --- MEDICINE CABINET & EXPIRATION TYPES ---

export type MedicineLocation =
  | 'armadietto_bagno'
  | 'cassetto_notte'
  | 'borsa_viaggio'
  | 'frigo'
  | 'altro'

export type MedicineUnit =
  | 'compresse'
  | 'bustine'
  | 'gocce'
  | 'ml'
  | 'pz'
  | 'confezione'

export interface Medicine {
  id: string
  name: string
  active_ingredient?: string
  dosage?: string
  form?: string
  aic?: string
  barcode?: string
  lot_number?: string
  expiration_date: string
  pao_days: number
  opened_at?: string
  effective_expiration_date: string
  location: MedicineLocation
  requires_cold_chain: boolean
  quantity: number
  unit: MedicineUnit
  min_threshold: number
  prescription_required: boolean
  notes?: string
  created_at: string
  updated_at: string
}

export interface DatamatrixParseResult {
  aic?: string
  gtin?: string
  expiration_date?: string
  lot_number?: string
  serial_number?: string
  raw_code: string
}

export interface AIFARecord {
  aic: string
  name: string
  active_ingredient: string
  dosage: string
  form: string
  unit: MedicineUnit
  default_quantity: number
  requires_cold_chain: boolean
  default_location: MedicineLocation
  default_pao_days: number
  prescription_required: boolean
}

export interface MedicineAlert {
  medicine_id: string
  medicine_name: string
  type: 'expired' | 'expiring_soon_7d' | 'expiring_soon_30d' | 'low_stock' | 'cold_chain_warning'
  severity: 'critical' | 'warning' | 'info'
  days_remaining: number
  message: string
  location: MedicineLocation
}

export type RoomOrientation = 'north' | 'south' | 'east' | 'west'
export type EmitterType = 'radiator' | 'floor_radiant' | 'fancoil'

export interface PredictiveZoneConfig {
  id?: string
  name: string
  area_id?: string
  trv_entity_key: string
  temp_sensor_key: string
  pressure_sensor_key?: string
  humidity_sensor_key?: string
  window_sensor_key?: string
  target_temperature: number
  orientation: RoomOrientation
  window_area_sqm: number
  emitter_type: EmitterType
  thermal_inertia_hours: number
  preheat_enabled: boolean
  solar_cutoff_enabled: boolean
  pressure_boost_enabled: boolean
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export interface PredictiveEvaluation {
  id: string
  zone_id: string
  zone_name: string
  current_temp: number
  target_temp: number
  current_pressure: number
  pressure_trend_hpa_hr: number
  humidity: number
  apparent_temp: number
  outdoor_temp_forecast: number
  direct_radiation_forecast: number
  estimated_solar_gain_kw: number
  cold_front_detected: boolean
  window_open: boolean
  calculated_setpoint: number
  mode: 'comfort' | 'cold_front_boost' | 'solar_cutoff' | 'preheat' | 'eco' | 'frost_protection'
  reason: string
  evaluated_at: string
}

export interface ZoneDashboardStatus {
  config: PredictiveZoneConfig
  latest_evaluation?: PredictiveEvaluation
}

export interface SystemPredictiveStatus {
  total_zones: number
  enabled_zones: number
  cold_front_active: boolean
  solar_cutoff_active: boolean
  zones: ZoneDashboardStatus[]
  evaluated_at: string
}

export interface ThermalProfile {
  zone_id: string
  heat_rate_per_hr: number
  cool_rate_per_hr: number
  outdoor_loss_coeff: number
  solar_boost_coeff: number
  samples_count: number
  last_trained_at: string
}

export interface ThermalCycleObservation {
  zone_id?: string
  cycle_type: 'heating' | 'cooling'
  start_temp: number
  end_temp: number
  avg_outdoor_temp?: number
  solar_radiation?: number
  duration_minutes: number
  observed_at?: string
}

export interface PreheatRecommendation {
  zone_id: string
  target_time: string
  target_temp: number
  current_temp: number
  outdoor_temp: number
  preheat_minutes: number
  start_time: string
  effective_heat_rate: number
  reason: string
}

export interface CoastDownRecommendation {
  zone_id: string
  departure_time: string
  comfort_limit_temp: number
  coast_down_minutes: number
  stop_time: string
  energy_saved_kwh: number
  reason: string
}

export interface FamilyFace {
  id: string
  name: string
  role?: string
  photo_url?: string
  enrolled_at?: string
  embedding?: number[]
}

export interface AuthorizedVehicle {
  plate_number: string
  owner_name: string
  vehicle_model?: string
  auto_open_gate: boolean
  enrolled_at?: string
}

export interface NetworkDiscoveredHost {
  ip: string
  mac: string
  hostname?: string
  vendor?: string
  suggested_plugin?: string
  confidence: number
  open_ports?: number[]
  first_seen_at?: string
  last_seen_at?: string
  is_iot: boolean
  metadata?: Record<string, string>
}

export interface NetworkLeakAlert {
  id: string
  source_ip: string
  source_mac?: string
  device_name?: string
  destination_ip: string
  port: number
  protocol: string
  detected_at: string
  reason: string
}

export interface NetworkGuardianLeaksResponse {
  count: number
  leaks: NetworkLeakAlert[]
  isolated_macs: string[]
}

// 27. OCPP Wallbox & Smart Solar EV Modulation
export type WallboxChargingMode = 'solar_only' | 'solar_optimized' | 'fast_charging' | 'scheduled_night'

export interface WallboxTelemetry {
  charge_point_id: string
  status: string
  active_power_w: number
  offered_amps: number
  delivered_amps: number
  vehicle_soc_pct: number
  energy_wh: number
  last_seen_at?: string
}

export interface WallboxModulationResult {
  charge_point_id: string
  mode: WallboxChargingMode
  target_amps: number
  expected_power_w: number
  solar_surplus_w: number
  home_grid_power_w: number
  reason: string
  timestamp: string
}

export interface WallboxStatusResponse {
  charge_point_id: string
  telemetry: WallboxTelemetry
  modulation: WallboxModulationResult
}

// 28. Vacation Presence Simulator & Commute Tracking
export interface VacationModeStatus {
  enabled: boolean
  jitter_minutes: number
}

export interface VacationSimulatedAction {
  id?: string
  entity_id: string
  command: string
  typical_time: string
  parameters?: Record<string, any>
}

export interface CommuteLocationUpdate {
  user_id: string
  latitude: number
  longitude: number
  speed_kmh: number
}

export interface CommuteStatus {
  user_id: string
  distance_km: number
  eta_minutes: number
  is_approaching: boolean
  preconditioning_active: boolean
  target_climate_temp?: number
  updated_at: string
}

// 29. Store & Marketplace Packages
export interface StorePackage {
  id: string
  package_id: string
  name: string
  type: 'wasm_plugin' | 'blueprint' | 'widget'
  tier?: 'verified' | 'community'
  review_status?: 'approved' | 'pending_review' | 'rejected'
  version: string
  author: string
  description: string
  repository_url?: string
  license?: string
  capabilities?: string[]
  installed: boolean
  installed_at?: string
  icon?: string
}



