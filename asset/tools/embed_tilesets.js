'use strict';
// Phaser không hỗ trợ external tileset reference (`.tsj` qua trường "source") — parser của Phaser
// literally in ra "External tilesets unsupported. Use Embed Tileset and re-export" và bỏ qua
// tileset đó (xem node_modules/phaser/src/tilemaps/parsers/tiled/ParseTilesets.js).
//
// Script này đọc asset/Maps/<map>.tmj (bản gốc, giữ external reference để còn mở lại bằng Tiled),
// inline nội dung từng .tsj vào thẳng mảng `tilesets`, sửa `image` thành đường dẫn tương đối tới
// frontend/public/assets/tiles/, rồi ghi bản đã embed vào frontend/public/assets/maps/<map>.tmj.
//
// Cách dùng:
//   node asset/tools/embed_tilesets.js                          # mặc định village_adventure
//   node asset/tools/embed_tilesets.js my_new_map               # xử lý asset/Maps/my_new_map.tmj

const fs = require('fs');
const path = require('path');

const mapName = process.argv[2] || 'village_adventure';

const MAP_SOURCE_PATH = path.join(__dirname, '..', 'Maps', `${mapName}.tmj`);
const MAP_SOURCE_DIR = path.dirname(MAP_SOURCE_PATH);
const OUT_PATH = path.join(__dirname, '..', '..', 'frontend', 'public', 'assets', 'maps', `${mapName}.tmj`);

if (!fs.existsSync(MAP_SOURCE_PATH)) {
  console.error(`File not found: ${MAP_SOURCE_PATH}`);
  process.exit(1);
}

const map = JSON.parse(fs.readFileSync(MAP_SOURCE_PATH, 'utf8'));

function isXml(content) {
  return content.trimStart().startsWith('<?xml') || content.trimStart().startsWith('<tileset');
}

function parseTsx(content) {
  const name = (content.match(/<tileset[^>]*\sname="([^"]*)"/) || [])[1] || '';
  const tilewidth = parseInt((content.match(/tilewidth="([^"]*)"/) || [])[1]) || 16;
  const tileheight = parseInt((content.match(/tileheight="([^"]*)"/) || [])[1]) || 16;
  const tilecount = parseInt((content.match(/tilecount="([^"]*)"/) || [])[1]) || 0;
  let columns = parseInt((content.match(/columns="([^"]*)"/) || [])[1]) || 0;
  const imageSource = (content.match(/<image[^>]*\ssource="([^"]*)"/) || [])[1] || '';
  const imagewidth = parseInt((content.match(/<image[^>]*\swidth="([^"]*)"/) || [])[1]) || 0;
  const imageheight = parseInt((content.match(/<image[^>]*\sheight="([^"]*)"/) || [])[1]) || 0;

  return { columns, image: imageSource, imageheight, imagewidth, name, tilecount, tileheight, tilewidth, margin: 0, spacing: 0 };
}

function normalizeTileset(tsj) {
  let columns = tsj.columns || 0;
  if (columns === 0 && tsj.imagewidth > 0 && tsj.tilewidth > 0) {
    columns = Math.floor(tsj.imagewidth / tsj.tilewidth);
  }
  let tilecount = tsj.tilecount || 0;
  if (tilecount === 0 && columns > 0 && tsj.imageheight > 0 && tsj.tileheight > 0) {
    tilecount = columns * Math.floor(tsj.imageheight / tsj.tileheight);
  }
  return { ...tsj, columns, tilecount };
}

map.tilesets = map.tilesets.map((entry) => {
  if (!entry.source) return entry;

  const tsjPath = path.resolve(MAP_SOURCE_DIR, entry.source);
  const content = fs.readFileSync(tsjPath, 'utf8');

  let tsj;
  if (isXml(content)) {
    tsj = normalizeTileset(parseTsx(content));
  } else {
    tsj = normalizeTileset(JSON.parse(content));
  }

  return {
    firstgid: entry.firstgid,
    columns: tsj.columns,
    image: `../tiles/${path.basename(tsj.image)}`,
    imageheight: tsj.imageheight,
    imagewidth: tsj.imagewidth,
    margin: tsj.margin || 0,
    name: tsj.name,
    spacing: tsj.spacing || 0,
    tilecount: tsj.tilecount,
    tileheight: tsj.tileheight,
    tilewidth: tsj.tilewidth,
  };
});

fs.mkdirSync(path.dirname(OUT_PATH), { recursive: true });
fs.writeFileSync(OUT_PATH, JSON.stringify(map, null, 1));
console.log('embedded tilesets written to', OUT_PATH);
