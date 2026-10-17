# Domus Extension: Widget Power Flow Wheel

Questo esempio illustra come sviluppare e confezionare un **Widget UI personalizzato** per la Dashboard di Domus.

---

## 1. Caratteristiche
- Rendering grafico SVG ad alta reattività con calcolo automatico della percentuale di autoconsumo solare.
- Adattivo per schermi mobile, tablet montati a parete (*kiosk wallmount*) e browser desktop.
- Si collega direttamente ai sensori di potenza di Domus Core (`sensor.produzione_fotovoltaico`, `sensor.potenza_rete`, `sensor.batteria_soc`).

---

## 2. Struttura
- `domus-extension.json`: Dichiara il tipo `widget`, la dimensione nella griglia (`card_size`) e le viste di destinazione supportate.
- `EnergyFlowWheel.vue`: Componente Vue 3 standard con TypeScript e Vanilla CSS glassmorfico coerente con il design system di Domus.
