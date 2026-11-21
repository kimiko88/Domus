import en from './en.json'

// Russian catalog. English is the fallback for legacy screens while their
// remaining hard-coded copy is migrated to vue-i18n.
const ru = {
  ...en,
  common: {
    ...en.common, done: 'Готово',
    ok: 'ОК', cancel: 'Отмена', close: 'Закрыть', save: 'Сохранить', edit: 'Изменить',
    delete: 'Удалить', search: 'Поиск...', loading: 'Загрузка...',
    error: 'Ошибка', success: 'Операция успешно выполнена',
    refresh: 'Обновить', back: 'Назад', all: 'Все', active: 'Активно',
    disabled: 'Отключено', name: 'Название', description: 'Описание',
    no_data: 'Нет данных',
    skipToContent: 'Перейти к основному содержимому',
    offlineReconnecting: 'Соединение с локальным хабом потеряно. Выполняется автоматическое подключение...',
    longTermStats: {"region":"Долгосрочная статистика","title":"Статистика за длительный период","rangeLabel":"Выбрать период","average":"Среднее","minimum":"Минимум","maximum":"Максимум","total":"Итого","chartAria":"Столбчатая диаграмма статистики","sevenDays":"7 дней","month":"Месяц","year":"Год"},
    twoFactor: {"title":"Двухфакторная аутентификация (2FA / TOTP)","close":"Закрыть окно","scanQr":"Отсканируйте этот QR-код приложением-аутентификатором (Google Authenticator, Apple Passwords, 1Password или Authy).","qrConfig":"QR-код настройки","qrSymbol":"Символ QR","manualKey":"Ключ вручную (Base32):","copied":"Скопировано!","copyKey":"Скопировать ключ","enterCode":"2. Введите 6-значный код из приложения:","codePlaceholder":"123456","verifying":"Проверка…","verifyEnable":"Проверить и включить","activated":"2FA успешно включена!","backupDescription":"Сохраните резервные коды в надёжном месте. Каждый код можно использовать только один раз, если вы потеряете устройство.","downloadCodes":"Скачать коды (.txt)","savedClose":"Я сохранил коды, закрыть","invalidCode":"Неверный код TOTP. Повторите попытку.","backupFileTitle":"РЕЗЕРВНЫЕ КОДЫ DOMUS 2FA","backupFileDate":"Дата","backupFileWarning":"Храните эти коды в безопасности."},
    alarmPanel: {"defaultName":"Охранная сигнализация","armedSuccess":"Охрана включена в режиме «{mode}»","armError":"Не удалось включить охрану","enterPinToDisarm":"Введите PIN-код для снятия с охраны","disarmedSuccess":"Охрана успешно снята","invalidPin":"Неверный или недействительный PIN-код","confirmPanic":"ВНИМАНИЕ: будет включена ручная SOS-сигнализация с сиренами. Продолжить?","panicError":"Не удалось активировать SOS","exitEntryDelay":"Задержка выхода / входа","triggered":"ТРЕВОГА АКТИВНА!","sirensActive":"СИГНАЛИЗАЦИЯ АКТИВНА — СИРЕНЫ ВКЛЮЧЕНЫ!","intrusionDetected":"Обнаружено вторжение","pendingHint":"Идёт задержка включения / отключения… Введите PIN-код!","keypadPin":"PIN-код клавиатуры","clearPin":"Очистить","deleteDigit":"Удалить последнюю цифру"},
    automationCard: {"toggleEnabled":"Включить или выключить автоматизацию","runManually":"Запустить вручную","run":"Запустить","executionHistory":"Просмотреть историю запусков","history":"История","edit":"Изменить правило","delete":"Удалить автоматизацию"},
    entityCard: {"decreaseTemperature":"Уменьшить температуру","increaseTemperature":"Увеличить температуру","turnOff":"Выключить {name}","turnOn":"Включить {name}","brightness":"Яркость","deactivate":"Отключить {name}","activate":"Активировать {name}"},
    sceneCard: {"activate":"Активировать","edit":"Изменить сценарий","delete":"Удалить сценарий"},
    appTopbar: {"toggleNavigation":"Переключить меню навигации","actionableNotifications":"Интерактивные уведомления дома","openNotifications":"Открыть интерактивные уведомления","openAssistant":"Открыть ИИ-ассистент Domus","assistant":"Ассистент","intercomTitle":"Видеодомофон и дверной звонок (WebRTC)","openIntercom":"Открыть видеодомофон","intercom":"Домофон","switchToLight":"Переключить на светлую тему","switchToDark":"Переключить на тёмную тему","activateLight":"Включить светлую тему","activateDark":"Включить тёмную тему","wsConnected":"WebSocket подключён","wsDisconnected":"WebSocket отключён","live":"В сети","offline":"Не в сети","logout":"Выйти"},
    notificationsView: {"title":"Интерактивные уведомления","subtitle":"Быстрые команды и интерактивные оповещения дома","close":"Закрыть уведомления","emptyTitle":"Нет ожидающих уведомлений","emptyDesc":"Все события и сигналы дома устранены или помещены в архив.","critical":"Критично / срочно","smart":"Умное уведомление","archive":"Архивировать","archiveNotification":"Архивировать уведомление: {title}","eventPreview":"Предпросмотр события"},
    notifications: {
      actionExecutedTitle: 'Действие выполнено',
      actionExecutedMessage: 'Команда успешно отправлена.',
      actionFailedMessage: 'Не удалось выполнить запрошенное действие.'
    },
    dismissNotification: "Закрыть уведомление",
    errors: { loadFailed: 'Не удалось загрузить данные. Повторите попытку.', createFailed: 'Не удалось создать элемент. Повторите попытку.', updateFailed: 'Не удалось обновить элемент. Повторите попытку.', deleteFailed: 'Не удалось удалить элемент. Повторите попытку.', actionFailed: 'Не удалось выполнить действие. Повторите попытку.', syncFailed: 'Не удалось выполнить синхронизацию. Повторите попытку.', exportFailed: 'Не удалось экспортировать данные. Повторите попытку.', pdfTooLarge: 'Размер PDF превышает ограничение 3 МБ.', imageTooLarge: 'Размер изображения превышает ограничение 4 МБ.', invalidPin: 'Неверный PIN-код.', fileTooLarge: 'Размер выбранного файла превышает {limit} МБ.', fileReadFailed: 'Не удалось прочитать выбранный файл.' },
    pwa: { updateAvailableTitle: "Доступно обновление", updateAvailableDescription: "Доступна новая версия Domus. Обновите приложение, чтобы применить последние изменения.", updateNow: "Обновить", later: "Позже", offlineReadyTitle: "Domus готова к работе офлайн", offlineReadyDescription: "Текущая версия сохранена и доступна без подключения к сети.", dismiss: "Закрыть" },
    wsNotifications: {"alarmTitle":"Сигнализация","alarmState":"Состояние сигнализации: {state}","updatedState":"обновлено","alarmTriggeredTitle":"ТРЕВОГА АКТИВИРОВАНА","alarmTriggeredMessage":"Сработала сигнализация о вторжении! Причина: {reason}","defaultAlarmReason":"Периметральный датчик","alarmDisarmedTitle":"Сигнализация снята","alarmDisarmedMessage":"Система сигнализации успешно снята с охраны.","tagScannedTitle":"Метка отсканирована","tagScannedMessage":"Метка \"{tag}\" успешно обработана.","deviceOnlineTitle":"Устройство в сети","deviceOnlineMessage":"{name} теперь в сети.","deviceOfflineTitle":"Устройство не в сети","deviceOfflineMessage":"{name} не в сети.","deviceDiscoveredTitle":"Устройство обнаружено","deviceDiscoveredMessage":"Обнаружено новое устройство: {name}","sceneActivatedTitle":"Сценарий активирован","sceneActivatedMessage":"Сценарий «{name}» активирован.","automationFailedTitle":"Ошибка автоматизации","automationFailedMessage":"Не удалось выполнить автоматизацию «{name}»: {error}","systemNotification":"Системное уведомление"},
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
    webPush: { title: 'Push-уведомления браузера', subtitle: 'Оповещения от сервера Domus на подключённые устройства.', active: 'Включены', inactive: 'Выключены', loadFailed: 'Не удалось загрузить настройки push-уведомлений.', unsupported: 'Этот браузер не поддерживает push-уведомления или service worker.', serverNotConfigured: 'На сервере не настроены ключи VAPID. См. руководство по настройке.', iosInstallHint: 'На iPhone или iPad сначала добавьте Domus на главный экран и откройте приложение через его значок.', privacyHint: 'Подписки привязаны к вашей учётной записи. Вы можете отключить уведомления здесь в любое время.', permission: 'Разрешение на уведомления', permissionStates: { granted: 'разрешено', denied: 'запрещено', default: 'не запрошено' }, enable: 'Включить push-уведомления', disable: 'Отключить уведомления', sendTest: 'Отправить тестовое уведомление', permissionDenied: 'Разрешение отклонено. Измените его в настройках браузера или устройства.', permissionDismissed: 'Вы не разрешили уведомления.', enabledSuccess: 'Push-уведомления включены для этого устройства.', enableFailed: 'Не удалось включить push-уведомления.', disabledSuccess: 'Это устройство больше не будет получать push-уведомления.', disableFailed: 'Не удалось отключить уведомления на этом устройстве.', preferencesSaved: 'Настройки уведомлений сохранены.', saveFailed: 'Не удалось сохранить настройки.', testSent: 'Тестовое уведомление отправлено.', testFailed: 'Не удалось отправить тестовое уведомление.', householdReminders: 'Домашние напоминания', tasks: 'Задачи к сроку', medicineExpiry: 'Срок годности лекарств', daysBefore: 'Дней до истечения срока', medicineStock: 'Минимальный запас лекарств', foodExpiry: 'Срок годности продуктов', foodStock: 'Минимальный запас продуктов', systemAlerts: 'Системные оповещения', alarms: 'Охранная сигнализация', emergencies: 'Аварии и утечки', deviceOffline: 'Устройства не в сети', automationFailures: 'Сбои автоматизаций', commandFailures: 'Ошибки команд', networkAlerts: 'Сетевые оповещения', cameraAlerts: 'Обнаружения камер', doorbellAlerts: 'Домофон и дверной звонок', financeAlerts: 'Финансовые оповещения', zeroDaysHint: 'Укажите 0 дней, чтобы отключить предварительное напоминание о сроке годности.' },
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
    coreVersion: "Версия ядра",
    webVersion: "Версия веб-интерфейса",
    database: "База данных",
    architecture: "Целевая архитектура",
    accessStatus: "Статус доступа",
    keyFingerprint: "Отпечаток ключа",
    duration15: "15 минут",
    duration30: "30 минут (по умолчанию)",
    duration60: "60 минут (1 час)",
    duration120: "120 минут (максимум 2 часа)",
    otaTitle: "Обновления прошивки OTA",
    otaDesc: "Обновляйте устройства Zigbee, ESPHome и Z-Wave одним нажатием без дополнительного программного обеспечения.",
    upToDate: "Обновлено",
    nfcTitle: "Физические NFC- и QR-метки",
    nfcDesc: "Сканируйте NFC-наклейки смартфоном, чтобы мгновенно запускать сценарии и автоматизации.",
    backupTitle: "Аварийное восстановление и зашифрованная резервная копия",
    backupPassphrase: "Парольная фраза шифрования (AES-256-GCM)",
    backupPlaceholder: "Введите надёжную парольную фразу...",
    lastSnapshots: "Последние сохранённые снимки:",
    vacationTitle: "Симуляция присутствия (отпуск)",
    vacationStatus: "Состояние режима отпуска",
    timeJitter: "Временное отклонение (jitter)",
    learningPattern: "Модель обучения",
    learningDays: "14 дней непрерывного обучения",
    rfHealthTitle: "Состояние устройств и диагностика RF",
    waterAnomalyTitle: "Аномалии водоснабжения и главный клапан",
    flowMonitoring: "Состояние мониторинга потока",
    flowNormal: "Норма (утечек нет)",
    nightlySafeStop: "Ночной порог безопасного отключения",
    continuousFlowDuration: "40 минут непрерывного потока",
    mainValveControl: "Управление главным клапаном",
    authorizedKeyPlaceholder: "Вставьте открытый SSH-ключ (необязательно)...",
    role: 'Роль', userId: 'ID пользователя', twoFactor: 'Двухфакторная аутентификация',
    enabled: 'Включена', notEnabled: 'Отключена', configure: 'Настроить', disable: 'Отключить',
    confirmDisableTwoFactor: 'Для отключения двухфакторной аутентификации необходимо подтвердить пароль и код второго фактора. Продолжить?',
    promptPassword: 'Введите текущий пароль', promptTwoFactorCode: 'Введите текущий код TOTP или код восстановления',
    twoFactorDisabled: 'Двухфакторная аутентификация успешно отключена.',
    twoFactorInitError: 'Не удалось настроить двухфакторную аутентификацию: {message}',
    twoFactorDisableError: 'Не удалось отключить двухфакторную аутентификацию: {message}',
    tagScanned: 'Метка «{name}» успешно отсканирована. Действие выполнено.',
    promoteLeaderSuccess: 'Повышение до лидера успешно выполнено!', promoteNodeSuccess: 'Узел успешно повышен!',
    backupSuccess: 'Зашифрованная резервная копия создана и сохранена в облаке/локальном NAS.',
    backupError: 'Не удалось создать резервную копию: {message}',
    confirmRestore: 'Вы действительно хотите запустить аварийное восстановление системы?',
    restoreSuccess: 'Восстановление завершено успешно. Система работает нормально.',
    restoreError: 'Ошибка восстановления: {message}', valveOpen: 'Главный клапан ОТКРЫТ', valveClosed: 'Главный клапан ЗАКРЫТ в целях безопасности',
    appearanceDescription: 'Настройте внешний вид и контрастность интерфейса Domus.',
    themes: { dark: 'Тёмная тема', light: 'Светлая тема', oled: 'Midnight OLED', system: 'Автоматически (система)' },
    themeDescriptions: {
      dark: 'Ночные синие и графитовые оттенки, подходящие для большинства помещений.',
      light: 'Чёткий интерфейс с высокой контрастностью для ярко освещённых помещений.',
      oled: 'Абсолютно чёрный цвет для OLED-дисплеев и снижения энергопотребления.',
      system: 'Автоматически использует цветовую схему устройства или браузера.'
    }
  },
  finance: { ...en.finance, ...{"expenseForm":{"editTitle":"Изменить расход","createTitle":"Добавить новый расход","close":"Закрыть окно","category":"Категория","selectCategory":"Выберите категорию","subcategory":"Подкатегория","paymentMethod":"Способ оплаты","paymentMethods":{"card":"Банковская карта","cash":"Наличные","bank_transfer":"Банковский перевод","direct_debit":"Автоматическое списание"},"description":"Описание / Название *","descriptionPlaceholder":"Например, покупки на неделю, счёт за электричество...","amount":"Сумма (€) *","date":"Дата расхода","costType":"Тип расходов","fixed":"Постоянный расход","variable":"Переменный расход","fixedHint":"Обязательные расходы (ипотека, страховка, подписки)","variableHint":"Гибкие расходы (покупки, потребление, досуг)","energy":"Оплаченная энергия (кВт·ч) — связь с Domus","energyPlaceholder":"Например, 360 кВт·ч для расчёта тарифа","gas":"Потребление газа (Smc)","gasPlaceholder":"Например, 85 Smc","contractExpiry":"Истечение договора (напоминание за 30 дней)","contractHint":"Domus напомнит за 30 дней, чтобы вы могли пересмотреть тариф.","foodWaste":"Просроченные / выброшенные продукты (€)","foodWastePlaceholder":"0,00 € (оценка стоимости пищевых отходов)","notes":"Дополнительные заметки","notesPlaceholder":"Детали чека или покупки...","cancel":"Отмена","update":"Обновить расход","save":"Сохранить расход","subcategories":{"groceries":"Супермаркет","bakery":"Пекарня","butcher":"Мясная лавка","restaurant_takeaway":"Ресторан / навынос","electricity":"Электричество","gas_heating":"Газ / отопление","water":"Вода","waste_tari":"Отходы (TARI)","telecom_fiber":"Интернет / телефон","fuel_petrol":"Бензин","fuel_diesel":"Дизель","ev_charging":"Зарядка электромобиля","toll_telepass":"Платные дороги","vehicle_maintenance":"Обслуживание автомобиля","vehicle_tax_bollo":"Транспортный налог","diy_hardware":"Ремонт / инструменты","appliances":"Бытовая техника","furniture_decor":"Мебель","repairs":"Ремонт","auto_insurance":"Автострахование","home_insurance":"Страхование жилья","health_insurance":"Медицинское страхование","life_insurance":"Страхование жизни","health_pharmacy":"Здоровье / аптека","education":"Образование","leisure_dining":"Досуг / рестораны","clothing":"Одежда","miscellaneous":"Разное"}},"budgetForm":{"title":"Установить месячный бюджет","close":"Закрыть окно","description":"Установите максимальный лимит расходов на текущий месяц. Domus предупредит, если расходы превысят 80% или заданный предел.","limit":"Месячный лимит расходов (€) *","placeholder":"Например, 1500","cancel":"Отмена","save":"Сохранить бюджет"},"kpis":{"region":"Основные финансовые показатели","totalSpent":"Всего потрачено","reduced":"Расходы снизились по сравнению с предыдущим периодом","increased":"Расходы выросли по сравнению с предыдущим периодом","previous":"по сравнению с предыдущим периодом","monthlyBudget":"Месячный бюджет","spent":"потрачено","max":"Макс.:","dailyAverage":"Средние расходы в день","averageFor":"Средний темп расходов за период: {period}","fixedVsVariable":"Постоянные и переменные","fixed":"Постоянные","variableShort":"Перем.","essential":"Обязательные:","flexible":"Гибкие:"},"insights":{"region":"Умные советы по экономии","title":"Умные советы по экономии","badge":"Совет Domus","potential":"Расчётная потенциальная экономия:","emptyTitle":"Отлично! Не обнаружено необычных расходов или потерь.","emptyDesc":"Потребление энергии, сроки годности и управление запасами оптимизированы.","saving":"Экономия:","actionAria":"Перейти к действию для: {title}","resolve":"Исправить и оптимизировать"},"charts":{"categoriesRegion":"Расходы по категориям","categoriesTitle":"Распределение по категориям (кольцевая диаграмма)","reset":"Сбросить фильтр","emptyCategory":"За этот период расходы не зарегистрированы.","chartAria":"Кольцевая диаграмма расходов по категориям","total":"Всего","trendRegion":"Сезонная динамика расходов за последние 12 месяцев","trendTitle":"Динамика расходов — последние 12 месяцев","trendSubtitle":"Сезонность счетов и потребления (зима и лето)","emptyTrend":"Исторические данные отсутствуют.","monthlyHistogram":"Гистограмма расходов по месяцам за 12 месяцев","legend":{"utilities":"💡 Счета и коммунальные услуги","food":"🛒 Продукты и покупки","transport":"⛽ Топливо и автомобиль","other":"🏷️ Прочее и постоянные"},"monthTotal":"Итого за месяц: {amount}"},"pageAria":"Домашний бюджет и управление расходами","title":"Домашний бюджет и расходы","subtitle":"Контролируйте расходы семьи, анализируйте потребление и разумно экономьте","selectPeriod":"Выберите период","periods":{"day":"День","week":"Неделя","month":"Месяц","year":"Год"},"budget":"Бюджет","budgetAria":"Установить месячный бюджет","newExpense":"Новый расход","newExpenseAria":"Добавить новый расход","transactionsRegion":"Транзакции и расходы","transactionsTitle":"Записанные транзакции и расходы","movementsCount":"Найдено операций: {count}","categoryFilter":"Фильтр по категории","searchPlaceholder":"Поиск расхода или супермаркета...","searchAria":"Поиск транзакций","noExpenses":"Расходы не найдены","noExpensesDesc":"Добавьте первый расход или измените параметры поиска.","recordNow":"Добавить расход","date":"Дата","description":"Описание","category":"Категория","type":"Тип","amount":"Сумма","actions":"Действия","expiry":"Истекает: {date}","fixed":"Постоянный","variable":"Переменный","editExpenseAria":"Изменить расход: {title}","deleteExpenseAria":"Удалить расход: {title}","confirmDelete":"Вы действительно хотите удалить этот расход?","categories":{"all":"Все категории","food":"Продукты и покупки","utilities":"Счета и коммунальные услуги","transport":"Топливо и автомобиль","home_maintenance":"Дом и материалы","insurance":"Страхование","other":"Другое / Прочее"}} },
  logbook: { ...en.logbook, ...{"title":"Журнал событий","subtitle":"Контекстная хроника активности умного дома","refresh":"Обновить","refreshing":"Загрузка...","filters":"Фильтры событий","searchPlaceholder":"Поиск события, объекта или пользователя...","searchAria":"Поиск в журнале событий","domainFilter":"Фильтр по типу","loading":"Загрузка событий...","emptyTitle":"Нет записей о событиях","emptyDesc":"Здесь будут отображаться события при изменении состояния объектов или автоматизаций.","contextUser":"Запущено пользователем {name}","contextUserUnknown":"Запущено пользователем","contextAutomation":"Запущено автоматизацией {name}","contextAutomationUnknown":"Запущено автоматизацией","contextScript":"Выполнено скриптом {name}","contextScriptUnknown":"Выполнено скриптом","contextSystem":"Системное событие","domains":{"all":"Все","light":"Освещение","switch":"Розетки","climate":"Климат","alarm":"Сигнализация","person":"Люди","automation":"Автоматизации"}} },
  zones: { ...en.zones, ...{"title":"Географические зоны и присутствие","subtitle":"Геозоны, периметры обнаружения и отслеживание устройств членов семьи","newZone":"Новая зона","createZoneAria":"Создать новую зону","mapAria":"Визуальная карта зон","mapTitle":"Интерактивная карта геозон","zonesConfigured":"Настроенные зоны","zonesListAria":"Список зон","activeZones":"Активные зоны","loading":"Загрузка зон...","empty":"Зоны не созданы. Для начала создайте Дом, Работу или Школу.","passive":"Пассивная","radius":"Радиус: {radius} м","deleteZoneAria":"Удалить зону {name}","trackedPeopleAria":"Отслеживаемые люди","trackedPeople":"Люди и отслеживание","noTrackedPeople":"Ни одному члену семьи не назначен трекер устройства.","vacationAria":"Симулятор присутствия в режиме отпуска","vacationTitle":"Симулятор присутствия — режим отпуска","vacationDesc":"Воспроизводит повседневные привычки (освещение, жалюзи и телевизор) с гауссовым отклонением ±{minutes} мин., чтобы затруднить вторжение во время вашего отсутствия.","active":"Включено","inactive":"Отключено","toggleVacationAria":"Включить или отключить режим отпуска","activeBanner":"Симуляция включена: привычки семьи автоматически воспроизводятся со случайными отклонениями.","inactiveBanner":"Режим отпуска отключён. Включите его, чтобы воспроизводить домашние привычки во время отсутствия.","simulatedActionsTitle":"Записанные симулированные действия","addAction":"Добавить симулируемое действие","addActionAria":"Добавить симулируемое действие","emptyActions":"Нет симулируемых действий. Добавьте освещение, жалюзи или телевизоры для воспроизведения в отпуске.","actionModalTitle":"Добавить симулируемое действие","entityId":"ID объекта (например, light.living_room)","command":"Команда (например, turn_on, turn_off)","typicalTime":"Обычное время (ЧЧ:ММ)","cancel":"Отмена","saveAction":"Сохранить действие","createModalTitle":"Создать географическую зону","zoneName":"Название зоны","zoneNamePlaceholder":"Например, Дом, Офис, Спортзал","latitude":"Широта","latitudePlaceholder":"Например, 45.4642","longitude":"Долгота","longitudePlaceholder":"Например, 9.1900","radiusLabel":"Радиус обнаружения: {radius} м","passiveZone":"Пассивная зона (не меняет основной статус присутствия)","saveZone":"Сохранить зону","invalidCoordinates":"Введите корректные координаты: широта от -90 до 90, долгота от -180 до 180.","confirmDelete":"Вы действительно хотите удалить эту зону?","zoneAria":"Зона {name}, радиус {radius} м"} },
  templates: { ...en.templates, ...{"title":"Шаблонные объекты и виртуальные датчики","subtitle":"Создавайте формулы, условную логику и датчики с расчётом в реальном времени","newSensor":"Новый шаблонный датчик","createSensorAria":"Создать шаблонный датчик","playgroundAria":"Проверка выражений шаблонов","playgroundTitle":"Проверка выражений в реальном времени (Jinja2 / Formula Evaluator)","evaluating":"Вычисление...","evaluate":"Вычислить шаблон ▶","editorLabel":"Редактор тестового шаблона","editorPlaceholder":"Например, {{ states('sensor.living_room_temp') | float * 1.8 + 32 }} °F","result":"Результат:","previewHint":"Введите шаблон и нажмите «Вычислить шаблон»","entitiesAria":"Настроенные шаблонные датчики","activeSensors":"Активные шаблонные датчики ({count})","loading":"Загрузка шаблонных датчиков...","empty":"Шаблонные датчики не настроены. Создайте собственные формулы или виртуальные датчики.","formula":"Формула / выражение:","unit":"Единица: {unit}","deviceClass":"Класс: {class}","deleteSensorAria":"Удалить шаблонный датчик {name}","confirmDelete":"Удалить этот виртуальный шаблонный датчик?","createTitle":"Создать шаблонный датчик","name":"Название датчика","namePlaceholder":"Например, Общая мощность дома","entityId":"ID объекта","entityPlaceholder":"sensor.total_power","domain":"Домен","sensorNumeric":"sensor (Числовой / строковый)","binarySensor":"binary_sensor (Логический вкл./выкл.)","stateTemplate":"Шаблон состояния","statePlaceholder":"Введите выражение состояния Jinja2","unitOfMeasurement":"Единица измерения","unitPlaceholder":"Например, W, kW, °C, %","deviceClassLabel":"Класс устройства","deviceClassPlaceholder":"power, energy, temperature","cancel":"Отмена","create":"Создать датчик","previewError":"Ошибка: {message}"} },
  medicines: { ...en.medicines, ...{"title":"Домашняя аптечка и сроки годности","subtitle":"Учёт лекарств, расчёт срока после вскрытия (PAO) и предупреждения о холодовой цепи","quickEntry":"Быстрое добавление","total":"Всего лекарств","expiring30":"Истекает в течение 30 дней","expired":"Просрочено","coldStorage":"Холодильник (2–8°C)","lowStock":"Мало запасов","alertsTitle":"Предупреждения аптечки ({count})","searchPlaceholder":"Поиск по названию, действующему веществу или коду AIC...","searchAria":"Поиск лекарств","clearSearchAria":"Очистить поиск","location":"Место хранения:","locationFilter":"Фильтр по месту хранения","locations":{"all":"Все места","cabinet":"Шкафчик в ванной","nightstand":"Прикроватная тумбочка","travel":"Сумка / поездка","fridge":"Холодильник (2–8°C)","other":"Другое"},"status":"Статус:","statusFilter":"Фильтр по статусу","statuses":{"all":"Все лекарства","expiring30":"Истекает в течение 30 дней","expiring7":"Истекает в течение 7 дней","expired":"Только просроченные","fridge":"Только холодильник","lowStock":"Только малый запас"},"emptyTitle":"Лекарства не найдены","emptyDesc":"Добавьте лекарства, отсканировав код Datamatrix или выполнив поиск в каталоге AIFA.","firstMedicine":"Добавить первое лекарство","fridgeTitle":"Хранить в холодильнике при 2–8°C","fridgeShort":"Холодильник 2–8°C","position":"Место хранения:","quantity":"Количество:","minimumStock":"Минимальный запас: {count}","openedOn":"Открыто {date} (PAO: {days} дн.)","effectiveExpiry":"Фактический срок годности:","sealed":"Запечатано (срок действия {days} дней после вскрытия)","openToday":"Отметить как открытое сегодня","consumeDose":"Использовать 1 дозу","editMedicine":"Изменить лекарство","removeMedicine":"Удалить лекарство","confirmDelete":"Вы действительно хотите удалить «{name}» из аптечки?","editTitle":"Изменить лекарство","addTitle":"Быстрое добавление лекарства","scanTitle":"Сканируйте GS1 Datamatrix или введите код AIC","datamatrixPlaceholder":"Отсканируйте или вставьте GS1 Datamatrix либо 9-значный код AIC","decode":"Декодировать","examples":"Примеры:","aifaSearch":"Быстрый поиск в каталоге лекарств AIFA:","aifaPlaceholder":"Введите название лекарства или действующее вещество...","medicineName":"Название лекарства *","medicineNamePlaceholder":"Например, Парацетамол 1000 мг","activeIngredient":"Действующее вещество","activeIngredientPlaceholder":"Например, Парацетамол","aicCode":"Код AIC (9 цифр)","aicPlaceholder":"Например, 034234035","packageExpiry":"Срок годности упаковки *","lotNumber":"Номер партии","lotPlaceholder":"Например, LOT982","remainingQuantity":"Остаток","unit":"Единица","units":{"tablets":"Таблетки","sachets":"Пакетики","drops":"Капли","ml":"мл","pieces":"Штуки","package":"Упаковка"},"storageLocation":"Место хранения","minimumAlert":"Минимальный запас для предупреждения","minimumPlaceholder":"Например, 2","paoTitle":"Срок после вскрытия (PAO)","paoDesc":"Для глазных капель, сиропов и мазей, срок годности которых отсчитывается с момента вскрытия.","paoDays":"Срок после вскрытия (дни)","paoPlaceholder":"0 = не применяется (например, 28)","alreadyOpened":"Упаковка уже открыта","openedDate":"Дата вскрытия","coldChain":"Требуется хранение в холодильнике (2–8°C)","prescription":"Требуется рецепт","cancel":"Отмена","saveChanges":"Сохранить изменения","register":"Сохранить лекарство","parseError":"Неизвестный или недействительный код","expiredStatus":"Просрочено ({date})","expiringSoon":"Истекает через несколько дней ({date})","expiring":"Скоро истекает ({date})","expiryShort":"Срок: {date}","locationsShort":{"cabinet":"Шкафчик в ванной","nightstand":"Прикроватная тумбочка","travel":"Сумка / поездка","fridge":"Холодильник (2–8°C)","other":"Другое"}} },
  login: { ...en.login, title: 'Войти в Domus', subtitle: 'Локальная система управления умным домом', username: 'Имя пользователя', password: 'Пароль', submit: 'Войти', error: 'Неверные данные или срок действия сеанса истёк', kioskPin: 'PIN режима киоска', passwordTab: 'Пароль', kioskTab: 'PIN-код киоска / гостя', unlockKiosk: 'Разблокировать киоск', missingCredentials: 'Введите имя пользователя и пароль', missingKioskPin: 'Введите PIN-код киоска', invalidKioskPin: 'PIN-код киоска недействителен или срок его действия истёк' },
  setup: { ...en.setup, title: 'Первоначальная настройка Domus', subtitle: 'Создайте учётную запись администратора для запуска хаба', adminUser: 'Имя администратора', adminPass: 'Пароль администратора', confirmPass: 'Подтвердите пароль', submit: 'Создать учётную запись и запустить хаб', passMismatch: 'Пароли не совпадают' , passwordMinLength: "Пароль должен содержать не менее 8 символов" , createAdminError: "Не удалось создать учётную запись администратора" , usernamePlaceholder: "admin" , passwordPlaceholder: "Не менее 8 символов" , confirmPasswordPlaceholder: "Повторите пароль" },
  dashboard: { ...en.dashboard,
    ...en.dashboard,
    title: 'Обзор дома', subtitle: 'Контролируйте локальные объекты и подключённые устройства.',
    totalEntities: 'Всего объектов', onlineDevices: 'Устройства в сети', activeAutomations: 'Активные сценарии',
    systemLoad: 'Нагрузка хаба', activeLights: 'Включённые лампы', activeSwitches: 'Активные розетки',
    onlineSensors: 'Датчики в сети', activeCount: 'Активно', detectedCount: 'Обнаружено',
    filterAll: 'Все', filterAlarm: 'Сигнализация', filterLights: 'Освещение', filterSwitches: 'Выключатели', filterSensors: 'Датчики',
    filterClimate: 'Климат', filterMarketplace: 'Каталог расширений',
    marketplaceTitle: 'Каталог и расширения', marketplaceSubtitle: 'Изолированные модули WASI, проверенные сценарии и пользовательские виджеты',
    exploreStore: 'Открыть магазин Domus', installedExtensions: 'Установленные расширения', showcaseBadge: 'Рекомендуем',
    allTypes: 'Все типы', wasmPlugins: 'Плагины WASM', blueprints: 'Шаблоны сценариев', widgets: 'Виджеты интерфейса',
    oneClickInstall: 'Установка в один клик', searchMarketplace: 'Поиск в каталоге...',
    noMarketplaceResults: 'Расширения не найдены', noMarketplaceResultsDesc: 'Измените поисковый запрос или фильтр категорий.',
    noEntities: 'Объекты не найдены', noEntitiesDesc: 'Для этого фильтра нет настроенных устройств. Подключите интеграцию или добавьте устройства, чтобы они появились здесь.',
    view: {
      ...en.dashboard.view,
      climateThermostat: 'Климат и термостат', batteryLabel: 'Батарея ({value} %)',
      defaultProfileLivingRoom: 'Планшет гостиной (основной)', defaultProfileKitchen: 'Планшет кухни и прихожей',
      defaultProfileNight: 'Планшет спальни и ночного режима', roomLivingRoom: 'Гостиная', roomKitchen: 'Кухня', roomBedroom: 'Спальня',
      customTitle: 'Пользовательская панель', customSubtitle: 'Интерактивные виджеты для настенных планшетов и режима киоска',
      selectTabletProfile: 'Выбрать профиль планшета', configureKiosk: 'Настроить режим киоска и заставку',
      configureKioskButton: 'Настроить киоск', startWallTablet: 'Включить режим настенного планшета',
      kioskMode: 'Режим киоска', exitKiosk: 'Выйти из режима киоска',
      heatingActive: 'Отопление включено', current: 'Текущее значение', target: 'Целевое значение',
      livingRoomTemperature: 'Температура в гостиной (динамика за 6 часов)',
      minimumTemperature: 'Мин.: {value}', risingTemperature: 'Повышается ({value})', maximumTemperature: 'Макс.: {value}',
      instantEnergyFlow: 'Текущий поток энергии', selfConsumption: '100% собственного потребления',
      solar: 'Солнечная энергия', feedIn: 'Отдача в сеть', home: 'Дом',
      sunnyWeather: '{temperature} · Солнечно', airQuality: 'Качество воздуха: {quality} ({value})', excellent: 'Отлично',
      activateDomusHint: 'Коснитесь экрана или подойдите ближе, чтобы активировать Domus',
      kioskSettingsTitle: 'Настройки киоска и настенного планшета', screensaverTimeout: 'Время бездействия до заставки',
      secondsQuickTest: '15 секунд (быстрая проверка)', seconds: '{value} сек.', minuteRecommended: '1 минута (рекомендуется)',
      minutes: '{value} мин.', neverAlwaysOn: 'Никогда (экран всегда включён)',
      screensaverTimeoutHint: 'Время без касаний, после которого экран затемняется или включается заставка.',
      wakeOnPresence: 'Пробуждение по камере или движению',
      wakeOnPresenceDesc: 'Использовать фронтальную камеру планшета для обнаружения приближающегося человека.',
      showClockDate: 'Показывать часы и дату', showClockDateDesc: 'Отображать крупные часы во время заставки.',
      showWeatherAirQuality: 'Показывать погоду и качество воздуха',
      showWeatherAirQualityDesc: 'Виджет погоды с местной температурой и индексом частиц PM2.5.',
      exitPin: 'PIN-код для выхода (необязательно)', exitPinPlaceholder: 'Например, 1234 (оставьте пустым, чтобы отключить)',
      exitPinHint: 'Не позволяет детям или посетителям выйти из режима киоска или открыть другие разделы.',
      saveConfiguration: 'Сохранить настройки', unlockKioskTitle: 'Разблокировать режим киоска',
      unlockKioskDesc: 'Введите PIN-код для выхода из режима планшета', invalidPin: 'Неверный PIN-код. Попробуйте ещё раз.',
      unlock: 'Разблокировать', entityFilters: 'Фильтры объектов и разделов панели',
      openStoreCatalog: 'Открыть полный каталог магазина Domus', searchExtensions: 'Поиск расширений в каталоге',
      clearSearch: 'Очистить поиск', filterModuleType: 'Фильтр по типу модуля',
      packageDetails: 'Сведения о пакете {name}', uninstallPackage: 'Удалить пакет {name}', installPackage: 'Установить пакет {name}',
      viewAllExtensions: 'Просмотреть все расширения в магазине Domus', closeModuleDetails: 'Закрыть сведения о модуле',
      author: 'Автор', license: 'Лицензия', installDate: 'Дата установки', wasiPermissions: 'Разрешения песочницы WASI',
      viewSourceFor: 'Просмотреть исходный код {name}', viewSource: 'Просмотреть исходный код'
    }
  },
  areas: { ...en.areas, icons: { livingRoom: 'Гостиная', kitchen: 'Кухня', bedroom: 'Спальня', bathroom: 'Ванная', office: 'Кабинет', garden: 'Сад', terrace: 'Терраса', garage: 'Гараж', cinema: 'Домашний кинотеатр / ТВ', other: 'Другое' }, title: 'Помещения дома', subtitle: 'Управляйте комнатами и зонами дома', newArea: 'Новое помещение', editArea: 'Изменить помещение', deleteArea: 'Удалить помещение', emptyTitle: 'Помещения не настроены', createFirst: 'Создать первое помещение', areaName: 'Название помещения', saveSuccess: 'Помещение сохранено', saveError: 'Не удалось сохранить помещение', deleteSuccess: 'Помещение удалено', deleteError: 'Не удалось удалить помещение', nameRequired: 'Укажите название помещения.', createArea: 'Создать помещение', areaNamePlaceholder: 'Например, гостиная', sortOrder: 'Порядок отображения' },
  devices: { ...en.devices, title: 'Устройства', subtitle: 'Все физические устройства, обнаруженные и подключённые к хабу', emptyTitle: 'Нет устройств', emptyDesc: 'Устройства с выбранными фильтрами не найдены.', view: {"refreshing":"Обновление…","refresh":"Обновить","scanning":"Сканирование...","scanLan":"Сканировать LAN","scanLanTitle":"Поиск новых IoT-устройств в локальной сети","pairMatterTitle":"Подключить устройство Matter с помощью QR-кода или кода сопряжения","pairMatter":"Подключить Matter / Thread","discoveredTitle":"Устройства, обнаруженные в локальной сети (Zero-Config)","configure":"Настроить","searchPlaceholder":"Поиск по имени, производителю, модели или IP…","integration":"Интеграция","allIntegrations":"Все интеграции","noFilteredDevices":"Нет устройств, соответствующих фильтрам","adjustFilters":"Измените поисковый запрос или фильтр интеграции.","resetFilters":"Сбросить фильтры","configureDiscovered":"Настроить обнаруженное устройство","close":"Закрыть","deviceName":"Название устройства","deviceNamePlaceholder":"Например, свет в гостиной, розетка стиральной машины","roomArea":"Комната / зона","manufacturer":"Производитель","ipAddress":"IP-адрес","recommendedIntegration":"Рекомендуемая интеграция","matterCommissioning":"Подключение Matter и Thread","matterIntro":"Подключите новое устройство Matter (розетку, лампу, замок или термостат) к локальной сети Thread Domus.","matterPairingCode":"Код сопряжения Matter (11 или 21 цифра)","matterCodePlaceholder":"Например, 34970-112233445","matterCodeHint":"Код указан на задней панели устройства или рядом с QR-кодом на упаковке.","matterNegotiating":"Согласование учётных данных Matter...","matterNegotiatingDesc":"Обмен ключами PASE, проверка сертификата DAC и подключение к сети Thread Domus.","matterPaired":"Устройство Matter подключено!","matterPairedDesc":"Узел аутентифицирован и работает в локальной mesh-сети Thread.","startPairing":"Начать сопряжение"} },
  integrations: { ...en.integrations, title: 'Интеграции', subtitle: 'Аппаратные мосты и облачные адаптеры, подключённые к Domus', refresh: 'Обновить', reconnect: 'Подключить снова', reconnecting: 'Повторное подключение: {name}…', reconnectError: 'Не удалось восстановить подключение', emptyTitle: 'Интеграции не настроены', emptyDesc: 'Настройте адаптеры MQTT, Zigbee2MQTT или ESPHome в файле конфигурации, чтобы подключить их.', view: { genericLanHost: 'Универсальный LAN-хост', selectedForLocalTuya: 'Выбрано для LocalTuya: {count}',"lanGuardianTitle":"LAN & IoT Guardian","smartLifeCloud":"Smart Life Cloud","bridgeInfo":"Аппаратные мосты и протоколы, подключённые к локальной системе Domus.","localScanTitle":"Сканирование локальной сети без облака","localScanDesc":"Обнаружение устройств Shelly, Philips Hue, Sonoff LAN, Tasmota и WLED в домашней сети Wi-Fi.","ipAddressPlaceholder":"IP-адрес (например, 192.168.1.150)","ipProbe":"Проверить IP","localOnly":"100% локально","oneClickEntities":"Объекты, созданные одним нажатием:","pendingPairingEmpty":"Нет устройств, ожидающих сопряжения","zeroCloud":"Без облака","protocolLabel":"Протокол","quickGuide":"Краткое руководство","supportedModels":"Поддерживаемые модели","networkGuardianTitle":"IoT Network Guardian и изоляция LAN","detectedLanHosts":"Обнаруженные узлы LAN","quarantinedDevices":"Устройства в карантине","wanLeaksCaught":"Перехваченные утечки WAN","externalAttemptsTitle":"Попытки внешнего подключения (утечки WAN)","noLeaksTitle":"Утечки не обнаружены","noLeaksDesc":"Все отслеживаемые устройства IoT соответствуют требованиям и работают строго автономно в локальной сети.","unlock":"Разблокировать","unblock":"Снять блокировку","quarantine":"Поместить в карантин","hostMapTitle":"Карта обнаруженных узлов LAN (OUI и отпечатки портов)","arpCacheEmpty":"В кэше ARP нет узлов. Нажмите «Сканировать ARP», чтобы обнаружить устройства.","isolated":"Изолировано","mac":"MAC","plugin":"Плагин","openPorts":"Открытые порты","noRecentProbe":"Недавних проверок нет","runTcpProbe":"Проверить TCP-порты, обычно используемые IoT (554, 1883, 6053, 502, 3671)","isolate":"Изолировать","tuyaSetupTitle":"Smart Life и Tuya (упрощённый режим UID + LAN)","tuyaSetupDesc":"Подключите Smart Life, отсканировав QR-код на iot.tuya.com (grant_type=1): без всплывающих окон, OEM-приложения и OAuth 2.0. При первом использовании Domus получает Local Key для полностью локальной работы в LAN.","refreshDevices":"Обновить устройства","disconnect":"Отключить","serverToServer":"Server-to-Server (grant_type=1) активен","tuyaConnectTitle":"Как подключить Smart Life к Domus (без OAuth 2.0)","dataCenterInstruction":"В правом верхнем углу над таблицей убедитесь, что выбран центр обработки данных Central Europe Data Center.","devicesAccountInstruction":"Откройте вкладку Devices, затем подкладку Link Tuya App Account.","qrAccountInstruction":"Не используйте Configure OAuth 2.0 Authorization. Нажмите синюю кнопку Add App Account, чтобы отобразить QR-код.","smartLifeScanInstruction":"Откройте Smart Life на смартфоне, перейдите в раздел «Я», нажмите сканер QR в правом верхнем углу, отсканируйте код, подтвердите вход, выберите Automatic Link, затем Read and Write и сохраните настройки.","copyCredentialsInstruction":"Скопируйте UID (начинается с eu…), а также Access ID и Access Secret со вкладки Overview и вставьте их ниже:","accessId":"Access ID / Client ID","accessIdPlaceholder":"Например, 3xt9q5…","accessSecret":"Access Secret / Client Secret","accessSecretPlaceholder":"Например, 8a4b6c…","smartLifeUid":"UID Smart Life (из Link App Account)","uidPlaceholder":"Например, eu167890123456abcdef","apiEndpoint":"Региональная конечная точка API","centralEurope":"Центральная Европа (рекомендуется)","westernEurope":"Западная Европа","westernAmerica":"Западная Америка","easternAmerica":"Восточная Америка","chinaDataCenter":"Центр обработки данных Китай","indiaDataCenter":"Центр обработки данных Индия","tuyaCloudUnavailable":"Облако Tuya недоступно","tuyaCredentialsHint":"Убедитесь, что ваш IP разрешён в Tuya IoT и введены правильные учётные данные.","loadingSmartLifeDevices":"Загрузка устройств Smart Life…","noDevicesForUid":"Для указанного UID устройства не найдены","verifyUidInstruction":"Убедитесь, что QR-код отсканирован в Tuya IoT (Devices → Link Tuya App Account) с правильной учётной записью Smart Life и выбран соответствующий центр обработки данных.","smartLifeDevices":"Устройства Smart Life","searchDevicesPlaceholder":"Поиск по имени, модели или ID…","filterTuyaDevices":"Фильтр устройств Tuya","selectVisible":"Выбрать видимые","deselectVisible":"Снять выбор с видимых","useInDomus":"Использовать устройство в Domus / LocalTuya","enabled":"Включено","localLanControl":"Локальное управление по LAN через local_key","lanLocal":"Локальная LAN","targetTemperature":"Целевая температура","setTemperature":"Установить температуру","localTuyaMode":"Режим LocalTuya · DP 4","automaticDp4":"Автоматический · DP4 = 0","manualDp4":"Ручной · DP4 = 1","valueFor":"Значение для {name}","noFilterMatches":"Устройства не соответствуют фильтрам","adjustSearchFilters":"Измените запрос или выберите другой фильтр. Текущий выбор будет сохранён.","resetFilters":"Сбросить фильтры","iot":"Устройства IoT","unlockButton":"Разблокировать","filterAll":"Все ({count})","filterOnline":"В сети ({count})","filterOffline":"Не в сети ({count})","filterSelected":"Выбрано ({count})","filterUnselected":"Не выбрано ({count})","clientId":"Client ID:","smartLifeUidLabel":"Smart Life UID:","endpointLabel":"Конечная точка:","devicesShown":"Показано: {count}","deviceOnline":"В сети","deviceOffline":"Не в сети","turnOff":"Выключить","turnOn":"Включить","chooseTemperature":"Выберите температуру","encodedValue":"отправляет","valueToSend":"Отправляемое значение","setpointMapAria":"Таблица преобразования уставки термостата","hideCommands":"Скрыть команды","availableApiCommands":"Команды, доступные через API","messages":{"tuyaConnectedTitle":"Smart Life подключён","deviceOfflineDesc":"Устройство не в сети. Проверьте питание и подключение к локальной сети.","deviceUnblockedTitle":"Блокировка устройства снята","tuyaConnectedDesc":"Учётная запись Smart Life успешно подключена.","tuyaConfigErrorTitle":"Ошибка конфигурации Tuya","tuyaConfigErrorDesc":"Не удалось сохранить настройки Smart Life. Проверьте учётные данные и повторите попытку.","integrationRemovedTitle":"Интеграция удалена","integrationRemovedDesc":"Конфигурация Smart Life удалена из Domus.","disconnectErrorTitle":"Не удалось отключить интеграцию","disconnectErrorDesc":"Не удалось удалить конфигурацию Smart Life.","localSelectionUpdatedTitle":"Выбор устройств обновлён","localSelectionUpdatedDesc":"С Domus и LocalTuya синхронизировано устройств: {count}.","localSyncErrorTitle":"Ошибка синхронизации","localSyncErrorDesc":"Не удалось обновить выбор локальных устройств.","tuyaCommandsUnavailableTitle":"Команды Tuya недоступны","tuyaCommandsUnavailableDesc":"Не удалось получить список команд, поддерживаемых устройством.","invalidValueTitle":"Недопустимое значение","invalidValueDesc":"Введите корректное числовое значение.","tuyaCommandErrorTitle":"Ошибка команды Tuya","communicationError":"Не удалось связаться с устройством. Проверьте подключение и повторите попытку.","thermostatModeUpdatedTitle":"Режим термостата обновлён","thermostatCommandErrorTitle":"Не удалось изменить режим термостата","lanCommunicationError":"Не удалось отправить команду через локальную сеть.","thermostatAuto":"Термостат переведён в автоматический режим.","thermostatManual":"Термостат переведён в ручной режим.","temperatureCommandMissingTitle":"Команда изменения температуры недоступна","temperatureCommandMissingDesc":"Устройство не предоставляет функцию настройки температуры.","deviceOfflineTitle":"Устройство не в сети","setpointUpdatedTitle":"Температура обновлена","setpointUpdatedDesc":"{name}: отправлена заданная температура {temperature} °C с кодированным значением {encoded}.","temperatureSetErrorTitle":"Не удалось установить температуру","directLanCommandTitle":"Отправлена команда по локальной сети","directLanCommandDesc":"Команда напрямую отправлена устройству {name} через локальную сеть.","commandSentTitle":"Команда отправлена в Tuya Cloud","commandSentLanTitle":"Команда отправлена по LAN","commandSentDesc":"{name}: команда {command} отправлена.","smartLifeCommandErrorTitle":"Ошибка команды Smart Life","scanFoundDevicesDesc":"В результате сканирования обнаружено устройств: {count}.","noDeviceTitle":"Устройства не найдены","noDeviceDesc":"По выбранному адресу или в выбранной сети устройства не обнаружены.","scanErrorTitle":"Ошибка сканирования сети","scanErrorDesc":"Не удалось завершить сканирование сети.","devicePairedTitle":"Устройство подключено","devicePairedDesc":"Устройство {name} добавлено в Domus.","pairErrorTitle":"Ошибка подключения устройства","pairErrorDesc":"Не удалось подключить выбранное устройство.","arpScanCompleteTitle":"Сканирование ARP завершено","arpScanCompleteDesc":"Сканирование ARP обнаружило узлов: {count}.","networkScanErrorTitle":"Ошибка сканирования сети","networkScanErrorDesc":"Не удалось просканировать узлы сети.","portProbeCompleteTitle":"Проверка портов завершена","portProbeCompleteDesc":"Проверка {ip} завершена. Открытые порты: {ports}.","portProbeErrorTitle":"Ошибка проверки портов","portProbeErrorDesc":"Не удалось проверить порты узла.","deviceQuarantinedTitle":"Устройство помещено в карантин","deviceQuarantinedDesc":"Устройство {mac} помещено в карантин.","quarantineErrorTitle":"Ошибка помещения в карантин","quarantineErrorDesc":"Не удалось поместить устройство в карантин.","deviceUnblockedDesc":"Устройству {mac} снова разрешён доступ к сети.","unblockErrorTitle":"Не удалось разблокировать устройство","unblockErrorDesc":"Не удалось удалить устройство из карантина.","stateUnavailableTitle":"Состояние недоступно","stateUnavailableDesc":"Не удалось переключить состояние {name}. Обновите состояние устройства и повторите попытку.","unsupportedTemperatureTitle":"Неподдерживаемая температура","unsupportedTemperatureDesc":"Выберите одно из доступных значений: 1, 1,5, 2, 7, 12, 17, 21 или 22 °C.","manualIsolationReason":"Устройство вручную изолировано службой Network Guardian."} } },
  history: { ...en.history, title: 'История и аналитика', subtitle: 'Отслеживайте изменения состояний и показания датчиков.', selectEntity: 'Выберите объект', timeRange: 'Период', noData: 'За выбранный период данных нет.', stats: 'Статистика периода', min: 'Минимум', max: 'Максимум', avg: 'Среднее', current: 'Текущее значение' },
  automations: { ...en.automations, title: 'Правила автоматизации', subtitle: 'Настройте события, условия и автоматические действия.', newAutomation: 'Новое правило', manualTrigger: 'Запустить вручную' },
  scenes: { ...en.scenes, title: 'Сценарии и режимы', subtitle: 'Применяйте настройки нескольких устройств одним нажатием.', newScene: 'Новый сценарий', activate: 'Активировать', saveSuccess: 'Сценарий сохранён', saveError: 'Не удалось сохранить сценарий', deleteSuccess: 'Сценарий удалён', deleteError: 'Не удалось удалить сценарий', emptyTitle: 'Сценарии не настроены', editor: {"editTitle":"Изменить сценарий","newTitle":"Новый сценарий","name":"Название сценария","namePlaceholder":"Например, режим кино, спокойной ночи...","icon":"Значок","icons":{"clapperboard":"Кинохлопушка","night":"Ночь","day":"День","cinema":"Кино","morning":"Утро"},"configuredActions":"Настроенные действия ({count})","addAction":"Добавить действие","removeAction":"Удалить действие","emptyActions":"Действия не добавлены. Нажмите «Добавить действие», чтобы начать.","save":"Сохранить сценарий","closeWindow":"Закрыть окно сценария","nameRequired":"Укажите название сценария.","actionRequired":"Добавьте хотя бы одно действие.","entityRequired":"Для каждого действия необходимо указать действительную сущность.","entityKey":"ID сущности","entityPlaceholder":"например, light.living_room","action":"Действие","actions":{"turnOn":"Включить","turnOff":"Выключить","toggle":"Переключить"}} },
  audit: { ...en.audit, title: 'Журнал аудита безопасности', subtitle: 'Неизменяемая запись входов, изменений конфигурации и автоматических действий.', actor: 'Источник', action: 'Действие', target: 'Объект', timestamp: 'Время', payload: 'Данные', noEntries: 'Записи аудита отсутствуют.', showPayload: 'Показать данные', hidePayload: 'Скрыть данные' },
  family: { ...en.family, "title":"Управление семьёй и разрешениями","subtitle":"Управляйте членами семьи, назначайте детальные разрешения и отправляйте приглашения на регистрацию.","membersTitle":"Члены семьи","invitesTitle":"Ожидающие приглашения","inviteButton":"Пригласить участника","username":"Имя пользователя","email":"Электронная почта","role":"Роль","joinedAt":"Дата добавления","actions":"Действия","changeRole":"Изменить роль","removeMember":"Удалить участника","confirmRemove":"Вы действительно хотите удалить этого члена семьи?","roles":{"admin":"Администратор","operator":"Оператор","viewer":"Наблюдатель","guest":"Гость","kid":"Ребёнок"},"rolesDesc":{"admin":"Полный доступ: настройки системы, управление участниками и полный контроль устройств.","operator":"Операционное управление: может управлять устройствами и запускать сценарии и автоматизации, но не изменять системные настройки.","viewer":"Только чтение: просмотр состояния дома и показаний датчиков в реальном времени без отправки команд.","guest":"Временный гостевой доступ или режим киоска с автоматическим сроком действия.","kid":"Безопасный доступ для ребёнка к разрешённым разделам, например панелям, мультимедиа и домашним делам; без управления устройствами и системных настроек."},"inviteModalTitle":"Пригласить нового члена семьи","selectRole":"Выберите роль","expiration":"Срок действия приглашения","hours24":"24 часа","hours48":"48 часов","days7":"7 дней","generateInvite":"Создать ссылку-приглашение","copyLink":"Копировать ссылку","linkCopied":"Ссылка для регистрации скопирована!","pendingInvites":"Ожидающие приглашения","noPendingInvites":"Нет активных приглашений на регистрацию.","expiresAt":"Истекает","revoke":"Отозвать","noMembers":"Участники не найдены.","tokenLabel":"Токен","errors":{"load":"Не удалось загрузить данные семьи.","guestRequired":"Укажите имя гостя и PIN-код.","createGuest":"Не удалось создать гостевой профиль.","generateInvite":"Не удалось создать приглашение.","updatePermissions":"Не удалось обновить разрешения.","updateRole":"Не удалось изменить роль участника.","removeMember":"Не удалось удалить участника.","revokeInvite":"Не удалось отозвать приглашение."},"permissions":{"notifications":"Уведомления","manage":"Управление разрешениями","title":"Разрешения: {name}","areasTitle":"Доступные помещения","allAreas":"Все помещения","noAreas":"Нет доступных помещений.","entitiesTitle":"Устройства и объекты","allEntities":"Все устройства и объекты","noEntities":"Нет доступных объектов.","dashboardsTitle":"Пользовательские панели","allDashboards":"Все панели","noDashboards":"Нет доступных пользовательских панелей.","viewsTitle":"Доступные разделы","allViews":"Все разделы","actionsTitle":"Разрешённые действия","controlDevices":"Управление устройствами","manageAutomations":"Управление автоматизациями","automationScopeHint":"Для управления автоматизациями необходимо разрешить управление устройствами и полный доступ ко всем помещениям и устройствам.","viewCameras":"Просмотр камер","manageSettings":"Управление системными настройками","hint":"Разрешения также проверяются сервером.","cancel":"Отмена","save":"Сохранить разрешения"},"view":{ ...en.family.view, "you":"(Вы)","guestProfilesTitle":"Гостевые профили и временные PIN-коды киоска","guestProfilesDesc":"Создавайте аккаунты с ограниченным доступом по PIN-коду или временным разблокированием режима киоска (например, для няни или гостя).","guestProfilesEmpty":"Нет активных гостевых профилей. Создайте временный PIN-код для посетителей.","removeGuest":"Удалить гостя","createGuestProfileTitle":"Создать гостевой профиль / режим киоска","profileName":"Имя профиля / посетителя","profileNamePlaceholder":"Например, няня или гость гостиной","numericPin":"Цифровой PIN-код (4–8 цифр)","temporaryValidity":"Срок действия (часы)","hours":"{value} ч","dayOne":"24 часа (1 день)","days":"{value} дн.","oneWeek":"7 дней (1 неделя)","enableKiosk":"Включить полноэкранный режим киоска для этого PIN-кода","createGuestButton":"Новый гость / PIN-код киоска","guestLabel":"Гость","guestKioskLabel":"Гость (киоск)","pinLabel":"PIN-код","expiresLabel":"Истекает"}},
  register: { ...en.register, title: 'Присоединиться к семье Domus', subtitle: 'Завершите регистрацию, чтобы управлять умным домом.', username: 'Имя пользователя', email: 'Электронная почта (необязательно)', password: 'Пароль', confirmPassword: 'Подтвердите пароль', submit: 'Завершить регистрацию', loading: 'Проверка приглашения...', errorInvalidInvite: 'Ссылка недействительна или срок её действия истёк.', errorPasswordMismatch: 'Пароли не совпадают.', success: 'Регистрация успешно завершена! Добро пожаловать в Domus.', backToLogin: 'Вернуться ко входу', usernamePlaceholder: "Например, ivan_ivanov", emailPlaceholder: "Например, ivan{'@'}example.ru" },
  energy: { ...en.energy, title: 'Энергия и солнечная генерация', solar: 'Солнечная генерация', battery: 'Аккумулятор', grid: 'Электросеть', house: 'Потребление дома', import: 'Потребление из сети', export: 'Отдача в сеть', charge: 'Зарядка', discharge: 'Разрядка', minPrice: 'Минимальная цена', avgPrice: 'Средняя цена за 24 ч', maxPrice: 'Максимальная цена', savings: 'Экономия' },
  cameras: { ...en.cameras,
    inferenceFps: 'Скорость обработки, кадров/с',
    latency: 'Задержка',
    activeChannels: 'Активные каналы',
    zonesAi: 'Зоны ИИ',
    aiGuard: 'Защита ИИ',
    zonesTitle: 'Зоны обнаружения движения',
    sensitivity: 'Чувствительность',
    timelineTitle: 'Временная шкала событий',
    clipButton: 'Сохранить клип',
    clipModalTitle: 'Просмотр видеоклипа',
    clipDuration: 'Длительность клипа',
    notifSync: 'Синхронизация уведомлений',
    zonesModalTitle: 'Настройка зон камеры',
 title: 'Видеонаблюдение и локальный ИИ', privacy: 'Конфиденциальность и безопасность', privacyDesc: 'Полностью локальная обработка, без облачной передачи', live: 'ЭФИР', active: 'Активно', disabled: 'Отключено', detected: 'Обнаружено', confidence: 'Уверенность', addZone: 'Добавить зону', noCameras: 'Камеры не настроены', view: { ...en.cameras.view, "grid":"Сетка","aiOverviewDescription":"Локальное распознавание лиц жителей с помощью косинусного сходства и автоматическое чтение номерных знаков для контроля доступа.","surveillanceWall":"Стена видеонаблюдения","biometricPlates":"Биометрия и распознавание номеров с ИИ","previous15":"Назад на 15 минут","pause":"Пауза","play":"Воспроизвести","next15":"Вперёд на 15 минут","playbackSpeed":"Скорость воспроизведения","today24h":"Сегодня · запись за 24 часа","aiDetected":"Обнаружено событие ИИ","localVisionEngine":"Локальный движок Domus Edge Vision","privacyGuarantee":"Гарантия конфиденциальности без облака","registeredFamily":"Зарегистрированные члены семьи","biometricProfilesActive":"Активные биометрические профили 128-D","authorizedPlates":"Разрешённые номерные знаки","automaticGate":"Автоматическое открытие ворот включено","similarityThreshold":"Порог сходства","angularTolerance":"Допуск по углу и наличию очков","hardwareAcceleration":"Аппаратное ускорение","edgeInference":"Локальный edge-анализ с точностью до кадра","familyProfiles":"Биометрические профили семьи","familyProfilesDesc":"Автоматическое распознавание членов семьи для сценариев и снятия охраны","registerFace":"Зарегистрировать новое лицо","noFamilyFaces":"На локальном хабе нет зарегистрированных лиц членов семьи.","embeddingActive":"Активно векторное представление","embedding128":"Векторное представление 128-D","testSimilarity":"Проверить сходство с локальной моделью ИИ","deleteProfile":"Удалить профиль","authorizedVehiclePlates":"Разрешённые номера автомобилей (ALPR / ANPR)","plateRecognitionDesc":"Оптическое распознавание номеров для своевременного открытия въездных и гаражных ворот","addPlate":"Добавить номер","noAuthorizedVehicles":"Нет автомобилей, зарегистрированных для автоматического доступа.","automaticGateOpen":"Автоматическое открытие ворот","removeAuthorizedPlate":"Удалить разрешённый номер","zeroFrameDrops":"Нет потери кадров WebRTC","realtimeInference":"Обработка кадров в реальном времени","streamingLatency":"Задержка потокового видео менее 200 мс","connectCameras":"Подключите камеры WebRTC, RTSP или HLS для просмотра трансляций и обнаружения объектов с помощью ИИ.","expandFullscreen":"Развернуть на весь экран","configureAiZones":"Настроить зоны движения ИИ","aiGuardActive":"Локальная защита ONNX AI активна","noMotionZones":"Зоны обнаружения движения не настроены.","noRecentAiEvents":"Недавние события ИИ не зарегистрированы.","viewAnimatedClip":"Просмотреть анимированный фрагмент события","porchDelivery":"Крыльцо и доставка посылок","vehicleDriveway":"Подъезд для автомобилей","cameraZones":"Зоны, заданные для камеры","registerBiometricProfile":"Зарегистрировать биометрический профиль","memberFullName":"Имя и фамилия члена семьи","memberPlaceholder":"Например, Анна Иванова","householdRole":"Роль в доме","resident":"Житель","administrator":"Администратор","frequentGuest":"Постоянный гость","vectorExtraction":"Извлечение вектора ONNX 128-D","edgeLocalProcessing":"Локальный алгоритм вычисляет векторную сигнатуру, не загружая изображения во внешние облачные сервисы.","addAuthorizedPlate":"Добавить разрешённый номер (ALPR)","licensePlateFormat":"Номер автомобиля (европейский или стандартный формат)","platePlaceholder":"Например, AB123CD","ownerReference":"Владелец / Примечание","ownerPlaceholder":"Например, Анна Иванова","vehicleModel":"Модель автомобиля","vehiclePlaceholder":"Например, Tesla Model Y, Fiat 500e…","localBiometricMatch":"Локальная биометрическая проверка","cosineSimilarity":"Косинусное сходство:","thresholdPassed":"Порог 65% превышен","accessConfirmed":"Доступ подтверждён","profileValidated":"Профиль проверен локальным движком NVR. Сценарии приветствия разблокированы.","testMatchButton":"Проверить совпадение","filterPerson":"Человек","filterVehicle":"Транспортное средство","filterPackage":"Посылка","wallSync":"СИНХРОНИЗАЦИЯ НЕСКОЛЬКИХ ЭКРАНОВ ({matrix})","cameraNumber":"КАМЕРА {number}: {name}","channelWaiting":"Канал {channel} ожидает подключения","filterAll":"Все ({count})","recording":"ЗАПИСЬ","cameraLabel":"Камера","confidenceLabel":"Уверенность","playClip":"Воспроизвести клип ({seconds} с)","storageNvr":"Хранилище NVR: {used} ГБ / {total} ГБ (циклический буфер WAL)","online":"В сети","streamingLatencyHint":"Задержка потоковой передачи менее 200 мс","cameraId":"ID камеры","directWebRtcStream":"Прямая трансляция WebRTC","sensitivityLabel":"Чувствительность","familyOwner":"Член семьи","privateVehicle":"Личный автомобиль","newZoneName":"Новая зона {count}","detectionTooltip":"Обнаружено: {label}, время {time} — камера {camera}","confirmDeleteFace":"Удалить биометрический профиль пользователя {name}?","confirmDeletePlate":"Удалить номерной знак {plate}?","detectionLabels":{"person":"Человек","vehicle":"Транспортное средство","package":"Посылка","animal":"Животное"},"saveProfile":"Сохранить профиль","autoOpenGateLights":"Автоматически открывать въездные ворота и включать освещение подъездной дорожки","registerPlate":"Зарегистрировать номер"} },
  shopping: { ...en.shopping, title: 'Общий список покупок семьи', newItem: 'Добавить товар', placeholderName: 'Название товара...', placeholderQty: 'Количество...', addBtn: 'Добавить в список', clearCompleted: 'Удалить выполненные', toBuy: 'Купить', completed: 'Выполнено', addedBy: 'Добавил', completedBy: 'Выполнил', view: { receiptScanner: {"documentType":"Тип документа","receipt":"Чек","invoice":"Счёт-фактура","title":"Сканирование чеков и счетов","subtitle":"Сделайте снимок на iPhone, выберите изображение или PDF. Domus распознает магазин, дату, товары и суммы; перед импортом всё можно исправить.","scanWithCamera":"Сканировать камерой","cameraHint":"Откройте заднюю камеру и полностью поместите чек в кадр","openingCamera":"Открываем камеру…","chooseFile":"Выбрать изображение или документ","fileHint":"Фото, изображения из Фото/Файлов или отсканированные PDF","cameraUnavailable":"Камера недоступна на этой странице. Можно выбрать снимок через системную камеру.","sourceImageTooLarge":"Исходное изображение превышает 32 МБ.","imageDimensionsTooLarge":"Размеры изображения слишком велики для безопасной обработки.","preparingImage":"Подготовка и сжатие изображения…","imageReady":"Изображение готово: проверьте предварительный просмотр и запустите OCR.","imagePrepareFailed":"Не удалось прочитать изображение. Сохраните его как JPEG или PNG и повторите попытку.","imageStillTooLarge":"Не удалось сжать изображение до 4 МБ. Попробуйте уменьшить разрешение.","pdfTooLarge":"PDF превышает ограничение 3 МБ.","pdfReady":"PDF готов к извлечению текста и OCR.","unsupportedFile":"Неподдерживаемый формат. Выберите изображение или PDF.","cameraNotReady":"Камера ещё не готова. Подождите немного.","captureFailed":"Не удалось сделать снимок. Попробуйте ещё раз.","ocrInProgress":"OCR выполняется на Raspberry Pi…","pdfInProgress":"PDF обрабатывается на Raspberry Pi…","analysisInProgress":"Анализ текста…","ocrFailed":"Не удалось распознать документ. Поверните изображение, выберите другое или вставьте текст.","imageReadyTitle":"Предварительный просмотр документа","removeFile":"Удалить выбранный файл","imagePreviewAlt":"Предпросмотр изображения чека","rotateImage":"Повернуть на 90°","readReceipt":"Распознать чек","readPDF":"Распознать PDF","privacy":"Преобразование и сжатие изображения выполняются в браузере. Файл отправляется на Raspberry Pi для OCR и не сохраняется как вложение.","cameraDialogTitle":"Сканирование документа","cameraDialogHint":"Используйте заднюю камеру. Держите телефон параллельно бумаге и захватите все края.","alignReceipt":"Расположите чек внутри рамки","cameraTip":"Избегайте бликов и теней. Положите чек на однотонный фон и приблизьте камеру, чтобы текст был чётким.","closeCamera":"Закрыть камеру","cancel":"Отмена","takePhoto":"Снять и обрезать","sampleHint":"Или попробуйте пример:","pasteText":"Вставить текст чека","pastePlaceholder":"Вставьте текст здесь… например PASTA 500G 1.29 € и ИТОГО 15.57 €.","analyzeText":"Анализировать текст","storeName":"Название магазина","documentDate":"Дата документа","total":"Итого","rescan":"Сканировать заново","itemsFound":"Строк товаров: {count}","confidence":"Оценка распознавания: {value}%","showOriginalText":"Показать распознанный OCR текст","product":"Товар","quantity":"Количество","unit":"Единица","price":"Сумма (€)","category":"Категория","actions":"Действия","removeProduct":"Удалить товар","addManualItem":"Добавить строку вручную","manualItem":"Новый товар","updatePantry":"Автоматически обновить кладовую","invoiceNoPantry":"Для счетов автоматическое обновление кладовой отключено.","updatePantryHint":"Добавлять распознанные товары или увеличивать существующие запасы.","addFinance":"Записать расход в Home Finance","addFinanceHint":"Создать расход на сумму документа.","importing":"Импорт…","confirmImport":"Подтвердить импорт"},
      quantityAria: "Количество",
      allSupermarkets: "Все магазины",
      markItem: "Отметить товар",
      bought: "Куплено",
      filterSupermarket: "Фильтр по магазину",
      markAsBought: "Отметить как купленное",
      recognizedAs: "Распознано как {name}",
      justNow: "Только что",
      itemsAdded: "Добавлено в кладовую: {count}",
      itemsUpdated: "Обновлено в кладовой: {count}",
      itemsTransferred: "В кладовую перемещено товаров: {count}",
      transferCompleted: "Перемещение завершено",
      "scanReceipt":"Сканировать чек","moveBoughtToPantry":"Переместить купленные товары в кладовую","moveShoppingToPantry":"Переместить покупки в кладовую","productName":"Название товара","supermarketPlaceholder":"Супермаркет / магазин","supermarketLabel":"Супермаркет или магазин","addToList":"Добавить в список","quickSupermarkets":"Быстрый выбор магазинов:","filterByStore":"Фильтр по магазину:","allStores":"Все магазины","moveCompletedHint":"Переместите товары в кладовую: количество можно прибавить к имеющимся запасам и указать срок годности.","loadIntoPantry":"Добавить в кладовую","moveItemToPantry":"Переместить товар в кладовую и указать срок годности","inPantry":"В кладовой","removeProduct":"Удалить товар","noItems":"В списке нет товаров","noProductsForStore":"Для магазина «{store}» товары не найдены.","allPurchasedOrEmpty":"Все товары куплены или список пуст.","receiptTitle":"Сканировать чек","receiptIntro":"Сделайте снимок камерой iPhone, выберите изображение или загрузите PDF, созданный через Файлы > Сканировать документы. Проверьте распознанные данные перед импортом.","closeScanner":"Закрыть сканер","compareWithList":"Сравнение со списком покупок","noAutoMatches":"Автоматических совпадений нет: вы можете проверить чек и подтвердить импорт.","closeNotification":"Закрыть уведомление","goToPantry":"Перейти в кладовую →"} },
  chores: { ...en.chores, title: 'Домашние дела', newChore: 'Новое дело', all: 'Все', pending: 'Ожидает', overdue: 'Просрочено', completed: 'Выполнено', due: 'Срок', assignedTo: 'Назначено', recurrence: 'Повторение', completeBtn: 'Отметить выполненным', view: { ...en.chores.view, "toDo":"К выполнению","viewModes":"Режимы просмотра задач","listView":"Список","calendarView":"Календарь","connectGoogleCalendar":"Подключить Google Calendar","newChore":"Новая задача","dueOrOverdue":"Срок истекает / Просрочено","activeTasks":"Активные задачи","markCompleted":"Отметить «{title}» как выполненную","deleteChore":"Удалить задачу","noChoresInView":"В этом представлении нет задач","emptyHint":"Добавьте новую домашнюю задачу или выберите другую категорию.","createChore":"Создать задачу","previousDay":"Предыдущий день","previousWeek":"Предыдущая неделя","previousMonth":"Предыдущий месяц","nextDay":"Следующий день","nextWeek":"Следующая неделя","nextMonth":"Следующий месяц","previousPeriod":"Предыдущий период","nextPeriod":"Следующий период","today":"Сегодня","calendarModes":"Режимы календаря","day":"День","week":"Неделя","month":"Месяц","urgent":"Срочно","high":"Высокий","medium":"Средний","low":"Низкий","openDay":"Открыть просмотр дня","addChoreToDay":"Добавить задачу на этот день","clickToComplete":"Нажмите, чтобы завершить","syncing":"Синхронизация...","syncNow":"Синхронизировать","mon":"Пн","tue":"Вт","wed":"Ср","thu":"Чт","fri":"Пт","sat":"Сб","sun":"Вс","allDay":"Весь день","add":"Добавить","noChoresForDate":"На эту дату задач нет","noChoresForDateHint":"Все задачи на этот день выполнены или не запланированы.","addReminderForDate":"Добавить напоминание на эту дату","newHouseholdChore":"Новая домашняя задача","taskTitle":"Название задачи","descriptionDetails":"Описание / Подробности","assignAndNotify":"Назначить исполнителя и уведомить","wholeFamily":"👨‍👩‍👧‍👦 Вся семья (уведомление всем участникам)","roomArea":"Комната / зона","wholeHome":"Весь дом / улица","priority":"Приоритет","dueDateTime":"Срок: дата и время","recurrence":"Повторение","noneOneTime":"Нет (однократно)","daily":"Каждый день","weekly":"Каждую неделю","monthly":"Каждый месяц","saveAndSchedule":"Сохранить и запланировать","syncChoresToGoogleCalendar":"Синхронизировать задачи Domus с Google Calendar","googleStep1":"Откройте Google Calendar в браузере или на смартфоне.","googleStep3":"Вставьте скопированный адрес и подтвердите действие «Добавить календарь».","importFromGoogleCalendar":"Импорт событий / задач из Google Calendar","googleIcalAddress":"Адрес iCal Google Calendar (.ics)","familyShort":"Семья","memberFallback":"Участник","noDueDate":"Без срока выполнения","completedTitle":"Задача выполнена","completedMessage":"Задача «{title}» выполнена.","completeFailed":"Не удалось отметить задачу выполненной.","deletedTitle":"Задача удалена","deletedMessage":"Задача удалена.","deleteFailed":"Не удалось удалить задачу.","createdTitle":"Задача запланирована","createdAssigned":"Напоминание сохранено: уведомление отправлено пользователю {name}.","createdFamily":"Напоминание сохранено: уведомление отправлено всей семье.","createFailed":"Не удалось создать задачу.","copiedTitle":"Ссылка скопирована","copiedMessage":"URL iCal скопирован. Вставьте его в Google Календарь.","copyFailed":"Не удалось скопировать ссылку.","syncSuccessTitle":"Синхронизация завершена","syncSuccessMessage":"Синхронизировано событий: {count}.","syncErrorTitle":"Ошибка синхронизации","syncFailed":"Не удалось синхронизировать Google Календарь.","markPending":"Отметить как невыполненную","markDone":"Отметить как выполненную","closeWindow":"Закрыть окно","titlePlaceholder":"Например, очистить фильтр робота-пылесоса, полить растения...","descriptionPlaceholder":"Инструкции или сведения о задаче","notifyAssigned":"Push-уведомления и напоминания будут отправлены пользователю {name}.","notifyFamily":"Уведомления о сроке и напоминания будут отправлены всей семье.","optionLow":"Низкий","optionMedium":"Средний","optionHigh":"Высокий","optionUrgent":"Срочный","cancel":"Отмена","copyDone":"Скопировано!","copyLink":"Копировать ссылку","optionOne":"ВАРИАНТ 1: ПОДПИСКА ИЗ GOOGLE КАЛЕНДАРЯ","optionTwo":"ВАРИАНТ 2: ИМПОРТ ИЗ GOOGLE КАЛЕНДАРЯ","calendarDesc":"Добавьте этот канал iCal в Google Календарь, Apple Calendar или Outlook, чтобы видеть актуальные домашние задачи.","step2":"В боковой панели рядом с «Другие календари» нажмите «+», затем «По URL».","importDesc":"Вставьте секретный или общедоступный адрес iCal Google Календаря, чтобы импортировать события в Domus.","dayTasksSingular":"запланированная задача","dayTasksPlural":"запланированные задачи","completedCount":"выполнено","openDayTitle":"Открыть вид дня","addChoreTitle":"Добавить задачу","addChoreForDay":"Добавить задачу на этот день","clickToCompleteTitle":"Нажмите, чтобы выполнить: {title}","deleteChoreAria":"Удалить задачу","assigneeFamily":"Вся семья" } },
  helpers: { ...en.helpers, title: 'Виртуальные объекты и помощники', newHelper: 'Новый помощник', boolean: 'Логический переключатель', counter: 'Счётчик', number: 'Числовой регулятор', timer: 'Таймер обратного отсчёта', view: { ...en.helpers.view, ...{"empty":"В базе данных нет настроенных помощников. Создайте новый с помощью кнопки выше.","delete":"Удалить помощник","active":"ВКЛЮЧЕНО","off":"ВЫКЛЮЧЕНО","value":"Значение:","start":"Запустить","pause":"Пауза","reset":"Сбросить","roomTarget":"Целевая температура комнаты","dynamicTrv":"Динамический TRV","sensor":"Датчик:","valve":"Клапан:","createTitle":"Создать виртуальный помощник","name":"Название помощника","namePlaceholder":"Например, Режим уборки","entityType":"Тип объекта","boolean":"Логический переключатель (input_boolean)","number":"Числовой регулятор (input_number)","timer":"Таймер обратного отсчёта","compensator":"Компенсатор термостата (Better Thermostat)","trv":"Термостатический клапан (TRV)","roomSensor":"Датчик температуры комнаты","windowSensor":"Датчик открытого окна (необязательно)","targetTemperature":"Целевая температура (°C)","offset":"Коэффициент смещения (1,0–2,5)","min":"Минимум","max":"Максимум","step":"Шаг","unit":"Единица","unitPlaceholder":"°C, %, кВт","duration":"Длительность (секунды)","cancel":"Отмена","create":"Создать помощник","deleteAria":"Удалить помощник","statusAria":"Состояние помощника","timerStates":{"active":"АКТИВЕН","paused":"ПРИОСТАНОВЛЕН","idle":"НЕАКТИВЕН"},"toggle":"Включить или выключить {name}" } } },
  floorplan: { ...en.floorplan, title: 'План дома', selectFloor: 'Выберите этаж', addFloor: 'Загрузить план этажа', editMarkers: 'Изменить метки устройств', view: { ...en.floorplan.view, "subtitle":"Нарисуйте и разместите комнаты дома, организуйте устройства и просматривайте план в реальном времени.","roomPlan":"План комнат","toggleHeatmap":"Показать градиент температуры или влажности","newFloor":"Новый этаж","createFirstFloor":"Создать первый этаж","roomEditor":"Редактор планировки комнат","addRoom":"Добавить комнату","generateStandardPlan":"Создать стандартный план","deleteRoom":"Удалить эту комнату","removeDeviceFromMap":"Убрать устройство с плана","availableDevices":"Доступные устройства","placeDeviceHint":"Выберите устройство и щёлкните по плану, чтобы разместить его.","addRoomTitle":"Добавить комнату на план","roomName":"Название комнаты","roomNamePlaceholder":"Например, гостиная, кухня, спальня","roomType":"Тип комнаты","roomColorIdentifier":"Цвет / архитектурный идентификатор","initialWidth":"Начальная ширина","initialHeight":"Начальная высота","placeRoom":"Разместить комнату","newFloorTitle":"Новый этаж","floorName":"Название этажа","floorNamePlaceholder":"Например, первый этаж, мансарда, сад","floorLevel":"Уровень этажа","floorLevelPlaceholder":"0 — первый этаж, 1 — второй этаж","svgMarkupOptional":"SVG-разметка плана (необязательно)","svgPlaceholder":"Вставьте сюда векторную SVG-разметку плана","saveFloor":"Сохранить этаж"} },
  media: { ...en.media, title: 'Мультирум-аудио и воспроизведение', masterVolume: 'Общая громкость', playing: 'Воспроизведение', idle: 'Ожидание', groupSpeakers: 'Объединить колонки', view: { ...en.media.view, ...{"add":"Добавить медиаплеер","broadcast":"Передать голосовое объявление","empty":"В базе данных нет зарегистрированных медиаплееров.","emptyDesc":"Зарегистрируйте динамики Apple AirPlay, устройства Google Cast, приёмники DLNA или локальные динамики.","registerFirst":"Зарегистрировать медиаплеер","deleteTitle":"Удалить из базы данных","deleteAria":"Удалить плеер","noTrack":"Ничего не воспроизводится","ready":"Динамик подключён и готов","pause":"Пауза","play":"Воспроизвести","broadcastTitle":"Синхронизированное голосовое объявление","close":"Закрыть окно","broadcastDesc":"Сообщение будет мгновенно синтезировано с помощью Piper TTS и передано на все динамики с приоритетной громкостью, временно прервав текущую музыку.","message":"Голосовое сообщение для трансляции:","messagePlaceholder":"Введите текст голосового сообщения...","cancel":"Отмена","send":"Передать сейчас","addTitle":"Зарегистрировать медиаплеер в базе данных","addDesc":"Добавляет аудиоустройство в таблицу SQLite media_players для мультирум-воспроизведения и голосовых уведомлений.","speakerName":"Название динамика *","speakerPlaceholder":"Например, Apple HomePod Mini Кухня, Sonos Гостиная","platform":"Платформа / протокол","cast":"Google Cast (Chromecast / Nest)","airplay":"Apple AirPlay 2 (HomePod / ресиверы)","dlna":"DLNA / UPnP (Hi-Fi / ТВ)","local":"Локальный динамик хаба (Jack / USB / DAC)","area":"Комната / зона","salotto":"Гостиная","cucina":"Кухня","camera":"Спальня","studio":"Кабинет","bagno":"Ванная","giardino":"Сад","saveDb":"Сохранить в базе данных","nameRequired":"Введите название медиаплеера","broadcastSuccess":"Объявление успешно передано на все динамики!","broadcastError":"Ошибка при передаче аудио","playerSuccess":"Динамик успешно зарегистрирован в базе данных!","playerError":"Ошибка регистрации плеера","confirmDelete":"Вы действительно хотите удалить «{name}» из базы данных Domus?","deleteFailed":"Не удалось удалить медиаплеер",closeWindow:"Закрыть окно",statePaused:"На паузе",stateStopped:"Остановлено",stateBuffering:"Загрузка",stateUnknown:"Неизвестное состояние"} } },
  mesh: { ...en.mesh, title: 'Радиодиагностика и карта Mesh', coordinator: 'Координатор', router: 'Маршрутизатор / ретранслятор', endDevice: 'Устройство с батареей', linkQuality: 'Качество связи', view: { ...en.mesh.view, "coordinator":"Координатор (шлюз)","router":"Маршрутизатор / ретранслятор (с питанием от сети)","endDevice":"Конечное устройство (на батарее)","excellent":"Отлично","good":"Хорошо","weak":"Слабо","devicesProtocolHint":"Здесь будут отображаться устройства, настроенные для этого протокола, с реальными показателями LQI.","nodeType":"Тип узла","linkQuality":"Качество связи (LQI)","remainingBattery":"Остаток заряда батареи","directBindingTitle":"Прямая привязка Zigbee (без хаба)","directBindingDesc":"Свяжите беспроводной пульт или кнопку напрямую с лампой или группой. Связь продолжит работать мгновенно, даже если сервер Domus выключен!","noBindings":"Активных прямых привязок нет. Создайте аппаратное соединение между выключателем и исполнительным устройством.","removeBinding":"Удалить привязку","newBindingTitle":"Новая прямая привязка Zigbee","sourceDevice":"Исходное устройство (пульт / выключатель)","sourceNamePlaceholder":"Название (например, прикроватный пульт)","deviceIdPlaceholder":"IEEE / ID устройства","destinationDevice":"Целевое устройство (лампа / реле / группа)","destinationNamePlaceholder":"Название (например, прикроватный свет)","functionalCluster":"Функциональный кластер","onOffCluster":"Включение / выключение (genOnOff)","levelCluster":"Регулировка яркости (genLevelCtrl)","colorCluster":"Цвет / температура (lightingColorCtrl)","createBinding":"Создать прямую привязку","noDevicesDetected":"Устройства {protocol} в mesh-сети не обнаружены","newDirectBinding":"Новая прямая привязка","nodeTypes":{"coordinator":"Координатор","router":"Маршрутизатор / ретранслятор","end_device":"Конечное устройство"},"sourcePrefix":"Источник","destinationPrefix":"Назначение","clusterPrefix":"Кластер" } },
  store: { ...en.store, title: 'Магазин Domus и расширения', installed: 'Установлено', available: 'Доступно в каталоге', install: 'Установить', open: 'Открыть приложение', uninstall: 'Удалить' , view: { ...en.store.view, "proposeCommunityPlugin":"Предложить плагин сообщества","searchPlaceholder":"Поиск по имени, автору, лицензии или ID пакета…","searchAria":"Поиск в каталоге расширений","clearSearch":"Очистить поиск","sourceVerification":"Источник и проверка:","verificationFilter":"Фильтр категории проверки","all":"Все","domusVerified":"Domus Verified","communityOpenSource":"Открытый код сообщества","pendingReview":"Ожидает проверки","moduleType":"Тип модуля:","extensionTypeFilter":"Фильтр типа расширения","allTypes":"Все типы","wasmPlugins":"Плагины WASM","blueprints":"Blueprint-сценарии","dashboardWidgets":"Виджеты панели","noSearchResults":"Ничего не найдено","noPackages":"Нет доступных пакетов","searchHint":"Попробуйте другие ключевые слова: «sunspec», «wmbus», «radar» или «modem».","packagesHint":"Здесь появятся пакеты каталога и предложения сообщества.","packageDetails":"Сведения о {name}","rejected":"Отклонено","community":"Сообщество","verified":"Проверено","author":"Автор","viewSourceFor":"Просмотреть исходный код {name}","openSourceCode":"Открытый исходный код","approvePackage":"Одобрить и опубликовать пакет","approve":"Одобрить","rejectPackage":"Отклонить пакет","reject":"Отклонить","proposeTitle":"Предложить плагин с открытым кодом","proposeSubtitle":"Отправить модуль WASI или blueprint на проверку сообщества","closeWindow":"Закрыть окно","openSourceRequirementTitle":"Требование к открытому исходному коду:","openSourceRequirementDesc":"Все плагины сообщества должны публиковать исходный код под свободной лицензией (например, на GitHub или GitLab), чтобы до одобрения можно было провести аудит безопасности песочницы.","packageName":"Название модуля","packageNamePlaceholder":"Например, счётчик воды M-Bus","packageId":"ID пакета (уникальный)","packageIdPlaceholder":"Например, domus.community.mbus-water","extensionType":"Тип расширения","wasmPluginOption":"Плагин WASM / WASI (песочница)","blueprintOption":"Сценарий автоматизации","widgetOption":"Виджет панели","authorCommunity":"Автор / команда сообщества","authorPlaceholder":"Например, GreenEnergy Labs","publicRepository":"Публичный URL репозитория с открытым кодом (GitHub/GitLab)","repositoryPlaceholder":"https://github.com/ваша-команда/domus-wasi-plugin","openSourceLicense":"Лицензия открытого кода","gplLicense":"GPL v3","bsdLicense":"BSD 3-Clause","mplLicense":"Mozilla Public License 2.0","requiredCapabilities":"Требуемые возможности WASI","descriptionFeatures":"Описание и функции","descriptionPlaceholder":"Опишите назначение модуля, поддерживаемое оборудование и сценарии использования…","publishing":"Публикация…","submitForReview":"Отправить на проверку","packageType":"Тип пакета","verificationTier":"Проверка и уровень","version":"Версия","authorPublisher":"Автор / издатель","softwareLicense":"Лицензия ПО","sourceCode":"Исходный код","viewPublicRepository":"Открыть публичный репозиторий","installationStatus":"Статус установки","installedActive":"Установлен и активен в системе","notInstalled":"Не установлен","sandboxPermissions":"Разрешения песочницы (возможности WASI)","verifiedSecurityTitle":"Безопасность Domus Verified:","verifiedSecurityDesc":"Выполнение в песочнице WebAssembly (WASI) с изолированной линейной памятью, строго контролируемыми разрешениями и без неавторизованных облачных вызовов.","uninstall":"Удалить","installOneClick":"Установить одним нажатием"} },
  schedules: { ...en.schedules, title: 'Расписания и календарь', newSchedule: 'Новое расписание', time: 'Время', days: 'Дни', target: 'Целевой объект', view: { ...en.schedules.view, "tabSelector":"Разделы термостата","weeklyTabAria":"Недельный термостат","weeklyTab":"Термостат","calendarTabAria":"Календарь и iCal","calendarEventsTab":"Календарь и события","liveThermostatAria":"Настройка умного термостата","liveThermostatTitle":"Настроить подключённый умный термостат","deviceLabel":"Устройство:","simulatedThermostat":"Термостат гостиной (симуляция)","actualTemperature":"Измеренная температура","continuousMonitoring":"Непрерывный мониторинг","desiredSetpoint":"Желаемая уставка","decreaseHalfDegree":"Уменьшить на 0,5 °C","decreaseTemperature":"Уменьшить температуру на 0,5 градуса","increaseHalfDegree":"Увеличить на 0,5 °C","increaseTemperature":"Увеличить температуру на 0,5 градуса","ecoPreset":"Эко 18 °C","normalPreset":"Обычный 20 °C","comfortPreset":"Комфорт 21,5 °C","boostPreset":"Ускоренный нагрев 23 °C","operatingMode":"Режим работы","heating":"Нагрев","automatic":"Автоматически","off":"Выключено","sendingCommand":"Отправка команды…","adjustThermostatNow":"Настроить термостат сейчас","weeklyProgrammer":"Недельное расписание","externalSensorAutomation":"Создать автоматизацию термостата по датчику","controlledEntity":"Управляемый объект:","profile":"Профиль:","newProfile":"Новый профиль","winterWorkProfile":"Зимний рабочий профиль","copyMondayToFriday":"Копировать понедельник на вторник–пятницу","copySaturdayToSunday":"Копировать субботу на воскресенье","saving":"Сохранение…","saveSchedule":"Сохранить расписание","addTimeSlot":"Добавить временной интервал","timeSlot":"Интервал","emptyTimeSlots":"Нет временных интервалов (поддержание Эко 18,0 °C)","syncedEventsTitle":"Синхронизированные события и триггеры автоматизации","importIcs":"Импорт .ics / CalDAV","noSyncedEvents":"В базе данных нет синхронизированных событий.","room":"Комната:","newTimeSlotTitle":"Новый временной интервал","closeWindow":"Закрыть окно","startTime":"Время начала","endTime":"Время окончания","targetTemperature":"Целевая температура (°C)","cancel":"Отмена","addSlot":"Добавить интервал","days":{"monday":"Понедельник","tuesday":"Вторник","wednesday":"Среда","thursday":"Четверг","friday":"Пятница","saturday":"Суббота","sunday":"Воскресенье"},"eventSources":{"entity":"Сущность Domus","tuya_local":"Tuya Local","tuya_cloud":"Tuya Cloud"},"manualOverrideDuration":"Длительность ручного переопределения","untilNextChange":"До следующего изменения","minutes30":"30 минут","hour1":"1 час","hours2":"2 часа","hours4":"4 часа","hours8":"8 часов","manualOverrideExpiryHint":"По истечении срока Domus восстановит предыдущую заданную температуру.","noProgrammableSetpoint":"Это устройство не поддерживает программируемую заданную температуру.","deleteTimeSlot":"Удалить временной интервал","deleteTimeSlotAria":"Удалить временной интервал","noThermostatTitle":"Термостат не выбран","noThermostatMessage":"Выберите подключённый умный термостат.","invalidOverrideDuration":"Длительность переопределения должна составлять от 0 до 1440 минут.","targetTemperatureUnsupported":"Запрошенная температура выходит за пределы диапазона термостата.","timedRestoreRequiresEntity":"Для отложенного восстановления устройство должно быть связано с сущностью Domus.","tuyaOffCommandUnavailable":"Устройство Smart Life не предоставляет распознанную команду выключения.","deviceNotControllable":"Выбранное устройство не связано с управляемой сущностью Domus.","commandSentTitle":"Команда термостату отправлена","commandUntilNextChange":"Запрошена заданная температура {temperature} °C до следующего изменения.","commandForDuration":"Запрошена заданная температура {temperature} °C на {duration} мин.; затем будет восстановлено предыдущее значение.","adjustmentErrorTitle":"Не удалось настроить термостат","adjustmentErrorMessage":"Не удалось связаться с умным термостатом.","invalidTimeTitle":"Неверное время","invalidTimeMessage":"Время окончания должно быть позже времени начала в тот же день.","invalidSetpointTitle":"Недопустимая заданная температура","invalidSetpointMessage":"Введите температуру от {min} до {max} °C.","overlappingSlotsTitle":"Перекрывающиеся интервалы","overlappingSlotsMessage":"Измените время: этот интервал пересекается с уже существующим интервалом выбранного дня.","timeSlotAddedTitle":"Временной интервал добавлен","timeSlotAddedMessage":"Интервал {start}–{end} ({temperature} °C) добавлен на день: {day}.","profileAppliedTitle":"Профиль применён","profileAppliedMessage":"Применён профиль «Зимний рабочий день» (комфорт утром и вечером, режим Eco днём).","copiedTitle":"Интервалы скопированы","copyMondayToWorkdaysMessage":"Интервалы понедельника скопированы на все рабочие дни (вторник–пятница).","copySaturdayToSundayMessage":"Интервалы субботы скопированы на воскресенье.","scheduleSavedTitle":"Расписание термостата сохранено","scheduleSavedMessage":"Расписание «{name}» обновлено.","scheduleSaveErrorTitle":"Ошибка сохранения","scheduleSaveErrorMessage":"Не удалось сохранить расписание. Повторите попытку.","newScheduleName":"Недельный термостат {number}","newThermostatTitle":"Новое расписание термостата","newThermostatMessage":"Создано: {name}.","calendarImportSuccess":"Календарь успешно импортирован.","calendarImportError":"Не удалось импортировать календарь iCal.","hvacModes":{"automatic":"Автоматически","heating":"Нагрев","off":"Выключено","manual":"Вручную","cooling":"Охлаждение","heatCool":"Нагрев и охлаждение","dry":"Осушение","fanOnly":"Только вентиляция","other":"Другой режим"}} },
  alarm: { ...en.alarm, title: 'Система охранной сигнализации', disarmed: 'Снята с охраны', armedHome: 'Дома', armedAway: 'Вне дома', armedNight: 'Ночной режим', triggered: 'ТРЕВОГА', pin: 'Защитный PIN-код', disarm: 'Снять с охраны', armHome: 'Режим дома', armAway: 'Режим вне дома', armNight: 'Ночной режим' },
  flowchart: { ...en.flowchart, conditionEditor: { logicalGroup: 'Логическая группа', condition: 'Условие', type: 'Тип', logicalOperators: 'Логические операторы', conditions: 'Условия', logicalTypes: { and: 'Все условия (AND)', or: 'Хотя бы одно условие (OR)', not: 'Ни одного условия (NOT)' }, leafTypes: { state: 'Состояние сущности', numericState: 'Числовой порог', time: 'Временной диапазон', trigger: 'Триггер' }, andHelp: 'Все дочерние условия должны быть истинными.', orHelp: 'Хотя бы одно дочернее условие должно быть истинным.', notHelp: 'Дочернее условие должно быть ложным.', removeCondition: 'Удалить условие {index}', addCondition: 'Добавить условие', depthLimit: 'Достигнута максимальная глубина вложенности, поддерживаемая валидатором.', notLimit: 'NOT допускает только одно дочернее условие.', selectEntity: 'Выберите сущность', statePlaceholder: 'on, off, open...', selectSensor: 'Выберите датчик', thresholdHelp: 'Укажите хотя бы один из двух порогов.', triggerPlaceholder: 'trigger_id', unsupportedHelp: 'Этот тип условия нельзя изменить с помощью визуальных элементов. Исходная конфигурация сохранена без изменений.' } },
  pantry: { ...en.pantry, "title":"Умная кладовая и книга рецептов","sections":"Разделы кладовой","inventoryTab":"Кладовая и запасы","recipesTab":"Книга рецептов","mealCalendarTab":"Календарь питания", view: { ...en.pantry.view, "title":"Умная кладовая и книга рецептов","sections":"Разделы кладовой","inventoryTab":"Кладовая и запасы","recipesTab":"Книга рецептов","mealCalendarTab":"Календарь питания","mealCalendar":{"previousWeek":"Предыдущая неделя","nextWeek":"Следующая неделя","today":"Сегодня","planMeal":"Запланировать приём пищи","addMealForDay":"Добавить: {meal} — {day}","addMeal":"Добавить: {meal}","deleteMeal":"Удалить приём пищи","scaledIngredients":"Количество ингредиентов рассчитано на {count} чел.","consumed":"Съедено","consumeTitle":"Отметить приём пищи и автоматически списать ингредиенты из кладовой.","consume":"Съесть","mealTypes":{"breakfast":"Завтрак","lunch":"Обед","snack":"Перекус","dinner":"Ужин","dessert":"Десерт"}},"recipeEditor":{"editTitle":"Изменить рецепт","newTitle":"Новый рецепт в книге","close":"Закрыть","title":"Название рецепта","titlePlaceholder":"Например, грибное ризотто, карбонара, запечённый лосось...","category":"Блюдо / категория","basePortions":"Базовые порции (человек)","portionsHint":"Количество рассчитано на указанное число человек и будет автоматически скорректировано в календаре.","preparationMinutes":"Подготовка (мин)","cookingMinutes":"Приготовление (мин)","ingredientsFor":"Ингредиенты (на {count} чел.)","addIngredient":"Добавить ингредиент","ingredientPlaceholder":"Название ингредиента (например, спагетти)","quantityPlaceholder":"Количество","optional":"Необязательно","optionalShort":"Опц.","removeIngredient":"Удалить ингредиент","stepsTitle":"Этапы и время приготовления","stepsTotal":"Общее время этапов: {minutes} мин","applyStepsTimeTitle":"Установить время приготовления равным сумме этапов","useAsCookTime":"Использовать как время приготовления","addStep":"Добавить этап","noSteps":"Этапы не заданы. Добавьте этап, чтобы описать приготовление.","stepNumber":"Этап {number}","stepMinutesLabel":"Длительность этапа {number} в минутах","zeroPlaceholder":"0","stepTimerHint":"Длительность таймера этапа в минутах (0 — без ожидания)","minutesShort":"мин","moveUp":"Переместить вверх","moveDown":"Переместить вниз","removeStep":"Удалить этап","stepPlaceholder":"Опишите действие на этапе {number} (например, обжарить лук, добавить вино, запечь...)","save":"Сохранить рецепт","create":"Создать рецепт","titleRequired":"Укажите название рецепта.","portionsRequired":"Количество базовых порций должно быть не менее 1.","ingredientRequired":"Добавьте в рецепт хотя бы один ингредиент.","categories":{"firstCourse":"🍝 Первое блюдо","mainCourse":"🥩 Второе блюдо","sideDish":"🥗 Гарнир","singleDish":"🍲 Блюдо на каждый день","breakfast":"☕ Завтрак","dessert":"🍰 Десерт","snack":"🥪 Перекус"},"units":{"grams":"г (граммы)","kilograms":"кг (килограммы)","pieces":"шт. (штуки)","milliliters":"мл (миллилитры)","liters":"л (литры)","pack":"упаковка"}}} },
  predictiveClimate: { ...en.predictiveClimate, view: { ...en.predictiveClimate.view,
    addZoneCta: 'Добавить зону',
    loadingZones: 'Загрузка температурных зон…',
    configureFirstZone: 'Настройте первую зону, чтобы начать предиктивное управление.',
    glazedAreaLabel: 'Площадь остекления',
    perceived: 'Ощущаемая температура',
    gain: 'Теплоприток',
    evaluateNow: 'Рассчитать сейчас',
    thermalModelStart: 'Модель теплового разгона',
    editAction: 'Изменить',
    deleteAction: 'Удалить',
    requiredPreheat: 'Необходимое время предварительного нагрева',
    startsAt: 'Начало',
    earlyShutdown: 'Досрочное отключение',
    stopsAt: 'Остановка',
    cycleTrainingDescription: 'Запишите фактический цикл нагрева или охлаждения, чтобы улучшить точность модели.',
"mainLabel":"Управление предиктивным термостатом","title":"Предиктивный многозонный термостат","reassessAllZones":"Пересчитать все зоны","addZone":"Добавить предиктивную зону","environmentSummary":"Сводка условий окружающей среды","activeZones":"Активные зоны","thermalZones":"Предиктивные температурные зоны","noZones":"Предиктивные зоны не настроены.","roomPerceived":"Комната / ощущаемая температура","baseTarget":"Базовая цель","calculatedSetpoint":"Рассчитанная уставка","calculateSetpoint":"Рассчитать уставку для этой комнаты","viewThermalModel":"Показать тепловую модель и оптимальное время запуска","editZone":"Изменить параметры зоны","deleteZone":"Удалить зону","zoneName":"Название зоны / комнаты","zoneNamePlaceholder":"Например, основная гостиная","trvEntity":"Сущность TRV / термостата","roomTemperatureSensor":"Датчик температуры комнаты","pressureSensor":"Датчик давления (барометр)","humiditySensor":"Датчик относительной влажности","windowContactSensor":"Датчик открытия окна (необязательно)","targetTemperature":"Целевая температура","mainExposure":"Основная ориентация","south":"Юг (сильное солнце во второй половине дня)","east":"Восток (утреннее солнце)","west":"Запад (вечернее солнце)","north":"Север (тень, без прямого солнца)","heatingEmitter":"Тип отопительного прибора","radiator":"Радиаторы (средняя инерция, 2 ч)","floorHeating":"Тёплый пол (высокая инерция, 4 ч)","fanCoil":"Фанкойл (быстрый отклик, 1 ч)","glazedArea":"Площадь остекления","coldFrontForecast":"Оповещение о холодном фронте (барометрический прогноз)","solarCutoff":"Упреждающее ограничение солнечного нагрева (защита от перегрева)","weatherPreheat":"Предварительный нагрев по прогнозу погоды (ожидается мороз)","zoneActive":"Зона активна","learnedThermalParameters":"Изученные тепловые параметры","learnedParametersTitle":"Изученные параметры (физическая Grey-box 1R-1C)","heatingRate":"Скорость нагрева","nominalActivePower":"При номинальной активной мощности","heatLossRate":"Скорость теплопотерь","normalizedDelta":"Нормализовано для наружной разницы 10°C","learningCycles":"Циклы обучения","emaUpdates":"Зарегистрированные обновления EMA","dynamicCalculators":"Динамические калькуляторы","preheatingTitle":"Предварительный нагрев (оптимальный запуск)","preheatingDesc":"Начинайте нагрев заранее, чтобы достичь комфорта точно к заданному времени","desiredTime":"Желаемое время","coastDownTitle":"Coast-down (тепловая инерция и остановка)","coastDownDesc":"Отключайте отопление перед уходом или сном, используя накопленное тепло","departureSleepTime":"Время ухода / сна","thermalCycleSimulator":"Симулятор теплового цикла","modelTrainingTitle":"Обучение модели на реальном цикле","cycleType":"Тип цикла","activeHeating":"Активный нагрев","passiveCooling":"Пассивное охлаждение","startTemperature":"Начальная температура","endTemperature":"Конечная температура","duration":"Продолжительность","minutes":"мин","estimated":"Оценка","calculating":"Выполняется расчёт...","evaluateAllZones":"Оценить все зоны","coldFrontDetected":"Обнаружен холодный фронт","pressureStable":"Стабильное давление","barometricPreheatActive":"Активен барометрический предварительный нагрев","noBarometricDrop":"Барометрическое давление не падает","solarCutoffActive":"Активно ограничение солнечного нагрева","moderateSun":"Умеренное солнце","solarSavingsActive":"Экономия активна в помещениях с солнечной экспозицией","standardSolarGain":"Стандартный солнечный приток","pressureTrend":"Тренд","saveConfiguration":"Сохранить настройки","entityListUnavailable":"Список устройств недоступен. Можно ввести ID сущностей вручную.","profileLoadFailed":"Не удалось загрузить тепловую модель.","preheatFailed":"Не удалось рассчитать предварительный нагрев.","coastDownFailed":"Не удалось рассчитать заблаговременное отключение отопления.","invalidSample":"Недостоверные данные. Проверьте направление изменения, температуру и длительность (15–720 минут).","modelUpdated":"Модель обновлена. Новая скорость: {rate} °C/ч (образцы: {samples}).","trainingFailed":"Не удалось обновить тепловую модель.","requiredFields":"Укажите название, термостат и датчик температуры.","targetTemperatureRange":"Целевая температура должна быть от 10 до 30 °C.","awayTemperatureRange":"Температура в отсутствие должна быть от 15 до 28 °C.","windowAreaRange":"Площадь остекления должна быть от 0 до 50 м².","saveFailed":"Не удалось сохранить. Проверьте устройства и повторите попытку.","confirmDeleteZone":"Вы действительно хотите удалить эту предиктивную зону?","modeLabels":{"coldFront":"Усиление при холодном фронте","solarCutoff":"Ограничение солнечного нагрева","frostProtection":"Защита от замерзания","preheat":"Предварительный нагрев","eco":"Экономичный режим","comfort":"Стандартный комфорт"},"orientations":{"south":"Юг","north":"Север","east":"Восток","west":"Запад"},"emitters":{"floorRadiant":"Тёплый пол","fancoil":"Фанкойл","radiator":"Радиаторы"},"editZoneTitle":"Изменить предиктивную зону","newZoneTitle":"Новая предиктивная зона","trvPlaceholder":"climate.termostat_gostinaya","temperaturePlaceholder":"sensor.temperatura_gostinaya","pressurePlaceholder":"sensor.davlenie_gostinaya","humidityPlaceholder":"sensor.vlazhnost_gostinaya","windowPlaceholder":"binary_sensor.okno_gostinaya","presenceSensorOptional":"Датчик присутствия (необязательно)","presencePlaceholder":"binary_sensor.prisutstvie_gostinaya","emptyRoomHint":"Когда в комнате никого нет, применяется энергосберегающая заданная температура.","awayTemperature":"Температура в отсутствие","profileTitle":"Тепловая модель и запуск","profileSubtitle":"Физическая модель 1R-1C, интеллектуальный предварительный нагрев и заблаговременное отключение с учётом инерции","loadingProfile":"Загрузка тепловых параметров...","recordCycle":"Записать цикл","trainingHelp":"При нагреве конечная температура должна быть выше начальной, при охлаждении — ниже. Допустимая длительность цикла: от 15 до 720 минут.","neverEvaluated":"Ещё не оценивалась","dateUnavailable":"Дата недоступна","lastEvaluated":"Последняя оценка: {date}","preheatNoAdvanceReason":"Текущая температура ({current} °C) уже достигла желаемого уровня комфорта ({target} °C); предварительный нагрев не требуется.","preheatScheduledReason":"Включите отопление на {minutes} мин раньше, в {start}, чтобы достичь {target} °C к {targetTime} (расчётная скорость {rate} °C/ч; температура снаружи {outdoor} °C).","coastDownNoShutdownReason":"Раннее отключение не требуется: температура уже ниже комфортного предела ({limit} °C).","coastDownScheduledReason":"Выключите отопление на {minutes} мин раньше, в {stop}, до выхода в {departure}; расчётная экономия энергии {energy} кВт·ч.","reasons":{"frostProtection":"Окно открыто: активна защита от замерзания для предотвращения потерь тепла.","eco":"Активен экономичный режим: заданная температура снижена для уменьшения расхода энергии.","solarCutoff":"Заданная температура заблаговременно снижена из-за прогнозируемого солнечного излучения ({radiation} Вт/м²).","coldFront":"Обнаружено падение атмосферного давления на {trend} гПа/ч; включена компенсация приближающегося холодного фронта.","preheat":"Активен упреждающий нагрев: прогнозируемая температура снаружи {outdoor} °C.","comfort":"Отопление работает в стандартном комфортном режиме.","default":"Активно предиктивное управление климатом."}} },
  sharedDashboard: { ...en.sharedDashboard, view: { ...en.sharedDashboard.view, "invalidToken":"Ссылка общего доступа недействительна или срок её действия истёк.","loading":"Загрузка общей панели…","accessDenied":"Доступ запрещён или срок действия истёк","loginToDomus":"Войти в Domus","readOnly":"Только чтение","guestAccess":"Гостевой доступ","widgetsAria":"Виджеты панели","active":"Активно","emptyWidgets":"В этой панели не настроено ни одного виджета."} },
  tour: { ...en.tour, stepsAria: "Шаги тура", goToStep: "Перейти к шагу {step}" },
}


Object.assign(ru.flowchart.view, {
  newAutomationName: 'Новая визуальная автоматизация',
  paletteCategories: { triggers: 'Триггеры', conditions: 'Условия', actionsFlow: 'Действия и сценарий' },
  palette: {
    stateChangeTitle: 'Изменение состояния', stateChangeDesc: 'Изменение состояния сущности, например выключателя или датчика',
    numericThresholdTitle: 'Числовой порог', timeTriggerTitle: 'Запланированное время',
    timeTriggerDesc: 'Точное время (HH:MM:SS), рассвет или закат', stateConditionTitle: 'Текущее состояние',
    stateConditionDesc: 'Проверить, активно ли устройство', timeConditionTitle: 'Временной интервал',
    timeConditionDesc: 'Только в заданный промежуток времени, например ночью', entityCommandTitle: 'Команда сущности',
    entityCommandDesc: 'Включить, выключить, открыть или настроить климат', activateSceneTitle: 'Активировать сценарий',
    activateSceneDesc: 'Применить световую атмосферу или профиль', delayTitle: 'Пауза',
    delayDesc: 'Запланированное ожидание перед следующим шагом', sendNotificationTitle: 'Отправить уведомление',
    sendNotificationDesc: 'Отправить приоритетное push-уведомление на смартфон', chooseTitle: 'Условный выбор',
    chooseDesc: 'Выполнить первую ветвь с выполненными условиями, предусмотрев запасной вариант',
    ifElseTitle: 'Если / Иначе', ifElseDesc: 'Выполнить разные последовательности в зависимости от условий',
    repeatTitle: 'Повторить', repeatDesc: 'Повторять заданное число раз, пока или до выполнения условия',
    parallelTitle: 'Параллельное выполнение', parallelDesc: 'Запустить несколько действий одновременно',
    climateProfileTitle: 'Климатический профиль', climateProfileDesc: 'Комфорт, Эко, Ночь, Отсутствие или заданная температура',
    emitEventTitle: 'Создать событие', emitEventDesc: 'Опубликовать событие приложения с данными'
  },
  branchContinue: 'Продолжить', timeFrom: 'С (HH:MM:SS)', timeTo: 'До (HH:MM:SS)',
  nestedConditions: 'Вложенные условия',
  nestedConditionsDescription: 'Настройте группы AND, OR и NOT, добавляя условия и другие группы. Неподдерживаемые типы сохраняются без изменений.',
  triggerId: 'ID триггера', nestedConfiguration: 'Вложенная конфигурация (JSON)',
  nestedConfigurationDescription: 'Редактируйте здесь ветви, условия и последовательности. Domus сохраняет вложенную структуру; некорректный JSON или неподдерживаемое действие блокирует сохранение.',
  nodeTypes: { trigger: 'ТРИГГЕР', condition: 'УСЛОВИЕ', action: 'ДЕЙСТВИЕ', delay: 'ПАУЗА', logic: 'ЛОГИКА' },
  validation: {
    missingNode: 'В сценарии есть ссылка на несуществующий узел.',
    invalidOrder: 'Неверный порядок: триггеры, условия и действия должны следовать порядку сценария.',
    singleOutput: 'У узла основной последовательности может быть только один выход. Используйте именованные порты IF/ELSE или CHOOSE для создания ветвей.',
    disconnectedNode: 'Узел «{title}» не подключён к сценарию. Подключите все условия и действия перед сохранением.',
    cycle: 'В сценарии обнаружен цикл. Удалите циклические связи перед сохранением.',
    missingBranchNode: 'Ветвь указывает на несуществующий узел.',
    missingBranchPort: 'Выход «{port}» больше не существует у узла «{node}».',
    conditionPortTarget: 'Порты условий можно подключать только к узлам условий.',
    actionBranchTarget: 'Ветви составных действий могут содержать только действия или паузы.',
    branchRejoin: 'Ветвь не может возвращаться в основную последовательность. Продублируйте последующие действия в каждой ветви.',
    branchMerge: 'У каждого узла внутри ветви должна быть ровно одна входящая связь. Объединение ветвей не поддерживается.',
    unsupportedTrigger: 'Триггер «{type}» пока нельзя изменить в визуальном редакторе. Сохранение заблокировано, чтобы не потерять исходную конфигурацию.',
    invalidLogicJson: 'Логическое условие содержит некорректный JSON.',
    logicObjectRequired: 'Логическое условие должно быть объектом JSON.',
    unsupportedCondition: 'Условие «{type}» пока нельзя изменить в визуальном редакторе. Сохранение заблокировано, чтобы не потерять исходную конфигурацию.',
    unsupportedAction: 'Это расширенное действие нельзя изменить в визуальном редакторе. Сохранение заблокировано, чтобы сохранить исходную конфигурацию.'
  },
  notifications: {
    updatedTitle: 'Автоматизация обновлена', updatedMessage: '«{name}» успешно сохранена.',
    createdTitle: 'Автоматизация создана', createdMessage: '«{name}» теперь активна на хабе.',
    saveErrorTitle: 'Ошибка сохранения', saveErrorMessage: 'Не удалось сохранить сценарий.',
    saveBeforeTestTitle: 'Сначала сохраните сценарий',
    saveBeforeTestMessage: 'Перед запуском симуляции сохраните автоматизацию. Тест не отправляет команды устройствам.',
    simulationSuccessTitle: 'Симуляция завершена', simulationSuccessMessage: 'Условия выполнены.',
    simulationFailedTitle: 'Симуляция не пройдена',
    simulationFailedMessage: 'Одно или несколько условий не выполнены. Проверьте подробности, чтобы найти причину.',
    simulationResultTitle: 'Результат симуляции', simulationErrorTitle: 'Ошибка симуляции',
    simulationErrorMessage: 'Не удалось смоделировать сценарий.', invalidFlowTitle: 'Некорректный сценарий',
    invalidFlowMessage: 'Исправьте связи перед экспортом.'
  }
});
Object.assign(ru.integrations.view, {
  pendingPairingInstructions: 'Нажмите «Сканировать сеть» или введите IP-адрес устройства Shelly, Sonoff или Tasmota, чтобы импортировать его напрямую без облака.',
  networkGuardianDescription: 'Активный мониторинг LAN, пассивный анализ ARP/DHCP и немедленное обнаружение несанкционированной передачи данных на внешние облачные серверы.',
  anomaliesRegistered: 'обнаруженных аномалий', noWritableFunctions: 'API не вернул доступных для записи функций этого устройства.',
  host: 'Узел', deviceTotals: 'всего · онлайн · офлайн'
});
Object.assign(ru.energy, {
  powerWheelTitle: 'Энергетические потоки в реальном времени', powerWheelSubtitle: 'Динамическое отображение энергетических потоков с анимированными частицами.',
  inverter: 'Инвертор', inverterStatus: 'Состояние инвертора', arbitrageSavingsToday: 'Экономия на арбитраже сегодня',
  economicalGridCharge: 'Экономичная зарядка от сети', batteryPeakShaving: 'Снижение пиков нагрузки аккумулятором',
  wallboxTitle: 'Интеллектуальная солнечная зарядка и Wallbox OCPP', activePower: 'Активная мощность',
  deliveredCurrent: 'Передаваемый ток', vehicleBatterySoc: 'Аккумулятор автомобиля (SoC)',
  sessionEnergy: 'Энергия сессии', chargingMode: 'Режим управления зарядкой', solarOnly: 'Только солнечная энергия (100% ФЭ)',
  active: 'Активно', solarOnlyDesc: 'Зарядка только за счёт чистого избытка солнечной энергии (не менее 6 А / 1,4 кВт), без потребления из сети.',
  solarOptimized: 'Оптимизированная солнечная зарядка (гибридная)',
  solarOptimizedDesc: 'Поддерживает непрерывные 6 А (1,4 кВт) и динамически увеличивает ток при избытке солнечной энергии.',
  fastCharge: 'Быстрая зарядка (максимальная мощность)',
  fastChargeDesc: 'Зарядка с максимально доступной договорной мощностью и защитой от перегрузки счётчика.',
  scheduledNight: 'Экономичная ночная зарядка',
  scheduledNightDesc: 'Планируйте зарядку в периодах F3 или в ночные часы с минимальным PUN для максимальной экономии.',
  balancingDecision: 'Решение алгоритма динамической балансировки:',
  tariffF1: 'F1 (пиковый период: пн–пт 08–19)', tariffF2: 'F2 (средний период: пн–пт 07–08 и 19–23, сб 07–23)',
  tariffF3: 'F3 (непиковый период: ночи, воскресенья и праздники)', electricity: 'Электричество'
});
Object.assign(ru.settings, {
  updated: 'Обновлено', otaEmpty: 'Нет устройств, зарегистрированных для OTA-обновлений.',
  nfcEmpty: 'В базе данных нет зарегистрированных NFC-меток.', testNfcScan: 'Проверить сканирование',
  backupDescription: 'Автоматическое локальное или удалённое резервное копирование на NAS/WebDAV с шифрованием AES-256-GCM. При аппаратном сбое восстановите данные одним нажатием.',
  vacationLightingDescription: 'Вместо включения света по фиксированному расписанию Domus воспроизводит привычки за последние 14 дней со случайным отклонением ±{minutes} мин.',
  batteryDiagnosticsDescription: 'Мониторинг батарей с предупреждением ниже 15% и диагностикой радиопомех Zigbee/Thread, включая рекомендации по расширению mesh-сети.',
  waterLeakDescription: 'Если счётчик обнаруживает непрерывный поток более 40 минут ночью, Domus предупреждает о возможной утечке и может закрыть главный электромагнитный клапан.'
});
Object.assign(ru.family, { roleUpdated: 'Роль обновлена', createGuestProfile: 'Создать гостевой профиль' });
Object.assign(ru.floorplan.view, {
  floors: 'Этажи', rooms: 'Комнаты', mapView: 'Вид карты', levelShort: 'Эт.',
  roomPrefix: 'Комната:', positionX: 'X:', positionY: 'Y:', widthShort: 'Ш:', heightShort: 'В:',
  editRoomHint: 'Нажмите на комнату на плане, чтобы изменить её положение, размер и цвет.',
  allDevicesPlaced: 'Все устройства уже размещены на этой карте!', finishEditing: 'Завершить редактирование',
  editPositions: 'Изменить позиции'
});
Object.assign(ru.shopping.categories, {
  alimentari: 'Продукты', frutta_verdura: 'Фрукты и овощи', bevande: 'Напитки',
  igiene: 'Личная гигиена', casa: 'Дом и уборка', altro: 'Другое'
});
Object.assign(ru.shopping.view, {
  moveBoughtToPantry: 'Переместить покупки в кладовую ({count})', allSupermarkets: 'Все ({count})',
  allItemsBought: 'Все товары из списка покупок приобретены!', markAsBought: 'Отметить как купленное',
  recognizedAs: 'распознано как «{name}»', quantityAria: 'Количество', markItem: 'Отметить {name} как {status}',
  filterSupermarket: 'Фильтр по {name}', justNow: 'Только что', toBuy: 'купить', bought: 'куплено'
});
Object.assign(ru.store.view, {
  pendingReview: 'На проверке', rejected: 'Отклонено', communityLabel: 'Сообщество', verifiedLabel: 'Проверено',
  cancel: 'Отмена', close: 'Закрыть', requiredPackageFields: 'Укажите название и ID пакета.',
  repositoryRequired: 'Требуется действительный URL публичного репозитория с открытым исходным кодом (например, GitHub или GitLab).',
  licenseRequired: 'Требуется действующая лицензия с открытым исходным кодом (например, Apache-2.0, MIT или GPL-3.0).',
  submissionFailed: 'Ошибка при публикации пакета.', rejectedByAdministrator: 'Отклонено администратором'
});
Object.assign(ru.dashboard.view, {
  customize: 'Настроить', newDashboard: '+ Создать', deleteDashboard: 'Удалить',
  ambientSetpoint: 'Заданная температура помещения', measuredTemperature: 'Измеренная температура',
  instantaneousPower: 'Текущая мощность дома', configuredEntity: 'Настроенная сущность Domus',
  customTextWidget: 'Пользовательский текстовый виджет', createCustomTitle: 'Создать пользовательскую панель',
  customizeTitle: 'Настроить панель', editorHint: 'Выберите название, виджеты и размеры сетки.',
  dashboardName: 'Название панели', dashboardNamePlaceholder: 'Например: Панель первого этажа',
  addWidget: 'Добавить виджет', widgetType: 'Тип виджета',
  widgetTypes: { thermostat: 'Термостат', temperature: 'Температура', energy: 'Энергия', entity: 'Сущность', text: 'Текст / Заметка' },
  widgetTitlePlaceholder: 'Название виджета (необязательно)', widgetTitleAria: 'Название виджета',
  entityIdPlaceholder: 'ID сущности (необязательно)', entityIdAria: 'ID сущности', add: 'Добавить',
  widgetsCount: 'Виджеты панели ({count})', noWidgets: 'Виджеты не добавлены. Можно создать пустую панель и настроить её позже.',
  removeWidget: 'Удалить', cancel: 'Отмена', saveDashboard: 'Сохранить панель', closeEditor: 'Закрыть редактор',
  deleteConfirm: 'Удалить панель «{title}»?', dashboardNameRequired: 'Введите название панели.',
  saveFailed: 'Не удалось сохранить панель. Повторите попытку.', deleteFailed: 'Не удалось удалить панель.'
});
Object.assign(ru.devices.view, {
  discoveryReady: 'устройств готовы к немедленной настройке.', ipPrefix: 'IP:', deviceCount: 'устройств',
  addToHome: 'Добавить в умный дом Domus.', addToDomus: 'Добавить в Domus'
});
Object.assign(ru.dashboardMarketplace, { community: 'Сообщество', verified: 'Проверено' });
Object.assign(ru.sharedDashboard, { expiresOn: 'Срок действия до:' });
Object.assign(ru.predictiveClimate.view, { close: 'Закрыть' });


Object.assign(ru.floorplan.view, {
  commandSentTitle: 'Команда отправлена', entityTurnedOn: '{name}: включено', entityTurnedOff: '{name}: выключено',
  commandFailed: 'Не удалось отправить команду', positionSavedTitle: 'Позиция сохранена',
  positionSavedMessage: 'Устройство размещено на плане', markerSaveFailed: 'Не удалось сохранить маркер',
  markerRemovedTitle: 'Устройство удалено', markerRemovedMessage: 'Устройство удалено с плана',
  markerRemoveFailed: 'Не удалось удалить маркер',
  roomNames: { living: 'Гостиная', room: 'Комната', livingOpen: 'Гостиная открытой планировки', kitchen: 'Кухня', masterBedroom: 'Спальня', bathroom: 'Ванная', office: 'Кабинет / офис' },
  roomCreatedTitle: 'Комната создана', roomCreatedMessage: 'Комната «{name}» добавлена на план',
  floorplanGeneratedTitle: 'План создан', floorplanGeneratedMessage: 'Подготовлено пять комнат со стандартной планировкой',
  roomRemovedTitle: 'Комната удалена', roomRemovedMessage: 'Комната удалена с плана',
  floorCreatedTitle: 'Этаж создан', floorCreatedMessage: 'Новый этаж добавлен на план', floorCreateFailed: 'Не удалось создать этаж'
});
Object.assign(ru.devices.view, {
  addToDomus: 'Добавить в Domus', scanCompletedTitle: 'Сканирование сети завершено',
  scanCompletedMessage: 'В локальной сети найдено устройств: {count}', scanErrorTitle: 'Ошибка сканирования',
  scanErrorMessage: 'Не удалось завершить сканирование', deviceAddedTitle: 'Устройство добавлено',
  deviceAddedMessage: '{name} теперь настроено в Domus.', addDeviceErrorTitle: 'Ошибка добавления устройства',
  addDeviceErrorMessage: 'Не удалось добавить устройство', matterConnectedTitle: 'Matter подключён',
  matterConnectedMessage: 'Устройство подключено к локальной сети Thread', devicesUpdatedTitle: 'Устройства обновлены',
  devicesUpdatedMessage: 'Список устройств и состояние интеграций обновлены.',
  refreshErrorTitle: 'Ошибка обновления', refreshErrorMessage: 'Не удалось обновить устройства.'
});
Object.assign(ru.dashboard.view, {
  extensionUninstalled: 'Расширение успешно удалено', extensionInstalled: 'Расширение установлено и активно локально',
  extensionToggleFailed: 'Не удалось изменить состояние пакета'
});
Object.assign(ru.store.view, { rejectedByAdministrator: 'Отклонено администратором' });
Object.assign(ru.integrations.view, { deviceTotals: 'всего · онлайн · офлайн' });


Object.assign(ru.energy, {
  nodeSolar: 'СОЛНЕЧНАЯ ЭНЕРГИЯ', nodeGrid: 'СЕТЬ', nodeHomeLoads: 'ПОТРЕБЛЕНИЕ ДОМА',
  nodeBattery: 'АККУМУЛЯТОР', nodeWallbox: 'ЗАРЯДНАЯ СТАНЦИЯ EV', standby: '0 Вт (Ожидание)',
  recalculateArbitrage: 'Пересчитать оптимальный арбитраж',
  wallboxSubtitle: 'Динамическое управление IEC 61851 на основе чистого избытка солнечной энергии с защитой от перегрузки счётчика',
  recalculateSolarModulation: 'Пересчитать солнечную модуляцию', updating: 'Обновление…',
  recalculateModulation: 'Пересчитать модуляцию', target: 'Целевое значение',
  pwmAvailableCurrent: 'Доступный ток PWM', station: 'Станция', solarSurplus: 'Избыток солнечной энергии',
  homeLoads: 'Потребление дома', total: 'Итого', cost: 'Стоимость'
});
Object.assign(ru.floorplan.view.roomNames, {
  kitchenDining: 'Кухня и столовая', bedroom: 'Спальня', masterBathroom: 'Главная ванная'
});
Object.assign(ru.devices.view, { filteredCount: '{filtered} из {total} устройств' });
Object.assign(ru.integrations.view, { ipLabel: 'IP', firmwareLabel: 'Прошивка' });
Object.assign(ru.shopping.view, {
  itemsAdded: 'Новых товаров: {count}', itemsUpdated: 'Обновлено товаров: {count}',
  itemsTransferred: 'Товары перенесены', transferCompleted: 'Покупки сохранены в кладовой ({summary})!'
});


Object.assign(ru.floorplan.view, {
  closeHeatmap: 'Закрыть тепловую карту', heatmap25d: 'Тепловая карта 2.5D', modeLabel: 'Режим'
});
Object.assign(ru.integrations.view, {
  tuyaCloudSetupInstruction: 'Создайте учётную запись на iot.tuya.com, откройте Cloud → Development и выберите проект Domus.'
});


Object.assign(ru.floorplan.view.roomNames, { newRoom: 'Новая комната' });
Object.assign(ru.floorplan.view.roomTypes, {
  living: 'Гостиная', kitchen: 'Кухня / Столовая', bedroom: 'Спальня', bathroom: 'Ванная',
  office: 'Кабинет / офис', hallway: 'Коридор / Прихожая', balcony: 'Терраса / Балкон'
});
Object.assign(ru.floorplan.view.colors, {
  oceanBlue: 'Океанский синий', emerald: 'Изумрудный', amber: 'Янтарный',
  nightPurple: 'Ночной фиолетовый', skyCyan: 'Небесно-голубой', coralPink: 'Кораллово-розовый'
});
Object.assign(ru.energy, {
  enableAutoSchedule: 'Включить автоматическое расписание',
  disableAutoSchedule: 'Отключить автоматическое расписание'
});


Object.assign(ru.flowchart.view.notifications, {
  simulationWithoutTriggersMessage: 'Условия выполнены. Триггеры не моделировались.'
});
Object.assign(ru.dashboard.view, {
  removeWidgetAria: 'Удалить виджет {title}'
});
Object.assign(ru.store.view, {
  communityOpenSource: 'Сообщество открытого ПО', domusVerified: 'Проверено Domus'
});


Object.assign(ru.guide, {
  searchAria: 'Поиск по разделам руководства',
  collapseSection: 'Свернуть раздел',
  expandSection: 'Развернуть раздел'
});
Object.assign(ru.pantry.view, {
  cookableRecipes: 'рецептов можно приготовить'
});
Object.assign(ru.medicines, {
  aic: 'Код AIC',
  lotNumber: 'Номер партии'
});
Object.assign(ru.automations.view, {
  webhookCreatedTitle: 'Входящий webhook создан',
  webhookCreatedMessage: 'Webhook «{name}» успешно создан.',
  webhookCreateFailed: 'Не удалось создать webhook.',
  webhookDeletedTitle: 'Webhook удалён',
  webhookDeletedMessage: 'Webhook «{name}» успешно удалён.',
  webhookDeleteFailed: 'Не удалось удалить webhook.',
  flowSavedTitle: 'Сценарий сохранён',
  flowSavedMessage: 'Автоматизация успешно создана из блок-схемы.'
});
Object.assign(ru.common.notifications, {
  actionExecutedTitle: 'Действие выполнено',
  actionExecutedMessage: 'Команда успешно отправлена.',
  actionFailedMessage: 'Не удалось выполнить запрошенное действие.'
});


Object.assign(ru.automations.editor, {
  titleEdit: 'Изменить автоматизацию', titleNew: 'Новая автоматизация', close: 'Закрыть окно автоматизации',
  name: 'Название автоматизации', namePlaceholder: 'Например, включить ночник', executionMode: 'Режим выполнения',
  description: 'Описание (необязательно)', descriptionPlaceholder: 'Опишите назначение этого правила…',
  cooldown: 'Интервал между запусками (секунды)', dslRule: 'Правило JSON DSL (триггеры, условия, действия)',
  climateProgram: 'Программа термостата', motionPreset: 'Шаблон движения', schedulePreset: 'Шаблон расписания',
  format: 'Форматировать', climateBuilderAria: 'Программа термостата на основе датчика',
  climateProgramTitle: 'Ежедневная программа по температуре',
  climateProgramDescription: 'В выбранное время считывается датчик и устанавливается одна из двух заданных температур.',
  loadingDevices: 'Загрузка устройств…',
  missingEntities: 'Нужны как минимум один датчик температуры и один термостат. Убедитесь, что они настроены в Domus.',
  sensorToRead: 'Датчик для считывания', currentTemperatureHint: 'Считывание текущей температуры в помещении.',
  thermostatToControl: 'Термостат для управления', profileDuration: 'Длительность профиля (минуты)',
  profileDurationHint: '0 = действует до следующего изменения по расписанию; большее значение восстановит предыдущую заданную температуру по истечении срока.',
  automaticProfiles: 'Автоматические профили',
  automaticProfileSummary: '07:00 Комфорт · 09:00 Эко · 18:00 Комфорт · 22:30 Ночь, ежедневно.',
  generateProfiles: 'Создать программу Комфорт/Эко/Ночь', weeklySchedule: 'Недельное расписание',
  weeklyScheduleHint: 'Выберите дни и время. При каждом запуске Domus считывает актуальную температуру и регулирует термостат.',
  addSlot: 'Добавить временной интервал', slot: 'Интервал {index}', removeSlot: 'Удалить интервал {index}',
  activeDays: 'Активные дни интервала {index}', time: 'Время', sensorThreshold: 'Порог датчика (°C)',
  belowThreshold: 'Если ниже порога (°C)', otherwiseSet: 'Иначе установить (°C)',
  generateRule: 'Создать правило DSL', jsonPlaceholder: 'JSON DSL…', cancel: 'Отмена', save: 'Сохранить автоматизацию',
  profileName: 'Профили термостата {name}',
  profileDescription: 'Недельная программа Комфорт/Эко/Ночь. Длительность профиля: {duration}.',
  untilNextChange: 'до следующего изменения по расписанию',
  restoreAfterMinutes: '{minutes} мин., затем восстановить предыдущую заданную температуру',
  climateName: 'Программа термостата {name}',
  weeklyDescription: 'Считывать {sensor} и регулировать {thermostat} по недельному расписанию: {summary}.',
  scheduleSummary: '{days} в {time}: ниже {threshold} °C → {cold} °C, иначе {warm} °C'
});
Object.assign(ru.automations.editor.executionModes, {
  single: 'Одиночный запуск (пропустить, если уже выполняется)',
  restart: 'Перезапуск (остановить и запустить заново)',
  queued: 'Очередь (поставить запросы в очередь)',
  parallel: 'Параллельно (одновременное выполнение)'
});
Object.assign(ru.automations.editor.errors, {
  entitiesLoad: 'Не удалось загрузить устройства.',
  thermostatRequired: 'Выберите термостат перед созданием профилей.',
  durationRange: 'Временная длительность должна быть от 0 до 1440 минут.',
  sensorAndThermostatRequired: 'Выберите датчик температуры и термостат.',
  slotRequired: 'Добавьте хотя бы один временной интервал.',
  invalidTime: 'Введите время в формате ЧЧ:ММ.',
  dayRequired: 'Выберите хотя бы один допустимый день для каждого интервала.',
  duplicateTime: 'Время {time} повторяется хотя бы для одного выбранного дня.',
  thresholdRange: 'Пороги и заданные температуры должны быть от 5 до 35 °C.',
  nameRequired: 'Укажите название автоматизации.',
  invalidJson: 'Некорректный синтаксис JSON.',
  triggerRequired: 'Правило должно содержать хотя бы один триггер.',
  actionRequired: 'Правило должно содержать хотя бы одно действие.'
});
Object.assign(ru.automations.validation, {
  objectExpected: '{path} должен быть объектом JSON.',
  arrayExpected: '{path} должен быть массивом.',
  minLength: '{path} должен содержать не менее {min} элементов.',
  required: '{path} является обязательным.',
  numberRange: '{path} должен быть числом JSON от {min} до {max}.',
  integer: '{path} должен быть целым числом.',
  maxDepth: '{path}: превышена максимальная глубина вложенности.',
  conditionType: '{path}: неподдерживаемый тип условия «{type}».',
  numericThreshold: '{path}: укажите верхний или нижний числовой порог.',
  timeRange: '{path}: укажите время начала, окончания или оба значения.',
  notSingleCondition: '{path}.conditions должен содержать ровно одно условие для NOT.',
  maxActions: 'Конфигурация превышает лимит в {max} вложенных действий.',
  actionType: '{path}: неподдерживаемый тип действия «{type}».',
  paramsObject: '{path}.params должен быть объектом JSON.',
  eventPayloadObject: '{path}.event_payload должен быть объектом JSON.',
  profile: '{path}.profile должен быть comfort, eco, night или away.',
  chooseBranch: '{path}: добавьте ветвь choose или действие по умолчанию.',
  repeatMode: '{path}.repeat: выберите только один режим — count, while или until.',
  repeatSequence: '{path}.repeat.sequence не может быть пустым.',
  jsonRequired: 'Введите полную JSON-конфигурацию для этого действия.',
  jsonInvalid: 'Некорректная конфигурация JSON.',
  typeMismatch: 'Поле type для этого узла должно иметь значение «{type}».'
});
Object.assign(ru.automations.graphErrors, {
  duplicatePort: 'Выход «{port}» имеет несколько соединений. У каждого порта может быть только одно назначение.',
  conditionCycle: 'Последовательность условий содержит цикл или повторно используемый узел.',
  conditionTarget: 'Порт условия может соединяться только с узлами условий.',
  logicJson: 'Логическое условие содержит некорректный JSON.',
  logicObject: 'Логическое условие должно быть объектом JSON.',
  unsupportedCondition: 'Условие «{type}» не поддерживается визуальным редактором.',
  conditionSingleOutput: 'Последовательность условий может иметь только один линейный выход.',
  actionCycle: 'Ветвь содержит цикл или повторно используемый узел.',
  missingNode: 'Ветвь содержит несуществующий узел.',
  unsupportedAction: 'Ветвь содержит неподдерживаемое действие.',
  notExecutable: 'Ветвь содержит узел, который нельзя выполнить как действие.',
  actionSingleOutput: 'Действие внутри ветви может иметь только один линейный выход.',
  parallelSingle: 'Каждая параллельная ветвь должна содержать одно действие: движок выполняет их параллельно.'
});
Object.assign(ru.automations.graphErrors.conditionTitles, {
  state: 'Условие состояния', numeric: 'Числовой порог', time: 'Временной интервал',
  trigger: 'Условие триггера', and: 'Группа AND', or: 'Группа OR', not: 'Группа NOT', nested: 'Вложенное условие'
});
Object.assign(ru.automations.graphErrors.actionTitles, {
  delay: 'Задержка', scene: 'Применить сценарий', service: 'Команда', advanced: 'Расширенное действие ({type})'
});
Object.assign(ru.automations.graphErrors, { delaySubtitle: 'Ожидание {seconds} с', actionSubtitle: '{service} → {target}' });

Object.assign(ru.flowchart.conditionEditor, {
  entityLabel: 'Сущность', stateLabel: 'Требуемое состояние', sensorLabel: 'Датчик',
  aboveLabel: 'Больше чем', belowLabel: 'Меньше чем', afterLabel: 'После',
  beforeLabel: 'До', triggerIdLabel: 'ID триггера'
});
Object.assign(ru.guide, {
  searchAria: 'Поиск по руководству', collapse: 'Свернуть раздел', expand: 'Развернуть раздел'
});
Object.assign(ru.pantry.view.kpis, {
  filterLowStock: 'Фильтровать товары с низким запасом',
  filterExpiring: 'Фильтровать товары с истекающим сроком',
  showCookable: 'Показать рецепты, которые можно приготовить сегодня',
  totalItems: 'Товары в кладовой', totalItemsSub: 'Отслеживаемые запасы',
  lowStock: 'Низкий запас', lowStockSub: 'Ниже минимального порога',
  expiring: 'Истекает через 3 дня', expiringSub: 'Скоро использовать',
  cookable: 'Что можно приготовить?', cookableSub: 'Все ингредиенты в наличии'
});
Object.assign(ru.satellites, {
  assistantTitle: 'Голосовой помощник Domus', engineBadge: 'Сначала локальная работа (Ollama LLM и эвристический NLP)',
  resetConversation: 'Сбросить разговор', clearHistory: 'Очистить историю разговора',
  closeAssistant: 'Закрыть голосового помощника', sectionNavigation: 'Разделы голосового помощника',
  suggestedCommands: 'Быстрые команды', listeningTitle: 'Прослушивание… нажмите, чтобы остановить',
  activateDictation: 'Включить голосовой ввод', stopDictation: 'Остановить голосовой ввод',
  askPlaceholder: 'Спросите Domus… (например, «Выключи весь свет, кроме кухни»)',
  sendCommand: 'Отправить команду голосовому помощнику', muteSatellite: 'Отключить микрофон спутника',
  unmuteSatellite: 'Включить микрофон спутника', testCommandPlaceholder: 'Команда для проверки в комнате «{room}»…',
  sending: 'Отправка…', wakeWord: 'Ключевое слово', nodesCount: 'Узлов: {count}',
  defaultTestPrompt: 'Включи свет', processingForRoom: 'Обработка для комнаты «{room}»: «{prompt}»…',
  voiceResponse: '{room} · Голосовой ответ: «{response}»',
  commandSucceeded: '{room} · Команда «{prompt}» успешно выполнена в контексте комнаты.',
  contextApplied: '{room} · Команда «{prompt}» применена к комнате.',
  quickChips: {
    lightsOff: 'Выключи весь свет', kitchenLight: 'Включи свет на кухне',
    temperature: 'Установи температуру 21 градус', goodnight: 'Включи сценарий «Спокойной ночи»',
    pastaTimer: 'Запусти таймер для пасты на 10 минут'
  },
  fallbackSatellites: {
    livingRoom: 'Настенный планшет в гостиной', kitchen: 'Умный дисплей на кухне', bedroom: 'Планшет у кровати',
    livingRoomArea: 'Гостиная', kitchenArea: 'Кухня', bedroomArea: 'Спальня'
  }
});
Object.assign(ru.satellites.fallbackSatellites, {
  livingRoomArea: 'Гостиная', kitchenArea: 'Кухня', bedroomArea: 'Спальня'
});
Object.assign(ru.satellites.quickChips, {
  lightsOff: 'Выключи весь свет', kitchenLight: 'Включи свет на кухне',
  temperature: 'Установи температуру 21 градус', goodnight: 'Включи сценарий «Спокойной ночи»',
  pastaTimer: 'Запусти таймер для пасты на 10 минут'
});
Object.assign(ru.schedules.view, {
  monday: 'Понедельник', tuesday: 'Вторник', wednesday: 'Среда', thursday: 'Четверг',
  friday: 'Пятница', saturday: 'Суббота', sunday: 'Воскресенье'
});

Object.assign(ru.integrations.view, { tuyaProjectIntro: 'Откройте свой проект на', tuyaProjectPath: 'Cloud → Development → откройте проект Domus' });


Object.assign(ru.automations, { cooldown: 'Время ожидания' });
Object.assign(ru.integrations.view, { deviceOnline: 'В сети', deviceOffline: 'Не в сети' });
Object.assign(ru.audit, { timestamp: 'Дата и время' });
Object.assign(ru.flowchart.view.nodeTypes, { trigger: 'ТРИГГЕР' });
Object.assign(ru.flowchart.conditionEditor.leafTypes, { trigger: 'Триггер' });
Object.assign(ru.pantry.view.mealCalendar.mealTypes, { dessert: 'Десерт' });
Object.assign(ru.pantry.view.recipeEditor.categories, { dessert: '🍰 Десерт', snack: '🥪 Перекус' });
Object.assign(ru.guide, { sec7RouteLabel: 'Панель управления и сигнализация' });
Object.assign(ru.store.view, { blueprints: 'Шаблоны' });
Object.assign(ru.dashboard, { marketplaceTitle: 'Каталог и расширения' });
Object.assign(ru.mesh, { router: 'Маршрутизатор / повторитель' });

Object.assign(ru.common, { notificationRegion: 'Уведомления' });

export default ru
