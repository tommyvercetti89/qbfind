import './style.css';
import {
    Search,
    OpenFile,
    ShowInExplorer,
    CopyPath,
    CopyToDesktop,
    GetPreview,
    GetLanguage,
    SetLanguage,
    GetAboutMessage,
    StartScan
} from '../wailsjs/go/main/App';
import * as wailsruntime from '../wailsjs/runtime/runtime';

// 1. Translation System Dictionary
const i18n = {
    en: {
        Open: "Open",
        Preview: "Preview",
        CopyPath: "Copy Path",
        Desktop: "Desktop",
        Explorer: "Explorer",
        Navigate: "Navigate",
        Copy: "Copy",
        StatusMinChars: "Type at least 2 characters. Index: {count} items{suffix}",
        StatusResults: "Showing {showing} results. Index: {count} items{suffix}",
        StatusIndex: "Index: {count} items{suffix}",
        WarnSelect: "Please select a file first.",
        SearchPlaceholder: "Type at least 2 characters to search (e.g. report.pdf or ext:png)...",
        Preparing: "Indexing in progress...",
        AboutTitle: "About QBFind",
    },
    tr: {
        Open: "Aç",
        Preview: "Önizle",
        CopyPath: "Yolu Kopyala",
        Desktop: "Masaüstü",
        Explorer: "Explorer",
        Navigate: "Yönlendir",
        Copy: "Kopyala",
        StatusMinChars: "En az 2 karakter yazın. İndeks: {count} öğe{suffix}",
        StatusResults: "{showing} sonuç gösteriliyor. İndeks: {count} öğe{suffix}",
        StatusIndex: "İndeks: {count} öğe{suffix}",
        WarnSelect: "Lütfen önce bir dosya seçin.",
        SearchPlaceholder: "Aramak için en az 2 karakter yazın (örn: rapor.pdf veya ext:png)...",
        Preparing: "İndeks hazırlanıyor...",
        AboutTitle: "QBFind Hakkında",
    }
};

// 2. Application State Variables
let currentLanguage = 'en';
let resultsList = [];
let selectedIndex = -1;
let indexedCount = 0;
let isScanning = false;
let scanSuffix = " (ready)";
let currentQuery = "";
let searchTimeout = null;

// 3. Select DOM Elements
const elSearchInput = document.getElementById('search-input');
const elClearSearch = document.getElementById('btn-clear-search');
const elResultsList = document.getElementById('results-list');
const elStatusIndicator = document.getElementById('status-indicator');
const elStatusText = document.getElementById('status-text');
const elLangBtn = document.getElementById('btn-lang');
const elThemeBtn = document.getElementById('btn-theme');
const elToastContainer = document.getElementById('toast-container');

// Toolbar Buttons
const elBtnOpen = document.getElementById('btn-open');
const elBtnPreview = document.getElementById('btn-preview');
const elBtnCopyPath = document.getElementById('btn-copy-path');
const elBtnDesktop = document.getElementById('btn-desktop');
const elBtnExplorer = document.getElementById('btn-explorer');
const elBtnRefresh = document.getElementById('btn-refresh');
const elBtnInfo = document.getElementById('btn-info');

// Preview Modal
const elPreviewModal = document.getElementById('preview-modal');
const elPreviewFileName = document.getElementById('preview-file-name');
const elPreviewFileSize = document.getElementById('preview-file-size');
const elPreviewContentBox = document.getElementById('preview-content-box');
const elBtnPreviewCopy = document.getElementById('btn-preview-copy');
const elBtnPreviewClose = document.getElementById('btn-preview-close');

// Info Modal
const elInfoModal = document.getElementById('info-modal');
const elInfoModalTitle = document.getElementById('info-modal-title');
const elInfoModalText = document.getElementById('info-modal-text');
const elBtnInfoClose = document.getElementById('btn-info-close');
const elBtnInfoOk = document.getElementById('btn-info-ok');

// Right-Click Context Menu DOM
const elContextMenu = document.getElementById('context-menu');
const elCtxOpen = document.getElementById('ctx-open');
const elCtxPreview = document.getElementById('ctx-preview');
const elCtxCopyPath = document.getElementById('ctx-copy-path');
const elCtxDesktop = document.getElementById('ctx-desktop');
const elCtxExplorer = document.getElementById('ctx-explorer');

// Window Controls Action Bindings (Wails British English spellings)
document.getElementById('btn-minimize').addEventListener('click', () => wailsruntime.WindowMinimise());
document.getElementById('btn-maximize').addEventListener('click', () => wailsruntime.WindowToggleMaximise());
document.getElementById('btn-close').addEventListener('click', () => wailsruntime.Quit());

// Persistent Theme Switching & Icon Rendering
const savedTheme = localStorage.getItem('theme') || 'dark';
if (savedTheme === 'light') {
    document.body.classList.add('light-theme');
    updateThemeIcon(true);
} else {
    updateThemeIcon(false);
}

elThemeBtn.addEventListener('click', () => {
    const isLight = document.body.classList.toggle('light-theme');
    localStorage.setItem('theme', isLight ? 'light' : 'dark');
    updateThemeIcon(isLight);
    showToast(
        currentLanguage === 'tr' 
            ? (isLight ? "Açık tema aktif edildi" : "Koyu tema aktif edildi")
            : (isLight ? "Light theme enabled" : "Dark theme enabled"),
        "info"
    );
});

function updateThemeIcon(isLight) {
    if (isLight) {
        // Show Moon icon in light mode to switch back to dark
        elThemeBtn.innerHTML = `<svg viewBox="0 0 24 24" class="theme-icon-moon" style="width: 14px; height: 14px;"><path fill="currentColor" d="M12 3c.132 0 .263 0 .393.007a7.5 7.5 0 0 0 7.92 12.446A9 9 0 1 1 12 3zm1 2.112A7.002 7.002 0 0 0 18.888 11 7 7 0 1 0 13 5.112z"/></svg>`;
    } else {
        // Show Sun icon in dark mode to switch to light
        elThemeBtn.innerHTML = `<svg viewBox="0 0 24 24" class="theme-icon-sun" style="width: 14px; height: 14px;"><path fill="currentColor" d="M12 7c-2.76 0-5 2.24-5 5s2.24 5 5 5 5-2.24 5-5-2.24-5-5-5zM2 13h2c.55 0 1-.45 1-1s-.45-1-1-1H2c-.55 0-1 .45-1 1s.45 1 1 1zm18 0h2c.55 0 1-.45 1-1s-.45-1-1-1h-2c-.55 0-1 .45-1 1s.45 1 1 1zM11 2v2c0 .55.45 1 1 1s1-.45 1-1V2c0-.55-.45-1-1-1s-1 .45-1 1zm0 18v2c0 .55.45 1 1 1s1-.45 1-1v-2c0-.55-.45-1-1-1s-1 .45-1 1zM5.99 4.58c-.39-.39-1.03-.39-1.41 0s-.39 1.03 0 1.41l1.06 1.06c.39.39 1.03.39 1.41 0s.39-1.03 0-1.41L5.99 4.58zm12.37 12.37c-.39-.39-1.03-.39-1.41 0s-.39 1.03 0 1.41l1.06 1.06c.39.39 1.03.39 1.41 0s.39-1.03 0-1.41l-1.06-1.06zm1.06-10.96c.39-.39.39-1.03 0-1.41s-1.03-.39-1.41 0l-1.06 1.06c-.39.39-.39 1.03 0 1.41s1.03.39 1.41 0l1.06-1.06zM7.05 18.01c.39-.39.39-1.03 0-1.41s-1.03-.39-1.41 0l-1.06 1.06c-.39.39-.39 1.03 0 1.41s1.03.39 1.41 0l1.06-1.06z"/></svg>`;
    }
}

// 4. Initial Startup Hook
window.addEventListener('DOMContentLoaded', async () => {
    // Load initial language
    const lang = await GetLanguage();
    currentLanguage = lang.toLowerCase() === 'tr' ? 'tr' : 'en';
    
    // Configure initial localization
    applyTranslations();
    updateLanguageToggleUI();
    
    // Focus search on start
    elSearchInput.focus();
    
    // Bind Wails events
    wailsruntime.EventsOn("status_update", (data) => {
        indexedCount = data.count;
        isScanning = data.scanning;
        scanSuffix = data.suffix;
        updateStatusBar();
    });

    wailsruntime.EventsOn("language_changed", (langCode) => {
        currentLanguage = langCode.toLowerCase() === 'tr' ? 'tr' : 'en';
        applyTranslations();
        updateLanguageToggleUI();
        updateStatusBar();
    });
});

// 5. Localization Functions & Tooltips Dictionary
const tooltipTranslations = {
    en: {
        "btn-lang": "Switch Language",
        "btn-theme": "Toggle Dark/Light Theme",
        "btn-minimize": "Minimize Window",
        "btn-maximize": "Maximize Window",
        "btn-close": "Close Application",
        "btn-open": "Open Selected (Enter)",
        "btn-preview": "Preview Text (Space)",
        "btn-copy-path": "Copy File Path (Ctrl+C)",
        "btn-desktop": "Copy to Desktop",
        "btn-explorer": "Reveal in Explorer",
        "btn-refresh": "Re-index System Drives",
        "btn-info": "About QBFind"
    },
    tr: {
        "btn-lang": "Dili Değiştir",
        "btn-theme": "Açık/Koyu Tema Değiştir",
        "btn-minimize": "Pencereyi Küçült",
        "btn-maximize": "Pencereyi Büyüt / Ekranı Kapla",
        "btn-close": "Uygulamayı Kapat",
        "btn-open": "Seçileni Aç (Enter)",
        "btn-preview": "Metni Önizle (Space)",
        "btn-copy-path": "Dosya Yolunu Kopyala (Ctrl+C)",
        "btn-desktop": "Masaüstüne Kopyala",
        "btn-explorer": "Dosya Konumunu Aç",
        "btn-refresh": "Sistem Disklerini Yeniden Tara",
        "btn-info": "QBFind Hakkında"
    }
};

function applyTranslations() {
    const t = i18n[currentLanguage];
    
    // Apply data-i18n tags
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (t[key]) el.textContent = t[key];
    });
    
    // Update placeholders
    elSearchInput.placeholder = t.SearchPlaceholder;

    // Localize custom responsive tooltips
    const tt = tooltipTranslations[currentLanguage];
    for (const [id, text] of Object.entries(tt)) {
        const el = document.getElementById(id);
        if (el) el.setAttribute('data-tooltip', text);
    }
}

function updateLanguageToggleUI() {
    elLangBtn.textContent = currentLanguage === 'tr' ? 'EN' : 'TR';
}

elLangBtn.addEventListener('click', () => {
    const newLang = currentLanguage === 'tr' ? 'en' : 'tr';
    SetLanguage(newLang);
});

// 6. Interactive Toast Notification Utility
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    toast.className = `toast-notification toast-${type}`;
    
    const icon = type === 'success' 
        ? '<svg viewBox="0 0 24 24" class="toast-icon"><path fill="currentColor" d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/></svg>'
        : '<svg viewBox="0 0 24 24" class="toast-icon"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>';
        
    toast.innerHTML = `${icon}<span>${message}</span>`;
    elToastContainer.appendChild(toast);
    
    // Autoremove after slideout
    setTimeout(() => {
        toast.classList.add('hide');
        setTimeout(() => toast.remove(), 250);
    }, 2000);
}

// 7. Status Bar Update loop
function updateStatusBar() {
    const t = i18n[currentLanguage];
    
    // Pulsing Neon Dot State
    if (isScanning) {
        elStatusIndicator.className = 'pulse-dot scanning';
    } else {
        elStatusIndicator.className = 'pulse-dot ready';
    }
    
    const countFormatted = formatNumber(indexedCount);
    
    if (currentQuery.length < 2) {
        elStatusText.textContent = t.StatusMinChars
            .replace('{count}', countFormatted)
            .replace('{suffix}', scanSuffix);
    } else {
        elStatusText.textContent = t.StatusResults
            .replace('{showing}', resultsList.length)
            .replace('{count}', countFormatted)
            .replace('{suffix}', scanSuffix);
    }
}

// 8. Dynamic Search trigger pipeline
elSearchInput.addEventListener('input', (e) => {
    currentQuery = e.target.value;
    
    if (currentQuery.length > 0) {
        elClearSearch.classList.add('visible');
    } else {
        elClearSearch.classList.remove('visible');
    }
    
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        performSearch(currentQuery);
    }, 150);
});

elSearchInput.addEventListener('keydown', (e) => {
    if (e.key === "Escape") {
        e.preventDefault();
        elClearSearch.click();
    }
})

elClearSearch.addEventListener('click', () => {
    elSearchInput.value = "";
    currentQuery = "";
    elClearSearch.classList.remove('visible');
    performSearch("");
    elSearchInput.focus();
});

async function performSearch(query) {
    if (query.trim().length < 2) {
        resultsList = [];
        selectedIndex = -1;
        renderListPlaceholder();
        updateStatusBar();
        return;
    }
    
    try {
        resultsList = await Search(query);
        selectedIndex = resultsList.length > 0 ? 0 : -1;
        renderResultsList();
        updateStatusBar();
    } catch (err) {
        console.error("Search error:", err);
    }
}

// 9. Premium DOM Rendering Functions
function renderListPlaceholder() {
    const t = i18n[currentLanguage];
    const countFormatted = formatNumber(indexedCount);
    
    elResultsList.innerHTML = `
        <div class="results-placeholder">
            <svg viewBox="0 0 24 24" class="pulse-icon"><path fill="currentColor" d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>
            <p>${t.StatusMinChars.replace('{count}', countFormatted).replace('{suffix}', scanSuffix)}</p>
        </div>
    `;
}

function renderResultsList() {
    if (resultsList.length === 0) {
        elResultsList.innerHTML = `
            <div class="results-placeholder">
                <svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
                <p>No results found for "${currentQuery}"</p>
            </div>
        `;
        return;
    }
    
    elResultsList.innerHTML = "";
    resultsList.forEach((e, idx) => {
        const row = document.createElement('div');
        row.className = `result-row ${idx === selectedIndex ? 'selected' : ''}`;
        row.dataset.index = idx;
        
        // Icon Badge Determination based on File Ext
        let badgeClass = 'badge-other';
        let badgeText = e.isDir ? 'DIR' : 'FILE';
        
        if (e.isDir) {
            badgeClass = 'badge-dir';
            badgeText = currentLanguage === 'tr' ? 'KLSR' : 'DIR';
        } else {
            const ext = e.lowerExt.replace('.', '');
            if (ext) {
                badgeText = ext.toUpperCase().slice(0, 4);
                if (['exe', 'bat', 'cmd', 'lnk', 'url'].includes(ext)) {
                    badgeClass = 'badge-exe';
                } else if (['go', 'py', 'js', 'ts', 'html', 'css', 'json', 'md', 'xml'].includes(ext)) {
                    badgeClass = 'badge-code';
                } else if (['txt', 'log', 'ini', 'csv', 'yaml', 'yml'].includes(ext)) {
                    badgeClass = 'badge-text';
                }
            }
        }
        
        const sizeFormatted = e.isDir ? '' : formatBytes(e.size);
        const dateFormatted = formatDate(e.modTime);
        
        row.innerHTML = `
            <div class="col-data name-col">
                <span class="file-icon-badge ${badgeClass}">${badgeText}</span>
                <span class="file-name" title="${e.name}">${e.name}</span>
            </div>
            <div class="col-data path-col" title="${e.path}">${e.path}</div>
            <div class="col-data size-col">${sizeFormatted}</div>
            <div class="col-data modified-col">${dateFormatted}</div>
            
            <!-- Floating Inline Hover Action triggers -->
            <div class="row-actions-trigger" style="--wails-draggable:none">
                <button class="row-action-btn quick-open" title="Open File">
                    <svg viewBox="0 0 24 24"><path fill="currentColor" d="M19 19H5V5h7V3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2v-7h-2v7zM14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3h-7z"/></svg>
                </button>
                <button class="row-action-btn quick-copy" title="Copy Path">
                    <svg viewBox="0 0 24 24"><path fill="currentColor" d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/></svg>
                </button>
            </div>
        `;
        
        // Row Interactive Listeners
        row.addEventListener('click', () => {
            selectRow(idx);
        });
        
        row.addEventListener('dblclick', () => {
            OpenFile(e.path);
        });
        
        // Inline Action button click binders
        row.querySelector('.quick-open').addEventListener('click', (ev) => {
            ev.stopPropagation();
            OpenFile(e.path);
        });
        
        row.querySelector('.quick-copy').addEventListener('click', async (ev) => {
            ev.stopPropagation();
            const msg = await CopyPath(e.path);
            showToast(msg);
        });
        
        elResultsList.appendChild(row);
    });
    
    // Auto-scroll to selected row
    const selectedEl = elResultsList.querySelector('.result-row.selected');
    if (selectedEl) {
        selectedEl.scrollIntoView({ block: 'nearest' });
    }
}

function selectRow(idx) {
    if (idx < 0 || idx >= resultsList.length) return;
    selectedIndex = idx;
    
    const rows = elResultsList.querySelectorAll('.result-row');
    rows.forEach((r, i) => {
        if (i === selectedIndex) {
            r.classList.add('selected');
            r.scrollIntoView({ block: 'nearest' });
        } else {
            r.classList.remove('selected');
        }
    });
}

// 10. Keydown Bindings for Fluid Keyboard Navigation
window.addEventListener('keydown', (e) => {
    // If a modal is open, let modal capture key events
    if (elPreviewModal.classList.contains('visible')) {
        if (e.key === 'Escape') {
            closePreviewModal();
            e.preventDefault();
        }
        return;
    }
    if (elInfoModal.classList.contains('visible')) {
        if (e.key === 'Escape' || e.key === 'Enter') {
            closeInfoModal();
            e.preventDefault();
        }
        return;
    }
    
    if (resultsList.length === 0) {
        if (e.key === 'Escape') {
            elSearchInput.value = "";
            currentQuery = "";
            elClearSearch.classList.remove('visible');
            performSearch("");
            e.preventDefault();
        }
        return;
    }
    
    switch (e.key) {
        case 'ArrowDown':
            if (selectedIndex < resultsList.length - 1) {
                selectRow(selectedIndex + 1);
            }
            e.preventDefault();
            break;
            
        case 'ArrowUp':
            if (selectedIndex > 0) {
                selectRow(selectedIndex - 1);
            }
            e.preventDefault();
            break;
            
        case 'Enter':
            if (selectedIndex >= 0) {
                OpenFile(resultsList[selectedIndex].path);
            }
            e.preventDefault();
            break;
            
        case ' ':
            // Preview selected item on Spacebar press
            if (selectedIndex >= 0 && e.target !== elSearchInput) {
                openPreview();
                e.preventDefault();
            }
            break;
            
        case 'Escape':
            // ESC key refocuses input and clears search
            if (document.activeElement === elSearchInput) {
                elSearchInput.value = "";
                currentQuery = "";
                elClearSearch.classList.remove('visible');
                performSearch("");
            } else {
                elSearchInput.focus();
            }
            e.preventDefault();
            break;
            
        case 'c':
        case 'C':
            // Ctrl+C copies path of selected item
            if (e.ctrlKey && selectedIndex >= 0) {
                triggerCopyPath();
                e.preventDefault();
            }
            break;
    }
});

// 11. Actions Logic & Trigger bindings
async function getSelected() {
    if (selectedIndex < 0 || selectedIndex >= resultsList.length) {
        showToast(i18n[currentLanguage].WarnSelect, 'info');
        return null;
    }
    return resultsList[selectedIndex];
}

async function triggerCopyPath() {
    const item = await getSelected();
    if (!item) return;
    const msg = await CopyPath(item.path);
    showToast(msg);
}

elBtnOpen.addEventListener('click', async () => {
    const item = await getSelected();
    if (item) OpenFile(item.path);
});

elBtnCopyPath.addEventListener('click', triggerCopyPath);

elBtnDesktop.addEventListener('click', async () => {
    const item = await getSelected();
    if (!item) return;
    const msg = await CopyToDesktop(item.path);
    showToast(msg);
});

elBtnExplorer.addEventListener('click', async () => {
    const item = await getSelected();
    if (item) ShowInExplorer(item.path);
});

elBtnRefresh.addEventListener('click', () => {
    StartScan(true);
    showToast(currentLanguage === 'tr' ? "Sistem diskleri yeniden taranıyor..." : "Re-scanning system drives...");
});

// 12. Modal Windows Controller
// Info Modal
elBtnInfo.addEventListener('click', async () => {
    const about = await GetAboutMessage();
    elInfoModalTitle.textContent = about.title;
    elInfoModalText.innerHTML = about.message.replace(/\n/g, '<br/>');
    elInfoModal.classList.add('visible');
});

elBtnInfoClose.addEventListener('click', closeInfoModal);
elBtnInfoOk.addEventListener('click', closeInfoModal);
elInfoModal.addEventListener('click', (e) => {
    if (e.target === elInfoModal) closeInfoModal();
});

function closeInfoModal() {
    elInfoModal.classList.remove('visible');
    elSearchInput.focus();
}

// Text Preview Modal
elBtnPreview.addEventListener('click', openPreview);

function formatPreviewText(fileName, text) {
    if (!/\.json$/i.test(fileName)) {
        return text;
    }
    try {
        return JSON.stringify(JSON.parse(text), null, 2);
    } catch {
        return text;
    }
}

async function openPreview() {
    const item = await getSelected();
    if (!item) return;
    
    elPreviewFileName.textContent = item.name;
    elPreviewFileSize.textContent = formatBytes(item.size);
    
    const extBadge = elPreviewModal.querySelector('.modal-file-badge');
    const ext = item.lowerExt.replace('.', '').toUpperCase();
    extBadge.textContent = ext || 'FILE';
    
    elPreviewContentBox.textContent = currentLanguage === 'tr' ? 'Yükleniyor...' : 'Loading content...';
    elPreviewModal.classList.add('visible');
    
    try {
        const res = await GetPreview(item.path, item.size, item.name);
        if (res.success) {
            elPreviewContentBox.textContent = formatPreviewText(item.name, res.text);
        } else {
            elPreviewContentBox.textContent = "Error reading file preview.";
        }
    } catch (err) {
        elPreviewContentBox.textContent = `Error: ${err}`;
    }
}

elBtnPreviewClose.addEventListener('click', closePreviewModal);
elPreviewModal.addEventListener('click', (e) => {
    if (e.target === elPreviewModal) closePreviewModal();
});

function closePreviewModal() {
    elPreviewModal.classList.remove('visible');
    elSearchInput.focus();
}

elBtnPreviewCopy.addEventListener('click', () => {
    const text = elPreviewContentBox.textContent;
    navigator.clipboard.writeText(text);
    showToast(currentLanguage === 'tr' ? "Önizleme metni panoya kopyalandı." : "Preview text copied to clipboard.");
});

// 13. Data Formatting Helpers
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(isoString) {
    const date = new Date(isoString);
    if (isNaN(date.getTime())) return '-';
    
    const pad = (n) => n.toString().padStart(2, '0');
    
    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    const hours = pad(date.getHours());
    const minutes = pad(date.getMinutes());
    
    return `${day}.${month}.${year} ${hours}:${minutes}`;
}

function formatNumber(num) {
    if (currentLanguage === 'tr') {
        return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ".");
    }
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// 14. Custom Right-Click Context Menu Logic
elResultsList.addEventListener('contextmenu', (e) => {
    const row = e.target.closest('.result-row');
    if (!row) return;
    
    e.preventDefault();
    const idx = parseInt(row.dataset.index);
    selectRow(idx);
    
    // Position and display context menu dynamically
    elContextMenu.style.left = `${e.clientX}px`;
    elContextMenu.style.top = `${e.clientY}px`;
    elContextMenu.style.display = 'block';
});

elCtxOpen.addEventListener('click', async () => {
    const item = await getSelected();
    if (item) OpenFile(item.path);
    hideContextMenu();
});

elCtxPreview.addEventListener('click', () => {
    openPreview();
    hideContextMenu();
});

elCtxCopyPath.addEventListener('click', () => {
    triggerCopyPath();
    hideContextMenu();
});

elCtxDesktop.addEventListener('click', async () => {
    const item = await getSelected();
    if (!item) return;
    const msg = await CopyToDesktop(item.path);
    showToast(msg);
    hideContextMenu();
});

elCtxExplorer.addEventListener('click', async () => {
    const item = await getSelected();
    if (item) ShowInExplorer(item.path);
    hideContextMenu();
});

document.addEventListener('click', (e) => {
    if (!elContextMenu.contains(e.target)) {
        hideContextMenu();
    }
});

function hideContextMenu() {
    elContextMenu.style.display = 'none';
}
