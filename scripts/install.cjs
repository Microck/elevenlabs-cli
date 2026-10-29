#!/usr/bin/env node
'use strict';

const { spawnSync } = require('node:child_process');
const { mkdirSync } = require('node:fs');
const path = require('node:path');

// Source packages compile the same Go command as make install. No unpublished
// release URLs or platform downloads are required for a local npm installation.
const root = path.resolve(__dirname, '..');
const { version } = require('../package.json');
const binary = path.join(root, 'bin', process.platform === 'win32' ? 'elevenlabs.exe' : 'elevenlabs');
mkdirSync(path.dirname(binary), { recursive: true });
const bundle = spawnSync('go', ['run', './cmd/bundle-galleton'], { cwd: root, stdio: 'inherit' });
if (bundle.error || bundle.status !== 0) {
  console.error('elevenlabs: preparing the pinned session engine requires Go 1.25+ and module download access.');
  process.exit(bundle.status || 1);
}
const build = spawnSync('go', ['build', '-tags', 'galleton_bundle', '-trimpath', '-ldflags', `-X main.version=${version}`, '-o', binary, './cmd/elevenlabs'], {
  cwd: root,
  stdio: 'inherit',
});
if (build.error) {
  console.error('elevenlabs: building from source requires Go 1.25 or newer on PATH.');
  process.exit(1);
}
process.exit(build.status ?? 1);
