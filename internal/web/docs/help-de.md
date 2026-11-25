
Shifts ist ein kollaboratives Schichtplanungstool, das auf größtmögliche Zugänglichkeit ausgelegt ist. Die Erfahrung zeigt, dass die Organisation größerer Gruppen dann am besten gelingt, wenn alle Beteiligten die Daten eigenständig bearbeiten können. Beispiele für dieses Prinzip sind etwa Etherpad oder webbasierte Tabellenkalkulations-Tools.

**Für Nutzende** Du willst nur eine Erklärung haben, wie du dich in Schichten eintragen kannst? Dann springe direkt zu [für Nutzende](#für-nutzende).

![Beispiel eines Schichtplans](/images/intro.png)


## Einführung

### Hintergrund und Ziele

Das Programm wurde entwickelt, weil bisherige Schichtplanungstools den Anforderungen größerer und komplexer organisierter kollektiver Veranstaltungen nicht gerecht wurden.

- Für Personen, die eine Schicht übernehmen möchten, soll das Tool _so einfach wie möglich zu bedienen_ sein. Je niedriger die Einstiegshürden, desto wahrscheinlicher ist es, dass Interessierte sich tatsächlich in eine oder mehrere Schichten eintragen. Shifts benötigt von Helfenden lediglich einen Namen – keine E-Mail-Adresse, kein Login.
- Kollaborative und kollektive Organisationsprozesse sollen vollständig abgebildet werden können. Die Applikation macht _bewusst möglichst wenig Vorgaben_ darüber, wie die Schichtplanung und -vergabe konkret gestaltet wird.
- Die Applikation soll auch auf Smartphones intuitiv und komfortabel bedienbar sein.
- Die Verwaltung _vieler Schichten_ soll problemlos möglich sein, ohne dass die Übersichtlichkeit und Kompaktheit der Darstellung darunter leidet.
- Organisator:innen (Admins) sollen durch umfangreiche Filter-, Export- und Berichtsfunktionen optimal unterstützt werden. Das Anlegen und Bearbeiten von Schichten soll dabei effizient und schnell von der Hand gehen.
- Ein Übergang auf papierbasierte Verwaltung ist jederzeit möglich. Bei vielen Veranstaltungen hat es sich bewährt, während des Events auf eine papierbasierte Schichtverwaltung umzusteigen. Shifts stellt den Schichtplan hierfür nicht nur als druckbaren Plan, sondern auch als Posterversion zur Verfügung.


### Features

- Personen tragen sich mit einem Namen, Nickname oder Pseudonym in eine Schicht ein. Eine E-Mail-Adresse oder ein Login werden nicht benötigt. Einmal verwendete Namen werden beim Eintragen in weitere Schichten automatisch vorgeschlagen.
- Jede Person kann sich selbst oder andere Personen jederzeit wieder austragen. Dies erleichtert die flexible Verwaltung von Schichten erheblich.
- Schichten sind in Schichttypen (z. B. Bar, Küche) gruppiert. Jeder Schichttyp kann mit einem Beschreibungstext versehen werden, der die jeweilige Aufgabe erläutert. Markdown wird hierfür unterstützt.
- Schichten
    - Verfügen über eine definierte Startzeit und Dauer.
    - Die Anzahl der benötigten Helfenden kann festgelegt werden. Daneben ist es möglich, Schichten mit unbegrenzter Teilnehmendenzahl zu erstellen. Dies kann für »All Hands on Deck«-Situationen wie den Abbau hilfreich sein.
    - Wie Schichttypen können auch einzelne Schichten mit einem Beschreibungstext (optional in Markdown) versehen werden. So lassen sich etwa die in dieser Zeit auftretenden Acts oder Bands kommunizieren.
- Filter: Der Schichtplan kann umfangreich gefiltert werden. Gezielt können bestimmte Schichttypen ein- und ausgeblendet oder nach dem Belegungsstatus der Schichten gefiltert werden. Auch eine Filterung nach Namen einzelner Helfer:innen ist möglich.
- Internationalisierung: Die Applikation ist aktuell auf Deutsch und Englisch verfügbar.
- Tagesgrenze: Viele Veranstaltungen dauern bis tief in die Nacht. Häufig werden dabei die frühen Morgenstunden organisatorisch noch dem vorherigen Tag zugeordnet. Shifts ermöglicht es, die Tagesgrenze für jeden Schichtplan individuell zu definieren.
- Adminzugang: Beim Erstellen eines Schichtplans wird automatisch ein Adminpasswort vergeben. Mit diesem können folgende Funktionen genutzt werden:
    - Anlegen, Bearbeiten und Duplizieren von Schichten und Schichttypen.
    - Ändern der Sortierung von Schichttypen.
    - Anpassen von Beschreibungstext, Titel, Tagesgrenze und weiteren Einstellungen des Schichtplans.
    - Export: Die Daten des Schichtplans können in verschiedenen Formaten exportiert werden, wobei optional die aktuelle Filterkonfiguration berücksichtigt werden kann. Weitere Informationen zu den Exportformaten finden sich unten.
- Koordination: Eine Seite für Admins, die bei schwierigen Entscheidungen hilft: wer eine beliebte Schicht bekommt (Erfahrung, Fairness, Konflikte) und wie eine Gruppe mit Arbeitsauftrag gerecht verteilt wird. Mit gereihten Vorschlägen samt Begründung, Auto-Verteilung mit Vorschau und Rückgängig, Personenfeldern, Anwesenheit und Arbeitszeitregeln. Die Koordination lässt sich pro Plan ein- und ausschalten; neue Pläne haben sie zunächst aus.
- Kollaborativer Planungsmodus: Ein optimierter Modus für die Schichtplanung in Teams und Crews. Weitere Informationen dazu finden sich weiter unten.
- Export: Nach Eingabe des Adminpassworts stehen verschiedene Exportformate zur Verfügung. Auf Wunsch kann dabei die aktuelle Filterkonfiguration berücksichtigt werden. Verfügbare Formate:
    - JSON: Exportiert alle Daten des Schichtplans im JSON-Format.
    - Offene Schichten als Textdatei: Exportiert eine Übersicht der offenen Schichten als Textdatei. Dies ist besonders hilfreich, wenn die Übersicht beispielsweise in einem Chat geteilt werden soll.
    - Timetable (PDF): Stellt die Schichten und deren Belegung pro Tag als Zeittafel (im Stil eines Stundenplans) dar. Dies unterstützt insbesondere die Planungsphase.
    - Poster (PDF): Bei vielen kollektiven Veranstaltungen ist es sinnvoll, nach Beginn des Events auf einen papierbasierten Schichtplan umzusteigen, in den sich Personen vor Ort eintragen können. Dieser Export erstellt einen solchen Schichtplan. Pro Tag und Schichttyp wird eine DIN-A4-Seite erstellt, die ausgedruckt und aufgehängt werden kann.
    - Report (PDF): Erstellt eine druckoptimierte Gesamtansicht des Schichtplans.
    - Personal Overview (PDF): Ein Dokument im DIN-A5-Format, das die Schichten einer einzelnen Person übersichtlich aufführt.


### Kollaborativer Planungsmodus

Bei Veranstaltungen lassen sich zwei grundsätzlich verschiedene Arten von Schichten unterscheiden. Viele Schichttypen können ohne größere Absprache von nahezu allen Beteiligten übernommen werden. Diese erfordern in der Regel kaum Vorbereitungen oder Koordination. Ein klassisches Beispiel hierfür ist etwa der Abwasch. Dem stehen Schichten gegenüber, die von organisierten Teams oder Crews übernommen werden. Diese erfordern häufig Vorbereitung und Abstimmung. Shifts möchte beiden Gruppen gerecht werden und bietet deshalb für Crews einen optionalen kollaborativen Planungsmodus an. Dieser kann in den Einstellungen des Schichtplans aktiviert werden.

Der kollaborative Planungsmodus unterscheidet sich vom normalen Modus, bei dem es darum geht, alle Schichten möglichst zügig nach dem First-Come-First-Served-Prinzip zu besetzen. Der kollaborative Modus basiert hingegen auf einem zweistufigen Prozess:

1. Alle Personen können zunächst Interesse an einer Schicht bekunden. Schichten können in diesem Modus bewusst »überbucht« werden. Auf diese Weise werden die Bedürfnisse und Wünsche aller Crewmitglieder transparent gemacht.
2. Auf Grundlage der bekundeten Interessen kann das Team nun entscheiden, welche Schichten konkret wie besetzt werden. Wer ausgewählt wird, wird als eingeteilt markiert, die übrigen als nicht berücksichtigt (sie werden nicht gelöscht, siehe »Koordination«). Finalisierte Schichten können vom Admin fixiert bzw. gesperrt werden. Danach sind sie für reguläre Nutzer:innen nicht mehr veränderbar. Fixierte Schichten werden farblich hervorgehoben.

Dadurch ist für alle Beteiligten jederzeit ersichtlich, welche Schichten definitiv zugeteilt sind. Da Personen sich nicht mehr eigenständig aus fixierten Schichten austragen können, werden sie motiviert, sich bei notwendigen Änderungen an ihre Crewkoordination zu wenden.


## Für Nutzende

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Ein- und Austragen in Schichten

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Zeittafel

In der Zeittafel werden leere Abschnitte eines Tages (vier Stunden oder mehr ohne Schicht) zu einem gestrichelten Band wie "5 Std. ohne Schichten (01:00–06:00)" zusammengeklappt. Ein Klick auf das Band zeigt die leere Zeit, ein weiterer Klick blendet sie wieder aus. Überall sonst ist die Zeit maßstabsgetreu. Aktive Filter zählen mit: Wenn Du auf einen Bereich filterst, sind mehr Abschnitte leer und werden zusammengeklappt.

### Filtern

### Live-Aktualisierung

Änderungen anderer Personen erscheinen im geöffneten Schichtplan automatisch, ohne dass du die Seite neu laden musst. Das gilt für Ein- und Austragungen, neue, geänderte und gelöschte Schichten und Schichttypen, Änderungen an den Plan-Einstellungen, das Sperren von Schichten und das Löschen des Plans. Meist dauert es weniger als eine Sekunde, bis die Änderung sichtbar ist.

- **Statusanzeige:** Ein kleiner Punkt in der Leiste des Schichtplans zeigt den Verbindungsstatus. »Live« bedeutet, dass Änderungen automatisch ankommen. »Verbinde …« zeigt, dass die Verbindung gerade aufgebaut wird. »Offline« bedeutet, dass die Verbindung abgebrochen ist. Die Seite versucht zwar, die Verbindung selbst wiederherzustellen. Gelingt das nicht, lädst du die Seite neu, um den aktuellen Stand zu sehen.
- **Zähler:** Die Leiste zeigt außerdem, wie viele Personen den Plan gerade ansehen, etwa »3 Personen sehen gerade zu«. Gezählt werden die geöffneten Tabs dieses Plans. Personen werden dabei weder erkannt noch angezeigt.
- **Geänderte Schichten:** Schichten, die sich gerade verändert haben, werden kurz hervorgehoben, damit du die Änderung schnell findest.
- **Offene Dialoge:** Wenn du gerade eine Schicht ein- oder austragen möchtest und die Schicht inzwischen voll oder gelöscht ist, warnt dich der Dialog. Er bleibt geöffnet, du entscheidest also selbst, wie es weitergeht.
- **Statistik:** Auch die Statistik aktualisiert sich live.

Die Live-Aktualisierung kann von den Betreiber:innen der Instanz abgeschaltet werden. Dann siehst du Änderungen erst nach dem Neuladen der Seite.

## Für Admins

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Neuer Plan anlegen

Beim Anlegen eines Plans wählst du die Zeitzone der Veranstaltung. Vorbelegt ist die Zeitzone der Instanz. Nach dem Anlegen lässt sie sich nicht mehr ändern.

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Schichttypen und Schichten anlegen und bearbeiten

Alle Zeiten sind Ortszeit der Veranstaltung und werden auf allen Geräten gleich angezeigt.

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Einstellungen
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Statistiken

Die Statistik öffnest du über »Statistik« in der Leiste des Schichtplans. Sie öffnet im selben Tab, mit »Zum Schichtplan« kommst du zurück. Alle Zahlen berechnet die Ansicht direkt im Browser aus den Plandaten.

Die Statistik hat vier Tabs:

- **Überblick**: Kennzahlen (Besetzungsgrad, offene Plätze, Personen, geplante und geleistete Stunden), die Besetzung pro Tag und pro Bereich als Balken sowie eine Heatmap über den Tag. Sie zeigt, wie viele Plätze zu welcher Stunde besetzt sind; fehlen Plätze, steht die höchste Zahl gleichzeitig fehlender Plätze in der Zelle (z. B. »−2«). Mit »Zeilen: Tage | Bereiche« schaltest du um: Bei »Bereiche« ist jede Zeile ein Bereich, gezeigt wird ein Tag, den du mit ‹ › wechselst. Ein Klick auf einen Balken setzt den Filter auf diesen Tag bzw. Bereich, ein Klick auf eine Heatmap-Zelle öffnet den Schichtplan an dieser Stelle. Eine Liste aller offenen Plätze bekommst du im Schichtplan mit dem Filter »Offene Plätze«.
- **Personen**: eine Tabelle aller Teilnehmenden mit Schichten, Stunden, Tagen, Bereichen, Nachtstunden und dem stärksten Tag. Die Spalten lassen sich sortieren, die Tabelle durchsuchen und auf Personen mit Warnungen oder mit nur einer Schicht eingrenzen. Eine Zeile aufklappen zeigt alle Schichten der Person und den Link »Im Plan zeigen«. Warnungen gibt es für:
    - Überschneidung: zwei Schichten derselben Person überlappen sich.
    - Ruhezeit: weniger als 8 Stunden zwischen dem Ende einer Schicht und dem Beginn der nächsten.
    - Belastung: mehr als 10 Stunden an einem Tag.

    Hat der Plan eine Mindestpause oder ein Tagesmaximum (im Tab »Einrichtung« der Koordination), ersetzen diese Werte die 8 und 10 Stunden. Stunden zählen nur Einsätze, nicht offene Wünsche in beliebten Schichten; nicht berücksichtigte Anmeldungen zählen gar nicht.

    Namen, die sich nur durch Groß- und Kleinschreibung, Leerzeichen oder Akzente unterscheiden, zählen als dieselbe Person. Ähnliche Namen, die sich in Satzzeichen oder Akzenten unterscheiden, bekommen einen Hinweis, dass sie vielleicht dieselbe Person sind. Als Admin kannst du die Tabelle als CSV-Datei herunterladen.
- **Verlauf**: die Anmeldekurve über die Zeit, mit dem Bedarf als Linie. Wenn in den letzten 7 Tagen genug Anmeldungen eingegangen sind, zeigt die Kurve außerdem, wann der Plan bei diesem Tempo voll wäre. Darunter steht die Verteilung der Stunden auf die Personen.
- **Rückblick**: die Zahlen für Dank, Ansprache oder Social-Media-Post: Personen, Stunden, Schichten und Anmeldungen, umgerechnet in Tage rund um die Uhr und Arbeitstage, dazu Highlights wie die meisten Schichten, Frühaufsteher:in, Nachtstunden, das Dream-Team (die meisten Schichten zusammen), die erste Anmeldung, die am schnellsten volle Schicht und Last-Minute-Anmeldungen. Mit »Als Text kopieren« kopierst du alles, auch die Namen, als Text und passt ihn danach an.

Die Filter für Bereich und Tag gelten für alle Tabs. Sie stehen in der Adresse der Seite, du kannst eine gefilterte Ansicht also direkt teilen oder neu laden.

Unbegrenzte Schichten zählen nicht zum Bedarf, nicht zu den offenen Plätzen und nicht zum Besetzungsgrad. Die Stunden ihrer Teilnehmenden zählen aber mit. Überbesetzte Schichten werden beim Besetzungsgrad höchstens mit 100 % gerechnet.

### Koordination

Die Koordinationsseite öffnest Du über »Koordination« in der Leiste des Schichtplans (nur für Admins). Sie hilft bei zwei Arten schwieriger Entscheidungen: beliebte Schichten, für die sich mehr Personen interessieren als Plätze da sind, und eine bekannte Gruppe, die eine Mindeststundenzahl leisten muss. Die Seite hat drei Tabs: »Verteilen«, »Personen« und »Einrichtung«. Alle Zahlen berechnet die Seite im Browser.

**Ein- und Ausschalten.** Die Koordination lässt sich für jeden Plan einzeln ein- und ausschalten. Du findest den Schalter »Koordination verwenden« in den Einstellungen des Plans. Neue Pläne sind zunächst ausgeschaltet. Eingeschaltet erscheint der Link »Koordination« in der Leiste des Plans. Ausgeschaltet verschwinden Link und Abschnitte, und beim Eintragen werden keine Personenfelder und keine Anwesenheit mehr abgefragt. Ausschalten löscht nichts: Angaben, Anmeldestatus, Beliebtheit und Belastungsfaktoren bleiben gespeichert. Öffnest Du die Koordinationsseite, während die Koordination ausgeschaltet ist, siehst Du einen Hinweis mit dem Knopf »Koordination einschalten«. Pläne, die schon Koordinationsdaten enthalten, sind nach dem Update eingeschaltet.

**Status der Anmeldung.** Jede Anmeldung hat einen Status: *angemeldet*, *eingeteilt* oder *nicht berücksichtigt*. In einer beliebten Schicht (mehr Interessierte als Plätze, oder für Schichttyp bzw. Schicht auf »Beliebt« gestellt), die nicht gesperrt ist, ist eine Anmeldung ein offener Wunsch. In allen anderen Schichten zählt sie sofort als Einsatz. Nicht berücksichtigte Personen bleiben gespeichert, weil sie für die Fairness zählen. Im Schichtplan sind sie standardmäßig ausgeblendet; der Filter »Nicht berücksichtigte anzeigen« zeigt sie. Nur Admins können sie entfernen, und die Person kann sich für diese Schicht nicht erneut eintragen (sie wird gebeten, sich an die Koordination zu wenden). In den Einstellungen des Plans legst Du fest, ob Teilnehmende nicht berücksichtigte Anmeldungen überhaupt sehen; standardmäßig sehen sie sie nicht.

**Verteilen.** Wähle eine Schicht, links in der Liste oder in einer der zeitlichen Ansichten (siehe unten). Die mittlere Spalte reiht die Kandidat:innen und nennt für jede Position den Grund. Die Reihung folgt diesen Regeln, in dieser Reihenfolge:

1. Braucht die Schicht erfahrene Personen und ist der Bedarf noch offen, kommen Personen mit der nötigen Stufe zuerst.
2. Wird eine Person in einer parallelen Schicht als erfahrene Person gebraucht (siehe »Engpass-Hinweis« unten), steht sie weiter unten.
3. Wer eine Regel verletzen würde (Pause zwischen Schichten, Höchststunden pro Tag, mehr als 125 % der Soll-Stunden), steht weiter unten.
4. Bei beliebten Schichten kommen Personen zuerst, die bisher weniger beliebte Schichten bekommen haben; danach die mit mehr abgelehnten Wünschen oder mehr Einteilungen ohne Wunsch.
5. Personen, die sich für die Schicht nicht interessiert haben (Quelle »Alle Personen«), kommen danach in der Reihenfolge, wie selten sie ohne Wunsch eingeteilt wurden. Danach zählt die geringste relative Last: Stunden mal Belastungsfaktor, im Verhältnis zum Soll.
6. Wer lieber am Stück arbeitet und eine direkt anschließende Schicht hat, kommt zuerst.
7. Den Rest entscheidet das Los (»Neu losen« mischt neu).

Wer die Schicht nicht übernehmen kann, steht unter »Nicht verfügbar« mit Grund: gleichzeitig in einer anderen Schicht (mit der Aktion »Umbuchen«), nicht anwesend oder bereits abgelehnt. »Vorschlag übernehmen« teilt die ersten freien Plätze ein, »Übrige ablehnen« lehnt alle übrigen ab (auf Wunsch mit Sperren der Schicht). Mit der Quelle »Alle Personen« kannst Du auch Personen einteilen, die sich nicht eingetragen haben.

**Gleichzeitig.** Über der Kandidatenliste zeigt die Leiste »Gleichzeitig« alle Schichten, die sich zeitlich mit der gewählten überschneiden. Jeder Chip nennt Uhrzeit, Typ und Belegung. Ist für eine dieser Schichten ein Erfahrungsbedarf offen, steht dort zusätzlich »Erfahrene x/y«. Ein Klick auf einen Chip wählt diese Schicht. Gibt es keine parallelen Schichten, entfällt die Leiste.

**Engpass-Hinweis.** Eine erfahrene Person ist in einer parallelen Schicht unverzichtbar, wenn diese Schicht ohne sie ihren Erfahrungsbedarf nicht mehr erfüllen könnte. Dann steht bei ihr ein Hinweis »⚠ Wird in {Schicht} gebraucht«. Der Hinweis ist nur ein Hinweis: Du kannst die Person trotzdem einteilen. Der eigene Erfahrungsbedarf der gewählten Schicht hat Vorrang. Die Auto-Verteilung füllt zuerst Schichten, denen erfahrene Personen fehlen, und nimmt Personen mit diesem Hinweis nur, wenn sonst niemand frei ist. Die Vorschau nennt solche Fälle unter »Hinweise«.

**Die Auto-Verteilung** füllt die Schichten des aktuellen Filters Schritt für Schritt und zeigt immer zuerst eine Vorschau. Sie ändert nie bestehende Einteilungen oder Ablehnungen, bucht niemanden um und behandelt alle Regeln als hart: Eine Schicht bleibt lieber unterbesetzt, als eine Regel zu verletzen. Offene Plätze erscheinen mit Grund. Nach dem Übernehmen nimmt »Rückgängig« die Änderung zurück, solange die Seite offen bleibt.

**Ansichten.** Die Liste ist die Standardansicht. Auf dem Desktop (ab etwa 1024 Pixel Breite) kommen über den Umschalter »Liste | Zeitleiste | Personen« zwei zeitliche Ansichten dazu. Auf dem Smartphone bleibt es bei der Liste. Beide zeigen jeweils einen Tag; mit ‹ › wechselst Du den Tag.

- **Zeitleiste:** Jede Zeile ist ein Bereich, jeder Balken eine Schicht mit ihrer Belegung. ⚠ zeigt offenen Erfahrungsbedarf, ★ Beliebtheit, 🔒 eine Sperre, ein »+« eine Überbelegung. Sich überschneidende Schichten desselben Bereichs liegen untereinander. Ein Klick auf einen Balken wählt die Schicht.
- **Personen:** Jede Zeile ist eine Person. Volle Balken sind ihre Einsätze, gestrichelte Balken offene Wünsche, graue Flächen Zeiten außerhalb ihrer Anwesenheit. Die Spalte der gewählten Schicht ist markiert. Mit »Nur frei während der gewählten Schicht« siehst Du nur Personen, die in dieser Zeit frei sind; ihr »+« trägt sie direkt in die Schicht ein. Ein Klick auf einen Namen markiert die Person. Diese Ansicht ist nicht der Tab »Personen«, sondern eine Ansicht im Tab »Verteilen«.

**Personen.** Listet alle mit Stunden, Last, Soll, Anwesenheit und den Personenfeldern. Die Erfahrungsstufe lässt sich direkt in der Zeile per Auswahl setzen, ohne das Dialogfenster der Person zu öffnen. Fehlt eine Angabe, nennt die Zeile das fehlende Feld. Du kannst Personen anlegen, eine Liste von Namen einfügen, Personen umbenennen (auch zwei Schreibweisen zusammenführen) und Angaben bearbeiten.

**Einrichtung.** Hier legst Du die Regeln des Plans fest. Die Abschnitte stehen in dieser Reihenfolge: Personenfelder, Erfahrung, Bereiche, Anwesenheit, Arbeitszeit.

- **Personenfelder** (Auswahl, Zahl, Ja/Nein, Text), die Teilnehmende oder nur die Koordination ausfüllen. Sie werden sofort gespeichert.
- **Erfahrung:** Wähle das Auswahlfeld, das als Erfahrung gilt. Die Reihenfolge der Optionen ist die Stufe, die niedrigste zuerst (z. B. »Neu < Erfahren < Profi«). Die Stufe gilt global für jede Person, nicht pro Bereich: Wer an der Technik Profi ist, ist an der Bar nicht automatisch Profi. Du setzt sie deshalb einmal pro Person, im Tab »Personen« oder im Dialogfenster der Person.
- **Bereiche:** Eine Tabelle mit allen Schichttypen und den Werten, die für ihre Schichten gelten:

  | Spalte | Bedeutung |
  |---|---|
  | Faktor | Belastungsfaktor: wie stark eine Stunde in diesem Bereich zählt (Standard 1). |
  | Beliebtheit | »Automatisch« (beliebt, sobald die Schicht überbucht ist), »Beliebt« oder »Nicht beliebt«. |
  | Erfahrene mindestens | Wie viele Personen mindestens mit der angegebenen Stufe eingeteilt sein müssen. Die Spalte erscheint nur, wenn ein Erfahrungsfeld gewählt ist. |

  Jede Zeile speichert, sobald Du das Feld verlässt. Die Werte gelten für alle Schichten des Bereichs; einzelne Schichten stellst Du im Schicht-Dialog abweichend ein.
- **Anwesenheit und Arbeitszeit:** Frage die Anwesenheit (von/bis) ab und setze die Arbeitszeitregeln: Soll-Stunden pro Person, Mindestpause und Höchststunden pro Tag. Diese Angaben speichert »Speichern«.

Datenschutz: Angaben, die Teilnehmende selbst machen, kann jede Person lesen, die den Namen kennt, weil es keinen Login gibt. Felder, die nur die Koordination ausfüllt, sehen nur Admins.


### Export
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

#### JSON
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

#### Offene Schichten als Textdatei
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

#### Timetable (PDF)
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

#### Poster (PDF)
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

#### Report (PDF)
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

#### Personal Overview (PDF)
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

## API für Entwickler:innen

Alle Funktionen des Frontends stehen auch als API zur Verfügung. Die Dokumentation findet sich unter [swagger/index.html](/swagger/index.html), die maschinenlesbare Swagger/OpenAPI-Spezifikation unter [swagger/doc.json](/swagger/doc.json).

