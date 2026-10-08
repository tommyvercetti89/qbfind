# QBFind Premium (Wails Edition)

Modern Wails/WebView2 edition of QBFind. This directory is a standalone Go module (`wailsapp`) and must be built with the Wails CLI, which also builds the Vite frontend.

## Prerequisites

*   Go 1.23 or higher
*   Node.js 18+ and npm
*   Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`

## Live Development

To run in live development mode, run `wails dev` in this directory. This runs a Vite development server with hot reload for frontend changes. A browser dev server is also available at http://localhost:34115 where you can call Go methods from devtools.

## Building

```powershell
wails build -clean -trimpath -ldflags "-s -w" -o QBFind_Premium.exe
```

`wails build` runs `npm install` and `vite build` automatically and generates `frontend/dist`, which is embedded into the binary with `//go:embed`. Because `frontend/dist` is not committed (it is build output), a plain `go build` **fails on a clean clone** until the frontend has been built. If you must use the Go toolchain directly:

```powershell
cd frontend
npm install
npm run build
cd ..
go build .
```

## Quality Checks

```powershell
go vet -unsafeptr=false ./...
```

`-unsafeptr=false` is intentional: the Win32 interop layer converts callback `uintptr` arguments back into `unsafe.Pointer` at the point of use (see the root `CONTRIBUTING.md` for the rationale). All other `go vet` analyzers stay enabled.
