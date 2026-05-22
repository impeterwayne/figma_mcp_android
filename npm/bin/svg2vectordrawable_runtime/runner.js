const fs = require('node:fs');
const path = require('node:path');
const converter = require('./index');

async function readStdin() {
  return new Promise((resolve, reject) => {
    const chunks = [];
    process.stdin.on('data', chunk => chunks.push(chunk));
    process.stdin.on('end', () => resolve(Buffer.concat(chunks).toString('utf8')));
    process.stdin.on('error', reject);
  });
}

(async () => {
  try {
    const raw = await readStdin();
    const payload = JSON.parse(raw || '{}');
    const xml = await converter(payload.svg, payload.options || {});
    if (!xml || typeof xml !== 'string') {
      throw new Error('Conversion did not produce XML');
    }
    process.stdout.write(JSON.stringify({ xml }));
  } catch (error) {
    const message = error && error.message ? error.message : String(error);
    process.stdout.write(JSON.stringify({ error: message }));
    process.exitCode = 1;
  }
})();
