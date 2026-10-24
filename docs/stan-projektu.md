# Stan projektu ai-desk

Stan na 23.10.2026, po pierwszym dniu pracy (27 commitów). Notatka po to, żeby wrócić do projektu i od razu wiedzieć, co jest, dlaczego tak, i co dalej.

## W skrócie

Działa cała strona komputerowa: daemon w Go dostaje zdarzenia z Claude Code przez hooki i pokazuje agentów na żywo w symulatorze ekranu 7″ w przeglądarce. Sprzętu (ESP32) jeszcze nie ma. Hooki są podpięte w globalnym `~/.claude/settings.json`, więc każda sesja Claude Code (terminal i aplikacja desktopowa) od razu pojawia się na ekranie, jeśli daemon działa.

Jak wrócić do pracy:

```bash
cd C:/Projects/ai-desk/daemon
go build -o aidesk.exe ./cmd/aidesk
./aidesk.exe serve
```

Potem otwórz http://127.0.0.1:7777 (ekran) i http://127.0.0.1:7777/gallery.html (wszystkie stworki i animacje).

## Co działa

**Dane o agentach (daemon, `internal/state`)**
- Status każdej sesji: pracuje, czeka na Ciebie (prośba o zgodę albo odpowiedź), skończył, bezczynny.
- Bieżąca akcja jako kod, np. `edit` + `auth.ts` (tłumaczy ją ekran).
- Tytuł sesji (ten sam co w aplikacji Claude), a bez tytułu początek pierwszego polecenia.
- Model, effort i miejsce uruchomienia (aplikacja albo terminal), czytane z końcówki pliku transkrypcji.
- Plan agenta (lista kroków) z obu narzędzi: `TodoWrite` w terminalu i `update_plan` w aplikacji desktopowej, łącznie z krótkimi aktualizacjami („następny krok”).
- Nazwa projektu to katalog główny repozytorium git, więc agent w podkatalogu nie zmienia nazwy.
- Bez limitu liczby sesji. Każda ma stałe miejsce (slot), a układ jest zapamiętywany per projekt w `layout.json`.

**Ekran (symulator, `internal/server/web`)**
- Układ zależny od liczby agentów: 1 = cały ekran, 2 = połówki, 3–6 w dwóch rzędach. Rzędy dobierane tak, żeby nie rozcinać grup projektów.
- Strony przy większej liczbie agentów: kropki, strzałki, przesunięcie palcem. Kropka pulsuje na pomarańczowo, gdy ktoś na tej stronie czeka.
- Nagłówek karty: `projekt / tytuł sesji`. Pod nim linia `Claude Code · Opus 5.5 · Medium effort`, bieżąca akcja, krok planu i czas.
- Pasek postępu z planu: krok 1 od 10%, kroki środkowe od n/k, ostatni ok. 15% przed końcem, pełznie w trakcie kroku, a 100% dopiero gdy agent skończy.
- Grupy projektów: agenci z jednego projektu są obok siebie we wspólnym pudełku, z wyblakłym kolorem grupy i nazwą projektu na kolorowej etykiecie.
- Przeciąganie kart myszką zamienia je miejscami.
- Ustawienia (zębatka): automatyczny lub ręczny wybór agentów, agentów na ekran (1–6), przeskakiwanie do agenta, który czeka, ukrywanie skończonych po X minutach, język.
- Tryb ręczny: nowe sesje nie wskakują same, „+” dodaje z listy (z pomarańczową kropką, gdy ukryty agent czeka), „×” zdejmuje z ekranu.
- Wszystkie teksty przez słownik `STRINGS`. Na razie tylko angielski.

**Stworki (`mascot.js`, `mascot.css`, `gallery.html`)**
- 4 stworki rysowane gładko (wektorowo, nie piksele): Ognik (Wisp), Kot, Żaba, Ślimak. Każdy ma inną sylwetkę (ogonek, uszy, oczy na wypukłościach, muszla), żeby agentów odróżniać na pierwszy rzut oka. 8 kolorów, każda karta na ekranie wygląda inaczej.
- Czapka czarodzieja ze złotą gwiazdką i różdżka.
- Pracuje: podskakuje, z różdżki lecą iskry w rytm podskoku.
- Czeka na Ciebie: pulsuje.
- Skończył lub bezczynny: śpi, oddycha, „Z” lecą do góry.
- Budzenie (1,8 s) i zasypianie (2,4 s): powieki stopniowo, mruganie, walka ze snem. Łagodne lądowanie, jeśli zmiana złapie stworka w powietrzu.
- Animacje liczą fazę od zegara, więc przerysowanie karty ich nie przerywa.

**Testy:** `go test ./...` w `daemon/` (stan sesji, plan, transkrypcja, układ, ustawienia, tryb ręczny przez HTTP).

## Najważniejsze decyzje i dlaczego

- **Go na komputer, C++ z LVGL na ESP32.** Użytkownik zna trochę Go, nie zna C++ ani Rusta. Go daje jeden plik exe dla klienta. Na ESP32 C++ i LVGL mają najlepsze wsparcie dla ekranów 7″. Rust rozważany, ale odłożony jako hobby na później.
- **Własne stworki, zero marki Anthropic.** Bez maskotki Claude Code i bez „Claude” w nazwie produktu (ryzyko prawne). „Działa z Claude Code” w opisie jest ok.
- **Daemon wysyła kody, a nie zdania.** Ekran tłumaczy przez słownik. Domyślnie angielski, polski do dopisania później (jeden wpis w `STRINGS` plus kod w `settings.Languages`).
- **Pixel art i animacje klatkami.** Ten styl jest tani dla ESP32: podmiana małych bitmap. Symulator powinien się tego trzymać, żeby był wiernym podglądem urządzenia.
- **Grupowanie po projekcie (repozytorium git).** Wspólne pudełko grupy ma być w przyszłości „areną” (patrz pomysły niżej).
- **Tryb ręczny i przypięcia trzyma daemon,** żeby urządzenie i podgląd zawsze pokazywały to samo.

## Czego się nauczyliśmy (pułapki)

- **Hooki Claude Code plus plik transkrypcji dają prawie wszystko.** Status, akcje, plan, tytuł, model i effort. Nie trzeba żadnego API.
- **Aplikacja desktopowa sama odhacza ostatni krok planu na końcu pracy,** bez wywołania narzędzia, które widzi hook. Dlatego plan jest czyszczony przy każdej nowej wiadomości.
- **Przerysowanie elementu restartuje animacje CSS.** Rozwiązanie: każda animacja dostaje ujemne opóźnienie liczone od zegara. To samo przyda się na ESP32: faza animacji ma wynikać z czasu, a nie z momentu rysowania.
- **Przejścia między stanami muszą startować od pozy, w której stworek akurat jest,** a następna animacja od początku swojego cyklu. Inaczej widać skoki.
- **Windows:** nie da się przebudować `aidesk.exe`, gdy daemon działa, więc najpierw `taskkill`. W Git Bash `echo` z backslashami psuje JSON, więc do testów używać `printf '%s'` i danych z pliku (`testdata/demo-events.jsonl`).
- **Daemon uruchomiony z sesji aplikacji Claude (np. przez agenta) widzi zwirtualizowany `%APPDATA%`**, więc zapisuje ustawienia i układ w `%LOCALAPPDATA%\Packages\Claude_pzs8sxrjxfjjc\LocalCache\Roaming\aidesk`, a nie w `%APPDATA%\aidesk`. Daemon uruchomiony samodzielnie z terminala ma więc osobne ustawienia. Do rozwiązania przy autostarcie (np. jawna ścieżka w `os.UserHomeDir()`).
- **Testowy daemon na innym porcie** (`AIDESK_ADDR`) z osobnym `AIDESK_CONFIG_DIR` nie rusza prawdziwych ustawień i układu. Port 7778 jest na tym komputerze zajęty, więc używać np. 7790.

## Kliknięcie w kartę: wyniki badania (23.10.2026)

Sprawdzone na żywo: **da się otworzyć konkretną sesję w aplikacji desktopowej Claude jednym linkiem.**

- Aplikacja Claude rejestruje w Windowsie linki `claude://`. Link **`claude://code/continue?session=<local_id>`** przełącza aplikację na wskazaną sesję. Test: link do „Sesja testowa” spowodował w logu aplikacji `LocalSessions.setFocusedSession: sessionId=local_be4b10bd-…`.
- `<local_id>` to wewnętrzne ID aplikacji (format `local_…`, sprawdzane regexem `^local_[A-Za-z0-9-]{1,64}$`), a **nie** UUID sesji, które dostajemy z hooków.
- **Powiązanie obu ID:** Claude Code dla każdej działającej sesji zapisuje `~/.claude/sessions/<pid>.json` z polami `sessionId` (UUID z hooków), `hostSessionId` (`local_…` z aplikacji), `entrypoint` (`claude-desktop` albo terminal), `pid`, `cwd`, `name` (tytuł) i `status`. Daemon może to czytać i znaleźć `local_…` dla karty. Pliki `<pid>.*.key` w tym folderze to klucze: nie czytać ich i nie wysyłać.
- Inne linki znalezione w aplikacji: `claude://code/continue?session=last` (ostatnia sesja), `claude://code/needs-input` (sesje czekające na użytkownika), `claude://code/new` (nowa sesja, z parametrami `q`/`prompt`, `folder`). `claude://resume?session=<UUID>` **importuje** sesję z terminala do aplikacji, a tego dla kart nie chcemy.
- **Sesje w terminalu:** do wyciągnięcia okna na wierzch przyda się `pid` z tego samego pliku (proces Claude Code, a od niego w górę do okna terminala). Wybór konkretnej karty w Windows Terminal czy VS Code pozostaje trudny.
- Ograniczenia: to nieudokumentowane zachowanie aplikacji, więc może się zmienić w nowej wersji. Aplikacja ma ustawienie, które wyłącza takie linki (w kodzie „entry points off”). Na Macu i Linuksie trzeba będzie sprawdzić osobno.

Plan wdrożenia: daemon czyta `~/.claude/sessions/*.json` i dokleja do sesji `host_id` oraz `pid`. Kliknięcie w kartę wysyła `POST /api/open`, a daemon dla sesji z aplikacji otwiera `claude://code/continue?session=<host_id>`, a dla terminala wyciąga okno na wierzch.

## Znane ograniczenia

- **Daemon trzeba uruchamiać ręcznie.** Nie ma autostartu ani ikony w zasobniku.
- **Kliknięcie w kartę nic nie robi** (jeszcze). Droga jest zbadana, patrz „Kliknięcie w kartę: wyniki badania”.
- **Zatwierdzanie z urządzenia** (zgoda na komendę) świadomie odłożone: każdy, kto dotknie ekranu, mógłby coś zatwierdzić.
- **Daemon nie pamięta stanu po restarcie.** Sesje wracają przy swojej następnej akcji, a trwający plan dopiero przy pełnej liście kroków. Przypięcia trybu ręcznego też giną.
- **Grupowanie po samej nazwie folderu:** dwa różne repozytoria o tej samej nazwie trafią do jednej grupy. Worktree z aplikacji desktopowej (`.claude/worktrees/...`) nie trafią do grupy swojego repozytorium.
- **Strony dzielą agentów po kolei,** więc grupa może zostać rozcięta między strony, chociaż zmieściłaby się na następnej.
- **Pole „agent” to na razie zawsze Claude Code.** Inne narzędzia (Codex, Gemini CLI) nie są obsługiwane.
- **Część animacji w symulatorze nie jest jeszcze „ESP-owa”:** płynne skalowanie „Z”, oddech przez skalowanie, przesunięcia ułamkowe, gładkie kształty SVG (na ESP32 do przerobienia na sprite'y albo rysunek w LVGL). Do zamiany na klatki przed portem na ESP32.
- **Ścieżka do `aidesk.exe` w hookach jest na sztywno** (`C:/Projects/ai-desk/daemon/aidesk.exe`). Przy przeniesieniu projektu trzeba ponownie uruchomić `aidesk.exe hooks-config` i podmienić hooki.

## Co dalej (proponowana kolejność)

1. **Wygoda na co dzień:** autostart daemona (ikona w zasobniku Windows) i używanie przez kilka dni przy normalnej pracy. Notować, czego brakuje.
2. **Kliknięcie w kartę przenosi do sesji:** dla aplikacji przez `claude://code/continue?session=<local_id>` (zbadane, działa), dla terminala wyciągnięcie okna na wierzch.
3. **Grupy liczone w daemonie:** pełna ścieżka repozytorium, worktree do głównego repozytorium, strony przenoszą całą grupę na następną stronę, jeśli się zmieści.
4. **Sprzęt:** zamówić płytkę ESP32-S3 z ekranem 7″ 800×480 (Waveshare albo Elecrow), opisać protokół PC↔urządzenie (NDJSON po USB) w `protocol/`, pierwszy firmware w C++ z LVGL. Najpierw test: kolorowy prostokąt i dotyk.
5. **Animacje pod ESP32:** zamienić animacje transformacji na klatki i ustawić całkowite skalowanie stworków.
6. **Stworki, krok po kroku:** różne animacje dla różnych akcji (edycja pliku, testy, szukanie), dopracowanie wyglądu paska postępu.
7. **Polski język** w słowniku.
8. **Walidacja pomysłu:** pogadać z 5 osobami pracującymi z agentami, nagrać 30-sekundowy filmik, zbierać zapisy na listę oczekujących.

## Pomysły na później

- **Walka jako animacja pracy:** stworki strzelają z łuku, walczą, a agenci z jednego projektu razem walczą z jednym wspólnym przeciwnikiem. Wspólne pudełko grupy to ich arena.
- **Motywy postaci** (średniowiecze: mag, wojownik, łucznik; mądrale: naukowiec, pisarz, filozof). Raz zrobione naraz i wycofane: robić po jednym, zaczynając od obecnego stworka.
- **Przycisk „uruchom nowego agenta” z urządzenia:** lista ostatnich projektów, a po wyborze nowy terminal z `claude` w tym folderze.
- **Wersja 5″** jako tańszy model.
