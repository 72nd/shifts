
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
2. Auf Grundlage der bekundeten Interessen kann das Team nun entscheiden, welche Schichten konkret wie besetzt werden. Die entsprechenden Änderungen werden im Schichtplan vorgenommen. Finalisierte Schichten können vom Admin fixiert bzw. gesperrt werden. Danach sind sie für reguläre Nutzer:innen nicht mehr veränderbar. Fixierte Schichten werden farblich hervorgehoben.

Dadurch ist für alle Beteiligten jederzeit ersichtlich, welche Schichten definitiv zugeteilt sind. Da Personen sich nicht mehr eigenständig aus fixierten Schichten austragen können, werden sie motiviert, sich bei notwendigen Änderungen an ihre Crewkoordination zu wenden.


## Für Nutzende

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Ein- und Austragen in Schichten

Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Filtern

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

![Beispiel einer Statistikansicht](/images/stats.png)

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

