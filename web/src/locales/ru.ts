import en from './en.json'

// Russian catalog. English is the fallback for legacy screens while their
// remaining hard-coded copy is migrated to vue-i18n.
const ru = {
  ...en,
  common: {
    ...en.common,
    ok: 'ОК', cancel: 'Отмена', save: 'Сохранить', edit: 'Изменить',
    delete: 'Удалить', search: 'Поиск...', loading: 'Загрузка...',
    error: 'Ошибка', success: 'Операция успешно выполнена',
    refresh: 'Обновить', back: 'Назад', all: 'Все', active: 'Активно',
    disabled: 'Отключено', name: 'Название', description: 'Описание',
    no_data: 'Нет данных'
  },
  nav: {
    ...en.nav,
    sectionOverview: 'Обзор', sectionHome: 'Дом и семья',
    sectionDevices: 'Устройства и помещения', sectionAutomations: 'Автоматизация и мультимедиа',
    sectionSystem: 'Система и безопасность', dashboard: 'Панель управления',
    customDashboards: 'Панели для планшетов', energy: 'Энергия и солнечная энергия',
    helpers: 'Виртуальные помощники', cameras: 'Камеры', schedules: 'Расписания',
    media: 'Мультирум-аудио', mesh: 'Карта Mesh', store: 'Магазин Domus',
    floorplan: 'План дома', chores: 'Домашние дела', shopping: 'Список покупок',
    medicines: 'Домашняя аптечка', predictiveClimate: 'Прогноз климата',
    areas: 'Помещения', devices: 'Устройства', integrations: 'Интеграции',
    automations: 'Автоматизация', scenes: 'Сценарии', history: 'История и показатели',
    audit: 'Журнал аудита безопасности', logbook: 'Журнал событий',
    zones: 'Зоны и геозоны', templates: 'Шаблоны сущностей', family: 'Семья',
    finance: 'Домашние финансы', pantry: 'Кладовая и рецепты',
    settings: 'Настройки', logout: 'Выйти', guide: 'Справка и руководство'
  },
  settings: {
    ...en.settings,
    title: 'Системные настройки',
    subtitle: 'Управление настройками хаба, языком интерфейса и диагностикой.',
    userProfile: 'Профиль пользователя', systemMetrics: 'Показатели работы хаба',
    language: 'Язык приложения', chooseLanguage: 'Выберите язык интерфейса',
    uptime: 'Время работы системы', databaseSize: 'Размер базы данных SQLite',
    mqttBroker: 'Состояние брокера MQTT', wsClients: 'Подключённые клиенты WebSocket',
    restart: 'Перезапустить службу', version: 'Версия Domus',
    osDiagnostics: 'Диагностика устройства Domus OS', hardware: 'Оборудование и ресурсы',
    socTemp: 'Температура SoC', cpuLoad: 'Загрузка процессора', cpuCores: 'Ядра процессора',
    memory: 'Оперативная память', storage: 'Дисковое пространство /data',
    activeSlot: 'Активный слот A/B', slotConfirmed: 'Подтверждено',
    slotUnconfirmed: 'Ожидает подтверждения', containers: 'Активные контейнеры',
    maintenanceMode: 'Режим обслуживания (SSH)',
    maintenanceDesc: 'Временный доступ SSH через порт 22 для диагностики. Автоматически отключается по истечении времени.',
    sshActive: 'SSH включён', sshDisabled: 'SSH отключён (безопасность по умолчанию)',
    enableSsh: 'Включить обслуживание', disableSsh: 'Отключить немедленно',
    durationMinutes: 'Продолжительность (минуты)', authorizedKey: 'Открытый SSH-ключ (необязательно)',
    remainingTime: 'Оставшееся время', exportDiagnostics: 'Экспорт диагностического отчёта',
    exportDesc: 'Скачать JSON-отчёт с телеметрией и очищенной конфигурацией (секреты удалены).',
    appearance: 'Внешний вид и тема',
    appearanceDescription: 'Настройте внешний вид и контрастность интерфейса Domus.',
    themes: { dark: 'Тёмная тема', light: 'Светлая тема', oled: 'Midnight OLED', system: 'Автоматически (система)' },
    themeDescriptions: {
      dark: 'Ночные синие и графитовые оттенки, подходящие для большинства помещений.',
      light: 'Чёткий интерфейс с высокой контрастностью для ярко освещённых помещений.',
      oled: 'Абсолютно чёрный цвет для OLED-дисплеев и снижения энергопотребления.',
      system: 'Автоматически использует цветовую схему устройства или браузера.'
    }
  },
  login: { ...en.login, title: 'Войти в Domus', subtitle: 'Локальная система управления умным домом', username: 'Имя пользователя', password: 'Пароль', submit: 'Войти', error: 'Неверные данные или срок действия сеанса истёк' },
  setup: { ...en.setup, title: 'Первоначальная настройка Domus', subtitle: 'Создайте учётную запись администратора для запуска хаба', adminUser: 'Имя администратора', adminPass: 'Пароль администратора', confirmPass: 'Подтвердите пароль', submit: 'Создать учётную запись и запустить хаб', passMismatch: 'Пароли не совпадают' },
  dashboard: { ...en.dashboard, title: 'Обзор дома', subtitle: 'Контролируйте локальные объекты и подключённые устройства.', totalEntities: 'Всего объектов', onlineDevices: 'Устройства в сети', activeAutomations: 'Активные сценарии', systemLoad: 'Нагрузка хаба', activeLights: 'Включённые лампы', activeSwitches: 'Активные розетки', onlineSensors: 'Датчики в сети', noEntities: 'Объекты не найдены', noMarketplaceResults: 'Расширения не найдены' },
  areas: { ...en.areas, title: 'Помещения дома', subtitle: 'Управляйте комнатами и зонами дома', newArea: 'Новое помещение', editArea: 'Изменить помещение', deleteArea: 'Удалить помещение', emptyTitle: 'Помещения не настроены', createFirst: 'Создать первое помещение', areaName: 'Название помещения', saveSuccess: 'Помещение сохранено', saveError: 'Не удалось сохранить помещение', deleteSuccess: 'Помещение удалено', deleteError: 'Не удалось удалить помещение' },
  devices: { ...en.devices, title: 'Устройства', subtitle: 'Все физические устройства, обнаруженные и подключённые к хабу', emptyTitle: 'Нет устройств', emptyDesc: 'Устройства с выбранными фильтрами не найдены.' },
  integrations: { ...en.integrations, title: 'Интеграции', subtitle: 'Аппаратные мосты и облачные адаптеры, подключённые к Domus', refresh: 'Обновить', reconnect: 'Подключить снова', reconnectError: 'Не удалось восстановить подключение', emptyTitle: 'Интеграции не настроены' },
  history: { ...en.history, title: 'История и аналитика', subtitle: 'Отслеживайте изменения состояний и показания датчиков.', selectEntity: 'Выберите объект', timeRange: 'Период', noData: 'За выбранный период данных нет.', stats: 'Статистика периода', min: 'Минимум', max: 'Максимум', avg: 'Среднее', current: 'Текущее значение' },
  automations: { ...en.automations, title: 'Правила автоматизации', subtitle: 'Настройте события, условия и автоматические действия.', newAutomation: 'Новое правило', manualTrigger: 'Запустить тест', executionHistory: 'История запусков', searchPlaceholder: 'Поиск по названию или описанию...', saveSuccess: 'Правило сохранено', saveError: 'Не удалось сохранить правило', deleteSuccess: 'Правило удалено', deleteError: 'Не удалось удалить правило', emptyTitle: 'Правила не найдены', noResults: 'По вашему запросу ничего не найдено.' },
  scenes: { ...en.scenes, title: 'Сценарии и режимы', subtitle: 'Применяйте настройки нескольких устройств одним нажатием.', newScene: 'Новый сценарий', activate: 'Активировать', saveSuccess: 'Сценарий сохранён', saveError: 'Не удалось сохранить сценарий', deleteSuccess: 'Сценарий удалён', deleteError: 'Не удалось удалить сценарий', emptyTitle: 'Сценарии не настроены' },
  audit: { ...en.audit, title: 'Журнал аудита безопасности', subtitle: 'Неизменяемая запись входов, изменений конфигурации и автоматических действий.', actor: 'Источник', action: 'Действие', target: 'Объект', timestamp: 'Время', payload: 'Данные', noEntries: 'Записи аудита отсутствуют.', showPayload: 'Показать данные', hidePayload: 'Скрыть данные' },
  family: { ...en.family, title: 'Семья и права доступа', subtitle: 'Управляйте участниками дома, назначайте права и отправляйте приглашения.', membersTitle: 'Участники семьи', invitesTitle: 'Ожидающие приглашения', inviteButton: 'Пригласить участника', username: 'Имя пользователя', email: 'Электронная почта', role: 'Роль', joinedAt: 'Дата добавления', actions: 'Действия', changeRole: 'Изменить роль', removeMember: 'Удалить участника', inviteModalTitle: 'Пригласить нового участника', selectRole: 'Выберите роль', expiration: 'Срок действия приглашения', generateInvite: 'Создать ссылку-приглашение', copyLink: 'Копировать ссылку', revoke: 'Отозвать', noMembers: 'Участники не найдены.' },
  register: { ...en.register, title: 'Присоединиться к семье Domus', subtitle: 'Завершите регистрацию, чтобы управлять умным домом.', username: 'Имя пользователя', email: 'Электронная почта (необязательно)', password: 'Пароль', confirmPassword: 'Подтвердите пароль', submit: 'Завершить регистрацию', loading: 'Проверка приглашения...', errorInvalidInvite: 'Ссылка недействительна или срок её действия истёк.', errorPasswordMismatch: 'Пароли не совпадают.', success: 'Регистрация успешно завершена! Добро пожаловать в Domus.', backToLogin: 'Вернуться ко входу' },
  energy: { ...en.energy, title: 'Энергия и солнечная генерация', solar: 'Солнечная генерация', battery: 'Аккумулятор', grid: 'Электросеть', house: 'Потребление дома', import: 'Потребление из сети', export: 'Отдача в сеть', charge: 'Зарядка', discharge: 'Разрядка', minPrice: 'Минимальная цена', avgPrice: 'Средняя цена за 24 ч', maxPrice: 'Максимальная цена', savings: 'Экономия' },
  cameras: { ...en.cameras, title: 'Видеонаблюдение и локальный ИИ', privacy: 'Конфиденциальность и безопасность', privacyDesc: 'Полностью локальная обработка, без облачной передачи', live: 'ЭФИР', active: 'Активно', disabled: 'Отключено', detected: 'Обнаружено', confidence: 'Уверенность', addZone: 'Добавить зону', noCameras: 'Камеры не настроены' },
  shopping: { ...en.shopping, title: 'Общий список покупок семьи', newItem: 'Добавить товар', placeholderName: 'Название товара...', placeholderQty: 'Количество...', addBtn: 'Добавить в список', clearCompleted: 'Удалить выполненные', toBuy: 'Купить', completed: 'Выполнено', addedBy: 'Добавил', completedBy: 'Выполнил' },
  chores: { ...en.chores, title: 'Домашние дела', newChore: 'Новое дело', all: 'Все', pending: 'Ожидает', overdue: 'Просрочено', completed: 'Выполнено', due: 'Срок', assignedTo: 'Назначено', recurrence: 'Повторение', completeBtn: 'Отметить выполненным' },
  helpers: { ...en.helpers, title: 'Виртуальные объекты и помощники', newHelper: 'Новый помощник', boolean: 'Логический переключатель', counter: 'Счётчик', number: 'Числовой регулятор', timer: 'Таймер обратного отсчёта' },
  floorplan: { ...en.floorplan, title: 'План дома', selectFloor: 'Выберите этаж', addFloor: 'Загрузить план этажа', editMarkers: 'Изменить метки устройств' },
  media: { ...en.media, title: 'Мультирум-аудио и воспроизведение', masterVolume: 'Общая громкость', playing: 'Воспроизведение', idle: 'Ожидание', groupSpeakers: 'Объединить колонки' },
  mesh: { ...en.mesh, title: 'Радиодиагностика и карта Mesh', coordinator: 'Координатор', router: 'Маршрутизатор / ретранслятор', endDevice: 'Устройство с батареей', linkQuality: 'Качество связи' },
  store: { ...en.store, title: 'Магазин Domus и расширения', installed: 'Установлено', available: 'Доступно в каталоге', install: 'Установить', open: 'Открыть приложение', uninstall: 'Удалить' },
  schedules: { ...en.schedules, title: 'Расписания и календарь', newSchedule: 'Новое расписание', time: 'Время', days: 'Дни', target: 'Целевой объект' },
  alarm: { ...en.alarm, title: 'Система охранной сигнализации', disarmed: 'Снята с охраны', armedHome: 'Дома', armedAway: 'Вне дома', armedNight: 'Ночной режим', triggered: 'ТРЕВОГА', pin: 'Защитный PIN-код', disarm: 'Снять с охраны', armHome: 'Режим дома', armAway: 'Режим вне дома', armNight: 'Ночной режим' }
}

export default ru
