# Domus Extension: Blueprint Illuminazione Circadiana

Questo pacchetto mostra come realizzare e distribuire un **Blueprint di Automazione** per il Domus Store.

---

## 1. Cos'è un Blueprint?
Un Blueprint è un modello di automazione generico e riutilizzabile: definisce la logica (trigger, condizioni ed azioni basate sul sole), esponendo selettori facili per l'utente finale (ad es. quali lampadine sincronizzare o la temperatura colore minima e massima desiderata).

---

## 2. Struttura
- `domus-extension.json`: Metadati del pacchetto nello store, ID e input selector.
- `blueprint.yaml`: La ricetta di automazione standard Domus valutata dall'engine (`internal/application/automation/`).

---

## 3. Installazione
Quando l'utente clicca **"Installa con 1 Click"** nello Store, Domus registra il Blueprint nel catalogo automazioni locale, consentendo di generare istanze personalizzate per ciascuna stanza (es. "Circadiano Salotto", "Circadiano Studio") con zero righe di codice YAML.
