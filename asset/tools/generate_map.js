'use strict';
const fs = require('fs');
const path = require('path');

const MAP_W = 50;
const MAP_H = 35;
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
  const ts = TILESETS.find(t => t.name === name);
  if (!ts) throw new Error('unknown tileset ' + name);
  if (localIndex < 0 || localIndex >= ts.tilecount) throw new Error(name + ' index out of range: ' + localIndex);
  return ts.firstgid + localIndex;
}

// Ground fill / fence / decor base tiles
const GID_GRASS = gidFor('Grass_Middle', 0);
const GID_PATH = gidFor('Path_Middle', 0);
const GID_WATER_FILL = gidFor('Water_Middle', 0);

// Water_Tile blob layout (confirmed by visual grid inspection):
// 0 TL corner, 1 top edge, 2 TR corner, 3 left edge, 4 center, 5 right edge, 6 BL corner, 7 bottom edge, 8 BR corner
const WATER = {
  TL: gidFor('Water_Tile', 0), T: gidFor('Water_Tile', 1), TR: gidFor('Water_Tile', 2),
  L: gidFor('Water_Tile', 3), C: gidFor('Water_Tile', 4), R: gidFor('Water_Tile', 5),
  BL: gidFor('Water_Tile', 6), B: gidFor('Water_Tile', 7), BR: gidFor('Water_Tile', 8),
};

// Fences.png confirmed layout (4 cols x 4 rows):
// 0 = lone vertical post, 1 = horizontal fence left end, 2 = horizontal fence mid rail, 3 = horizontal fence right end
const FENCE = {
  POST: gidFor('Fences', 0),
  H_LEFT: gidFor('Fences', 1),
  H_MID: gidFor('Fences', 2),
  H_RIGHT: gidFor('Fences', 3),
};

const CHEST_GID = gidFor('Chest', 0);

// Outdoor_Decor_Free.png confirmed layout (7 cols x 12 rows, row*7+col)
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

// House_1_Wood_Base_Blue.png is a single 6x8 tile stamp (confirmed by grid inspection)
const HOUSE_W = 6, HOUSE_H = 8;
function houseGid(localIdx) { return gidFor('House_1_Wood_Base_Blue', localIdx); }

// Oak_Tree.png is a single 4x5 tile stamp. Canopy = rows 0-2, trunk = rows 3-4 (confirmed by grid inspection)
const OAKTREE_W = 4, OAKTREE_H = 5;
function oakTreeGid(localIdx) { return gidFor('Oak_Tree', localIdx); }

// Oak_Tree_Small.png contains two 2x3 medium-tree stamps side by side.
// Tree A = local cols {2,3}, Tree B = local cols {4,5}. Canopy = rows 0-1, trunk = row 2.
function oakSmallGid(col, row) { return gidFor('Oak_Tree_Small', row * 6 + col); }

function blankGrid(w, h) {
  return Array.from({ length: h }, () => new Array(w).fill(0));
}

const ground = blankGrid(MAP_W, MAP_H);
const decoBelow = blankGrid(MAP_W, MAP_H);
const objects = blankGrid(MAP_W, MAP_H);
const decoAbove = blankGrid(MAP_W, MAP_H);

function fillRect(grid, x0, y0, w, h, gid) {
  for (let y = y0; y < y0 + h; y++) {
    for (let x = x0; x < x0 + w; x++) {
      grid[y][x] = gid;
    }
  }
}
function setTile(grid, x, y, gid) { grid[y][x] = gid; }

// ---------------------------------------------------------------------
// 1. Ground: base grass fill
// ---------------------------------------------------------------------
fillRect(ground, 0, 0, MAP_W, MAP_H, GID_GRASS);

// ---------------------------------------------------------------------
// 2. Ground: pond (left side), Water_Tile blob border + Water_Middle fill
// ---------------------------------------------------------------------
const POND = { x: 3, y: 4, w: 9, h: 10 }; // x:3-11, y:4-13
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
// 3. Ground: road network (min. width 2, here 2-3 tiles)
// ---------------------------------------------------------------------
const HOUSE = { x: 22, y: 12 }; // 6x8 stamp -> occupies x22-27, y12-19
const DOOR_X = HOUSE.x + 2; // local col 2 -> global x24
const GATE_X0 = 23, GATE_X1 = 25; // 3-tile wide entrance path, aligned with the door

fillRect(ground, GATE_X0, 20, GATE_X1 - GATE_X0 + 1, MAP_H - 20, GID_PATH); // vertical: y20..34
fillRect(ground, 9, 20, 42 - 9 + 1, 2, GID_PATH); // horizontal: y20-21, x9..42

// ---------------------------------------------------------------------
// 4. Objects: house (single 6x8 stamp, fully solid)
// ---------------------------------------------------------------------
for (let row = 0; row < HOUSE_H; row++) {
  for (let col = 0; col < HOUSE_W; col++) {
    const gid = houseGid(row * HOUSE_W + col);
    setTile(objects, HOUSE.x + col, HOUSE.y + row, gid);
  }
}

// ---------------------------------------------------------------------
// 5. Trees: big Oak_Tree (hills) + Oak_Tree_Small (hills / pond edge)
// ---------------------------------------------------------------------
function placeBigOakTree(originX, originY) {
  for (let row = 0; row < OAKTREE_H; row++) {
    for (let col = 0; col < OAKTREE_W; col++) {
      const gid = oakTreeGid(row * OAKTREE_W + col);
      const gx = originX + col, gy = originY + row;
      if (row <= 2) setTile(decoAbove, gx, gy, gid); // canopy, rows 0-2
      else setTile(objects, gx, gy, gid); // trunk, rows 3-4
    }
  }
  // trunk collision: local rows 3-4, cols 1-2 (confirmed non-blank tiles)
  return { x: originX + 1, y: originY + 3, w: 2, h: 2 };
}

function placeSmallOakTree(originX, originY, variant) {
  const colOffset = variant === 'A' ? 2 : 4;
  for (let row = 0; row < 3; row++) {
    for (let col = 0; col < 2; col++) {
      const gid = oakSmallGid(colOffset + col, row);
      const gx = originX + col, gy = originY + row;
      if (row <= 1) setTile(decoAbove, gx, gy, gid); // canopy, rows 0-1
      else setTile(objects, gx, gy, gid); // trunk, row 2
    }
  }
  return { x: originX, y: originY + 2, w: 2, h: 1 };
}

const treeCollisions = [];
treeCollisions.push(placeBigOakTree(37, 4));   // hill 1
treeCollisions.push(placeBigOakTree(37, 22));  // hill 2
treeCollisions.push(placeSmallOakTree(43, 5, 'A'));  // hill 1
treeCollisions.push(placeSmallOakTree(43, 23, 'B')); // hill 2
treeCollisions.push(placeSmallOakTree(12, 6, 'A'));  // near pond
treeCollisions.push(placeSmallOakTree(13, 15, 'B')); // near pond, south edge

// ---------------------------------------------------------------------
// 6. Rocks / stump / log / lamp / chest (Objects, each with 1x1 collision)
// ---------------------------------------------------------------------
const propCollisions = [];
function placeProp(x, y, gid, layer) {
  setTile(layer, x, y, gid);
  propCollisions.push({ x, y, w: 1, h: 1 });
}
placeProp(39, 10, DECOR.ROCK_CLUSTER, objects);
placeProp(42, 9, DECOR.ROCK_CLUSTER_2, objects);
placeProp(41, 11, DECOR.ROCK_SMALL, objects);
placeProp(39, 28, DECOR.ROCK_CLUSTER_3, objects);
placeProp(42, 27, DECOR.ROCK_CLUSTER, objects);
placeProp(41, 29, DECOR.ROCK_SMALL, objects);
placeProp(14, 17, DECOR.STUMP, objects);
placeProp(15, 18, DECOR.LOG, objects);
placeProp(28, 19, CHEST_GID, objects);

// lamp: 3-tile vertical stack, collision only on the base tile
setTile(objects, 21, 17, DECOR.LAMP_TOP);
setTile(objects, 21, 18, DECOR.LAMP_MID);
setTile(objects, 21, 19, DECOR.LAMP_BASE);
propCollisions.push({ x: 21, y: 19, w: 1, h: 1 });

// ---------------------------------------------------------------------
// 7. Entrance fence gate (south border), gap at GATE_X0..GATE_X1
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
  return { x: x0, y: row, w: x1 - x0 + 1, h: 1 };
}
const fenceCollisions = [
  placeFenceRun(15, 22, FENCE_ROW),
  placeFenceRun(26, 33, FENCE_ROW),
];
setTile(objects, 14, FENCE_ROW, FENCE.POST);
setTile(objects, 34, FENCE_ROW, FENCE.POST);
fenceCollisions.push({ x: 14, y: FENCE_ROW, w: 1, h: 1 });
fenceCollisions.push({ x: 34, y: FENCE_ROW, w: 1, h: 1 });

// ---------------------------------------------------------------------
// 8. DecorationBelow: flat, non-blocking clutter (flowers, sprigs)
// ---------------------------------------------------------------------
const belowDecor = [
  [21, 23, DECOR.TULIP_RED], [21, 22, DECOR.POT_RED],
  [26, 23, DECOR.TULIP_YELLOW], [26, 22, DECOR.POT_YELLOW],
  [20, 22, DECOR.BLOOM_RED], [27, 22, DECOR.BLOOM_YELLOW],
  [15, 7, DECOR.WHITE_FLOWER_1], [8, 15, DECOR.WHITE_FLOWER_2],
  [17, 6, DECOR.GRASS_SPRIG_1], [30, 8, DECOR.GRASS_SPRIG_2],
  [12, 25, DECOR.GRASS_SPRIG_3], [33, 30, DECOR.GRASS_SPRIG_1],
  [45, 15, DECOR.GRASS_SPRIG_2], [5, 20, DECOR.GRASS_SPRIG_3],
  [37, 15, DECOR.WHITE_FLOWER_1], [16, 28, DECOR.TULIP_RED],
];
for (const [x, y, gid] of belowDecor) setTile(decoBelow, x, y, gid);

// ---------------------------------------------------------------------
// Object layers: Collision / SpawnPoints / NPCSpawns
// ---------------------------------------------------------------------
let nextObjId = 1;
function rectObject(name, type, tx, ty, tw, th, properties) {
  const obj = {
    id: nextObjId++, name, type, x: tx * TS, y: ty * TS, width: tw * TS, height: th * TS,
    rotation: 0, visible: true,
  };
  if (properties) obj.properties = properties;
  return obj;
}
function pointObject(name, type, tx, ty, properties) {
  const obj = {
    id: nextObjId++, name, type, x: tx * TS, y: ty * TS, width: 0, height: 0,
    rotation: 0, visible: true, point: true,
  };
  if (properties) obj.properties = properties;
  return obj;
}

const collisionObjects = [];
// House footprint minus the door gap (door = local col2 -> global x24, bottom row -> global y19)
collisionObjects.push(rectObject('house_upper', 'wall', HOUSE.x, HOUSE.y, HOUSE_W, HOUSE_H - 1, null));
collisionObjects.push(rectObject('house_base_left', 'wall', HOUSE.x, HOUSE.y + HOUSE_H - 1, DOOR_X - HOUSE.x, 1, null));
collisionObjects.push(rectObject('house_base_right', 'wall', DOOR_X + 1, HOUSE.y + HOUSE_H - 1, HOUSE.x + HOUSE_W - (DOOR_X + 1), 1, null));
// pond
collisionObjects.push(rectObject('pond_water', 'water', POND.x, POND.y, POND.w, POND.h, null));
// trees + props + fences
treeCollisions.forEach((r, i) => collisionObjects.push(rectObject('tree_' + i, 'obstacle', r.x, r.y, r.w, r.h, null)));
propCollisions.forEach((r, i) => collisionObjects.push(rectObject('prop_' + i, 'obstacle', r.x, r.y, r.w, r.h, null)));
fenceCollisions.forEach((r, i) => collisionObjects.push(rectObject('fence_' + i, 'wall', r.x, r.y, r.w, r.h, null)));

const spawnPointObjects = [
  pointObject('player_spawn', 'spawn', 24, 32, [{ name: 'facing', type: 'string', value: 'north' }]),
];

const npcSpawnObjects = [
  pointObject('npc_spawn_01', 'animal', 39, 9, [{ name: 'species', type: 'string', value: 'sheep' }]),
  pointObject('npc_spawn_02', 'animal', 41, 12, [{ name: 'species', type: 'string', value: 'chicken' }]),
  pointObject('npc_spawn_03', 'animal', 39, 27, [{ name: 'species', type: 'string', value: 'cow' }]),
  pointObject('npc_spawn_04', 'animal', 41, 30, [{ name: 'species', type: 'string', value: 'pig' }]),
  pointObject('npc_spawn_05', 'animal', 9, 16, [{ name: 'species', type: 'string', value: 'chicken' }]),
  pointObject('npc_spawn_06', 'villager', 13, 16, [{ name: 'role', type: 'string', value: 'fisher' }]),
  pointObject('npc_spawn_07', 'villager', 30, 22, [{ name: 'role', type: 'string', value: 'villager' }]),
  pointObject('npc_spawn_08', 'villager', 18, 22, [{ name: 'role', type: 'string', value: 'villager' }]),
  pointObject('npc_spawn_09', 'villager', 25, 28, [{ name: 'role', type: 'string', value: 'villager' }]),
  pointObject('npc_spawn_10', 'villager', 25, 6, [{ name: 'role', type: 'string', value: 'villager' }]),
];

// ---------------------------------------------------------------------
// Assemble the .tmj
// ---------------------------------------------------------------------
function tileLayer(name, grid, id) {
  const data = [];
  for (let y = 0; y < MAP_H; y++) for (let x = 0; x < MAP_W; x++) data.push(grid[y][x]);
  return {
    id, name, type: 'tilelayer', width: MAP_W, height: MAP_H,
    x: 0, y: 0, opacity: 1, visible: true, data,
  };
}
function objectLayer(name, id, objs) {
  return {
    id, name, type: 'objectgroup', draworder: 'topdown',
    x: 0, y: 0, opacity: 1, visible: true, objects: objs,
  };
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
  nextlayerid: 8,
  nextobjectid: nextObjId,
  layers: [
    tileLayer('Ground', ground, 1),
    tileLayer('DecorationBelow', decoBelow, 2),
    tileLayer('Objects', objects, 3),
    tileLayer('DecorationAbove', decoAbove, 4),
    objectLayer('Collision', 5, collisionObjects),
    objectLayer('SpawnPoints', 6, spawnPointObjects),
    objectLayer('NPCSpawns', 7, npcSpawnObjects),
  ],
  tilesets: TILESETS.map(t => ({ firstgid: t.firstgid, source: t.source })),
};

const outPath = path.join(__dirname, '..', 'Maps', 'village_adventure.tmj');
fs.writeFileSync(outPath, JSON.stringify(map, null, 1));
console.log('written', outPath);
console.log('max gid used check follows in validator');
