import fs from 'node:fs';
import path from 'node:path';
import crypto from 'node:crypto';

// Read ZIP central directory sizes, including the actual compressed APK contribution.
// APK signing blocks do not change central-directory entries.
function report(file) {
  const buffer = fs.readFileSync(file);
  let end = buffer.length - 22;
  const limit = Math.max(0, buffer.length - 65557);
  while (end >= limit && buffer.readUInt32LE(end) !== 0x06054b50) end--;
  if (end < limit) throw new Error(`Invalid APK ZIP: ${file}`);
  const count = buffer.readUInt16LE(end + 10);
  let offset = buffer.readUInt32LE(end + 16);
  const entries = [];
  for (let index = 0; index < count; index++) {
    if (buffer.readUInt32LE(offset) !== 0x02014b50) throw new Error('Invalid central directory');
    const compressed = buffer.readUInt32LE(offset + 20);
    const uncompressed = buffer.readUInt32LE(offset + 24);
    const nameLength = buffer.readUInt16LE(offset + 28);
    const extraLength = buffer.readUInt16LE(offset + 30);
    const commentLength = buffer.readUInt16LE(offset + 32);
    const name = buffer.toString('utf8', offset + 46, offset + 46 + nameLength);
    entries.push({ name, compressed, uncompressed });
    offset += 46 + nameLength + extraLength + commentLength;
  }
  const groups = {};
  for (const entry of entries) {
    const group = entry.name.startsWith('lib/') ? entry.name.split('/').slice(0, 2).join('/')
      : /kernel_blob|isolate_snapshot/.test(entry.name) ? 'Dart debug data'
      : /\.dex$/.test(entry.name) ? 'Java and Kotlin bytecode'
      : /\.(?:ttf|otf)$/.test(entry.name) ? 'Fonts'
      : 'Other resources';
    groups[group] ??= { compressed: 0, uncompressed: 0 };
    groups[group].compressed += entry.compressed;
    groups[group].uncompressed += entry.uncompressed;
  }
  return {
    file: path.resolve(file), bytes: buffer.length,
    megabytes: Number((buffer.length / 1_000_000).toFixed(2)),
    mebibytes: Number((buffer.length / 1024 ** 2).toFixed(2)),
    sha256: crypto.createHash('sha256').update(buffer).digest('hex'),
    architectures: [...new Set(entries.filter(e => e.name.startsWith('lib/')).map(e => e.name.split('/')[1]))],
    hasDebugKernel: entries.some(e => e.name.endsWith('kernel_blob.bin')),
    hasTestSQLite: entries.some(e => e.name.endsWith('/libsqlite3.so')),
    groups,
    largestEntries: entries.sort((a, b) => b.compressed - a.compressed).slice(0, 15),
  };
}

if (process.argv.length < 3) throw new Error('Usage: node scripts/apk-size-report.mjs path/to/app.apk [...]');
console.log(JSON.stringify(process.argv.slice(2).map(report), null, 2));
