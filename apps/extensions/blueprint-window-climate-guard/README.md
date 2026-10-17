# Domus Extension: Blueprint Guardia Clima & Infissi

Questo Blueprint fornisce un meccanismo fondamentale di risparmio energetico e domotica green.

---

## Funzionamento
- Quando un sensore di contatto magnetico Zigbee/Z-Wave/Matter rileva una finestra aperta per oltre 90 secondi (parametro personalizzabile), invia il comando `climate.set_hvac_mode: "off"` al termostato o alla valvola TRV della stanza.
- Alla chiusura della finestra, la climatizzazione viene riattivata automaticamente.
