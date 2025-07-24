När man använder `moar` så ska bakgrundsinläsningen stanna efter 20k rader.

Men om moar tailar ska readern fortsätta läsa tills man inte tailar längre.

# Information som behöver skickas

- När man skapar readern behöver man kunna ställa in hur många rader som ska
  läsas innan paus
- Readern behöver berätta för pagern när den pausar.
- Pagern behöver kunna säga till readern:
  - Jag visar sista raden och väntar på fler
  - Jag väntar inte på fler rader just nu

# UI

- När readern läser visar vi den vanliga spinnern
- När readern pausar visar vi en ny pausspinner
- När readern tycker att den är klar visar vi ingen spinner

# Blandade randfall

- Vad händer ifall readern har läst in en komplett fil, men sedan vid pollning
  hittar fler rader? Antar att den borde pausa då med.

# Implementation

- Gör tester för readerns nya beteende
  - Ställ in paus efter N rader
  - Kolla att readern berättar när den pausar
  - Kolla att readern kan bli tillsagd att fortsätta
  - Kolla att readern berättar när den startar igen
- Gör samma tester fast för situationen när readern pollar och hittar fler rader
- Se till att pagern tar emot de nya meddelandena från readern och visar rätt
  spinner
- Låt pagern informera readern om sin status, väntar den sig fler rader eller
  inte?
