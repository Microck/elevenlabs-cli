import assert from 'node:assert/strict';
import { readFile, readdir, stat } from 'node:fs/promises';
import { createServer } from 'node:http';
import { join, resolve } from 'node:path';
import { staticClient } from 'fumadocs-core/search/client/orama-static';

const output = resolve('out');
const htmlFiles = [];
async function collect(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) await collect(path);
    else if (entry.name.endsWith('.html')) htmlFiles.push(path);
  }
}
await collect(output);
assert(htmlFiles.length >= 12, 'Expected the home page and documentation pages');
for (const path of htmlFiles) {
  const html = await readFile(path, 'utf8');
  for (const match of html.matchAll(/(?:href|src)="(\/(?!\/)[^"?#]*)(?:[?#][^"]*)?"/g)) {
    assert(match[1].startsWith('/elevenlabs-web-cli/'), `Unprefixed URL: ${match[1]}`);
    const target = join(output, decodeURIComponent(match[1].slice('/elevenlabs-web-cli'.length))); 
    const info = await stat(target);
    if (info.isDirectory()) await stat(join(target, 'index.html'));
  }
}
const index = await readFile(join(output, 'api/search'));
assert((await readFile(join(output, 'llms.txt'), 'utf8')).includes('/elevenlabs-web-cli/docs/auth'));
const server = createServer((_request, response) => {
  response.setHeader('Content-Type', 'application/json');
  response.end(index);
});
await new Promise((done) => server.listen(0, '0.0.0.0', done));
try {
  const address = server.address();
  const client = staticClient({ from: `http://127.0.0.1:${address.port}/api/search` });
  for (const query of ['install', 'authentication', 'dialogue', 'profile']) {
    const results = await client.search(query);
    assert(results.length > 0, `No search results for ${query}`);
    assert(results.some((result) => result.url.startsWith('/docs/')));
    console.log(`${query}: ${results.length} results`);
  }
} finally {
  await new Promise((done, reject) => server.close((error) => error ? reject(error) : done()));
}
console.log(`Verified ${htmlFiles.length} HTML pages, local links, search and llms.txt`);
