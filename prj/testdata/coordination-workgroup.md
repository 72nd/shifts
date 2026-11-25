# Testdaten: Arbeitsgruppe mit Arbeitsauftrag (Problemfeld 2)

Zum Prüfen der Koordination, wenn eine bekannte Gruppe eine Mindeststundenzahl leisten muss.

## Aufbau

1. Neuen Plan anlegen (Zeitraum Fr 17.07.2026 bis So 19.07.2026) und die Beispielschichten (Küchendienst) behalten oder
   mit `example-shifts.csv` ersetzen.
2. Koordination öffnen, Tab **Felder & Regeln**:
   - Soll-Stunden pro Person: 12
   - Mindestpause: 1:00
   - Höchstens pro Tag: 8 h
   - Anwesenheit: „Teilnehmende geben an“
   - Feld „Erfahrung“ (Auswahl: Neu, Mittel, Erfahren), als Erfahrungsfeld setzen
3. Tab **Personen**, „Namen einfügen“, diese Liste einfügen (20 Namen):

```
Aaron
Bea
Carlos
Dilara
Elias
Fatima
Gregor
Hanna
Ivo
Jule
Karim
Lena
Mats
Nora
Oskar
Pia
Quentin
Rosa
Samir
Tilda
```

4. Anwesenheit anpassen:
   - **Carlos**: anwesend ab Sa 18.07. 12:00 (späte Anreise, keine Freitag-Schichten)
   - **Rosa**: anwesend ab Sa 18.07. 12:00
   - **Hanna**: „Arbeitet lieber am Stück“ aktivieren
5. Tab **Verteilen**, Quelle „Alle Personen“, alle Schichten filtern, „Auto-Verteilung…“.

## Erwartung

- Die Last ist ausgeglichen (Spannweite höchstens längste Schicht mal größter Faktor).
- Carlos und Rosa bekommen keine Freitag-Schichten.
- Pausen- und Tagesregel werden eingehalten; Plätze, die dadurch nicht besetzbar sind, erscheinen in der Vorschau.
- Hanna bekommt zusammenhängende Schichten.
