# Dev Dojo — Oktober bis Mitte Dezember 2026

**Das Projekt:** Ein GUI-Editor für Hugo- und Astro-Projekte. Zielgruppe sind
Leute, die *keine* Entwickler sind — die Lücke, die praktisch jeder Static
Site Generator offen lässt. Go im Backend, Vue + TypeScript im Frontend.

**Warum dieses Projekt:** Es trifft exakt den Stack, für den du dich bewirbst,
es lässt sich vorführen (ein GUI screencastet, ein CLI nicht), und du kennst
die Domäne bereits aus dem SSG.

**Zeitraum:** Mo 05.10. – So 14.12.2026 (10 Wochen)
**Budget:** 5h Code/Tag, Mo–Sa · 1h Lesen/Tag

**Arbeitsname:** `staticierge-studio` (oder was dir besser gefällt — aber
entscheide es in Woche 1 und ändere es nicht mehr)

---

## Architektur-Entscheidung (Woche 1, einmal)

**Empfehlung: lokaler Go-Server + Vue-SPA im Browser. Wails erst später,
wenn überhaupt.**

Begründung: Du brauchst Dateisystemzugriff, deshalb muss ein lokaler Prozess
laufen — aber du brauchst kein natives Fenster. Ein lokaler Server kostet
dich nichts an Funktionalität und erspart dir Cross-Platform-Builds,
macOS-Code-Signing und Wails-spezifisches Debugging. Das ist realistisch eine
gesparte Woche.

Halte die Grenze zwischen Backend und UI trotzdem sauber (HTTP/JSON, keine
Logik im Frontend), dann ist ein späteres Wails-Wrapping ein Zwei-Tage-Job
statt eines Rewrites.

⚠️ **Wenn du Wails trotzdem willst:** entscheide es jetzt, nicht im November.
Ein Umbau mittendrin kostet dich das Release.

---

## Scope-Grenze — was v1.0 **nicht** enthält

Das hier ist der wichtigste Abschnitt des Dokuments. "IDE" ist ein
unbegrenzter Begriff und frisst zehn Wochen ohne Ergebnis.

**Nicht in v1.0:**
Plugin-System · Theme-Designer · Git-Integration · Multi-User · Cloud-Sync ·
eigener Markdown-Parser · Unterstützung für weitere SSGs · Mobile ·
Kollaboration · Auth · Undo über Sessions hinweg

**In v1.0, sonst nichts:**
Projekt öffnen · Content-Baum durchsuchen · Markdown mit Frontmatter
bearbeiten · Live-Preview · Assets per Drag & Drop · Build/Serve ausführen
und Output anzeigen · an *ein* Ziel publishen

Jede Idee, die während der zehn Wochen aufkommt, wandert in `IDEAS.md` und
wird dort ohne Diskussion geparkt.

---

## Arbeitsweise (jede Woche)

| Tag | Fokus |
|---|---|
| **Mo** | Konzept + Design der Woche, Spikes |
| **Di/Mi** | Implementieren |
| **Do** | Refactoring + Idiom-Pass |
| **Fr** | Tests, `-race`, CI grün halten |
| **Sa** | Cleanup, Doku, Blogpost, Screenshot |

**Ab Tag 1 dabei, nicht nachgerüstet:** Tests, GitHub Actions CI,
`golangci-lint`, strukturierte Logs, ein `CHANGELOG.md`.

---

## Woche 1–2 (05.–18.10.) — Fundament

**Ziel:** Ein Projekt öffnen und seinen Inhalt sehen.

- Repo, Modulstruktur, CI, Linter, Makefile
- Vue 3 + TypeScript + Vite Setup, Backend serviert das gebaute Frontend
- Projekt-Erkennung: ist das ein Hugo- oder ein Astro-Projekt?
  (`config.toml`/`hugo.toml` vs. `astro.config.mjs`, Ordnerkonventionen)
- Dateisystem-Layer: Content-Verzeichnis rekursiv lesen, **mit sauberer
  Behandlung von Symlinks und Path-Traversal** — der Nutzer soll nie aus
  seinem Projektordner ausbrechen können
- HTTP-API definieren (`/api/project`, `/api/content`, …), einmal ordentlich
- Erste Vue-Ansicht: Dateibaum

🎯 **Meilenstein:** Ordner auswählen → Baum mit allen Content-Dateien.

📓 **Blog:** "Warum ich zwei Wochen ins Fundament stecke, bevor etwas sichtbar wird"

---

## Woche 3–4 (19.10.–01.11.) — Content & Frontmatter

**Ziel:** Dateien nicht nur sehen, sondern verwalten.

- Frontmatter-Parser (die Library aus September-Woche 2 wiederverwenden —
  YAML *und* TOML, beides kommt vor)
- Metadaten im UI anzeigen: Titel, Datum, Tags, Draft-Status
- Sortieren, Filtern, Volltextsuche über Content
- Neue Seite anlegen (aus Archetype/Template), umbenennen, löschen
- **Atomares Schreiben** (temp file + rename) — du fasst die Dateien echter
  Menschen an, ein halb geschriebener Post ist Datenverlust

🎯 **Meilenstein:** Content-Verwaltung ohne Texteditor möglich.

📓 **Blog:** "Frontmatter parsen klingt trivial — bis TOML dazukommt"

---

## Woche 5–6 (02.–15.11.) — Editor & Preview

**Ziel:** Das Kernstück. Hier wird das Ding real.

- CodeMirror 6 als Markdown-Editor einbinden (Syntax-Highlighting,
  Keybindings)
- Frontmatter als **Formular** statt als Text — genau das ist der Mehrwert
  für Nicht-Entwickler
- Live-Preview, gerendert vom Backend
- Assets: Bild per Drag & Drop → landet in `static/`/`public/`, Pfad wird im
  Markdown eingesetzt
- Autosave mit Debounce, "unsaved changes"-Zustand, Undo im Editor

⚠️ Die Preview ist die Stelle, an der du Zeit verlieren kannst. Rendere
zunächst nur Markdown, nicht die echten SSG-Templates. Template-getreue
Preview ist ein v2-Thema.

🎯 **Meilenstein:** Ein Post lässt sich vollständig ohne Terminal schreiben.

📓 **Blog:** "Frontmatter als Formular — die eine Idee hinter dem Editor"

---

## Woche 7–8 (16.–29.11.) — Build & Serve

**Ziel:** Deine Concurrency-Woche aus September zahlt sich aus.

- `hugo`/`astro` als Subprozess starten (`os/exec`) mit Context-Cancellation
- Build-Output **live streamen** in die UI (SSE oder WebSocket)
- Dev-Server starten, stoppen, Status anzeigen; Preview-Proxy
- Fehler aus dem Build parsen und im UI an der richtigen Datei anzeigen
- Sauberes Aufräumen: kein verwaister Prozess, wenn der Editor beendet wird

Das ist der technisch anspruchsvollste Teil und im Interview die beste
Geschichte: Prozessverwaltung, Streaming, Cancellation, Shutdown.

🎯 **Meilenstein:** Bearbeiten → Build → Ergebnis sehen, alles in der App.

📓 **Blog:** "Subprozesse, Streaming und Context — Go bei der Arbeit"

---

## Woche 9 (30.11.–06.12.) — Publish

**Ziel:** Ein Weg nach draußen. **Einer.**

- Ein Deploy-Ziel, vollständig: GitHub Pages *oder* Netlify *oder* rsync/SFTP
- Konfiguration im UI, Credentials sicher ablegen (nicht im Klartext im
  Projektordner)
- Fortschritt und Fehler sichtbar machen
- Der Weg muss von "Text tippen" bis "Seite ist online" ohne Terminal gehen

🎯 **Meilenstein:** Eine echte Website, publiziert komplett aus dem Tool.

📓 **Blog:** "Von Markdown zu live — der letzte Meter"

---

## Woche 10 (07.–14.12.) — Härten und ausliefern

Keine neuen Features. Null.

- Testabdeckung dort erhöhen, wo es wirklich weh täte (Dateischreiben,
  Pfadbehandlung, Prozessverwaltung)
- Fehlermeldungen für Nicht-Entwickler lesbar machen
- README: Problem, Screenshots, Quickstart, Architektur-Überblick, bewusste
  Scope-Entscheidungen
- **Screencast, 2–3 Minuten.** Das ist das Artefakt, das in Bewerbungen
  tatsächlich angeklickt wird — wichtiger als der Code darunter.
- Release bauen (Binaries für macOS/Linux/Windows), `v1.0.0` taggen
- Abschluss-Blogpost

🎯 **Meilenstein:** v1.0.0 released, Screencast online.

---

## Lesen — Oktober bis Dezember (1h/Tag)

Deine Liste ist zu groß und ~70% davon zahlt nicht auf dieses Ziel ein.
Ehrliche Sortierung:

### Lesen

**Oktober — *100 Go Mistakes* zu Ende** (die Kapitel, die im September
offen blieben) **+ *Efficient Go* selektiv**, sobald du in Woche 7/8
Subprozesse und Streaming baust. Nur die Kapitel zu Profiling und
Allocations, nicht linear.

**November/Dezember — *Designing Data-Intensive Applications*.**
Das ist das einzige Buch auf deiner Liste, das direkt auf
Senior-Backend-**Interviews** einzahlt. System-Design-Fragen in Deutschland
kommen fast alle aus diesem Stoff: Replikation, Partitionierung,
Transaktionen, Konsistenz. Kapitel 1–9 reichen.

⚠️ Prüf deine Ausgabe: die **2. Auflage** (Kleppmann & Riccomini) ist im
März 2026 erschienen und deutlich aktueller. Wenn du die 1. hast, lohnt der
Nachkauf hier ausnahmsweise.

**Nebenbei, wenn du im Frontend steckst:** *100 TypeScript Mistakes*,
kapitelweise nach Bedarf. Kein Durcharbeiten.

### Ausdrücklich nicht jetzt

*Go Design Patterns* — GoF-Muster auf Go übertragen ist überwiegend ein
Anti-Pattern; Go löst diese Probleme anders (Interfaces, Funktionen,
Embedding). Das Buch würde dir Gewohnheiten antrainieren, die dir ein
Go-Reviewer wieder austreibt. **Überspringen.**

*Distributed Services with Go* und *Build an Orchestrator in Go* — beides gut,
aber beides sind eigene Projekte. Sie konkurrieren mit dem Editor um dieselben
Stunden. Nach dem Release, nicht davor.

*Data Quality Fundamentals · Continuous API Management · Build Real-Time
Analytics Systems · Building an Event-Driven Data Mesh · Architecture
Modernization · Architecting for Scale · Foundations of Scalable Systems ·
Deciphering Data Architectures · Network Programmability & Automation* —
alle off-path für "Senior Backend/Go-Stelle im Januar". Mehrere davon
überschneiden sich ohnehin stark mit DDIA. Ins Regal, nicht in den Plan.

**Einzige Ausnahme:** *API Design Patterns* (Geewax) — ein Nachmittag
Querlesen im Dezember. API-Design kommt in Backend-Interviews verlässlich vor.

**Codewars:** 30 Minuten an den Kurz-Lesetagen. Ab November auf TypeScript
umstellen, damit die Frontend-Geläufigkeit bis zu den Gesprächen sitzt.

---

## Ab 15. Dezember — Bewerbungsblock

Steht hier, damit es nicht wieder hinten runterfällt. Kein Code mehr.

- Lebenslauf und Profile (LinkedIn, Xing, GitHub) neu aufsetzen
- Auszeit-Narrativ: bewusste Auszeit, genutzt für strukturierte Weiterbildung
  — mit Projekt, Blog und Release als Beleg
- Zielliste: 30–40 Firmen, Raum Augsburg/München, plus Remote
- Interviewvorbereitung: System Design (DDIA-Stoff), Go-Fragen,
  "erzähl von einem schwierigen Bug"
- **Erste Bewerbungen raus in der ersten Januarwoche**, nicht "im Januar mal
  anfangen"

---

## Puffer

Der Plan ist auf 5h/Tag gerechnet und hat bewusst **keine** Reserve
eingebaut — die Reserve bist du. Wenn eine Woche schiefgeht, wird Scope
gestrichen, nicht die Deadline verschoben und nicht am Wochenende
nachgeholt. Der Screencast am 14.12. ist der einzige unverschiebbare Punkt.

Ein Plan, der nur bei voller Intensität aufgeht, geht in der ersten
schlechten Woche dauerhaft kaputt. Vier gute Stunden schlagen sechs
erzwungene.
