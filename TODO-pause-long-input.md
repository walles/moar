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

# State som readern behöver hålla

- Tycker pagern att vi ska dansa eller pausa?
- Är vi över eller under lästa-rader-gränsen för att pausa?

Vi ska skicka meddelanden till pagern när:

- OK: Vi går över radantalsgränsen in i paus-zonen. Sätt paus till true och
  skicka uppdateringsmeddelandet.
- Vi är över radantalsgränsen, och byter på uppmaning mellan att dansa och att
  pausa.

# Blandade randfall

- OK, test skrivet: Vad händer ifall readern har läst in en komplett fil, men
  sedan vid pollning hittar fler rader? Antar att den borde pausa då med.

# Implementation

- OK: Gör tester för readerns nya beteende
  - OK: Ställ in paus efter N rader
  - OK: Kolla att readern berättar när den pausar
  - OK: Kolla att readern kan bli tillsagd att fortsätta
  - OK: Kolla att readern berättar när den startar igen
- OK: Gör samma tester fast för situationen när readern pollar och hittar fler rader
- OK: Skriv om paustesterna så att det pagern berättar för readern är vilket
  radnummer den siktar på
- OK (alla utom large-git-log-patch.txt): Se till att testerna går igenom
- OK Se till att pagern tar emot de nya meddelandena från readern och visar rätt
  spinner
- OK: Ta bort kanalen som readern postar på när pausstatus ändras, pagern pollar
  så den behövs inte
- OK: Låt pagern informera readern om sin status, väntar den sig fler rader
  eller inte?
- OK: Om pagern kommer tillräckligt nära det pausade slutet på de inlästa
  raderna, se till att readern ligger en lagom bit framför.
- Fundera på UIet, ska vi verkligen visa antalet rader om vi med flit pausat
  inläsningen? Dels vet vi inte om det stämmer, och dels rullar inte siffrorna
  så det är inte uppenbart för användaren att datan är inkomplett.
