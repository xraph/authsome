/**
 * JavaScript Build Script for AuthSome Dashboard
 * Bundles Preline + custom components into a single file
 */

import * as esbuild from 'esbuild';
import { readFileSync, writeFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

async function build() {
  console.log('📦 Building dashboard JavaScript bundle...');

  // Read the existing custom JS files
  const pinesComponents = readFileSync(
    join(__dirname, '../static/js/pines-components.js'),
    'utf-8'
  );
  
  const dashboardJs = readFileSync(
    join(__dirname, '../static/js/dashboard.js'),
    'utf-8'
  );

  // Create the bundle entry point
  const bundleContent = `
// ===== Preline UI =====
import 'preline';

// ===== Custom Pines Components =====
${pinesComponents}

// ===== Dashboard Utilities =====
${dashboardJs}

// ===== Initialize Preline =====
document.addEventListener('DOMContentLoaded', () => {
  // Preline auto-initializes, but we can manually trigger if needed
  if (window.HSStaticMethods) {
    window.HSStaticMethods.autoInit();
  }
});

// Re-initialize Preline after Alpine.js updates the DOM
document.addEventListener('alpine:initialized', () => {
  setTimeout(() => {
    if (window.HSStaticMethods) {
      window.HSStaticMethods.autoInit();
    }
  }, 100);
});

console.log('🚀 AuthSome Dashboard bundle loaded (Preline + Alpine.js)');
`;

  // Write temporary entry file
  const entryFile = join(__dirname, 'src/.bundle-entry.js');
  writeFileSync(entryFile, bundleContent);

  try {
    // Bundle with esbuild
    await esbuild.build({
      entryPoints: [entryFile],
      bundle: true,
      minify: true,
      sourcemap: false,
      target: ['es2020'],
      format: 'iife',
      outfile: join(__dirname, '../static/js/bundle.js'),
      external: [],
      logLevel: 'info',
    });

    console.log('✅ JavaScript bundle created: static/js/bundle.js');
  } catch (error) {
    console.error('❌ Build failed:', error);
    process.exit(1);
  }
}

build();

