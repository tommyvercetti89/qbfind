import './fonts.css';
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
    StartScan,
    GetExclude,
    SetExclude,
    GetDrives,
    SetDrives,
    GetHistory,
    AddHistory,
    GetSavedSearches,
    ToggleSavedSearch,
    CutFiles,
    DeleteToRecycleBin,
    RenameFile,
    StartDrag,
    CheckForUpdate,
    OpenURL,
    GetReleasesURL
} from '../wailsjs/go/main/App';
import * as wailsruntime from '../wailsjs/runtime/runtime';

import { i18n, tooltipTranslations } from './i18n.js';

// 2. Application State Variables
let currentLanguage = 'en';
let resultsList = [];
let selectedIndex = -1;
let indexedCount = 0;
let isScanning = false;
let scanSuffix = " (ready)";
let currentQuery = "";
let searchTimeout = null;
let searchSeq = 0;
let lastIndexCount = -1;

// 3. Select DOM Elements
const elSearchInput = document.getElementById('search-input');
const elClearSearch = document.getElementById('btn-clear-search');
const elSearchHistory = document.getElementById('search-history-list');
const elBtnSaveSearch = document.getElementById('btn-save-search');
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
const elBtnPreviewOpen = document.getElementById('btn-preview-open');
const elBtnPreviewCopy = document.getElementById('btn-preview-copy');
const elBtnPreviewClose = document.getElementById('btn-preview-close');
let previewItemPath = "";

// Settings Modal
const elSettingsModal = document.getElementById('settings-modal');
const elSettingsExclude = document.getElementById('settings-exclude-input');
const elSettingsDrives = document.getElementById('settings-drives-input');
const elBtnSettings = document.getElementById('btn-settings');
const elBtnSettingsSave = document.getElementById('btn-settings-save');
const elBtnSettingsClose = document.getElementById('btn-settings-close');
const elBtnSettingsCancel = document.getElementById('btn-settings-cancel');

// Info Modal
const elInfoModal = document.getElementById('info-modal');
const elInfoModalTitle = document.getElementById('info-modal-title');
const elInfoModalText = document.getElementById('info-modal-text');
const elBtnInfoClose = document.getElementById('btn-info-close');
const elBtnInfoOk = document.getElementById('btn-info-ok');
const elBtnCheckUpdate = document.getElementById('btn-check-update');

// Right-Click Context Menu DOM
const elContextMenu = document.getElementById('context-menu');
const elCtxOpen = document.getElementById('ctx-open');
const elCtxPreview = document.getElementById('ctx-preview');
const elCtxCopyPath = document.getElementById('ctx-copy-path');
const elCtxDesktop = document.getElementById('ctx-desktop');
const elCtxExplorer = document.getElementById('ctx-explorer');
const elCtxCut = document.getElementById('ctx-cut');
const elCtxRename = document.getElementById('ctx-rename');
const elCtxDelete = document.getElementById('ctx-delete');

// Rename Modal
const elRenameModal = document.getElementById('rename-modal');
const elRenameInput = document.getElementById('rename-input');
const elBtnRenameSave = document.getElementById('btn-rename-save');
const elBtnRenameClose = document.getElementById('btn-rename-close');
const elBtnRenameCancel = document.getElementById('btn-rename-cancel');
let renameTarget = null;

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
    const t = i18n[currentLanguage];
    showToast(isLight ? t.ThemeEnabledLight : t.ThemeEnabledDark, "info");
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
    renderListPlaceholder();
    refreshSearchSuggestions();
    
    // Focus search on start
    elSearchInput.focus();
    
    // Bind Wails events
    wailsruntime.EventsOn("status_update", (data) => {
        indexedCount = data.count;
        isScanning = data.scanning;
        scanSuffix = data.suffix;
        updateStatusBar();

        if (currentQuery.trim().length < 2) {
            renderListPlaceholder();
            return;
        }
        if (data.count !== lastIndexCount) {
            lastIndexCount = data.count;
            scheduleSearch(currentQuery);
        }
    });

    wailsruntime.EventsOn("language_changed", (langCode) => {
        currentLanguage = langCode.toLowerCase() === 'tr' ? 'tr' : 'en';
        applyTranslations();
        updateLanguageToggleUI();
        updateStatusBar();
        if (currentQuery.trim().length < 2) {
            renderListPlaceholder();
        }
    });
});


function applyTranslations() {
    const t = i18n[currentLanguage];
    
    // Apply data-i18n tags
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        if (t[key]) el.textContent = t[key];
    });
    
    // Update placeholders and accessibility labels
    elSearchInput.placeholder = t.SearchPlaceholder;
    elSearchInput.setAttribute('aria-label', t.SearchPlaceholder);
    elResultsList.setAttribute('aria-label', t.ResultsAria);

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
    
    if (currentQuery.trim().length < 2) {
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
    
    scheduleSearch(currentQuery);
});

function scheduleSearch(query) {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        performSearch(query);
    }, 150);
}

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
    
    const previousPath = selectedIndex >= 0 && selectedIndex < resultsList.length ? resultsList[selectedIndex].path : "";
    const seq = ++searchSeq;
    try {
        const results = await Search(query);
        if (seq !== searchSeq) {
            return;
        }
        resultsList = results;
        let keepIndex = -1;
        if (previousPath) {
            keepIndex = resultsList.findIndex((e) => e.path.toLowerCase() === previousPath.toLowerCase());
        }
        selectedIndex = keepIndex >= 0 ? keepIndex : (resultsList.length > 0 ? 0 : -1);
        renderResultsList();
        updateStatusBar();
        AddHistory(query).then(() => refreshSearchSuggestions()).catch(() => {});
    } catch (err) {
        console.error("Search error:", err);
    }
}

async function refreshSearchSuggestions() {
    try {
        const [history, saved] = await Promise.all([GetHistory(), GetSavedSearches()]);
        const items = [];
        const seen = new Set();
        for (const query of [...saved, ...history]) {
            const key = query.toLowerCase();
            if (key && !seen.has(key)) {
                seen.add(key);
                items.push(query);
            }
        }
        elSearchHistory.replaceChildren(...items.map((query) => {
            const option = document.createElement('option');
            option.value = query;
            return option;
        }));
    } catch {
        // suggestions are non-critical
    }
}

async function toggleSavedCurrentSearch() {
    const query = currentQuery.trim();
    if (query.length < 2) {
        return;
    }
    const saved = await ToggleSavedSearch(query);
    const isSaved = saved.some((q) => q.toLowerCase() === query.toLowerCase());
    const t = i18n[currentLanguage];
    showToast(isSaved ? t.SearchSaved : t.SearchUnsaved);
    refreshSearchSuggestions();
}

elBtnSaveSearch.addEventListener('click', toggleSavedCurrentSearch);

// 9. Premium DOM Rendering Functions
function renderListPlaceholder() {
    const t = i18n[currentLanguage];
    const countFormatted = formatNumber(indexedCount);
    resultsWindow = null;
    renderedStart = -1;
    renderedEnd = -1;
    
    elResultsList.innerHTML = `
        <div class="results-placeholder">
            <svg viewBox="0 0 24 24" class="pulse-icon"><path fill="currentColor" d="M15.5 14h-.79l-.28-.27C15.41 12.59 16 11.11 16 9.5 16 5.91 13.09 3 9.5 3S3 5.91 3 9.5 5.91 16 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/></svg>
            <p>${t.StatusMinChars.replace('{count}', countFormatted).replace('{suffix}', scanSuffix)}</p>
        </div>
    `;
}

const ROW_HEIGHT = 38;
const ROW_OVERSCAN = 8;
let resultsWindow = null;
let renderedStart = -1;
let renderedEnd = -1;

function renderResultsList() {
    resultsWindow = null;
    renderedStart = -1;
    renderedEnd = -1;

    if (resultsList.length === 0) {
        const placeholder = document.createElement('div');
        placeholder.className = 'results-placeholder';
        placeholder.innerHTML = `
            <svg viewBox="0 0 24 24"><path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/></svg>
            <p></p>
        `;
        placeholder.querySelector('p').textContent = `${i18n[currentLanguage].NoResults} "${currentQuery}"`;
        elResultsList.replaceChildren(placeholder);
        return;
    }

    const windowEl = document.createElement('div');
    windowEl.style.position = 'relative';
    windowEl.style.height = `${resultsList.length * ROW_HEIGHT}px`;
    elResultsList.replaceChildren(windowEl);
    resultsWindow = windowEl;
    renderVisibleRows(true);
    scrollToSelected();
}

function renderVisibleRows(force) {
    if (!resultsWindow) return;
    const total = resultsList.length;
    const viewport = elResultsList.clientHeight || 400;
    const scrollTop = elResultsList.scrollTop;
    const start = Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - ROW_OVERSCAN);
    const end = Math.min(total, Math.ceil((scrollTop + viewport) / ROW_HEIGHT) + ROW_OVERSCAN);
    if (!force && start === renderedStart && end === renderedEnd) return;
    renderedStart = start;
    renderedEnd = end;

    const fragment = document.createDocumentFragment();
    for (let idx = start; idx < end; idx++) {
        const row = buildResultRow(resultsList[idx], idx);
        row.style.position = 'absolute';
        row.style.top = `${idx * ROW_HEIGHT}px`;
        row.style.left = '0';
        row.style.right = '0';
        row.style.height = `${ROW_HEIGHT}px`;
        fragment.appendChild(row);
    }
    resultsWindow.replaceChildren(fragment);
}

function buildResultRow(e, idx) {
    const row = document.createElement('div');
    row.className = `result-row ${idx === selectedIndex ? 'selected' : ''}`;
    row.dataset.index = idx;
    row.setAttribute('role', 'option');
    row.setAttribute('aria-selected', idx === selectedIndex ? 'true' : 'false');

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
            <span class="file-name"></span>
        </div>
        <div class="col-data path-col"></div>
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

    const nameEl = row.querySelector('.file-name');
    nameEl.textContent = e.name;
    nameEl.title = e.name;
    const pathEl = row.querySelector('.path-col');
    pathEl.textContent = e.path;
    pathEl.title = e.path;

    // Native OLE drag-out (CF_HDROP) handled by the Go backend
    row.draggable = true;
    row.addEventListener('dragstart', (ev) => {
        ev.preventDefault();
        StartDrag([e.path]).catch(() => {});
    });

    row.addEventListener('click', () => {
        selectRow(idx);
    });

    row.addEventListener('dblclick', () => {
        OpenFile(e.path);
    });

    row.querySelector('.quick-open').addEventListener('click', (ev) => {
        ev.stopPropagation();
        OpenFile(e.path);
    });

    row.querySelector('.quick-copy').addEventListener('click', async (ev) => {
        ev.stopPropagation();
        const msg = await CopyPath(e.path);
        showToast(msg);
    });

    return row;
}

elResultsList.addEventListener('scroll', () => renderVisibleRows(false));
window.addEventListener('resize', () => {
    renderedStart = -1;
    renderVisibleRows(true);
});

function scrollToSelected() {
    if (selectedIndex < 0) return;
    const top = selectedIndex * ROW_HEIGHT;
    const bottom = top + ROW_HEIGHT;
    const viewTop = elResultsList.scrollTop;
    const viewBottom = viewTop + elResultsList.clientHeight;
    if (top < viewTop) {
        elResultsList.scrollTop = top;
    } else if (bottom > viewBottom) {
        elResultsList.scrollTop = bottom - elResultsList.clientHeight;
    }
}

function selectRow(idx) {
    if (idx < 0 || idx >= resultsList.length) return;
    selectedIndex = idx;
    scrollToSelected();
    renderVisibleRows(true);
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
    if (elSettingsModal.classList.contains('visible')) {
        if (e.key === 'Escape') {
            closeSettingsModal();
            e.preventDefault();
        }
        return;
    }
    if (elRenameModal.classList.contains('visible')) {
        if (e.key === 'Escape') {
            closeRenameModal();
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
            if (selectedIndex >= 0 && e.target !== elSearchInput) {
                OpenFile(resultsList[selectedIndex].path);
                e.preventDefault();
            }
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
            // Ctrl+C copies path of selected item (allow native copy inside the search box)
            if (e.ctrlKey && selectedIndex >= 0 && e.target !== elSearchInput) {
                triggerCopyPath();
                e.preventDefault();
            }
            break;

        case 'd':
        case 'D':
            if (e.ctrlKey) {
                toggleSavedCurrentSearch();
                e.preventDefault();
            }
            break;

        case 'x':
        case 'X':
            if (e.ctrlKey && selectedIndex >= 0 && e.target !== elSearchInput) {
                cutSelectedItem();
                e.preventDefault();
            }
            break;

        case 'Delete':
            if (selectedIndex >= 0 && e.target !== elSearchInput) {
                deleteSelectedItem();
                e.preventDefault();
            }
            break;

        case 'F2':
            if (selectedIndex >= 0 && e.target !== elSearchInput) {
                openRenameModal();
                e.preventDefault();
            }
            break;
    }
});

async function cutSelectedItem() {
    const item = await getSelected();
    if (!item) return;
    const msg = await CutFiles([item.path]);
    showToast(msg);
}

async function deleteSelectedItem() {
    const item = await getSelected();
    if (!item) return;
    const msg = await DeleteToRecycleBin([item.path]);
    showToast(msg);
    if (currentQuery.trim().length >= 2) {
        scheduleSearch(currentQuery);
    }
}

function openRenameModal() {
    getSelected().then((item) => {
        if (!item) return;
        renameTarget = item;
        elRenameInput.value = item.name;
        elRenameModal.classList.add('visible');
        elRenameInput.focus();
        elRenameInput.select();
    });
}

function closeRenameModal() {
    elRenameModal.classList.remove('visible');
    renameTarget = null;
    elSearchInput.focus();
}

async function commitRename() {
    if (!renameTarget) return;
    const target = renameTarget;
    const msg = await RenameFile(target.path, elRenameInput.value);
    showToast(msg);
    closeRenameModal();
    if (currentQuery.trim().length >= 2) {
        scheduleSearch(currentQuery);
    }
}

elBtnRenameClose.addEventListener('click', closeRenameModal);
elBtnRenameCancel.addEventListener('click', closeRenameModal);
elRenameModal.addEventListener('click', (e) => {
    if (e.target === elRenameModal) closeRenameModal();
});
elBtnRenameSave.addEventListener('click', commitRename);
elRenameInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        e.preventDefault();
        commitRename();
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
    lastIndexCount = -1;
    StartScan(true);
    if (currentQuery.trim().length >= 2) {
        scheduleSearch(currentQuery);
    }
    showToast(i18n[currentLanguage].Rescanning);
});

// 12. Modal Windows Controller
// Info Modal
elBtnInfo.addEventListener('click', async () => {
    const about = await GetAboutMessage();
    elInfoModalTitle.textContent = about.title;
    elInfoModalText.innerHTML = about.message.replace(/\n/g, '<br/>');
    if (about.version) {
        document.getElementById('about-version').textContent =
            `${i18n[currentLanguage].VersionLabel} ${about.version} (Wails Edition)`;
    }
    elInfoModal.classList.add('visible');
});

elBtnInfoClose.addEventListener('click', closeInfoModal);
elBtnInfoOk.addEventListener('click', closeInfoModal);
elBtnCheckUpdate.addEventListener('click', async () => {
    const t = i18n[currentLanguage];
    try {
        const latest = await CheckForUpdate();
        if (latest) {
            showToast(t.UpdateAvailable.replace('{version}', latest));
            const url = await GetReleasesURL();
            OpenURL(url);
        } else {
            showToast(t.UpToDate);
        }
    } catch {
        showToast(t.UpdateFailed, 'info');
    }
});
elInfoModal.addEventListener('click', (e) => {
    if (e.target === elInfoModal) closeInfoModal();
});

function closeInfoModal() {
    elInfoModal.classList.remove('visible');
    elSearchInput.focus();
}

// Settings Modal controller
elBtnSettings.addEventListener('click', async () => {
    elSettingsExclude.value = await GetExclude();
    elSettingsDrives.value = await GetDrives();
    elSettingsModal.classList.add('visible');
    elSettingsExclude.focus();
});

elBtnSettingsClose.addEventListener('click', closeSettingsModal);
elBtnSettingsCancel.addEventListener('click', closeSettingsModal);
elSettingsModal.addEventListener('click', (e) => {
    if (e.target === elSettingsModal) closeSettingsModal();
});
elBtnSettingsSave.addEventListener('click', async () => {
    await SetExclude(elSettingsExclude.value);
    const msg = await SetDrives(elSettingsDrives.value);
    showToast(msg);
});

function closeSettingsModal() {
    elSettingsModal.classList.remove('visible');
    elSearchInput.focus();
}

// Text Preview Modal
elBtnPreview.addEventListener('click', openPreview);

async function openPreview() {
    const item = await getSelected();
    if (!item) return;
    
    previewItemPath = item.path;
    elPreviewFileName.textContent = item.name;
    elPreviewFileSize.textContent = formatBytes(item.size);
    
    const extBadge = elPreviewModal.querySelector('.modal-file-badge');
    const ext = item.lowerExt.replace('.', '').toUpperCase();
    extBadge.textContent = ext || 'FILE';
    
    const t = i18n[currentLanguage];
    elPreviewContentBox.textContent = t.LoadingPreview;
    elPreviewModal.classList.add('visible');
    
    try {
        const res = await GetPreview(item.path, item.size, item.name);
        elPreviewContentBox.textContent = res.text;
    } catch (err) {
        elPreviewContentBox.textContent = `${t.ErrorText}: ${err}`;
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

elBtnPreviewOpen.addEventListener('click', () => {
    if (previewItemPath) {
        OpenFile(previewItemPath);
    }
});

async function copyText(text) {
    if (navigator.clipboard) {
        try {
            await navigator.clipboard.writeText(text);
            return true;
        } catch {
            // fall through to the legacy path
        }
    }
    try {
        const area = document.createElement('textarea');
        area.value = text;
        area.style.position = 'fixed';
        area.style.opacity = '0';
        document.body.appendChild(area);
        area.select();
        const ok = document.execCommand('copy');
        area.remove();
        return ok;
    } catch {
        return false;
    }
}

elBtnPreviewCopy.addEventListener('click', async () => {
    const text = elPreviewContentBox.textContent;
    const ok = await copyText(text);
    const t = i18n[currentLanguage];
    showToast(ok ? t.PreviewCopied : t.CopyFailedToast, ok ? 'success' : 'info');
});

// 13. Data Formatting Helpers
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

const dateFormatters = {
    en: new Intl.DateTimeFormat('en-GB', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }),
    tr: new Intl.DateTimeFormat('tr-TR', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
};

function formatDate(isoString) {
    const date = new Date(isoString);
    if (isNaN(date.getTime())) return '-';
    return dateFormatters[currentLanguage].format(date);
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
    
    // Position and display context menu, clamped to the viewport
    elContextMenu.style.display = 'block';
    const menuW = elContextMenu.offsetWidth;
    const menuH = elContextMenu.offsetHeight;
    const x = Math.max(4, Math.min(e.clientX, window.innerWidth - menuW - 4));
    const y = Math.max(4, Math.min(e.clientY, window.innerHeight - menuH - 4));
    elContextMenu.style.left = `${x}px`;
    elContextMenu.style.top = `${y}px`;
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

elCtxCut.addEventListener('click', () => {
    cutSelectedItem();
    hideContextMenu();
});

elCtxRename.addEventListener('click', () => {
    openRenameModal();
    hideContextMenu();
});

elCtxDelete.addEventListener('click', () => {
    deleteSelectedItem();
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
