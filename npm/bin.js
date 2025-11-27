#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');

const platform = process.platform === 'win32' ? 'Windows' : process.platform === 'darwin' ? 'Darwin' : 'Linux';
const binaryName = platform === 'Windows' ? 'codex-mcp-go.exe' : 'codex-mcp-go';
const binaryPath = path.join(__dirname, binaryName);

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit'
});

child.on('exit', (code) => {
  process.exit(code);
});