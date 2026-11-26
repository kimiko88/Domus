# Xiaomi Home Cloud: migrazione al Plugin Manager

## Ambito

La configurazione Open Platform di Xiaomi Home viene ora gestita dall'istanza persistente `xiaomi_home_cloud` del Plugin Manager.

L'identificativo dell'istanza rimane `integration_xiaomi_home_cloud`, esattamente come l'identificativo di integrazione usato dai dispositivi, dalle entità e dagli eventi già sincronizzati. Il record storico è già presente nella tabella `integrations` e ha un formato JSON compatibile: l'avvio del Manager lo ripristina direttamente, senza copiare i dati in una seconda tabella o modificare l'identificativo.

## Dati conservati

La configurazione cifrata dell'istanza contiene:

- credenziali dell'app Xiaomi Open Platform e regione;
- URL pubblico, redirect callback e segreti callback/eventi;
- stato OAuth temporaneo anti-CSRF;
- access token, refresh token, MAC key/algoritmo e scadenza;
- identificativo e nickname dell'account collegato.

I campi token, stato OAuth e segreti interni sono marcati `Secret`/`Internal`: non vengono pubblicati nello schema visibile del catalogo Plugin Manager. Le API di stato continuano a restituire solo informazioni non segrete e identificativi mascherati.

## Comportamento runtime

- `GET /api/v1/integrations/xiaomi-home/cloud/status` legge la configurazione decifrata dal Manager.
- `PUT /api/v1/integrations/xiaomi-home/cloud/config` crea l'istanza al primo salvataggio oppure aggiorna atomicamente i campi configurati.
- `/authorize`, callback OAuth, refresh token, dispositivi e comandi MIoT mantengono gli endpoint esistenti.
- Gli aggiornamenti OAuth usano `Manager.UpdateConfigMutate`, che cifra e persiste la nuova configurazione sotto il lock lifecycle.
- L'handler recupera l'adapter corrente per ID a ogni creazione del client; non conserva un puntatore obsoleto quando il Manager sostituisce l'adapter dopo un aggiornamento.
- `POST /disconnect` elimina token e dati di sessione, ma conserva credenziali applicative, callback e configurazione di base.
- Se il Plugin Manager non è disponibile, resta il percorso di compatibilità precedente sul repository cifrato delle integrazioni.

## Verifiche aggiunte

I test coprono il ripristino del record storico, la cifratura senza segreti in chiaro, la configurazione da zero, la presenza dell'adapter supervisionato, la rotazione della configurazione, la risoluzione dell'adapter aggiornato e la disconnessione selettiva. I test del Registry verificano inoltre che Xiaomi Home Cloud sia visibile nel catalogo ma che i campi OAuth interni non siano esposti.
