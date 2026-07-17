'use strict';
// Phaser không hỗ trợ external tileset reference (`.tsj` qua trường "source") — parser của Phaser
// literally in ra "External tilesets unsupported. Use Embed Tileset and re-export" và bỏ qua
// tileset đó (xem node_modules/phaser/src/tilemaps/parsers/tiled/ParseTilesets.js).
//
// Script này đọc asset/Maps/village_adventure.tmj (bản gốc, giữ external reference để còn mở lại
// bằng Tiled), inline nội dung từng .tsj vào thẳng mảng `tilesets`, sửa `image` thành đường dẫn
// tương đối tới frontend/public/assets/tiles/, rồi ghi bản đã embed vào
// frontend/public/assets/maps/village_adventure.tmj — bản duy nhất frontend thực sự load.
//
// Chạy lại script này mỗi khi asset/Maps/village_adventure.tmj đổi (thêm tileset mới, sửa layer...).

const fs = require('fs');
const path = require('path');

const MAP_SOURCE_PATH = path.join(__dirname, '..', 'Maps', 'village_adventure.tmj');
const MAP_SOURCE_DIR = path.dirname(MAP_SOURCE_PATH);
const OUT_PATH = path.join(__dirname, '..', '..', 'frontend', 'public', 'assets', 'maps', 'village_adventure.tmj');

const map = JSON.parse(fs.readFileSync(MAP_SOURCE_PATH, 'utf8'));

map.tilesets = map.tilesets.map((entry) => {
  if (!entry.source) return entry; // đã embed sẵn, giữ nguyên

  const tsjPath = path.resolve(MAP_SOURCE_DIR, entry.source);
  const tsj = JSON.parse(fs.readFileSync(tsjPath, 'utf8'));

  return {
    firstgid: entry.firstgid,
    columns: tsj.columns,
    image: `../tiles/${path.basename(tsj.image)}`,
    imageheight: tsj.imageheight,
    imagewidth: tsj.imagewidth,
    margin: tsj.margin,
    name: tsj.name,
    spacing: tsj.spacing,
    tilecount: tsj.tilecount,
    tileheight: tsj.tileheight,
    tilewidth: tsj.tilewidth,
  };
});

fs.mkdirSync(path.dirname(OUT_PATH), { recursive: true });
fs.writeFileSync(OUT_PATH, JSON.stringify(map, null, 1));
console.log('embedded tilesets written to', OUT_PATH);
