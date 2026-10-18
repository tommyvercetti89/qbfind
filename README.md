# QBFind

[![Go Version](https://img.shields.io/github/go-mod/go-version/tommyvercetti89/qbfind?logo=go)](https://golang.org)
[![License](https://img.shields.io/github/license/tommyvercetti89/qbfind)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Windows-blue?logo=windows)](https://microsoft.com)

QBFind is a Windows desktop file finder focused on fast indexing, low resource usage, and practical search filters.

## Editions

- **QBFind Classic (root)**: Native Win32 UI, pure Go, lightweight executable.
- **QBFind Premium (`/wailsapp`)**: Modern Wails/WebView2 UI with Go backend and Vite frontend.

## Core Features

- Parallel filesystem indexing with persistent cache under `%APPDATA%/QBFind`.
- Rich query support (`ext:`, `name:`, `path:`, `size:`, `date:` and glob patterns).
- Ranked search results and configurable exclusion rules.
- Built-in text preview and bilingual interface support (English/Turkish).

## Requirements

- **Classic**: Go version declared in `go.mod` (currently Go 1.26.3).
- **Premium**: Go 1.23+, Node.js 18+, and Wails CLI (`v2.11.0`).

## Quick Start

### Classic (repository root)

```powershell
go run .
go build -ldflags "-s -w -H windowsgui" -o QBFind_Classic.exe
```

### Premium (`/wailsapp`)

```powershell
cd wailsapp
wails dev
wails build -clean -trimpath -ldflags "-s -w" -o QBFind_Premium.exe
```

> `frontend/dist` is generated at build time and not committed. On clean clones, build the frontend (or run `wails build`) before plain `go build` in `wailsapp`.

## Quality Checks

```powershell
# root
go vet -unsafeptr=false ./...
go test ./...

# wailsapp
cd wailsapp
go vet -unsafeptr=false ./...
```

## Contributing

Please keep root code Win32-focused, keep Wails-specific code under `/wailsapp`, and preserve English/Turkish localization for user-facing strings.

## License

MIT — see [LICENSE](LICENSE).
