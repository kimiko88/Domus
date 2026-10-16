#!/usr/bin/env python3
import json
import os
import re

LANGUAGES = ['it', 'en', 'fr', 'es', 'de']

MOBILE_EXTRA = {
    'app_name': {
        'it': 'Domus', 'en': 'Domus', 'fr': 'Domus', 'es': 'Domus', 'de': 'Domus'
    },
    'connection_mode_lan': {
        'it': 'LAN Locale', 'en': 'Local LAN', 'fr': 'LAN Local', 'es': 'LAN Local', 'de': 'Lokales LAN'
    },
    'connection_mode_remote': {
        'it': 'TLS Remoto (Fuori Casa)', 'en': 'Remote TLS (Away)', 'fr': 'TLS Distant (Hors Maison)', 'es': 'TLS Remoto (Fuera de Casa)', 'de': 'Remote-TLS (Unterwegs)'
    },
    'connection_connecting': {
        'it': 'Connessione in corso...', 'en': 'Connecting...', 'fr': 'Connexion en cours...', 'es': 'Conectando...', 'de': 'Verbindung wird hergestellt...'
    },
    'connection_disconnected': {
        'it': 'Disconnesso', 'en': 'Disconnected', 'fr': 'Déconnecté', 'es': 'Desconectado', 'de': 'Getrennt'
    },
    'connection_connected': {
        'it': 'Connesso', 'en': 'Connected', 'fr': 'Connecté', 'es': 'Conectado', 'de': 'Verbunden'
    },
    'login_title': {
        'it': 'Accesso a Domus', 'en': 'Sign in to Domus', 'fr': 'Connexion à Domus', 'es': 'Iniciar sesión en Domus', 'de': 'Bei Domus anmelden'
    },
    'login_subtitle': {
        'it': 'Connettiti al tuo hub locale o remoto', 'en': 'Connect to your local or remote hub', 'fr': 'Connectez-vous à votre hub local ou distant', 'es': 'Conéctate a tu hub local o remoto', 'de': 'Verbinden Sie sich mit Ihrem lokalen oder Remote-Hub'
    },
    'login_server_address': {
        'it': 'Indirizzo Hub (LAN o TLS Remoto)', 'en': 'Hub Address (LAN or Remote TLS)', 'fr': 'Adresse du Hub (LAN ou TLS distant)', 'es': 'Dirección del Hub (LAN o TLS remoto)', 'de': 'Hub-Adresse (LAN oder Remote-TLS)'
    },
    'login_discover_hub': {
        'it': 'Cerca Hub in rete...', 'en': 'Search Hub on network...', 'fr': 'Rechercher le Hub sur le réseau...', 'es': 'Buscar Hub en la red...', 'de': 'Hub im Netzwerk suchen...'
    },
    'login_button': {
        'it': 'Accedi', 'en': 'Sign In', 'fr': 'Se Connecter', 'es': 'Iniciar Sesión', 'de': 'Anmelden'
    },
    'login_error_credentials': {
        'it': 'Credenziali non valide o hub non raggiungibile', 'en': 'Invalid credentials or hub unreachable', 'fr': 'Identifiants invalides ou hub inaccessible', 'es': 'Credenciales inválidas o hub inaccesible', 'de': 'Ungültige Anmeldedaten oder Hub nicht erreichbar'
    },
    'floorplan_title': {
        'it': 'Planimetria Interattiva', 'en': 'Interactive Floorplan', 'fr': "Plan d'Étage Interactif", 'es': 'Plano Interactivo', 'de': 'Interaktiver Grundriss'
    },
    'floorplan_select_floor': {
        'it': 'Seleziona Piano', 'en': 'Select Floor', 'fr': "Sélectionner l'étage", 'es': 'Seleccionar Piso', 'de': 'Etage auswählen'
    },
    'floorplan_position_device': {
        'it': 'Posiziona Dispositivo', 'en': 'Place Device', 'fr': "Placer l'appareil", 'es': 'Colocar Dispositivo', 'de': 'Gerät platzieren'
    },
    'floorplan_drag_hint': {
        'it': "Trascina l'icona per regolare la posizione sulla mappa", 'en': 'Drag the icon to adjust position on the map', 'fr': "Faites glisser l'icône pour ajuster la position sur la carte", 'es': 'Arrastra el icono para ajustar la posición en el mapa', 'de': 'Ziehen Sie das Symbol, um die Position auf der Karte anzupassen'
    },
    'chores_karma_points': {
        'it': 'Punti Karma Famiglia', 'en': 'Family Karma Points', 'fr': 'Points Karma Famille', 'es': 'Puntos Karma Familiar', 'de': 'Familien-Karma-Punkte'
    },
    'chores_pending_tasks': {
        'it': 'in attesa', 'en': 'pending', 'fr': 'en attente', 'es': 'pendientes', 'de': 'ausstehend'
    },
    'chores_no_karma': {
        'it': 'Nessun punto karma accumulato', 'en': 'No karma points earned yet', 'fr': 'Aucun point karma accumulé', 'es': 'No hay puntos de karma acumulados', 'de': 'Noch keine Karma-Punkte gesammelt'
    },
    'chores_empty_list': {
        'it': 'Nessuna faccenda presente', 'en': 'No chores found', 'fr': 'Aucune tâche présente', 'es': 'No hay tareas disponibles', 'de': 'Keine Aufgaben vorhanden'
    },
    'chores_filter_all': {
        'it': 'Tutti', 'en': 'All', 'fr': 'Tous', 'es': 'Todos', 'de': 'Alle'
    },
    'chores_filter_todo': {
        'it': 'Da Fare', 'en': 'To Do', 'fr': 'À faire', 'es': 'Por hacer', 'de': 'Zu erledigen'
    },
    'chores_filter_completed': {
        'it': 'Completati', 'en': 'Completed', 'fr': 'Terminés', 'es': 'Completados', 'de': 'Erledigt'
    },
    'chores_new_task': {
        'it': 'Nuova Faccenda', 'en': 'New Chore', 'fr': 'Nouvelle Tâche', 'es': 'Nueva Tarea', 'de': 'Neue Aufgabe'
    },
    'chores_priority_low': {
        'it': 'Bassa', 'en': 'Low', 'fr': 'Basse', 'es': 'Baja', 'de': 'Niedrig'
    },
    'chores_priority_medium': {
        'it': 'Media', 'en': 'Medium', 'fr': 'Moyenne', 'es': 'Media', 'de': 'Mittel'
    },
    'chores_priority_high': {
        'it': 'Alta', 'en': 'High', 'fr': 'Haute', 'es': 'Alta', 'de': 'Hoch'
    },
    'chores_priority_urgent': {
        'it': 'Urgente', 'en': 'Urgent', 'fr': 'Urgente', 'es': 'Urgente', 'de': 'Dringend'
    },
    'chores_assigned_to': {
        'it': 'Assegnato a', 'en': 'Assigned to', 'fr': 'Assigné à', 'es': 'Asignado a', 'de': 'Zugewiesen an'
    },
    'chores_room': {
        'it': 'Stanza / Area', 'en': 'Room / Area', 'fr': 'Pièce / Zone', 'es': 'Habitación / Área', 'de': 'Raum / Bereich'
    },
    'shopping_title': {
        'it': 'Lista della Spesa', 'en': 'Shopping List', 'fr': 'Liste de Courses', 'es': 'Lista de Compras', 'de': 'Einkaufsliste'
    },
    'shopping_empty': {
        'it': 'La lista della spesa è vuota', 'en': 'Shopping list is empty', 'fr': 'La liste de courses est vide', 'es': 'La lista de compras está vacía', 'de': 'Die Einkaufsliste ist leer'
    },
    'shopping_clear_completed': {
        'it': 'Rimuovi completati', 'en': 'Clear completed', 'fr': 'Effacer les terminés', 'es': 'Eliminar completados', 'de': 'Erledigte entfernen'
    },
    'energy_title': {
        'it': 'Flusso Energetico', 'en': 'Energy Flow', 'fr': "Flux d'Énergie", 'es': 'Flujo de Energía', 'de': 'Energiefluss'
    },
    'assistant_title': {
        'it': 'Assistente Domus', 'en': 'Domus Assistant', 'fr': 'Assistant Domus', 'es': 'Asistente Domus', 'de': 'Domus Assistent'
    },
    'assistant_placeholder': {
        'it': 'Chiedi qualcosa o impartisci un comando...', 'en': 'Ask something or give a command...', 'fr': 'Demandez quelque chose ou donnez un ordre...', 'es': 'Pregunta algo o da una orden...', 'de': 'Fragen Sie etwas oder geben Sie einen Befehl...'
    },
    'settings_language': {
        'it': 'Lingua', 'en': 'Language', 'fr': 'Langue', 'es': 'Idioma', 'de': 'Sprache'
    },
    'settings_language_system': {
        'it': 'Predefinita di sistema', 'en': 'System default', 'fr': 'Par défaut du système', 'es': 'Predeterminado del sistema', 'de': 'Systemstandard'
    }
}

def to_snake(name):
    # Convert camelCase to snake_case
    s1 = re.sub('(.)([A-Z][a-z]+)', r'\1_\2', name)
    s2 = re.sub('([a-z0-9])([A-Z])', r'\1_\2', s1).lower()
    return re.sub(r'[^a-zA-Z0-9_]', '_', s2)

def flatten(obj, prefix=''):
    items = {}
    for k, v in obj.items():
        key = f"{prefix}_{to_snake(k)}" if prefix else to_snake(k)
        if isinstance(v, dict):
            items.update(flatten(v, key))
        elif isinstance(v, str):
            items[key] = v
    return items

def escape_android(text):
    if not isinstance(text, str):
        text = str(text)
    text = text.replace('&', '&amp;')
    text = text.replace('<', '&lt;')
    text = text.replace('>', '&gt;')
    text = text.replace("'", "\\'")
    text = text.replace('"', '\\"')
    text = text.replace('\n', '\\n')
    return text

def escape_ios(text):
    if not isinstance(text, str):
        text = str(text)
    text = text.replace('\\', '\\\\')
    text = text.replace('"', '\\"')
    text = text.replace('\n', '\\n')
    return text

def main():
    base_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
    locales_dir = os.path.join(base_dir, 'web', 'src', 'locales')

    data = {}
    for lang in LANGUAGES:
        json_path = os.path.join(locales_dir, f"{lang}.json")
        with open(json_path, 'r', encoding='utf-8') as f:
            d = json.load(f)
        flat = flatten(d)
        for extra_key, translations in MOBILE_EXTRA.items():
            if lang in translations:
                flat[extra_key] = translations[lang]
        data[lang] = flat

    # Find common keys
    all_keys = sorted(data['it'].keys())

    # 1. Generate Android strings
    res_base = os.path.join(base_dir, 'mobile', 'android', 'app', 'src', 'main', 'res')
    for lang in LANGUAGES:
        folder_name = 'values' if lang == 'it' else f'values-{lang}'
        folder_path = os.path.join(res_base, folder_name)
        os.makedirs(folder_path, exist_ok=True)
        xml_path = os.path.join(folder_path, 'strings.xml')

        lines = ['<?xml version="1.0" encoding="utf-8"?>', '<resources>']
        for k in all_keys:
            val = data[lang].get(k, data['it'].get(k, ''))
            escaped = escape_android(val)
            lines.append(f'    <string name="{k}">{escaped}</string>')
        lines.append('</resources>')
        lines.append('')

        with open(xml_path, 'w', encoding='utf-8') as f:
            f.write('\n'.join(lines))
        print(f"Written Android {xml_path} ({len(all_keys)} strings)")

    # Also write values-it explicitly for multi-locale devices
    folder_it = os.path.join(res_base, 'values-it')
    os.makedirs(folder_it, exist_ok=True)
    xml_path_it = os.path.join(folder_it, 'strings.xml')
    lines_it = ['<?xml version="1.0" encoding="utf-8"?>', '<resources>']
    for k in all_keys:
        val = data['it'].get(k, '')
        escaped = escape_android(val)
        lines_it.append(f'    <string name="{k}">{escaped}</string>')
    lines_it.append('</resources>')
    lines_it.append('')
    with open(xml_path_it, 'w', encoding='utf-8') as f:
        f.write('\n'.join(lines_it))
    print(f"Written Android {xml_path_it}")

    # 2. Generate iOS Localizable.strings in DomusKit and domus app
    ios_destinations = [
        os.path.join(base_dir, 'mobile', 'ios', 'Sources', 'DomusKit', 'Resources'),
        os.path.join(base_dir, 'mobile', 'ios', 'domus', 'domus')
    ]

    for dest in ios_destinations:
        for lang in LANGUAGES:
            lproj = os.path.join(dest, f"{lang}.lproj")
            os.makedirs(lproj, exist_ok=True)
            strings_file = os.path.join(lproj, 'Localizable.strings')

            lines = [f'/* Domus Localizable Strings ({lang}) */']
            for k in all_keys:
                val = data[lang].get(k, data['it'].get(k, ''))
                escaped = escape_ios(val)
                lines.append(f'"{k}" = "{escaped}";')
            lines.append('')

            with open(strings_file, 'w', encoding='utf-8') as f:
                f.write('\n'.join(lines))
            print(f"Written iOS {strings_file} ({len(all_keys)} strings)")

if __name__ == '__main__':
    main()
