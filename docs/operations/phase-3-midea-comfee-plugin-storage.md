# Midea / Comfee SmartHome LAN: migrazione al Plugin Manager

## Identità e compatibilità

- Tipo pubblico: `midea_smarthome_local`.
- ID stabile: `integration_midea_smarthome`, invariato rispetto ai record e ai dispositivi già salvati.
- Il Plugin Manager usa il campo interno `devices` per la configurazione cifrata dei dispositivi.
- La migrazione non introduce una nuova tabella né cambia gli ID usati nel Device Registry e nelle Entity.

I record esistenti hanno già il tipo `midea_smarthome_local` e il payload `{"devices":[...]}`: il Manager può ripristinarli direttamente. Il costruttore dell'handler mantiene un percorso di lettura compatibile per ricostruire la vista applicativa e il Manager crea la propria istanza runtime dallo stesso contenuto cifrato. Il binding controlla che ogni dispositivo configurato sia presente nel runtime gestito.

## Confine del plugin

Il plugin gestisce esclusivamente il trasporto LAN M-Smart V2/V3 per il profilo A1 (deumidificatori Midea/Comfee):
- driver V2 e V3 separati;
- mapping dispositivi;
- credenziali V3 specifiche del dispositivo;
- discovery/lettura/comandi attraverso l'adapter Midea già esistente.

Il campo `devices` è marcato `Secret` e `Internal`. La configurazione è cifrata dal Manager; il campo viene omesso dal catalogo e non compare nei metadati delle istanze. Il modello `Device`, le risposte REST, gli eventi e i log non espongono token o key. L'istanza resta singola perché la tabella dispositivi legacy è legata all'ID stabile dell'integrazione.

Non viene introdotto un login Midea cloud. Il token e la key V3 sono materiale di autenticazione LAN per il singolo apparecchio. Le password dell'app consumer, account cloud e refresh token non fanno parte del flusso implementato.

## Lifecycle

- All'avvio, `Manager.Start` ripristina la riga abilitata, costruisce entrambi i driver, valida la configurazione dei dispositivi e avvia il runtime supervisionato.
- L'handler storico riceve il Plugin Manager e risolve l'adapter corrente tramite l'ID stabile, evitando di continuare a usare un adapter sostituito.
- Aggiunta, sostituzione o rimozione di dispositivi aggiornano atomicamente il campo interno `devices`; il Manager crea e verifica l'adapter candidato prima di sostituire quello attivo.
- Quando l'ultimo dispositivo viene eliminato, l'istanza viene disabilitata e il runtime viene fermato. L'ID e il record di stato restano disponibili.
- I percorsi REST esistenti sotto `/api/v1/integrations/midea` non cambiano.

## Test aggiunti

- manifest pubblico singleton e campo dispositivo internal/secret;
- creazione factory con l'ID storico e rifiuto di un ID alternativo;
- limiti al numero dei dispositivi, protocollo ammesso, validazione endpoint/credenziali;
- ripristino della configurazione V3 cifrata da un record esistente;
- verifica che token, key e indirizzo LAN non siano in chiaro nel record e che i segreti non siano esposti dalle API del Plugin Manager;
- sostituzione dell'adapter dopo rotazione credenziali/aggiunta device e stop dell'adapter precedente;
- disconnessione, cancellazione della configurazione attiva e disabilitazione dell'istanza.

I test unitari e di integrazione simulata non sostituiscono il collaudo su dispositivi fisici. Prima di dichiarare la compatibilità generale occorre verificare almeno un dispositivo V2 e uno V3 e mantenere la compatibilità limitata ai profili/modelli realmente testati.

## Correzione del blocco CI di partenza

La compilazione del repository era bloccata da una seconda dichiarazione di `CloudPluginType` e `CloudIntegrationID` in `internal/plugins/xiaomi_home/adapters.go`. È stata rimossa la coppia duplicata, mantenendo le costanti originali. La riga `go: downloading` e i pacchetti WASM contrassegnati `[no test files]` non erano errori: il fallimento concreto era il redeclare che impediva di compilare `cmd/domus`, `internal/plugins/xiaomi_home` e i test che dipendevano da questi package.
