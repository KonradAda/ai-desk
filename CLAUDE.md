# ai-desk

Fizyczny hub na biurko do pilnowania agentów AI (na start: Claude Code). Mały ekran dotykowy (7″, 800×480), podłączony do komputera kablem USB, pokazuje do 6 aktywnych sesji agentów: co robią, czy czekają na zgodę, czy skończyły. Dotknięcie karty przenosi użytkownika do właściwej sesji na komputerze.

Cel biznesowy: gotowe urządzenie „wyjmij z pudełka, podłącz, działa”, a nie zestaw DIY. Projekt poboczny z ambicją na małą firmę. Najpierw prototyp dla siebie, potem walidacja rynku (filmik, lista oczekujących), dopiero potem mała seria.

## Architektura

```
Claude Code ──hooki──▶ aidesk hook ──HTTP localhost──▶ aidesk daemon ──USB serial──▶ ESP32-S3 + ekran 7″
                                                           │                              │
                                                           └──▶ symulator w przeglądarce   │
                                                           ◀────── dotknięcie karty ──────┘
```

1. **Hooki Claude Code**: konfiguracja w `settings.json` wywołuje `aidesk hook` przy zdarzeniach (`SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `Notification`, `Stop`, `SessionEnd`). Hook czyta JSON ze stdin i przekazuje go do daemona.
2. **Daemon (`aidesk`)**: jeden binarny plik w Go. Trzyma stan wszystkich sesji, wysyła go do urządzenia po USB i serwuje symulator ekranu w przeglądarce. Obsługuje akcje z urządzenia (np. przeniesienie fokusu do sesji).
3. **Firmware**: ESP32-S3 z ekranem 7″, C++ z LVGL. Rysuje karty i animowane stworki. Wysyła z powrotem dotknięcia.

## Technologie

- **Daemon i hook: Go.** Jeden statyczny plik exe, łatwa dystrybucja do klientów (Windows, macOS, Linux), szybki start (ważne, bo hook odpala się przy każdym użyciu narzędzia przez agenta), dobre biblioteki do portu szeregowego.
- **Firmware: C++, PlatformIO, LVGL.** LVGL to standard dla UI na ESP32. Rust na ESP32 z LVGL jest jeszcze niedojrzały.
- **Symulator: zwykły HTML i JS** serwowany przez daemona. Ten sam układ co na urządzeniu, żeby rozwijać UI bez sprzętu.

## Struktura katalogów

```
daemon/      Go: cmd/aidesk (CLI), internal/state (stan sesji), internal/server (HTTP, SSE, symulator w web/), internal/hook
firmware/    PlatformIO: ESP32-S3, LVGL
protocol/    specyfikacja komunikatów PC <-> urządzenie
design/      stworki (sprite'y), makiety ekranu
docs/        notatki, decyzje, badanie rynku
```

## Model stanu sesji

Każda sesja ma: `id` (session_id z Claude Code), `project` (nazwa katalogu z `cwd`), `status`, `action` (krótki opis bieżącej czynności, np. „Edytuje auth.ts”), `started_at`, `steps` (np. 3/7, z listy zadań agenta, jeśli jest), `mascot` (kolor i typ stworka, stały dla danego projektu).

Statusy: `working`, `waiting` (czeka na zgodę lub odpowiedź), `done`, `error`, `idle`.

Agent nie wie, ile zostało mu do końca. Nie pokazujemy procentów ani paska postępu, tylko status, czas, bieżącą akcję i kroki listy zadań.

## Protokół PC <-> urządzenie

JSON rozdzielany znakiem nowej linii (NDJSON) po USB CDC. Daemon wysyła pełny stan (lista kart), a urządzenie odsyła zdarzenia (`tap`, `approve`, `open`). Szczegóły w `protocol/` (do napisania).

## Zasady

- **Hook nigdy nie może spowolnić ani zablokować agenta.** Krótki timeout, zawsze kończy się kodem 0, gdy daemon nie działa, po cichu nic nie robi.
- **Żadnej marki Anthropic w produkcie.** Bez nazwy „Claude” w nazwie produktu, bez maskotki Claude Code (Clawd), bez ich logo i kolorystyki. Własne stworki to nasza marka. „Działa z Claude Code” w opisie jest ok.
- **Zatwierdzanie z urządzenia** (np. zgoda na komendę) to dopiero późniejsza wersja. Wymaga przemyślenia bezpieczeństwa: każdy, kto dotknie ekranu, może coś zatwierdzić. W v1 tylko podgląd i przeniesienie do sesji.
- **Najpierw Claude Code, ale bez zamykania drogi do innych narzędzi.** Źródła danych za interfejsem, żeby dało się dodać Codex, Gemini CLI itd.
- Kod, nazwy i komentarze po angielsku. Dokumentacja i notatki projektowe po polsku.
- Commity i PR-y bez dopisków o Claude: żadnego `Co-Authored-By: Claude` ani „Generated with Claude Code”.

## Roadmapa

1. Daemon i hooki, stan sesji widoczny w terminalu i w symulatorze w przeglądarce.
2. Używanie na co dzień przez 2–3 tygodnie, poprawki UI.
3. Firmware na ESP32-S3 7″, połączenie po USB.
4. Własne stworki i animacje (pracuje, czeka, śpi, błąd).
5. Obudowa (druk 3D), filmik, lista oczekujących.
6. Mała seria w przedsprzedaży, wtedy CE i formalności firmowe.

## Komendy

Wszystko z katalogu `daemon/`:

```bash
go test ./...                              # testy
go build -o aidesk.exe ./cmd/aidesk        # build
./aidesk.exe serve                         # daemon + symulator na http://127.0.0.1:7777
./aidesk.exe hooks-config                  # blok "hooks" do wklejenia w ~/.claude/settings.json
```

Demo bez prawdziwych agentów (przy działającym daemonie):

```bash
while IFS= read -r l; do printf '%s' "$l" | ./aidesk.exe hook; done < testdata/demo-events.jsonl
```

Zmienne środowiskowe: `AIDESK_ADDR` (domyślnie `127.0.0.1:7777`), `AIDESK_DEBUG=1` (hook wypisuje błędy na stderr zamiast milczeć).

Uwaga przy testach w Git Bash: `echo` z backslashami psuje ścieżki Windows w JSON. Używaj `printf '%s'` i danych z pliku.
