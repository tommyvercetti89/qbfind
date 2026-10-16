# QBFind Premium (Wails Edition)

This directory contains the Premium edition of QBFind. It is a standalone Go module (`module wailsapp`) with a Wails backend and a Vite frontend.

## Location and scope

- Module root: `/home/runner/work/qbfind/qbfind/wailsapp`
- Frontend project: `/home/runner/work/qbfind/qbfind/wailsapp/frontend`

## Prerequisites

- Windows environment (application sources use `//go:build windows`).
- Go 1.23 (from `/home/runner/work/qbfind/qbfind/wailsapp/go.mod`).
- Node.js 22 (same major version used in repository CI workflows).
- Wails CLI v2.11.0:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0
```

## Frontend setup

From `/home/runner/work/qbfind/qbfind/wailsapp/frontend`:

```powershell
npm install
```

Available frontend scripts (from `frontend/package.json`):

```powershell
npm run dev
npm run build
npm run preview
npm run test:i18n
```

## Development workflow

For normal development, run from `/home/runner/work/qbfind/qbfind/wailsapp`:

```powershell
wails dev
```

`wails dev` starts the desktop app and frontend dev workflow together.

## Production build

From `/home/runner/work/qbfind/qbfind/wailsapp`:

```powershell
wails build -clean -trimpath -ldflags "-s -w -X main.AppVersion=v0.0.0" -o QBFind_Premium.exe
```

Release automation in `.github/workflows/release.yml` also uses `wails build -nsis` to produce an installer executable.

## Validation commands

From `/home/runner/work/qbfind/qbfind/wailsapp`:

```powershell
wails build -clean -trimpath -ldflags "-s -w -X main.AppVersion=dev" -o QBFind_Premium.exe
go vet -unsafeptr=false ./...
go test ./...
go test -run=^$ -bench=. -benchtime=1x ./...
npm --prefix frontend run test:i18n
```

## Generated files and embedding behavior

- `main.go` embeds `frontend/dist` using `//go:embed all:frontend/dist`.
- Because of that embed directive, plain `go vet`, `go test`, or `go build` can fail on a clean clone until frontend assets are built.
- `wails build` handles frontend install and frontend build via `wails.json` (`frontend:install` and `frontend:build`).
- `frontend/dist` and `build/bin` are ignored in `/home/runner/work/qbfind/qbfind/wailsapp/.gitignore` and should not be committed.

## Configuration and cache paths

The Premium edition currently stores:

- Settings at `%APPDATA%\\QBFind\\settings.txt`
- Index cache at `%APPDATA%\\QBFind\\index.cache.gz`

These paths match the Classic edition.
