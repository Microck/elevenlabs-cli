#!/usr/bin/env node
'use strict';

const { spawn } = require('node:child_process');
const path = require('node:path');

const binary = path.resolve(__dirname, '..', 'bin', process.platform === 'win32' ? 'elevenlabs.exe' : 'elevenlabs');
const child = spawn(binary, process.argv.slice(2), { stdio: 'inherit' });
child.on('error', () => {
  console.error('elevenlabs: executable unavailable. Reinstall the package with Go 1.25 or newer on PATH.');
  process.exitCode = 1;
});
// Forward interruption to the Go process so it can clean up partial downloads.
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => child.kill(signal));
}
child.on('exit', (code, signal) => {
  process.exitCode = code ?? (signal === 'SIGINT' ? 130 : 1);
});
