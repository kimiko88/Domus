# Fase 3 – Storage OAuth per istanza Tuya Cloud

## Obiettivo della tranche

La roadmap cloud viene affrontata per passi indipendenti. Questa tranche migra i token Smart Life OAuth e la configurazione OpenAPI del vecchio option flow Tuya Cloud al repository cifrato delle istanze Plugin Manager. Le tabelle legacy `tuya_user_oauth_connections` e `tuya_cloud_config` vengono ritirate dopo migrazione riuscita; la tabella `tuya_user_oauth_states` resta per lo state CSRF monouso.

## Modello dati

- L'istanza singleton `tuya_cloud` contiene un campo interno `smart_life_oauth` con una mappa di account indicizzata dall'ID utente Domus.
- Ogni account memorizza UID Smart Life, access token, refresh token e scadenza.
- L'intero JSON dell'istanza è cifrato dal `EnvelopeCipher` prima della persistenza; non viene creato un secondo archivio token in chiaro.
- Le credenziali App Authorization (`oauth_client_id` / `oauth_secret`) restano distinte dalle credenziali Tuya OpenAPI (`client_id` / `secret`). Un'istanza OAuth-only può esistere senza inventare o riutilizzare credenziali OpenAPI; il client OpenAPI si attiva solo quando vengono configurate le sue chiavi.
- I campi `oauth_client_id`, `oauth_secret`, `oauth_identifier` e `smart_life_oauth` sono dichiarati interni nello schema. Il Registry li valida, ma `Manager.ListTypes` li filtra dal catalogo mostrato nella UI.
- Gli endpoint pubblici delle istanze continuano a restituire solo metadati; il nuovo accessor `Manager.GetConfig` è destinato esclusivamente a consumer fidati del livello applicativo e non va esposto via HTTP.
- Le modifiche interne passano da `Manager.UpdateConfigMutate`, che serializza lettura/modifica/cifratura sotto il lock del lifecycle. Le modifiche pubbliche con `Manager.UpdateConfig` preservano i campi interni già salvati e non possono sovrascriverli o cancellarli.

## Option flow Tuya OpenAPI

L'endpoint amministrativo esistente `/api/v1/integrations/tuya-cloud/config` usa ora il Plugin Manager come unica sorgente di verità quando il runtime è disponibile:

- `GET` legge client ID, UID ed endpoint dalla configurazione cifrata e non restituisce mai il client secret.
- `PUT` salva credenziali OpenAPI dopo validazione, con aggiornamento atomico e replace supervisionato dell'adapter. `POST` resta alias temporaneo per i client web già distribuiti.
- `DELETE` elimina soltanto `client_id`, `secret`, `uid` e `schema` OpenAPI. Non elimina l'istanza Tuya, il client App Authorization né i token Smart Life OAuth.
- Gli aggiornamenti vengono eseguiti tramite `Manager.UpdateConfigMutate`, quindi i campi interni OAuth sono conservati anche quando l'utente salva dal vecchio form di configurazione.

## Flusso di connessione e refresh

1. L'amministratore avvia l'autorizzazione.
2. Il callback usa lo state monouso associato all'utente Domus; la tabella `tuya_user_oauth_states` rimane l'archivio CSRF, non contiene token Smart Life.
3. I token ricevuti vengono salvati nella configurazione cifrata dell'istanza attraverso `Manager.UpdateConfigMutate`.
4. Refresh ordinario e refresh dopo rifiuto token aggiornano la stessa entry utente in modo serializzato; un amministratore non sovrascrive le credenziali di un altro.
5. Il disconnect rimuove solo l'entry dell'utente che ha richiesto la disconnessione.

## Migrazione compatibile

All'avvio del `TuyaCloudHandler`, se è presente la vecchia tabella `tuya_cloud_config`:
- il client secret viene decifrato dal vecchio `EnvelopeCipher`;
- le impostazioni OpenAPI vengono unite alla configurazione dell'istanza `tuya_cloud` usando `UpdateConfigMutate`, preservando i campi OAuth interni;
- se esiste già una configurazione Plugin Manager completa, questa resta autoritativa e la vecchia riga viene ritirata come duplicato obsoleto;
- la tabella legacy viene eliminata soltanto dopo che la nuova configurazione è stata salvata o confermata.

Se la migrazione fallisce, la tabella e la riga legacy restano disponibili per un nuovo tentativo. Le nuove installazioni con Plugin Manager non creano più `tuya_cloud_config`. Le installazioni senza Plugin Manager conservano temporaneamente il percorso legacy per compatibilità.

Per OAuth, al primo accesso a un account non ancora presente nella configurazione Plugin Manager, il handler:
- legge i token legacy dalla tabella `tuya_user_oauth_connections`;
- li decifra con il cipher esistente;
- li salva nell'istanza tramite `Manager.UpdateConfigMutate`, che cifra la configurazione completa;
- elimina la vecchia riga solo dopo la conferma della persistenza nuova.

In caso di errore nella scrittura nuova, la riga legacy non viene cancellata. La migrazione è quindi ripetibile senza perdere la sessione dell'utente. La tabella `tuya_user_oauth_states` resta il solo archivio SQL del flusso OAuth, per lo state CSRF monouso.

## Hardening e test

- I dati OAuth interni sono esclusi dallo schema pubblico di configurazione.
- I token non vengono aggiunti a `Instance`, log o risposte API.
- Il test di migrazione verifica che la riga legacy scompaia solo dopo la scrittura cifrata e che i blob/risultati pubblici non contengano i token.
- Il test multi-account verifica isolamento per utente e disconnect selettivo.
- I test dell'option flow coprono migrazione `tuya_cloud_config`, installazione nuova, GET/PUT/POST/DELETE, conservazione dei token OAuth quando vengono rimosse le credenziali OpenAPI e assenza dei segreti nelle risposte.
- Il test OAuth-only verifica che le credenziali App Authorization non siano copiate nei campi OpenAPI.
- Il test di aggiornamento pubblico verifica che una modifica ordinaria delle impostazioni Tuya preservi i campi interni non mostrati nel form.
- I test precedenti del percorso legacy restano attivi per installazioni che non hanno il Plugin Manager.

## Lavoro ancora aperto

- Migrare Xiaomi Home Cloud, Imou Life, VeSync e Midea a storage account/istanza con refresh e callback supervisionati.
- Completare prove end-to-end con progetto IoT Tuya reale, permessi API User Permission Package e hardware Smart Life. I test automatici usano trasporti simulati e non dimostrano la disponibilità dei pacchetti API del progetto Tuya.

Questa tranche non introduce sandbox per plugin Go nativi: continuano a eseguire con i privilegi del processo Domus.
