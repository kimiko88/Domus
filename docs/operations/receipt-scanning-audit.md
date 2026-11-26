# Audit scansione scontrini e fatture

## Flusso verificato

1. Il componente Vue gestisce fotocamera, selezione file, anteprima e rotazione.
2. Le immagini vengono decodificate nel browser (inclusi HEIC/HEIF), ridimensionate e convertite in JPEG entro 4 MiB prima dell'upload.
3. I PDF sono limitati a 3 MiB. Il backend usa Poppler per estrarre il livello testuale e, quando il PDF è una scansione, rasterizza le pagine una alla volta e invoca Tesseract.
4. Le immagini inviate direttamente all'API devono essere JPEG o PNG, al massimo 4 MiB e 24 megapixel.
5. Tesseract usa i dati lingua italiani e inglesi, una sola scansione OCR alla volta, timeout di 20 secondi e output testuale massimo di 64 KiB.
6. Il testo è analizzato dal parser euristico condiviso da scontrini e fatture. L'utente può correggere i dati prima dell'import.
7. L'import può aggiornare la dispensa e/o registrare la spesa in Home Finance. Le fatture non aggiornano automaticamente le quantità della dispensa.

## Difetti corretti in questo audit

- La parola generica carta non viene più trattata come motivo di esclusione: prodotti come carta igienica e carta da cucina vengono conservati, mentre le righe di pagamento con carta, bancomat e POS sono escluse.
- Gli importi italiani con separatore delle migliaia (1.234,56) vengono convertiti correttamente; sono tollerati anche importi senza raggruppamento e la notazione 1,234.56.
- La scelta del totale dà precedenza alle etichette più specifiche (totale complessivo, da pagare, importo pagato) invece di prendere semplicemente il valore numericamente più alto. Subtotali e metodi di pagamento non sovrascrivono il totale finale.
- Il punteggio di confidenza viene ridotto quando il totale esplicito non è coerente con la somma delle righe estratte.
- La formattazione dei nomi usa rune Unicode e conserva le lettere accentate iniziali.
- I PDF oltre 20 pagine vengono rifiutati esplicitamente. In precedenza il limite di 20 pagine poteva troncare silenziosamente il documento e produrre un import parziale.
- L'import pre-valida tutti gli articoli, unità e categorie prima di modificare la dispensa. Le righe duplicate per nome (senza distinzione maiuscole/minuscole) vengono aggregate; un conflitto fra unità viene restituito prima di qualsiasi scrittura.
- Le scritture della dispensa vengono compensate in caso di errore successivo, inclusa la mancata registrazione della spesa in Home Finance. Se la compensazione stessa fallisce, l'errore di rollback viene mantenuto nella risposta applicativa.
- Gli errori di validazione, i conflitti di unità e l'indisponibilità del modulo Finance hanno ora risposte HTTP distinte (400, 409 e 503).
- Il frontend invia una chiave di idempotenza stabile durante il retry dello stesso import, così una risposta persa non causa normalmente un secondo inserimento entro la finestra della cache HTTP.

## Dipendenze runtime

L'immagine Docker installa poppler-utils, tesseract-ocr e i dati lingua italiani/inglesi. deployments/scripts/install.sh installa o verifica gli stessi strumenti sulle installazioni native Debian/Raspberry Pi. Non è necessario installare un servizio OCR cloud o inviare scontrini a un provider esterno.

## Limiti residui

- Il parser non usa un catalogo prodotti o un modello AI: estrae righe con euristiche su nomi, quantità e prezzi. Le descrizioni abbreviate dei registratori di cassa, prezzi collocati su righe separate e impaginazioni insolite possono ancora produrre risultati errati. L'anteprima va sempre verificata prima della conferma.
- L'OCR usa Tesseract con segmentazione automatica a blocco singolo (--psm 6); non esegue correzione prospettica, rilevamento del documento o deskew avanzato. Foto sfocate, riflessi, caratteri termici molto chiari e scontrini piegati possono degradare i risultati.
- La cache HTTP d'idempotenza è in memoria e ha la finestra predefinita di 60 secondi. La chiave protegge il retry immediato, ma non è un archivio persistente degli import e non sopravvive al riavvio del server.
- Non viene conservata una copia del documento o un registro storico completo dello scontrino. Il testo OCR è restituito nella risposta di parsing, ma non è archiviato come documento consultabile.
- I test automatici coprono parser, validazioni, rollback compensativo, limiti e rifiuto dei formati non supportati. Non sostituiscono una prova end-to-end con scontrini reali di Esselunga, Coop, Conad, Lidl e altri punti vendita, né un collaudo con fotografie reali scattate da iPhone/Android in condizioni di luce diverse.

## Collaudo consigliato prima del rilascio

Usare almeno: scontrino termico fotografato, PDF con testo incorporato, PDF scansionato, scontrino lungo con totale sopra 999,99 €, righe duplicate dello stesso prodotto, prodotti non alimentari con parola carta, sconti/depositi e articolo con unità incompatibile rispetto alla dispensa. Controllare nome, quantità, unità, prezzo riga, totale e saldo della dispensa/Finance dopo la conferma.
