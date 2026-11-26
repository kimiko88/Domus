# Imou Life: migrazione al Plugin Manager

## Ambito

Imou Life viene ora gestito dal tipo plugin pubblico `imou_life`, con l'istanza singola e stabile `integration_imou_life`. L'ID è lo stesso usato dal record cifrato esistente: non vengono ricreati né rinominati integrazioni, dispositivi o relazioni.

## Configurazione cifrata

L'istanza mantiene:
- Access Key, Secret Key e data center del progetto Imou;
- URL H5 di consenso e base HTTPS pubblica;
- capability casuali separate per callback OAuth e callback eventi;
- access token e refresh token dell'utente delegato;
- token amministrativo di progetto con scadenza;
- stato e categorie di event push;
- scadenze delle due famiglie di token.

Project Secret, token e callback capability sono marcati come segreti interni nel manifest. La API pubblica del catalogo omette tutti i campi interni e non restituisce la configurazione cifrata delle istanze.

## Ripristino e compatibilità

- Il Manager riconosce il record storico `integration_imou_life` perché il suo tipo persistito `imou_life` è ora registrato nel Registry.
- Il JSON cifrato preesistente è compatibile con lo schema; viene ripristinato direttamente all'avvio, senza migrazione in un'altra tabella.
- I percorsi REST esistenti, inclusi `/config`, `/authorize`, callback OAuth, dispositivi, preview, registrazioni, eventi e `/disconnect`, restano invariati.
- Quando un'istanza nasce dal catalogo generico, i callback secret mancanti vengono generati e scritti atomicamente alla prima lettura amministrativa.
- Il percorso repository/cipher precedente rimane come fallback solo quando il Plugin Manager non è disponibile.

## Lifecycle e sessioni

- Configurazione e refresh usano `Manager.UpdateConfigMutate`, preservando gli aggiornamenti concorrenti e cifrando la configurazione aggiornata.
- Dopo il refresh del token utente o la generazione del token amministrativo, l'handler risolve nuovamente l'adapter supervisionato prima della successiva chiamata cloud.
- `POST /disconnect` revoca localmente la sessione delegata, eliminando access token, refresh token e relativa scadenza. Mantiene le credenziali del progetto, i callback, la configurazione push e il token amministrativo di progetto (che è indipendente dalla sessione dell'utente).

## Verifiche aggiunte

I test coprono il ripristino del record cifrato storico, la configurazione da zero, la redazione dei segreti nelle risposte, l'aggiornamento atomico dei token, la sostituzione dell'adapter, la disconnessione selettiva, il bootstrap dei callback da una istanza creata nel catalogo e la conservazione dei callback dopo una modifica pubblica.
