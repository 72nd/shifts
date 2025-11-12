# Shifts – The friendly collaborative shift planning tool

Shifts ist ein kollaboratives Schichtplanungstool, was versucht möglichst zugänglich zu sein.


## Hintergrund und Ziele

Das Programm wurde geschaffen, weil es bisher kein Schichtplanungstool gab, welches die Anforderungen von größeren bzw. komplexeren kollektiv organisierten Veranstaltungen erfüllte.

- Für Menschen, die eine Schicht übernehmen möchten, soll das Tool _so einfach wie möglich zu bedienen_ sein. Je weniger Steine man interessierten Personen in den Weg legt, umso wahrscheinlicher ist es, dass sie sich tatsächlich in eine oder mehrere Schichten eintragen. Shifts braucht von einer helfenden Person nur ein Name. Keine Mailadresse, kein Login.
- Kollaborative und kollektive Organisationsprozesse sollen damit abgebildet werden. Die Applikation soll _möglichst wenig Vorgaben_ machen, wie der Prozess der Schichtplanung und -vergabe gestaltet ist.
- Die Applikation soll auch auf einem Smartphone gut und einfach zu bedienen sein.
- Es sollte kein Problem sein, _viele Schichten_ zu verwalten und dennoch eine möglichst kompakte und übersichtliche Darstellung zu haben.
- Menschen aus der Orga (Admin) sollen möglichst gut bei ihrer Arbeit unterstützt werden. Reichhaltige Filter, Export und Berichtsfunktionen. Das Anlegen und Bearbeiten von Schichten soll möglichst schnell von der Hand gehen.


## Features

- Menschen tragen sich mit einem Namen/Nick/Pseudonym in eine Schicht ein. Keine Mailadresse oder Login wird benötigt. Einmal verwendet, wird der Name beim Eintragen in weiteren Schichten vorgeschlagen.
- Jede Person kann sich oder andere Personen jederzeit wieder austragen. Dies vereinfacht das Verwalten von Schichten.
- Schichten sind in Schichttypen (z.B. Bar) gruppiert. Jeder Schichttyp kann mit einem Beschreibungstext versehen werden, welcher die Aufgabe erklärt. Markdown wird hierfür unterstützt.
- Schichten
    - Haben eine definierte Startzeit und Dauer.
    - Anzahl der Helfenden kann definiert werden. Daneben ist es auch möglich Schichten zu erstellen, welche beliebig viele Helfende haben können. Dies kann für »All Hands on Deck« Situationen wie der Abbau hilfreich sein.
    - Wie bei den Schichttypen kann auch jede Schicht mit einem Beschreibungstext (optional in Markdown) versehen werden. Denkbar ist etwa die in dieser Zeit auftretenden Acts oder Bands zu kommunizieren.
- Filter: Der Schichtplan kann umfangreich gefiltert werden. So können gezielt bestimmte Schichttypen ein- und ausgeblendet werden oder nach den Belegungsstatus der Schichten gefiltert werden. Auch nach den Namen einzelner Helfer:innen kann gefiltert werden.
- i18n: Die Applikation ist aktuell in Deutsch und Englisch verfügbar.
- Tagesgrenze: Viele Veranstaltungen dauern lange in die Nacht. Oft werden hierbei die frühen Morgenstunden organisatorisch noch dem vorherigen Tag zugeordnet. Shifts erlaubt es deshalb, die Tagesgrenze für jeden Schichtplan individuell zu definieren.
- Adminzugang: Beim Erstellen eines neuen Schichtplans wird jedem Schichtplan ein Adminpwasswort zugewiesen. Damit kann danach auf folgende Funktionen zugegriffen werden.
    - Anlegen, Bearbeiten, Duplizieren von Schichten und Schichttypen.
    - Die Sortierung von Schichttypen ändern.
    - Beschreibungstext, Text, Tagesgrenze und weitere Einstellungen des Schichtplans anpassen.
    - Export: Exportiert die Daten des Schichtplans in verschiedenen Formaten. Hierbei kann optional die aktuelle Filterkonfiguration berücksichtigt werden. Weitere Informationen zu den Exportformaten finden sich unten.
- Kollaborativer Planungsmodus: Optimierter Modus für die Schichtplanung in Teams und Crews. Weitere Informationen weiter Unten.
- Export. Ist nach Eingabe des Adminpassworts möglich. Wenn gewünscht, kann beim Export die aktuelle Filterkonfiguration berücksichtigt werden. Verfügbare Formate:
    - JSON: Exportiert alle Daten des Schichtplans in JSON-Format.
    - Offene Schichten als Textdatei: Exportiert eine Übersicht der offenen Schichten als Textdatei. Dies ist besonders hilfreich, wenn eine Übersicht über offene Schichten z.B. in einen Chat gepostet werden soll.
    - Timetable (PDF): Stellt die Schichten und deren Belegung pro Tag als eine Zeittafel (»Stundenplan«) dar. Dies hilft insbesondere bei der Planung.
    - Poster (PDF): Bei vielen kollektiven Veranstaltungen ist es sinnvoll, nach Beginn der Veranstaltung auf einen papierbasierten Schichtplan umzusteigen, in welchen sich Menschen vor Ort eintragen können. Dieser Export erstellt solch einen Schichtplan. Je Tag und Schichttyp wird eine A4-Seite erstellt. Diese können danach gedruckt und aufgehängt werden.
    - Report (PDF): Erstellt eine für den Druck optimierte Ansicht des Schichtplans.
    - Personal Overview (PDF): Ein Dokument im A5-Format, welches die Schichten einer Person übersichtlich aufführt.


## Kollaborativer Planungsmodus

Auf Veranstaltungen gibt es zwei unterschiedliche Arten von Schichten. Viele Schichttypen können ohne große Absprache von fast allen Menschen übernommen werden. Diese erfordern in der Regel kaum Vorbereitungen oder Absprachen. Klassische Beispiele ist etwa der Abwasch. Dem gegenüber stehen Schichten, welche von organisierten Teams oder Crews übernommen werden. Diese erfordern oft Vorbereitung und Absprachen. Shifts möchte beiden Gruppen gerecht werden und bietet deshalb Crews den optionalen kollaborativen Planungsmodus an. Dieser kann in den Einstellungen des Schichtplans aktiviert werden.

Der kollaborative Planungsmodus steht im Gegensatz zum normalen Modus, bei welchem es darum geht, möglichst schnell alle Schichten nach dem First-Come-First-Served-Prinzip zu besetzen. Der kollaborative Modus hingegen geht von einem zweistufigen Prozess aus.

1. Alle Personen können Interesse an einer Schicht bekunden. Deshalb können Schichten in diesem Modus bewusst »überfüllt«/overbooked werden. Auf diese Art kann werden die Bedürfnisse und Wünsche aller Crewmitglieder offensichtlich gemacht.
2. Auf Grundlage der Interessen kann nun das Team entscheiden, welche Schichten nun konkret besetzt werden. Die Schichten werden dann im Schichtplan entsprechend geändert. Fertig verteilte Schichten können dann vom Admin fixiert/gesperrt werden. Danach sind sie von normalen Usern nicht mehr veränderbar. Die fixierten Schichten werden farblich hervorgehoben.

Dadurch ist es zu jeder Zeit für alle Beteiligten offensichtlich, welche Schichten definitiv zugeteilt sind. Dadurch, dass Personen sich selbst nicht mehr aus Schichten austragen können, werden diese motiviert, sich bei Änderungen an ihre Crewkoordination zu melden.