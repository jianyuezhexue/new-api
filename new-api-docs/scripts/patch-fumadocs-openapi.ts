// Patch fumadocs-openapi to fix SSR hydration mismatch
// The original code uses 'https://loading' as base URL during SSR,
// causing "http://localhost:3001" vs "https://loading" mismatch.
// This replaces both branches with empty string to produce relative URLs.
import { readFile, writeFile } from 'node:fs/promises';

const TARGET = 'node_modules/fumadocs-openapi/dist/ui/operation/usage-tabs/client.js';

try {
  let content = await readFile(TARGET, 'utf8');
  const original = /typeof window !== 'undefined'\s*\n\s*\? window\.location\.origin\s*\n\s*: 'https:\/\/loading'/;
  if (original.test(content)) {
    content = content.replace(original, "''");
    await writeFile(TARGET, content, 'utf8');
    console.log('✅ Patched fumadocs-openapi hydration issue');
  } else {
    console.log('ℹ️  fumadocs-openapi already patched or uses different pattern');
  }
} catch (err) {
  console.error('❌ Failed to patch fumadocs-openapi:', err);
}
