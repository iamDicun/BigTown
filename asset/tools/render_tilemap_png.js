'use strict';
// Rasterizes a Tiled .tmj map (tile layers only, external .tsj tileset refs) into a flat RGBA PNG.
// No image dependency beyond Node's built-in zlib — decodes/encodes PNG chunks by hand since all
// source tile sheets in asset/ are 8-bit non-interlaced colorType 2 (RGB) or 6 (RGBA).
//
// Usage: node render_tilemap_png.js <input.tmj> <output.png>

const fs = require('fs');
const path = require('path');
const zlib = require('zlib');

const PNG_SIGNATURE = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);

function readChunks(buf) {
  if (!buf.subarray(0, 8).equals(PNG_SIGNATURE)) throw new Error('not a PNG file');
  const chunks = [];
  let offset = 8;
  while (offset < buf.length) {
    const length = buf.readUInt32BE(offset);
    const type = buf.toString('ascii', offset + 4, offset + 8);
    const data = buf.subarray(offset + 8, offset + 8 + length);
    chunks.push({ type, data });
    offset += 12 + length; // length + type + data + crc
  }
  return chunks;
}

function decodePNG(buf) {
  const chunks = readChunks(buf);
  const ihdrChunk = chunks.find((c) => c.type === 'IHDR');
  if (!ihdrChunk) throw new Error('missing IHDR');
  const width = ihdrChunk.data.readUInt32BE(0);
  const height = ihdrChunk.data.readUInt32BE(4);
  const bitDepth = ihdrChunk.data.readUInt8(8);
  const colorType = ihdrChunk.data.readUInt8(9);
  const interlace = ihdrChunk.data.readUInt8(12);
  if (bitDepth !== 8) throw new Error('only 8-bit PNGs supported, got bitDepth=' + bitDepth);
  if (interlace !== 0) throw new Error('interlaced PNGs not supported');

  const channelsByColorType = { 0: 1, 2: 3, 4: 2, 6: 4 };
  const channels = channelsByColorType[colorType];
  if (!channels) throw new Error('unsupported colorType ' + colorType + ' (palette PNGs not supported)');

  const idat = Buffer.concat(chunks.filter((c) => c.type === 'IDAT').map((c) => c.data));
  const raw = zlib.inflateSync(idat);

  const bpp = channels; // bitDepth 8 => 1 byte per channel
  const rowBytes = width * channels;
  const out = Buffer.alloc(rowBytes * height);
  let inPos = 0;

  function prevByte(row, x, c) {
    return x < 0 ? 0 : out[(row) * rowBytes + x * channels + c];
  }

  for (let y = 0; y < height; y++) {
    const filterType = raw[inPos];
    inPos += 1;
    for (let x = 0; x < width; x++) {
      for (let c = 0; c < channels; c++) {
        const rawByte = raw[inPos];
        inPos += 1;
        const a = x > 0 ? out[y * rowBytes + (x - 1) * channels + c] : 0;
        const b = y > 0 ? out[(y - 1) * rowBytes + x * channels + c] : 0;
        const cc = x > 0 && y > 0 ? out[(y - 1) * rowBytes + (x - 1) * channels + c] : 0;
        let value;
        switch (filterType) {
          case 0: value = rawByte; break;
          case 1: value = rawByte + a; break;
          case 2: value = rawByte + b; break;
          case 3: value = rawByte + Math.floor((a + b) / 2); break;
          case 4: {
            const p = a + b - cc;
            const pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - cc);
            const pred = pa <= pb && pa <= pc ? a : pb <= pc ? b : cc;
            value = rawByte + pred;
            break;
          }
          default: throw new Error('unsupported filter type ' + filterType);
        }
        out[y * rowBytes + x * channels + c] = value & 0xff;
      }
    }
  }

  // Normalize to RGBA
  const rgba = Buffer.alloc(width * height * 4);
  for (let i = 0; i < width * height; i++) {
    if (channels === 4) {
      out.copy(rgba, i * 4, i * 4, i * 4 + 4);
    } else if (channels === 3) {
      rgba[i * 4] = out[i * 3];
      rgba[i * 4 + 1] = out[i * 3 + 1];
      rgba[i * 4 + 2] = out[i * 3 + 2];
      rgba[i * 4 + 3] = 255;
    } else if (channels === 1) {
      rgba[i * 4] = out[i];
      rgba[i * 4 + 1] = out[i];
      rgba[i * 4 + 2] = out[i];
      rgba[i * 4 + 3] = 255;
    }
  }

  return { width, height, data: rgba };
}

function crc32Of(typeAndData) {
  return zlib.crc32(typeAndData);
}

function makeChunk(type, data) {
  const typeBuf = Buffer.from(type, 'ascii');
  const lenBuf = Buffer.alloc(4);
  lenBuf.writeUInt32BE(data.length, 0);
  const crcBuf = Buffer.alloc(4);
  crcBuf.writeUInt32BE(crc32Of(Buffer.concat([typeBuf, data])), 0);
  return Buffer.concat([lenBuf, typeBuf, data, crcBuf]);
}

const BPP = 4; // RGBA, 8-bit

function filterRow(type, cur, prev, rowBytes) {
  const out = Buffer.alloc(rowBytes);
  for (let x = 0; x < rowBytes; x++) {
    const a = x >= BPP ? cur[x - BPP] : 0;
    const b = prev ? prev[x] : 0;
    const c = x >= BPP && prev ? prev[x - BPP] : 0;
    let value;
    switch (type) {
      case 0: value = cur[x]; break;
      case 1: value = cur[x] - a; break;
      case 2: value = cur[x] - b; break;
      case 3: value = cur[x] - Math.floor((a + b) / 2); break;
      case 4: {
        const p = a + b - c;
        const pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c);
        const pred = pa <= pb && pa <= pc ? a : pb <= pc ? b : c;
        value = cur[x] - pred;
        break;
      }
      default: throw new Error('bad filter type');
    }
    out[x] = value & 0xff;
  }
  return out;
}

function sumOfAbsSigned(buf) {
  let sum = 0;
  for (let i = 0; i < buf.length; i++) {
    const v = buf[i] < 128 ? buf[i] : buf[i] - 256;
    sum += Math.abs(v);
  }
  return sum;
}

function buildRawNone(data, width, height) {
  const rowBytes = width * 4;
  const raw = Buffer.alloc((rowBytes + 1) * height);
  for (let y = 0; y < height; y++) {
    raw[y * (rowBytes + 1)] = 0;
    data.copy(raw, y * (rowBytes + 1) + 1, y * rowBytes, y * rowBytes + rowBytes);
  }
  return raw;
}

function buildRawAdaptive(data, width, height) {
  const rowBytes = width * 4;
  const raw = Buffer.alloc((rowBytes + 1) * height);
  let prevRow = null;
  for (let y = 0; y < height; y++) {
    const curRow = data.subarray(y * rowBytes, y * rowBytes + rowBytes);
    let bestType = 0;
    let bestFiltered = filterRow(0, curRow, prevRow, rowBytes);
    let bestScore = sumOfAbsSigned(bestFiltered);
    for (let type = 1; type <= 4; type++) {
      const filtered = filterRow(type, curRow, prevRow, rowBytes);
      const score = sumOfAbsSigned(filtered);
      if (score < bestScore) {
        bestScore = score;
        bestType = type;
        bestFiltered = filtered;
      }
    }
    raw[y * (rowBytes + 1)] = bestType;
    bestFiltered.copy(raw, y * (rowBytes + 1) + 1);
    prevRow = curRow;
  }
  return raw;
}

function encodePNG({ width, height, data }) {
  // Flat pixel-art (large runs of identical bytes) often compresses better with no
  // per-row filtering at all, while smooth/gradient art benefits from adaptive filtering.
  // Try both, keep whichever deflates smaller.
  const idatNone = zlib.deflateSync(buildRawNone(data, width, height), { level: 9 });
  const idatAdaptive = zlib.deflateSync(buildRawAdaptive(data, width, height), { level: 9 });
  const idatData = idatAdaptive.length < idatNone.length ? idatAdaptive : idatNone;

  const ihdrData = Buffer.alloc(13);
  ihdrData.writeUInt32BE(width, 0);
  ihdrData.writeUInt32BE(height, 4);
  ihdrData.writeUInt8(8, 8); // bit depth
  ihdrData.writeUInt8(6, 9); // color type RGBA
  ihdrData.writeUInt8(0, 10); // compression
  ihdrData.writeUInt8(0, 11); // filter
  ihdrData.writeUInt8(0, 12); // interlace

  return Buffer.concat([
    PNG_SIGNATURE,
    makeChunk('IHDR', ihdrData),
    makeChunk('IDAT', idatData),
    makeChunk('IEND', Buffer.alloc(0)),
  ]);
}

function resolveTilesets(tmj, tmjDir) {
  return tmj.tilesets.map((entry) => {
    if (entry.source) {
      const tsjPath = path.resolve(tmjDir, entry.source);
      const tsjDir = path.dirname(tsjPath);
      const tsj = JSON.parse(fs.readFileSync(tsjPath, 'utf8'));
      return {
        firstgid: entry.firstgid,
        tilecount: tsj.tilecount,
        columns: tsj.columns,
        margin: tsj.margin || 0,
        spacing: tsj.spacing || 0,
        tilewidth: tsj.tilewidth,
        tileheight: tsj.tileheight,
        imagePath: path.resolve(tsjDir, tsj.image),
      };
    }
    // embedded tileset
    return {
      firstgid: entry.firstgid,
      tilecount: entry.tilecount,
      columns: entry.columns,
      margin: entry.margin || 0,
      spacing: entry.spacing || 0,
      tilewidth: entry.tilewidth,
      tileheight: entry.tileheight,
      imagePath: path.resolve(tmjDir, entry.image),
    };
  });
}

function findTileset(tilesets, gid) {
  let best = null;
  for (const ts of tilesets) {
    if (ts.firstgid <= gid && (!best || ts.firstgid > best.firstgid)) best = ts;
  }
  return best;
}

function blitOver(dst, dstW, dstH, dstX, dstY, src, srcW, srcH, srcX, srcY, tileW, tileH) {
  for (let y = 0; y < tileH; y++) {
    const py = dstY + y;
    if (py < 0 || py >= dstH) continue;
    for (let x = 0; x < tileW; x++) {
      const px = dstX + x;
      if (px < 0 || px >= dstW) continue;
      const sIdx = ((srcY + y) * srcW + (srcX + x)) * 4;
      const dIdx = (py * dstW + px) * 4;
      const srcA = src[sIdx + 3] / 255;
      if (srcA <= 0) continue;
      if (srcA >= 1) {
        dst[dIdx] = src[sIdx];
        dst[dIdx + 1] = src[sIdx + 1];
        dst[dIdx + 2] = src[sIdx + 2];
        dst[dIdx + 3] = 255;
        continue;
      }
      const dstA = dst[dIdx + 3] / 255;
      const outA = srcA + dstA * (1 - srcA);
      for (let c = 0; c < 3; c++) {
        const s = src[sIdx + c];
        const d = dst[dIdx + c];
        dst[dIdx + c] = outA > 0 ? Math.round((s * srcA + d * dstA * (1 - srcA)) / outA) : 0;
      }
      dst[dIdx + 3] = Math.round(outA * 255);
    }
  }
}

function renderTmjToPng(tmjPath, outPath) {
  const tmjDir = path.dirname(tmjPath);
  const tmj = JSON.parse(fs.readFileSync(tmjPath, 'utf8'));
  const { tilewidth: tw, tileheight: th, width: mapW, height: mapH } = tmj;

  const tilesets = resolveTilesets(tmj, tmjDir);
  const imageCache = new Map();
  function loadImage(imgPath) {
    if (!imageCache.has(imgPath)) imageCache.set(imgPath, decodePNG(fs.readFileSync(imgPath)));
    return imageCache.get(imgPath);
  }

  const canvasW = mapW * tw;
  const canvasH = mapH * th;
  const canvas = Buffer.alloc(canvasW * canvasH * 4); // transparent

  const GID_FLIP_MASK = 0xe0000000;

  for (const layer of tmj.layers) {
    if (layer.type !== 'tilelayer') continue;
    for (let y = 0; y < layer.height; y++) {
      for (let x = 0; x < layer.width; x++) {
        const rawGid = layer.data[y * layer.width + x];
        const gid = rawGid & ~GID_FLIP_MASK;
        if (gid === 0) continue;
        const ts = findTileset(tilesets, gid);
        if (!ts) throw new Error('no tileset found for gid ' + gid);
        const localId = gid - ts.firstgid;
        const col = localId % ts.columns;
        const row = Math.floor(localId / ts.columns);
        const srcX = ts.margin + col * (ts.tilewidth + ts.spacing);
        const srcY = ts.margin + row * (ts.tileheight + ts.spacing);
        const img = loadImage(ts.imagePath);
        blitOver(canvas, canvasW, canvasH, x * tw, y * th, img.data, img.width, img.height, srcX, srcY, ts.tilewidth, ts.tileheight);
      }
    }
  }

  const png = encodePNG({ width: canvasW, height: canvasH, data: canvas });
  fs.mkdirSync(path.dirname(outPath), { recursive: true });
  fs.writeFileSync(outPath, png);
  console.log('rendered', outPath, canvasW + 'x' + canvasH);
}

if (require.main === module) {
  const [, , inArg, outArg] = process.argv;
  if (!inArg || !outArg) {
    console.error('Usage: node render_tilemap_png.js <input.tmj> <output.png>');
    process.exit(1);
  }
  renderTmjToPng(path.resolve(inArg), path.resolve(outArg));
}

module.exports = { renderTmjToPng, decodePNG, encodePNG };
