import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { i18n, tooltipTranslations } from '../src/i18n.js';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const html = readFileSync(join(root, 'index.html'), 'utf8');
const htmlKeys = [...new Set([...html.matchAll(/data-i18n="([^"]+)"/g)].map((m) => m[1]))];

const enKeys = Object.keys(i18n.en).sort();
const trKeys = Object.keys(i18n.tr).sort();
const ttEn = Object.keys(tooltipTranslations.en).sort();
const ttTr = Object.keys(tooltipTranslations.tr).sort();

let failed = false;
const fail = (msg) => {
    console.error('FAIL: ' + msg);
    failed = true;
};

if (JSON.stringify(enKeys) !== JSON.stringify(trKeys)) {
    fail('i18n en/tr key sets differ');
}
if (JSON.stringify(ttEn) !== JSON.stringify(ttTr)) {
    fail('tooltip en/tr key sets differ');
}
for (const key of htmlKeys) {
    if (!i18n.en[key]) fail(`index.html data-i18n="${key}" missing in i18n.en`);
    if (!i18n.tr[key]) fail(`index.html data-i18n="${key}" missing in i18n.tr`);
}
for (const [, id] of html.matchAll(/id="(btn-[^"]+)"[^>]*data-tooltip=/g)) {
    if (!tooltipTranslations.en[id]) fail(`data-tooltip on #${id} missing in tooltipTranslations.en`);
    if (!tooltipTranslations.tr[id]) fail(`data-tooltip on #${id} missing in tooltipTranslations.tr`);
}

if (failed) {
    process.exit(1);
}
console.log(`i18n OK: ${enKeys.length} keys, ${htmlKeys.length} html refs, ${ttEn.length} tooltips`);
