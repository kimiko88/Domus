# Domus Extension: BESS Arbitrage WASM Plugin

Questa estensione ufficiale Domus dimostra come implementare un **plugin ad altissime prestazioni in WebAssembly (WASI)** che gira in sandbox isolata su Domus Core senza richiedere container Docker.

---

## 1. Struttura dell'Estensione
- `domus-extension.json`: Manifest contenente metadati, ID pacchetto, autorizzazioni richieste (`capabilities`) e parametri di configurazione esposti all'utente.
- `main.go`: Codice sorgente compilabile in WASM contenente l'export della funzione `domus_evaluate_arbitrage`.
- `plugin.wasm`: Binario WebAssembly pre-compilato pronto per il runtime.

---

## 2. Come compilare il Plugin WASM
Il codice può essere compilato direttamente utilizzando [TinyGo](https://tinygo.org/):

```bash
tinygo build -o plugin.wasm -target=wasi -no-debug -opt=z main.go
```

Il binario risultante pesa meno di 150 KB e viene caricato in memoria da Domus Core con tempo di istanziazione inferiore a 1 millisecondo.

---

## 3. Registrazione nello Store
Inserendo il pacchetto in `store_packages` o nel repository del marketplace, l'estensione viene presentata con un click per l'installazione immediata:
```json
{
  "id": "pkg_bess_arbitrage",
  "package_id": "domus.energy.bess-arbitrage",
  "name": "BESS Dynamic Tariff & Energy Arbitrage Optimizer",
  "type": "wasm_plugin",
  "version": "1.2.0",
  "author": "Domus Energy Labs"
}
```
