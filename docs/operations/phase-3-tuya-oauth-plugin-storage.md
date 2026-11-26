# Fase 3 – Storage OAuth e configurazione OpenAPI per istanza Tuya Cloud

## Obiettivo della tranche

La roadmap cloud viene affrontata per passi indipendenti. Questa tranche migra i token di collegamento Smart Life OAuth dal vecchio archivio per-utente `tuya_user_oauth_connections` al repository delle istanze Plugin Manager. La stessa tranche include anche la migrazione dell'option flow Tuya OpenAPI (`GET/POST/DELETE /integrations/tuya-cloud/config`) dal vault separato alla configurazione cifrata dell'istanza.

## Modello dati

- L'istanza singleton `tuya_cloud` contiene le impostazioni pubbliche OpenAPI (`client_id`, `base_url`, `uid`, `schema`) e il segreto OpenAPI `secret` cifrato assieme all'intera configurazione.
- Il campo interno `smart_life_oauth` contiene una mappa di account indicizzata dall'ID utente Domus. Ogni account memorizza UID Smart Life, access token, refresh token e scadenza.
- L'intero JSON dell'istanza è cifrato dal `EnvelopeCipher` prima della persistenza; non viene creato un secondo archivio token in chiaro.
- Le credenziali App Authorization (`oauth_client_id` / `oauth_secret`) restano distinte dalle credenziali Tuya OpenAPI (`client_id` / `secret`). Un'istanza OAuth-only può esistere senza inventare o riutilizzare credenziali OpenAPI; il client OpenAPI si attiva solo quando vengono configurate le sue chiavi.
- I campi `oauth_client_id`, `oauth_secret`, `oauth_identifier` e `smart_life_oauth` sono dichiarati interni nello schema. Il Registry li valida, ma `Manager.ListTypes` li filtra dal catalogo mostrato nella UI.
- Gli endpoint pubblici delle istanze continuano a restituire solo metadati; il nuovo accessor `Manager.GetConfig` è destinato esclusivamente a consumer fidati del livello applicativo e non va esposto via HTTP.
- Le modifiche interne passano da `Manager.UpdateConfigMutate`, che serializza lettura/modifica/cifratura sotto il lock del lifecycle. Le modifiche pubbliche con `Manager.UpdateConfig` preservano i campi interni già salvati e non possono sovrascriverli o cancellarli.

## Flusso di connessione e refresh

1. L'amministratore avvia l'autorizzazione.
2. Il callback usa lo state monouso associato all'utente Domus; la tabella `tuya_user_oauth_states` rimane l'archivio CSRF, non contiene token Smart Life.
3. I token ricevuti vengono salvati nella configurazione cifrata dell'istanza attraverso `Manager.UpdateConfigMutate`.
4. L'option flow Tuya OpenAPI legge le impostazioni da `Manager.GetConfig`; `POST /config` aggiorna i campi OpenAPI attraverso `Manager.UpdateConfigMutate`, senza sostituire i dati OAuth interni.
5. Refresh ordinario e refresh dopo rifiuto token aggiornano la stessa entry utente in modo serializzato; un amministratore non sovrascrive le credenziali di un altro.
6. Il disconnect rimuove solo l'entry dell'utente che ha richiesto la disconnessione.
7. `DELETE /config` rimuove soltanto le credenziali OpenAPI (`client_id`, `secret`, `uid`, `schema`); non elimina l'istanza e non scollega gli account OAuth.

## Migrazione compatibile

Per Smart Life OAuth, al primo accesso a un account non ancora presente nella configurazione Plugin Manager, il handler:
- legge i token legacy dalla tabella `tuya_user_oauth_connections`;
- li decifra con il cipher esistente;
- li salva nell'istanza tramite `Manager.UpdateConfigMutate`, che cifra la configurazione completa;
- elimina la vecchia riga solo dopo la conferma della persistenza nuova.

In caso di errore nella scrittura nuova, la riga legacy non viene cancellata. La migrazione è quindi ripetibile senza perdere la sessione dell'utente. Le installazioni senza Plugin Manager conservano il percorso legacy di compatibilità.

La configurazione App Authorization viene copiata nei campi interni cifrati dell'istanza quando questa viene preparata per il flusso OAuth. Le variabili d'ambiente restano la sorgente di bootstrap.

Per la configurazione OpenAPI legacy, `TuyaCloudHandler.WithStorage` verifica l'esistenza della vecchia tabella, decifra la riga `default`, crea o aggiorna l'istanza Tuya e cancella la riga solo dopo il commit cifrato. Le installazioni managed non creano la tabella e non vi scrivono più; essa resta disponibile solo per la modalità di compatibilità senza Plugin Manager.

## Hardening e test

- I dati OAuth interni sono esclusi dallo schema pubblico di configurazione.
- I token non vengono aggiunti a `Instance`, log o risposte API.
- Il test di migrazione verifica che la riga legacy scompaia solo dopo la scrittura cifrata e che i blob/risultati pubblici non contengano i token.
- Il test multi-account verifica isolamento per utente e disconnect selettivo.
- Il test OAuth-only verifica che le credenziali App Authorization non siano copiate nei campi OpenAPI.
- Il test di aggiornamento pubblico verifica che una modifica ordinaria delle impostazioni Tuya preservi i campi interni non mostrati nel form.
- I test Tuya Cloud verificano migrazione della vecchia riga OpenAPI, lettura senza esporre `secret`, salvataggio tramite Plugin Manager, conservazione di `schema` e token OAuth durante gli aggiornamenti e cancellazione selettiva delle credenziali OpenAPI.
- I test precedenti del percorso legacy restano attivi per installazioni che non hanno il Plugin Manager.

## Lavoro ancora aperto

- Migrare Xiaomi Home Cloud, Imou Life, VeSync e Midea a storage account/istanza con refresh e callback supervisionati.
- Completare prove end-to-end con progetto IoT Tuya reale, permessi API User Permission Package e hardware Smart Life. I test automatici usano trasporti simulati e non dimostrano la disponibilità dei pacchetti API del progetto Tuya.

Questa tranche non introduce sandbox per plugin Go nativi: continuano a eseguire con i privilegi del processo Domus.
