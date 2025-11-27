const fs = require('fs');
const path = require('path');
const axios = require('axios');
const tar = require('tar');
const unzipper = require('unzipper');
const { execSync } = require('child_process');

const packageJson = require('./package.json');
const version = packageJson.version;
const REPO = 'w31r4/codex-mcp-go';

const platformMap = {
  'darwin': 'Darwin',
  'linux': 'Linux',
  'win32': 'Windows'
};

const archMap = {
  'x64': 'x86_64',
  'arm64': 'arm64'
};

const platform = platformMap[process.platform];
const arch = archMap[process.arch];

if (!platform || !arch) {
  console.error(`Unsupported platform: ${process.platform} ${process.arch}`);
  process.exit(1);
}

const extension = platform === 'Windows' ? 'zip' : 'tar.gz';
const binaryName = platform === 'Windows' ? 'codex-mcp-go.exe' : 'codex-mcp-go';
const fileName = `codex-mcp-go_${platform}_${arch}.${extension}`;
const downloadUrl = `https://github.com/${REPO}/releases/download/v${version}/${fileName}`;

console.log(`Downloading ${downloadUrl}...`);

async function download() {
  const writer = fs.createWriteStream(fileName);

  try {
    const response = await axios({
      url: downloadUrl,
      method: 'GET',
      responseType: 'stream'
    });

    response.data.pipe(writer);

    return new Promise((resolve, reject) => {
      writer.on('finish', resolve);
      writer.on('error', reject);
    });
  } catch (error) {
    console.error(`Error downloading binary: ${error.message}`);
    process.exit(1);
  }
}

async function extract() {
  console.log(`Extracting ${fileName}...`);
  
  if (extension === 'zip') {
    fs.createReadStream(fileName)
      .pipe(unzipper.Extract({ path: '.' }))
      .on('close', () => {
        cleanup();
      });
  } else {
    await tar.x({
      file: fileName,
      cwd: '.'
    });
    cleanup();
  }
}

function cleanup() {
  fs.unlinkSync(fileName);
  if (platform !== 'Windows') {
    fs.chmodSync(binaryName, 0o755);
  }
  console.log('Installation complete!');
}

download().then(extract);