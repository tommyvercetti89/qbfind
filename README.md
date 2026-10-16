# QBFind

QBFind is a Windows desktop file finder written in Go. The repository contains two editions that share the same indexing and query model:

- **Classic (Win32)**: native Win32 UI in the repository root.
- **Premium (Wails)**: Wails + WebView2 UI in `/home/runner/work/qbfind/qbfind/wailsapp`.

## Supported platform

- Microsoft Windows (the application code is guarded with `//go:build windows`).

## Editions

| Edition | Location | UI stack | Build output |
| --- | --- | --- | --- |
| Classic (Win32) | `/home/runner/work/qbfind/qbfind` | Native Win32 controls and message loop | `QBFind_Classic.exe` |
| Premium (Wails) | `/home/runner/work/qbfind/qbfind/wailsapp` | Go backend, Wails runtime, Vite frontend | `QBFind_Premium.exe` |

## Major features

Both editions implement the same core search and indexing behavior:

- Concurrent filesystem indexing with a worker pool sized from CPU count (bounded between 2 and 16 workers).
- Cached index loading on startup from `%APPDATA%\\QBFind\\index.cache.gz`.
- Query language with support for:
  - extension filters: `ext:pdf`, `extension:pdf`, `.pdf`, `*.pdf`
  - name and path filters: `name:report`, `path:documents`, `folder:photos`
  - glob patterns: `report*.pdf`
  - size filters: `size:>10mb`, `size:1mb-10mb`
  - date filters: `date:today`, `date:>=2024-01-01`
  - regex filters: `regex:`, `pathregex:`
  - fuzzy token matching with `~token`
- Ranking based on path priority, extension class, and match quality.
- Exclusion patterns configured with semicolon-separated values (for example `node_modules;.git;cache*`).
- English and Turkish UI support.

Additional capabilities in current UI implementations include result actions such as opening files, showing in Explorer, copying paths, and text preview handling through Windows shell integration.

## Architecture overview

### Shared backend model

The Classic and Premium modules both include:

- `indexer.go`: scanning, cache save/load, and live folder watch updates.
- `search.go`: query parsing, filtering, and top-N ranked result selection.
- `settings.go`: persisted settings for language, exclusions, drives, history, and saved searches.

### Classic edition (Win32)

- `quickfind.go`: process setup, Win32 constants, and application entry point.
- `ui.go`, `listview.go`, `settingswindow.go`: native window creation and interaction.
- `actions.go`, `ole.go`: shell actions and drag/drop interoperability.

### Premium edition (Wails)

- `wailsapp/main.go`: Wails app setup and embedded frontend assets.
- `wailsapp/app.go`: bound Go methods for frontend actions.
- `wailsapp/frontend`: Vite project (`index.html`, `src`, `tests`, `wailsjs`).

## Repository structure

```text
/home/runner/work/qbfind/qbfind
├─ .github/workflows/        # CI and release workflows
├─ go.mod                    # Classic module (Go 1.26.3)
├─ *.go                      # Classic Win32 source and tests
├─ CONTRIBUTING.md
├─ LICENSE
└─ wailsapp/
   ├─ go.mod                 # Premium module (Go 1.23)
   ├─ main.go app.go *.go
   ├─ wails.json
   ├─ build/
   └─ frontend/
      ├─ package.json
      ├─ src/
      ├─ tests/
      └─ wailsjs/
```

## Prerequisites

### Classic edition

- Go version from `/home/runner/work/qbfind/qbfind/go.mod` (`go 1.26.3`).

### Premium edition

- Go version from `/home/runner/work/qbfind/qbfind/wailsapp/go.mod` (`go 1.23`).
- Node.js 22 (same major version used in CI workflows).
- Wails CLI v2.11.0:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
```

## Development

### Classic (Win32)

Run from `/home/runner/work/qbfind/qbfind`:

```powershell
go run .
```

### Premium (Wails)

Run from `/home/runner/work/qbfind/qbfind/wailsapp`:

```powershell
wails dev
```

## Production builds

### Classic (Win32)

From `/home/runner/work/qbfind/qbfind`:

```powershell
go build -ldflags "-s -w -H windowsgui" -o QBFind_Classic.exe .
```

### Premium (Wails)

From `/home/runner/work/qbfind/qbfind/wailsapp`:

```powershell
wails build -clean -trimpath -ldflags "-s -w -X main.AppVersion=v0.0.0" -o QBFind_Premium.exe
```

The release workflow also builds an NSIS installer with `wails build -nsis`.

## Testing and quality checks

CI is defined in `/home/runner/work/qbfind/qbfind/.github/workflows/ci.yml` and runs on Windows.

From `/home/runner/work/qbfind/qbfind`:

```powershell
go vet -unsafeptr=false ./...
go test ./...
go test -run=^$ -bench=. -benchtime=1x ./...
```

From `/home/runner/work/qbfind/qbfind/wailsapp`:

```powershell
wails build -clean -trimpath -ldflags "-s -w -X main.AppVersion=dev" -o QBFind_Premium.exe
go vet -unsafeptr=false ./...
go test ./...
go test -run=^$ -bench=. -benchtime=1x ./...
npm --prefix frontend run test:i18n
```

`wails build` is run first because `wailsapp/main.go` embeds `frontend/dist` with `//go:embed all:frontend/dist`.

## Configuration and cache behavior

Both editions currently use the same configuration and cache paths:

- Settings: `%APPDATA%\\QBFind\\settings.txt`
- Index cache: `%APPDATA%\\QBFind\\index.cache.gz`

The index cache includes a version field and is treated as stale after 30 days.

## Contributing

See `/home/runner/work/qbfind/qbfind/CONTRIBUTING.md` for contribution rules and workflow.

## License

This project is licensed under the MIT License. See `/home/runner/work/qbfind/qbfind/LICENSE`.
