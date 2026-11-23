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

## 5. Stato del codice in questo branch

Questo branch aggiunge:
- tassonomia multi-apparecchio e capacità generiche;
- profili estendibili per codice prodotto, incluso il profilo iniziale deumidificatore A1;
- validazione dei comandi e dei limiti specifici del dispositivo;
- contratto multi-driver, discovery validata, registry in memoria e lock per dispositivo;
- semantica write-once con verifica indipendente read-back;
- test unitari che coprono deumidificatori, validazione capability, discovery, errori di rete, mancata applicazione dei comandi e assenza del driver.

**Non è ancora una integrazione end-to-end pronta da attivare dall'interfaccia.** Nel branch non sono inclusi i driver di trasporto M-Smart V1/V2/V3, il bootstrap account SmartHome, la persistenza cifrata delle credenziali, gli handler REST, l'auto-provisioning nel registry SQLite né la UI delle integrazioni. La presenza di una categoria nel modello non equivale alla disponibilità di un driver per quel prodotto.

La ragione è tecnica: le implementazioni complete e aggiornate per tutte le famiglie Midea individuate sono principalmente librerie community non Go; i client Go disponibili esaminati sono specifici per alcuni climatizzatori V3 e non possono essere riutilizzati onestamente per lavatrici, frigoriferi o deumidificatori senza una verifica. La fase successiva dovrà integrare un driver per ciascuna famiglia di protocollo, iniziare dal M-Smart V2/V3 e dalla famiglia A1 dei deumidificatori, aggiungere l'onboarding cifrato e collegare le entità create al sistema Domus già esistente.
~~~