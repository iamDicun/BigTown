'use strict';
// Generates a decorative Cute Fantasy "diorama" map used only as the login/register page
// background (never loaded by the game's Phaser runtime, so no Collision/SpawnPoints/NPCSpawns
// layers and no embed_tilesets.js step — render_tilemap_png.js reads the external .tsj refs
// directly and rasterizes this straight to a PNG).
//
// Tile catalog/gid math and the Water_Tile / Fences / Oak_Tree / Oak_Tree_Small /
// Outdoor_Decor_Free layouts below are the same confirmed layouts used by generate_map.js.

const fs = require('fs');
const path = require('path');

const MAP_W = 64;
const MAP_H = 36;
const TS = 16;

const TILESETS = [
  { name: 'Grass_Middle', source: '../Tiles/Grass_Middle.tsj', firstgid: 1, tilecount: 1 },
  { name: 'Path_Middle', source: '../Tiles/Path_Middle.tsj', firstgid: 2, tilecount: 1 },
  { name: 'Water_Middle', source: '../Tiles/Water_Middle.tsj', firstgid: 3, tilecount: 1 },
  { name: 'Water_Tile', source: '../Tiles/Water_Tile.tsj', firstgid: 4, tilecount: 18 },
  { name: 'House_1_Wood_Base_Blue', source: '../Tiles/House_1_Wood_Base_Blue.tsj', firstgid: 22, tilecount: 48 },
  { name: 'Oak_Tree', source: '../Tiles/Oak_Tree.tsj', firstgid: 70, tilecount: 20 },
  { name: 'Oak_Tree_Small', source: '../Outdoor decoration/Oak_Tree_Small.tsj', firstgid: 90, tilecount: 18 },
  { name: 'Fences', source: '../Outdoor decoration/Fences.tsj', firstgid: 108, tilecount: 16 },
  { name: 'Chest', source: '../Outdoor decoration/Chest.tsj', firstgid: 124, tilecount: 1 },
  { name: 'Outdoor_Decor_Free', source: '../Outdoor decoration/Outdoor_Decor_Free.tsj', firstgid: 125, tilecount: 84 },
];

function gidFor(name, localIndex) {
  const ts = TILESETS.find((t) => t.name === name);
  if (!ts) throw new Error('unknown tileset ' + name);
  if (localIndex < 0 || localIndex >= ts.tilecount) throw new Error(name + ' index out of range: ' + localIndex);
  return ts.firstgid + localIndex;
}

const GID_GRASS = gidFor('Grass_Middle', 0);
const GID_PATH = gidFor('Path_Middle', 0);
const GID_WATER_FILL = gidFor('Water_Middle', 0);

const WATER = {
  TL: gidFor('Water_Tile', 0), T: gidFor('Water_Tile', 1), TR: gidFor('Water_Tile', 2),
  L: gidFor('Water_Tile', 3), C: gidFor('Water_Tile', 4), R: gidFor('Water_Tile', 5),
  BL: gidFor('Water_Tile', 6), B: gidFor('Water_Tile', 7), BR: gidFor('Water_Tile', 8),
};

const FENCE = {
  POST: gidFor('Fences', 0),
  H_LEFT: gidFor('Fences', 1),
  H_MID: gidFor('Fences', 2),
  H_RIGHT: gidFor('Fences', 3),
};

const CHEST_GID = gidFor('Chest', 0);

const DECOR = {
  GRASS_SPRIG_1: gidFor('Outdoor_Decor_Free', 0),
  GRASS_SPRIG_2: gidFor('Outdoor_Decor_Free', 1),
  GRASS_SPRIG_3: gidFor('Outdoor_Decor_Free', 2),
  WHITE_FLOWER_1: gidFor('Outdoor_Decor_Free', 7),
  WHITE_FLOWER_2: gidFor('Outdoor_Decor_Free', 8),
  STUMP: gidFor('Outdoor_Decor_Free', 14),
  ROCK_SMALL: gidFor('Outdoor_Decor_Free', 15),
  ROCK_CLUSTER: gidFor('Outdoor_Decor_Free', 16),
  ROCK_CLUSTER_2: gidFor('Outdoor_Decor_Free', 21),
  ROCK_CLUSTER_3: gidFor('Outdoor_Decor_Free', 22),
  LOG: gidFor('Outdoor_Decor_Free', 49),
  LAMP_TOP: gidFor('Outdoor_Decor_Free', 32),
  LAMP_MID: gidFor('Outdoor_Decor_Free', 39),
  LAMP_BASE: gidFor('Outdoor_Decor_Free', 46),
  TULIP_RED: gidFor('Outdoor_Decor_Free', 56),
  TULIP_YELLOW: gidFor('Outdoor_Decor_Free', 57),
  POT_RED: gidFor('Outdoor_Decor_Free', 58),
  POT_YELLOW: gidFor('Outdoor_Decor_Free', 59),
  BLOOM_RED: gidFor('Outdoor_Decor_Free', 63),
  BLOOM_YELLOW: gidFor('Outdoor_Decor_Free', 64),
};

const HOUSE_W = 6, HOUSE_H = 8;
function houseGid(localIdx) { return gidFor('House_1_Wood_Base_Blue', localIdx); }

const OAKTREE_W = 4, OAKTREE_H = 5;
function oakTreeGid(localIdx) { return gidFor('Oak_Tree', localIdx); }

function oakSmallGid(col, row) { return gidFor('Oak_Tree_Small', row * 6 + col); }

function blankGrid(w, h) {
  return Array.from({ length: h }, () => new Array(w).fill(0));
}
function fillRect(grid, x0, y0, w, h, gid) {
  for (let y = y0; y < y0 + h; y++) for (let x = x0; x < x0 + w; x++) grid[y][x] = gid;
}
function setTile(grid, x, y, gid) { grid[y][x] = gid; }

const ground = blankGrid(MAP_W, MAP_H);
const decoBelow = blankGrid(MAP_W, MAP_H);
const objects = blankGrid(MAP_W, MAP_H);
const decoAbove = blankGrid(MAP_W, MAP_H);

// ---------------------------------------------------------------------
// 1. Ground: base grass fill
// ---------------------------------------------------------------------
fillRect(ground, 0, 0, MAP_W, MAP_H, GID_GRASS);

// ---------------------------------------------------------------------
// 2. Ground: pond, top-left corner
// ---------------------------------------------------------------------
const POND = { x: 3, y: 4, w: 11, h: 9 };
fillRect(ground, POND.x + 1, POND.y + 1, POND.w - 2, POND.h - 2, GID_WATER_FILL);
setTile(ground, POND.x, POND.y, WATER.TL);
setTile(ground, POND.x + POND.w - 1, POND.y, WATER.TR);
setTile(ground, POND.x, POND.y + POND.h - 1, WATER.BL);
setTile(ground, POND.x + POND.w - 1, POND.y + POND.h - 1, WATER.BR);
for (let x = POND.x + 1; x < POND.x + POND.w - 1; x++) {
  setTile(ground, x, POND.y, WATER.T);
  setTile(ground, x, POND.y + POND.h - 1, WATER.B);
}
for (let y = POND.y + 1; y < POND.y + POND.h - 1; y++) {
  setTile(ground, POND.x, y, WATER.L);
  setTile(ground, POND.x + POND.w - 1, y, WATER.R);
}

// ---------------------------------------------------------------------
// 3. Ground: path — vertical gate path down from the cottage door + a
//    horizontal road crossing the lower third (village_adventure T-junction vibe)
// ---------------------------------------------------------------------
const HOUSE = { x: 44, y: 6 }; // 6x8 stamp -> x44-49, y6-13
const DOOR_X = HOUSE.x + 2; // global x46
const GATE_X0 = DOOR_X - 1, GATE_X1 = DOOR_X + 1; // 3-tile wide

fillRect(ground, GATE_X0, HOUSE.y + HOUSE_H, GATE_X1 - GATE_X0 + 1, MAP_H - (HOUSE.y + HOUSE_H), GID_PATH);
fillRect(ground, 9, 29, 50 - 9 + 1, 2, GID_PATH);

// ---------------------------------------------------------------------
// 4. Objects: cottage (6x8 stamp)
// ---------------------------------------------------------------------
for (let row = 0; row < HOUSE_H; row++) {
  for (let col = 0; col < HOUSE_W; col++) {
    setTile(objects, HOUSE.x + col, HOUSE.y + row, houseGid(row * HOUSE_W + col));
  }
}

// ---------------------------------------------------------------------
// 5. Trees: big Oak_Tree + Oak_Tree_Small clusters top-right / bottom-right / near pond
// ---------------------------------------------------------------------
function placeBigOakTree(originX, originY) {
  for (let row = 0; row < OAKTREE_H; row++) {
    for (let col = 0; col < OAKTREE_W; col++) {
      const gid = oakTreeGid(row * OAKTREE_W + col);
      const gx = originX + col, gy = originY + row;
      if (row <= 2) setTile(decoAbove, gx, gy, gid);
      else setTile(objects, gx, gy, gid);
    }
  }
}
function placeSmallOakTree(originX, originY, variant) {
  const colOffset = variant === 'A' ? 2 : 4;
  for (let row = 0; row < 3; row++) {
    for (let col = 0; col < 2; col++) {
      const gid = oakSmallGid(colOffset + col, row);
      const gx = originX + col, gy = originY + row;
      if (row <= 1) setTile(decoAbove, gx, gy, gid);
      else setTile(objects, gx, gy, gid);
    }
  }
}

placeBigOakTree(56, 1);        // top-right hill
placeBigOakTree(1, 19);        // left side, below pond
placeBigOakTree(55, 24);       // bottom-right corner
placeSmallOakTree(15, 5, 'A'); // pond edge
placeSmallOakTree(50, 8, 'B'); // beside cottage
placeSmallOakTree(6, 26, 'A'); // near garden
placeSmallOakTree(60, 28, 'B');// bottom-right cluster

// ---------------------------------------------------------------------
// 6. Rocks / stump / log / lamp / chest
// ---------------------------------------------------------------------
setTile(objects, 60, 4, DECOR.ROCK_CLUSTER);
setTile(objects, 62, 2, DECOR.ROCK_CLUSTER_2);
setTile(objects, 61, 6, DECOR.ROCK_SMALL);
setTile(objects, 59, 32, DECOR.ROCK_CLUSTER_3);
setTile(objects, 62, 31, DECOR.ROCK_CLUSTER);
setTile(objects, 61, 34, DECOR.ROCK_SMALL);
setTile(objects, 5, 17, DECOR.STUMP);
setTile(objects, 6, 18, DECOR.LOG);
setTile(objects, 51, 13, CHEST_GID);

setTile(objects, 43, 11, DECOR.LAMP_TOP);
setTile(objects, 43, 12, DECOR.LAMP_MID);
setTile(objects, 43, 13, DECOR.LAMP_BASE);

// ---------------------------------------------------------------------
// 7. Garden fence run, bottom-left, with a small flower bed behind it
// ---------------------------------------------------------------------
const FENCE_ROW = 33;
function placeFenceRun(x0, x1, row) {
  for (let x = x0; x <= x1; x++) {
    let gid;
    if (x === x0) gid = FENCE.H_LEFT;
    else if (x === x1) gid = FENCE.H_RIGHT;
    else gid = FENCE.H_MID;
    setTile(objects, x, row, gid);
  }
}
placeFenceRun(4, 13, FENCE_ROW);
setTile(objects, 3, FENCE_ROW, FENCE.POST);
setTile(objects, 14, FENCE_ROW, FENCE.POST);

const gardenBed = [
  [5, 31, DECOR.TULIP_RED], [5, 32, DECOR.POT_RED],
  [7, 31, DECOR.BLOOM_RED], [9, 31, DECOR.TULIP_YELLOW],
  [9, 32, DECOR.POT_YELLOW], [11, 31, DECOR.BLOOM_YELLOW],
  [12, 32, DECOR.TULIP_RED],
];
for (const [x, y, gid] of gardenBed) setTile(decoBelow, x, y, gid);

// ---------------------------------------------------------------------
// 8. Scattered grass sprigs / wildflowers for texture
// ---------------------------------------------------------------------
const scatter = [
  [18, 3, DECOR.GRASS_SPRIG_1], [22, 8, DECOR.WHITE_FLOWER_1], [27, 4, DECOR.GRASS_SPRIG_2],
  [33, 10, DECOR.WHITE_FLOWER_2], [38, 3, DECOR.GRASS_SPRIG_3], [17, 15, DECOR.GRASS_SPRIG_2],
  [24, 19, DECOR.WHITE_FLOWER_1], [30, 15, DECOR.GRASS_SPRIG_1], [37, 20, DECOR.GRASS_SPRIG_3],
  [42, 22, DECOR.WHITE_FLOWER_2], [16, 24, DECOR.GRASS_SPRIG_3], [20, 28, DECOR.GRASS_SPRIG_1],
  [35, 27, DECOR.WHITE_FLOWER_1], [52, 20, DECOR.GRASS_SPRIG_2], [55, 15, DECOR.WHITE_FLOWER_2],
  [58, 12, DECOR.GRASS_SPRIG_1], [24, 24, DECOR.GRASS_SPRIG_2], [40, 27, DECOR.GRASS_SPRIG_3],
  [8, 8, DECOR.WHITE_FLOWER_1], [2, 24, DECOR.GRASS_SPRIG_1], [53, 27, DECOR.GRASS_SPRIG_3],
  [46, 17, DECOR.WHITE_FLOWER_2], [49, 22, DECOR.GRASS_SPRIG_1],
];
for (const [x, y, gid] of scatter) setTile(decoBelow, x, y, gid);

// ---------------------------------------------------------------------
// Assemble the .tmj (tile layers only — decorative-only, never loaded by Phaser)
// ---------------------------------------------------------------------
function tileLayer(name, grid, id) {
  const data = [];
  for (let y = 0; y < MAP_H; y++) for (let x = 0; x < MAP_W; x++) data.push(grid[y][x]);
  return { id, name, type: 'tilelayer', width: MAP_W, height: MAP_H, x: 0, y: 0, opacity: 1, visible: true, data };
}

const map = {
  compressionlevel: -1,
  width: MAP_W,
  height: MAP_H,
  tilewidth: TS,
  tileheight: TS,
  infinite: false,
  orientation: 'orthogonal',
  renderorder: 'right-down',
  type: 'map',
  tiledversion: '1.12.2',
  version: '1.10',
  nextlayerid: 5,
  nextobjectid: 1,
  layers: [
    tileLayer('Ground', ground, 1),
    tileLayer('DecorationBelow', decoBelow, 2),
    tileLayer('Objects', objects, 3),
    tileLayer('DecorationAbove', decoAbove, 4),
  ],
  tilesets: TILESETS.map((t) => ({ firstgid: t.firstgid, source: t.source })),
};

const outPath = path.join(__dirname, '..', 'Maps', 'login_background.tmj');
fs.writeFileSync(outPath, JSON.stringify(map, null, 1));
console.log('written', outPath);
