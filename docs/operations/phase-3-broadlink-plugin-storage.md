# BroadLink Local: istanza persistente nel Plugin Manager

## Obiettivo

Portare il lifecycle di BroadLink sotto il Plugin Manager senza cambiare la configurazione dei telecomandi già accoppiati, gli ID dei device, gli entity key o le API HTTP esistenti.

## Identità e compatibilità

- Tipo plugin: `broadlink_local`.
- ID storico per le installazioni adottate: `broadlink_local`.
- Singleton: le tabelle `broadlink_devices` e `broadlink_commands` sono condivise e non sono ancora isolate per integration ID.
- Nessun account BroadLink, password cloud o token remoto viene introdotto.

Durante l'avvio Domus controlla se esistono record nella tabella legacy `broadlink_devices`. Solo in presenza di almeno un telecomando accoppiato viene creata l'istanza persistente singleton; installazioni nuove o tabelle vuote non ricevono una configurazione vuota non richiesta. Se esiste già un'istanza BroadLink disabilitata, non viene riabilitata automaticamente.

## Persistenza e sicurezza

Le tabelle legacy vengono mantenute:
- `broadlink_devices` contiene MAC, IP, modello, device ID e chiave AES cifrata;
- `broadlink_commands` contiene i codici IR/RF appresi, che restano nel database locale;
- la configurazione dell'istanza nel Plugin Manager contiene solo `{}` e viene comunque salvata tramite il normale meccanismo cifrato del Manager.

La chiave AES non viene copiata nel catalogo, nel payload REST delle istanze o nei log. La migrazione non altera le chiavi già cifrate né cancella i codici appresi.

## Lifecycle e API

- L'istanza persistente viene ripristinata da `Manager.Start` e supervisionata come gli altri plugin.
- La route legacy `/api/v1/integrations/broadlink-local` resta compatibile.
- L'handler usa il Manager per risolvere l'adapter corrente a ogni richiesta.
- Disable/delete rimuove il runtime dalla risoluzione HTTP; l'handler restituisce HTTP 503 anziché continuare a usare un puntatore obsoleto.
- Dopo un enable o un restart dal Plugin Manager, le successive richieste usano il nuovo adapter senza ricreare il router.
- La route amministrativa resta disponibile anche quando l'istanza è disabilitata, così l'amministratore può riabilitarla dal Plugin Manager.

## Test

- nessuna creazione automatica quando la tabella legacy non esiste o è vuota;
- adozione automatica con una configurazione persistita cifrata quando esiste almeno un device;
- ripristino dei device paired dal database precedente;
- istanza già disabilitata non riabilitata dall'avvio;
- handler usa il provider del Manager, non un puntatore legacy residuo;
- disable immediato rende le API BroadLink indisponibili con HTTP 503.

I test automatizzati non sostituiscono un collaudo IR/RF su un telecomando fisico. Non è stata eseguita una trasmissione verso hardware reale durante questa migrazione.
