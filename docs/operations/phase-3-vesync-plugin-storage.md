# VeSync Cloud: migrazione al Plugin Manager

## Identità e compatibilità

- Tipo pubblico: \`vesync_cloud\`.
- ID singolo e stabile: \`integration_vesync_cloud\`.
- Il formato cifrato storico resta \`{"session": {...}}\`; l'istanza viene ripristinata automaticamente dal Plugin Manager all'avvio.
- Non è richiesta una migrazione SQL o la cancellazione/ricreazione dell'integrazione esistente.

La password VeSync è un input transitorio del flusso HTTP di login. Non viene conservata in chiaro né cifrata nel database: dopo il login Domus conserva solo il token cloud, l'account ID e i metadati di paese/regione/fuso orario. L'attuale API VeSync integrata non restituisce un refresh token separato né espone un endpoint di refresh documentato; quando il provider rifiuta una sessione scaduta/revocata, l'utente deve riconnettere l'account. Domus non simula né inventa un refresh token.

## Schema del plugin

Il manifest espone \`http_timeout\` come impostazione non segreta opzionale (default 8 secondi, intervallo 1–20 secondi). Il campo \`session\` è \`json\`, \`secret\` e \`internal\`: partecipa alla validazione e alla factory, ma viene rimosso dal catalogo e dalle API pubbliche.

La factory rifiuta:
- ID istanza diverso da \`integration_vesync_cloud\`;
- sessioni parziali (account ID senza token o viceversa);
- regioni diverse da \`EU\` e \`US\`;
- fusi orari IANA non validi;
- timeout fuori dall'intervallo ammesso.

## Gestione dell'account

L'handler storico resta come API di compatibilità (\`/api/v1/integrations/vesync\`), ma ottiene il client dall'adapter corrente ogni volta. Questo evita che un token aggiornato lasci l'handler con un client associato a un'istanza già sostituita.

Il login usa un client temporaneo solo quando l'istanza non esiste ancora o è stata disabilitata. Dopo aver autenticato l'account e verificato la lista dei dispositivi, la sessione viene salvata attraverso il Manager:
- creazione della prima istanza con ID stabile;
- aggiornamento atomico del solo campo interno \`session\`, preservando le impostazioni pubbliche;
- riabilitazione dell'istanza dopo un nuovo login;
- cancellazione della sessione e disabilitazione dell'istanza al disconnect.

Le richieste di dispositivi e comandi richiedono una istanza gestita e attiva. Una istanza mancante/disabilitata non viene aggirata creando un client autonomo.

## Verifiche automatiche

La tranche include test per:
- identità pubblica, ID singleton, lifecycle e timeout dell'adapter;
- rifiuto di sessioni parziali e regioni non supportate;
- ripristino di un record cifrato storico con il formato già in uso;
- occultamento di token e account ID nelle API di stato;
- creazione, disconnessione e riconnessione dell'istanza persistente;
- rimozione del campo \`session\` dallo schema restituito al catalogo.

Questi test non sostituiscono le prove di login e controllo su account/dispositivi fisici VeSync. Il client usa endpoint cloud non documentati pubblicamente e la compatibilità può cambiare lato provider.
