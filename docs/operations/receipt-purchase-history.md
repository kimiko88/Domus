# Storico acquisti e gestione scorte

## Conferma di uno scontrino o di una fattura

Il parser prova a identificare il negozio usando prima un elenco di catene note (Esselunga, Coop, Conad, Lidl, Carrefour, Eurospin, Ipercoop, Pam, MD, Dpiù, Famila, Sigma, CRAI, Todis, Bennet, NaturaSì, Rossetto e altre) e poi un'euristica sull'intestazione per riconoscere negozi indipendenti e farmacie. Il nome resta sempre modificabile nell'anteprima OCR; non va considerato certo quando il punteggio di confidenza è basso.

Ogni documento confermato viene salvato nello storico, anche quando le opzioni di aggiornamento automatico della dispensa e di registrazione nel Bilancio Casa sono disattivate. L'utente può quindi usare lo scanner anche solo come archivio digitale dei propri acquisti.

Per ogni documento vengono conservati:
- negozio, data e tipo documento (scontrino/fattura);
- righe originali dei prodotti con quantità, unità, prezzo unitario, totale riga e categoria;
- totale documento e punteggio di confidenza OCR;
- riferimento all'eventuale movimento creato in Home Finance e scelte di importazione.

Il file immagine/PDF e il testo OCR originale non vengono memorizzati nello storico. In particolare, questo evita di conservare automaticamente codici fiscali, indirizzi, numeri di cassa o testo relativo ai pagamenti.

## Modifica dello storico

In Lista della Spesa è disponibile il riquadro **Storico degli acquisti**. Da qui si può riaprire un documento salvato e modificare negozio, data, tipo, totale, prodotti, quantità, unità, prezzi e categorie; si possono anche aggiungere o rimuovere righe.

Se l'importazione originale aveva creato un movimento in Home Finance, la modifica aggiorna anche titolo, data, importo, categoria e descrizione del movimento collegato. Se il salvataggio dello storico fallisce dopo la modifica finanziaria, il servizio prova a ripristinare il movimento precedente.

**La modifica dello storico non ricarica la dispensa.** Dopo l'acquisto, alcuni prodotti potrebbero essere già stati consumati; applicare nuovamente le righe dello scontrino altererebbe in modo errato le scorte. Le quantità correnti si modificano separatamente.

## Modifica delle scorte presenti

Nella scheda Dispensa, la quantità visualizzata è modificabile direttamente: seleziona il valore, inserisci la nuova quantità e conferma con Invio o uscendo dal campo. È ancora possibile usare i pulsanti `+` e `−` per piccoli aggiustamenti o aprire la scheda di modifica per cambiare anche nome, categoria, unità, soglia minima, scadenza, posizione, codice a barre e costo.

Le modifiche parziali sono unite ai dati correnti del prodotto prima della richiesta HTTP: una correzione della sola quantità non sovrascrive gli altri campi.

## API e persistenza

- `GET /api/v1/receipts?limit=50` elenca gli acquisti salvati (limite massimo 200).
- `POST /api/v1/receipts/import` salva il documento e applica solo gli aggiornamenti richiesti.
- `PUT /api/v1/receipts/{id}` modifica un documento esistente e sincronizza l'eventuale movimento finanziario.
- `PUT /api/v1/pantry/items/{id}` modifica tutti i dati di un prodotto della dispensa.

La migrazione SQLite `000026_receipt_history` crea la tabella dello storico e gli indici per data, negozio e collegamento finanziario. Le righe dei prodotti sono memorizzate come JSON strutturato, separate dal magazzino corrente.

## Regole e limiti

- I dati OCR sono euristici: catene non note, loghi stilizzati, nomi troncati o intestazioni su più righe possono richiedere una correzione manuale.
- Il nome negozio è limitato a 150 caratteri; ogni documento può contenere al massimo 500 righe.
- I prezzi e le quantità devono essere numerici e finiti; le unità e le categorie devono appartenere ai valori supportati da Domus.
- L'aggiornamento dello storico non aggiunge o sottrae quantità alla dispensa. Correggi esplicitamente la quantità attuale nella scheda Dispensa.
- La rimozione dei file originali OCR è intenzionale; lo storico non costituisce una copia fiscale certificata o un archivio documentale sostitutivo.

## Verifiche automatiche

I test coprono il riconoscimento di catene e negozi indipendenti, il rifiuto di intestazioni come `FATTURA` come nome del negozio, il salvataggio/lettura/modifica SQLite, la mancata persistenza del testo OCR grezzo, l'aggiornamento del movimento finanziario collegato e la garanzia che modificare uno scontrino storico non modifichi nuovamente la scorta.