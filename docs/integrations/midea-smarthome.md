# Midea SmartHome / MSmartHome

Questo documento descrive il modello tecnico previsto da Domus per l'app Midea SmartHome / MSmartHome e per gli elettrodomestici dei marchi OEM compatibili. La presenza del marchio Midea o la possibilità di controllare un apparecchio dall'app non garantiscono, da sole, che il suo protocollo sia accessibile a Domus.

## 1. Prima regola: individuare il protocollo reale

SmartHome è un ecosistema di prodotti, non un protocollo unico. Apparecchi Midea, Comfee e marchi OEM possono utilizzare moduli Wi-Fi e firmware diversi.

| Trasporto | Percorso | Credenziali | Strategia |
|---|---|---|---|
| M-Smart LAN V1 | Discovery LAN e comunicazione locale | In genere senza token/key V3 | Locale |
| M-Smart LAN V2 | Discovery UDP e connessione TCP locale | In genere senza token/key V3 | Locale |
| M-Smart LAN V3 | Discovery UDP e connessione TCP autenticata | Token e key per apparecchio, da importare o acquisire durante onboarding | Locale dopo onboarding |
| Firmware basato su Tuya | Adapter Tuya Local / Cloud già presente in Domus | Credenziali Tuya e local key, quando disponibile | Riutilizzo adapter Tuya |
| Midea Open Platform | API cloud per partner | Applicazione partner approvata e OAuth | Integrazione cloud separata e opzionale |

Le implementazioni M-Smart di riferimento utilizzano discovery UDP sulle porte 6445 e, per alcune famiglie, 20086; la connessione dati è normalmente TCP sulla porta 6444. Sono dettagli osservati nelle implementazioni pubbliche, non una garanzia valida per tutti i modelli.

Le API consumer usate da applicazioni di terze parti per ottenere token e key V3 non coincidono con la Midea Open Platform ufficiale. Sono interfacce non documentate e possono cambiare o essere dismesse. Le credenziali non devono essere inviate a proxy cloud non controllati. Quando possibile, l'onboarding deve importare una coppia token/key già ottenuta e protetta; il normale controllo deve poi rimanere sulla LAN.

Riferimenti tecnici:
- Home Assistant Midea: https://www.home-assistant.io/integrations/midea
- Midea Local: https://github.com/midea-lan/midea-local
- Midea AC LAN: https://github.com/wuwentao/midea_ac_lan
- Deumidificatori Midea / Comfee: https://github.com/nbogojevic/midea-beautiful-air
- Midea Open Platform v3: https://iot.midea.com/docs/control-midea-cloud-devices/cloud-2-cloud-v3-api.html

## 2. Apparecchi e capacità

Il modello Domus deve poter rappresentare tutte le famiglie riconosciute e supportate dal protocollo effettivo, senza limitarsi ai climatizzatori. Le categorie comprendono:

- Climatizzatori, pompe di calore, controller MDV e sistemi di aria fresca.
- **Deumidificatori** e umidificatori.
- Purificatori, Air Box, ventilatori, ventilatori da soffitto e luci.
- Lavatrici a carica frontale e dall'alto, asciugatrici, lavastoviglie e sterilizzatori.
- Forni, microonde, forni a vapore, cuociriso, pentole a pressione, slow cooker e tostapane.
- Frigoriferi, scaldacqua, apparecchi per acqua potabile, cappe e apparecchi da bagno.
- Altre famiglie OEM quando viene aggiunto un profilo verificato.

Il catalogo dei tipi è una tassonomia, non una promessa di controllo universale. Per ogni singolo modello e firmware il driver deve leggere il report delle capacità o il relativo thing-model e intersecarlo con il profilo del prodotto. Se un apparecchio non espone un campo scrivibile, Domus non deve mostrarne il controllo. I modelli sconosciuti restano identificabili come non classificati, senza comandi inventati.

### Deumidificatori

Il deumidificatore è una categoria di primo livello, distinta da climatizzatore e sensore generico. Il profilo iniziale A1 prevede i seguenti campi, da confermare sul report del dispositivo:

- Accensione e spegnimento.
- Modalità di funzionamento.
- Umidità ambientale rilevata e umidità obiettivo.
- Temperatura rilevata.
- Velocità ventilatore.
- Serbatoio pieno / allarme serbatoio.
- Vita filtro, energia ed eventuali codici errore.

Il profilo di base marca scrivibili soltanto power, mode, target humidity e fan speed. I modi consentiti e i limiti di umidità effettivi devono arrivare dal modello o dal dispositivo; non si deve assumere che tutti i deumidificatori accettino gli stessi setpoint. Il test usa un apparecchio simulato con intervallo 30–80% RH per verificare la validazione.

## 3. Architettura Domus

~~~text
SmartHome / MSmartHome
        |
        | onboarding controllato / import credenziali V3
        v
Archivio credenziali cifrato Domus
        |
        v
Identificazione modello, firmware e protocollo
        |
        +-------------+-------------+--------------+
        |             |             |              |
     M-Smart V1    M-Smart V2    M-Smart V3     Tuya based
        |             |             |              |
        +-------------+-------------+--------------+
                      |
              Interfaccia Driver Go
                      |
              Adapter Midea Domus
         discovery / lettura / comandi / stato
                      |
              Device & Entity Registry
                      |
             Aree, clima, storico e automazioni
~~~

Il contratto Driver separa il protocollo concreto dal dominio Domus:

- Protocol identifica la famiglia di trasporto.
- Discover individua gli apparecchi sulla LAN autorizzata.
- Read acquisisce uno snapshot dello stato corrente.
- Execute invia un singolo comando. Un timeout o una risposta ambigua non autorizzano a ripetere il comando.

L'adapter mantiene un lock per apparecchio, valida ciascun comando rispetto alle capacità scrivibili, esegue il write una sola volta e richiede una lettura indipendente per verificare lo stato. Se la lettura fallisce o il dispositivo non applica il valore, l'operazione restituisce un errore esplicito e non dichiara un successo.

Il tipo Device contiene solo informazioni pubbliche del dispositivo e non token, key o password. Le credenziali V3 devono essere conservate nel repository integrazioni cifrato tramite EnvelopeCipher e passate al driver soltanto in memoria. Non devono mai essere incluse in log, eventi, JSON di risposta o dump diagnostici.

## 4. Sicurezza e affidabilità

1. La discovery è limitata alla LAN autorizzata, con timeout e context; non deve scansionare reti pubbliche.
2. Rifiutare indirizzi pubblici, loopback, multicast e non specificati come destinazioni locali.
3. Non utilizzare mai l'IP restituito dal cloud come destinazione di un socket LAN.
4. Cifrare token/key V3 a riposo. Eliminare la password dell'app dopo l'onboarding e non registrarla nei log.
5. Proteggere configurazione e comandi con ruolo amministratore; le risposte sensibili devono usare Cache-Control no-store.
6. Serializzare operazioni sul singolo apparecchio e permettere letture concorrenti su dispositivi diversi.
7. I comandi fisici sono at-most-once. In caso di esito incerto non ritentare automaticamente: rileggere lo stato oppure richiedere un'azione esplicita dell'utente.
8. Usare polling con intervallo limitato, jitter e backoff; non generare scritture SQLite ad alta frequenza. La telemetria climatica deve passare dalla raccolta e aggregazione già presenti in Domus.
9. I log debug devono contenere protocollo, durata, classe di errore e identificativo Domus pseudonimizzato, mai credenziali, MAC, seriali o payload di autenticazione.
10. I test con fixture protocollo devono usare catture sanitizzate; i test hardware devono essere separati dai test unitari.

## 5. API amministrativa e flusso di configurazione

Tutti gli endpoint sono sotto `/api/v1/integrations/midea` e richiedono un utente amministratore.

| Metodo | Endpoint | Scopo |
|---|---|---|
| GET | `/status` | Stato configurazione, categorie gestite e numero di dispositivi |
| POST | `/discovery/scan` | Scansione locale UDP su porte 6445 e 20086 |
| GET | `/devices` | Inventario configurato, senza token/key |
| POST | `/devices` | Associazione di un dispositivo A1 rilevato |
| POST | `/devices/{id}/refresh` | Lettura dello stato e sincronizzazione delle entità |
| POST | `/devices/{id}/commands` | Comando con verifica indipendente read-back |
| DELETE | `/devices/{id}` | Rimozione credenziali, adapter ed entità Domus |

Configurazione di un M-Smart V2:

~~~json
{
  "id": "123456789",
  "name": "Deumidificatore cantina",
  "brand": "Comfee",
  "model": "MDDF16WF",
  "type_code": "A1",
  "host": "192.168.1.50",
  "port": 6444,
  "protocol": "midea_msmart_v2"
}
~~~

Per M-Smart V3 aggiungere `token` (128 caratteri esadecimali) e `key` (64 caratteri esadecimali), specifici del singolo apparecchio. Il client UI non richiede la password dell'app SmartHome. Il server rifiuta la configurazione se ID, IP, porta, tipo e protocollo non corrispondono all'ultima discovery eseguita da Domus; questo vincolo evita di trasformare l'endpoint in un proxy generico per gli host privati.

Esempio comando:

~~~json
{
  "values": {
    "power": true,
    "target_humidity": 45,
    "mode": "continuous",
    "fan_speed": "medium"
  }
}
~~~

Per il reset filtro usare l'azione `reset_filter`. I comandi ammessi sono verificati contro il profilo A1 e i limiti del dispositivo. L'adapter non ripete mai un comando fisico in caso di timeout: l'esito incerto viene restituito al chiamante.

## 6. Entità Domus e funzionalità

Durante l'associazione Domus crea un dispositivo normale nel registry (integration_id=`integration_midea_smarthome`) e quattro entità:

- `humidifier.midea_<id>`: potenza, modalità, velocità ventola, limiti e attributi.
- `sensor.midea_<id>_humidity`: umidità rilevata in percentuale.
- `sensor.midea_<id>_temperature`: temperatura ambiente in °C.
- `binary_sensor.midea_<id>_tank_full`: stato serbatoio pieno.

Le misurazioni di temperatura e umidità sono quindi utilizzabili dal sistema Clima & Storico, incluse le aggregazioni esistenti. Le entità sono collegate al device e possono essere assegnate alle Aree dalla normale sezione Dispositivi. Le variazioni pubblicano eventi sul bus Domus per automazioni e client collegati.

## 7. Stato effettivo dei driver

In questo branch è operativo il driver locale **M-Smart A1 per deumidificatori**, con trasporto V2 e V3, discovery LAN, gestione token/key V3, lettura, comandi e verifica read-back. La UI amministrativa consente scansione, associazione, configurazione, controllo e rimozione. Le credenziali V3 sono conservate nel campo cifrato della tabella integrazioni; non sono incluse nel registry device né nelle risposte API.

La tassonomia Domus prevede molte altre categorie Midea (climatizzatori, purificatori, lavatrici, asciugatrici, lavastoviglie, frigoriferi, forni, umidificatori, ventilatori e apparecchi per acqua), ma **non significa che i relativi driver siano già implementati**. La discovery visualizza gli apparecchi riconosciuti e marca come non supportati quelli per cui manca un driver verificato. La presente versione associa e controlla solo il tipo A1. I prodotti Midea che usano firmware Tuya vanno invece collegati tramite gli adapter Tuya già presenti in Domus.

Non è ancora implementato il bootstrap automatico di token/key tramite account consumer SmartHome, né il protocollo M-Smart V1 o i driver delle altre categorie. Per V3 l'utente deve fornire un token/key già ottenuto per quel dispositivo con un metodo affidabile. Le API ufficiali cloud Midea sono un percorso partner separato e non vengono simulate con credenziali consumer.

## 8. Test e validazione hardware

I test unitari coprono:
- frame A1 query/set, checksum e CRC8;
- decodifica dello stato di un deumidificatore, umidità, temperatura, ventola e serbatoio;
- cifratura outer packet, digest MD5 e envelope 0x8370 V3;
- discovery V2/V3, sorgenti LAN non private e datagrammi manomessi;
- capability e limiti del deumidificatore;
- comandi at-most-once e verifica read-back;
- configurazione admin API, persistenza cifrata, ripristino al riavvio e provisioning di device/entità;
- test UI su discovery, configurazione e comando dell'umidità.

I test di protocollo usano fixture sintetiche/sanificate. **Non sostituiscono il test hardware**: prima di dichiarare compatibile un modello reale vanno verificati discovery, token V3, frame di stato e ciascun comando con quel modello/firmware, misurando gli effetti sul dispositivo e confrontando le letture di ritorno.


## 7. Persistenza tramite Plugin Manager

Dalla migrazione alla fase Plugin Manager, Midea / Comfee M-Smart è registrato come plugin pubblico a istanza singola:

- **Tipo plugin:** `midea_smarthome_local`.
- **ID persistente:** `integration_midea_smarthome`, mantenuto invariato per conservare i riferimenti dei dispositivi e delle entità già creati.
- **Configurazione gestita:** campo `devices`, marcato `Secret` e `Internal`; il Plugin Manager lo cifra a riposo e non lo restituisce nel catalogo pubblico.
- **Credenziali M-Smart V3:** token e key restano associati al singolo apparecchio e sono ricostruiti nei driver soltanto in memoria. La struttura pubblica `Device`, le API di inventario e gli eventi non contengono token/key.
- **Lifecycle:** il Manager ripristina i driver V2/V3 all'avvio, sostituisce l'adapter quando la configurazione cambia e lo rimuove dal runtime quando l'integrazione viene disabilitata.
- **Compatibilità:** gli endpoint amministrativi `/api/v1/integrations/midea` non cambiano; i record cifrati già presenti vengono letti e riutilizzati senza cambiare l'ID storico.

Il provider implementato è locale: non vengono salvate password dell'app Midea, credenziali di account cloud o refresh token. Le credenziali token/key V3 sono materiale di autenticazione LAN del singolo dispositivo, non una sessione cloud. Un record senza dispositivi può rimanere disabilitato per conservare lo stato dell'integrazione, senza mantenere un driver attivo.

La registrazione è limitata ai dispositivi A1 e alla discovery LAN recente. M-Smart V2/V3 mantengono i driver esistenti; Comfee viene trattato come marchio OEM dello stesso protocollo, non come un provider separato. I modelli non A1 rimangono visibili in discovery ma non vengono dichiarati controllabili da questa tranche.

### Test della migrazione

I test automatizzati verificano: creazione dell'istanza gestita, ripristino di una configurazione V3 cifrata, assenza di token/key negli elenchi pubblici, campi interni esclusi dal catalogo, sostituzione dell'adapter dopo una modifica, arresto del vecchio adapter, disconnessione con eliminazione delle credenziali dalla configurazione attiva e mantenimento dell'ID stabile.

I test non sostituiscono ancora la verifica fisica con almeno un apparecchio M-Smart V2 e uno V3, inclusi Comfee, né l'interoperabilità con modelli diversi dal profilo A1.
