# Contributing to QBFind

First off, thank you for taking the time to contribute! 🎉

QBFind is built to be a fast, lightweight, and modern desktop search application for Windows. We distribute it in two editions: the **🚀 Win32 Classic Edition** (pure Go, Cgo-free) and the **✨ Wails Premium Edition** (GPU-accelerated modern GUI).

We welcome contributions to both editions. By contributing, you help make QBFind faster, more feature-rich, and beautiful for everyone.

---

## 🗺️ How Can I Contribute?

### 1. Reporting Bugs & Issues
*   Check if the bug has already been reported in the **Issues** tab.
*   If not, open a new issue. Provide a clear description, steps to reproduce, your Windows version, and logs/errors if applicable.

### 2. Suggesting Features
*   Open an issue with the tag `enhancement`.
*   Explain the utility of the feature and how it benefits the community (e.g. support for searching zip file archives, adding new shortcut keys, etc.).

### 3. Submitting Code Changes (Pull Requests)
*   **Step 1**: Fork the repository.
*   **Step 2**: Clone your fork locally:
    ```bash
    git clone https://github.com/YOUR-USERNAME/qbfind.git
    ```
*   **Step 3**: Create a separate branch for your work:
    ```bash
    git checkout -b feature/amazing-new-feature
    # or
    git checkout -b fix/resolve-scrolling-bug
    ```
*   **Step 4**: Implement your changes adhering to our **Technical Coding Rules** below.
*   **Step 5**: Test both editions to ensure no regressions occur.
*   **Step 6**: Commit your changes using descriptive, conventional commit messages:
    ```bash
    git commit -m "feat(wails): add zip file preview support"
    ```
*   **Step 7**: Push to your branch and open a **Pull Request (PR)** against our `main` branch.

---

## 🛠️ Technical Coding Rules

To maintain high code quality, performance, and cross-compatibility, please follow these guidelines:

### A. 🚀 Rules for Win32 Classic Edition (Root Folder)
*   **100% Cgo-Free**: We do not use Cgo or external third-party C/C++ DLLs. All procedures must be loaded dynamically from Windows system DLLs (`user32`, `kernel32`, `gdi32`, `shell32`, `comctl32`, `ole32`) via Go's `syscall.NewLazyDLL` or `lazyProc.Call`.
*   **Memory Management**: When allocating virtual memory or COM interfaces, always release allocations (`CoTaskMemFree`, `GlobalFree`) to prevent memory leaks in the background indexer.
*   **Native UI Bounds**: Keep custom sizing and layouts responsive within the `wndProc` WM_SIZE event routing.

### B. ✨ Rules for Wails Premium Edition (`/wailsapp`)
*   **Vanilla Core Design**: To avoid unnecessary bundler overhead, do not install Tailwind CSS or large JS UI libraries. Leverage Vanilla CSS Custom Properties (Variables) inside `style.css` to build highly responsive, custom-themeable UI layouts.
*   **Fluid UX**: Ensure all new interactive elements contain hover states, smooth CSS transitions (`cubic-bezier`), and custom tooltips (`data-tooltip`).
*   **Draggable Windows**: Ensure that custom draggable zones maintain `style="--wails-draggable:drag"` and interactive buttons use `style="--wails-draggable:none"` so that the application remains easy to position.

### C. 🌍 Bilingual Translations Constraint
*   QBFind is a bilingual application. 
*   If you add any text label, button, tooltip, right-click menu item, or alert popup, you **MUST** add localizations for both:
    *   **English (`en`)**
    *   **Turkish (`tr`)**
*   Update these in the translation dictionaries in both `settings.go` (for Win32 Classic) and `main.js` (for Wails Premium).

### D. ✅ Local Quality Checks
Run these before opening a PR:

```powershell
# Classic edition (repository root)
go vet -unsafeptr=false ./...
go build .

# Premium edition
cd wailsapp
go vet -unsafeptr=false ./...
wails build
```

**Why `-unsafeptr=false`?** QBFind talks to Win32/COM directly. Callback signatures deliver pointer arguments as `uintptr` (for example the `wndProc` `lParam` and the OLE vtable callbacks in `ole.go`), and the codebase converts them back with `unsafe.Pointer` at the point of use. Go's `unsafeptr` analyzer flags that pattern by design; the conversions are intentional and the pointed-to memory is owned by Windows for the duration of the callback. Only this analyzer is disabled — every other `go vet` check stays enabled.

### E. 📦 Windows Resources (Classic Edition)
*   The Classic executable metadata (icon, version info) is generated into `resource_windows_amd64.syso` from `versioninfo.json`.
*   If you change `versioninfo.json`, regenerate the resource file with:
    ```powershell
    go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
    goversioninfo -o resource_windows_amd64.syso versioninfo.json
    ```

### F. 🌍 i18n Parity (Premium Edition)
*   Keep `en` and `tr` dictionaries in `wailsapp/frontend/src/i18n.js` in sync. Every `data-i18n` key used in `index.html` must exist in both languages.
*   Verify with `npm --prefix wailsapp/frontend run test:i18n` (also enforced in CI).

---

## 🤝 Need Help?

If you have any questions, feel free to open a discussion in our repository or reach out to the maintainers. Let's make QBFind the ultimate file finder together!
