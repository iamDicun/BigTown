'use strict';
const fs = require('fs');
const path = require('path');

const MAP_PATH = path.join(__dirname, '..', 'Maps', 'village_adventure.tmj');
const MAP_DIR = path.dirname(MAP_PATH);

let failures = 0;
let warnings = 0;
function fail(msg) { failures++; console.log('[FAIL] ' + msg); }
function warn(msg) { warnings++; console.log('[WARN] ' + msg); }
function ok(msg) { console.log('[OK]   ' + msg); }

if (!fs.existsSync(MAP_PATH)) {
  console.log('[FAIL] map file not found: ' + MAP_PATH);
  process.exit(1);
}
const map = JSON.parse(fs.readFileSync(MAP_PATH, 'utf8'));

// 1. Basic structure -------------------------------------------------
if (map.type !== 'map') fail('root type is not "map"');
if (map.orientation !== 'orthogonal') fail('orientation must be orthogonal for a top-down map');
if (map.tilewidth !== 16 || map.tileheight !== 16) fail('tile size is not 16x16');
if (map.width !== 50 || map.height !== 35) fail('map size is not 50x35, got ' + map.width + 'x' + map.height);
else ok('map size 50x35, tile size 16x16');

// 2. Tileset path resolution ------------------------------------------
if (!Array.isArray(map.tilesets) || map.tilesets.length === 0) {
  fail('no tilesets referenced');
} else {
  let allResolved = true;
  const gidRanges = [];
  for (const ts of map.tilesets) {
    const resolved = path.resolve(MAP_DIR, ts.source);
    if (!fs.existsSync(resolved)) {
      fail('tileset source does not resolve to a file: ' + ts.source + ' -> ' + resolved);
      allResolved = false;
      continue;
    }
    const tsj = JSON.parse(fs.readFileSync(resolved, 'utf8'));
    const imgResolved = path.resolve(path.dirname(resolved), tsj.image);
    if (!fs.existsSync(imgResolved)) {
      fail('tileset "' + ts.source + '" image does not resolve: ' + tsj.image + ' -> ' + imgResolved);
      allResolved = false;
    }
    gidRanges.push({ name: tsj.name, firstgid: ts.firstgid, lastgid: ts.firstgid + tsj.tilecount - 1 });
  }
  if (allResolved) ok('all ' + map.tilesets.length + ' tileset sources and images resolve on disk');
  map.__gidRanges = gidRanges;
}

// 3. Required layers ---------------------------------------------------
const REQUIRED = [
  ['Ground', 'tilelayer'],
  ['DecorationBelow', 'tilelayer'],
  ['Objects', 'tilelayer'],
  ['DecorationAbove', 'tilelayer'],
  ['Collision', 'objectgroup'],
  ['SpawnPoints', 'objectgroup'],
  ['NPCSpawns', 'objectgroup'],
];
const byName = {};
for (const l of map.layers || []) byName[l.name] = l;
for (const [name, type] of REQUIRED) {
  const layer = byName[name];
  if (!layer) { fail('missing required layer: ' + name); continue; }
  if (layer.type !== type) { fail('layer "' + name + '" has type ' + layer.type + ', expected ' + type); continue; }
  ok('layer "' + name + '" present (' + type + ')');
}

// 4. Tile layer data integrity + gid range check ------------------------
function gidOwner(gid) {
  if (gid === 0) return 'empty';
  for (const r of map.__gidRanges || []) if (gid >= r.firstgid && gid <= r.lastgid) return r.name;
  return null;
}
for (const [name] of REQUIRED.filter(r => r[1] === 'tilelayer')) {
  const layer = byName[name];
  if (!layer) continue;
  if (layer.data.length !== map.width * map.height) {
    fail('layer "' + name + '" data length ' + layer.data.length + ' != width*height ' + (map.width * map.height));
    continue;
  }
  let badGid = 0;
  for (const gid of layer.data) if (gidOwner(gid) === null) badGid++;
  if (badGid > 0) fail('layer "' + name + '" has ' + badGid + ' tile(s) with a gid not covered by any declared tileset');
  else ok('layer "' + name + '" gids all resolve to a declared tileset');
}

// 5. Spawn points --------------------------------------------------------
const spawnLayer = byName['SpawnPoints'];
const npcLayer = byName['NPCSpawns'];
if (spawnLayer) {
  const playerSpawns = (spawnLayer.objects || []).filter(o => o.name === 'player_spawn');
  if (playerSpawns.length !== 1) fail('expected exactly 1 object named "player_spawn" in SpawnPoints, found ' + playerSpawns.length);
  else ok('player_spawn present at (' + (playerSpawns[0].x / 16) + ',' + (playerSpawns[0].y / 16) + ') tile coords');
}
if (npcLayer) {
  const count = (npcLayer.objects || []).length;
  if (count < 10) fail('expected at least 10 NPC/animal spawns, found ' + count);
  else ok(count + ' NPC/animal spawn points present');
}

// 6. Collision must not cover the door gap or the road/gate gap ---------
function tileRectsOverlap(a, b) {
  return a.x0 < b.x1 && a.x1 > b.x0 && a.y0 < b.y1 && a.y1 > b.y0;
}
function objToTileRect(o) {
  return { x0: o.x / 16, y0: o.y / 16, x1: (o.x + o.width) / 16, y1: (o.y + o.height) / 16 };
}
const collisionLayer = byName['Collision'];
if (collisionLayer && spawnLayer) {
  // Sample the path/gate corridor tiles that must remain walkable: the door threshold and the south gate opening.
  const criticalPoints = [
    { label: 'door threshold (24,19)', x: 24, y: 19 },
    { label: 'south gate opening (24,33)', x: 24, y: 33 },
    { label: 'player_spawn tile (24,32)', x: 24, y: 32 },
  ];
  for (const cp of criticalPoints) {
    const rect = { x0: cp.x, y0: cp.y, x1: cp.x + 1, y1: cp.y + 1 };
    const blocking = (collisionLayer.objects || []).filter(o => tileRectsOverlap(objToTileRect(o), rect));
    if (blocking.length > 0) fail('collision object(s) [' + blocking.map(b => b.name).join(', ') + '] block critical tile ' + cp.label);
    else ok('critical tile ' + cp.label + ' is free of collision');
  }
}

// 7. Reachability flood-fill: every non-collision tile must be reachable from player_spawn
function buildBlockedGrid() {
  const blocked = Array.from({ length: map.height }, () => new Array(map.width).fill(false));
  for (const o of (collisionLayer.objects || [])) {
    const x0 = Math.floor(o.x / 16), y0 = Math.floor(o.y / 16);
    const x1 = Math.ceil((o.x + o.width) / 16), y1 = Math.ceil((o.y + o.height) / 16);
    for (let y = y0; y < y1; y++) for (let x = x0; x < x1; x++) {
      if (y >= 0 && y < map.height && x >= 0 && x < map.width) blocked[y][x] = true;
    }
  }
  return blocked;
}
if (collisionLayer && spawnLayer) {
  const blocked = buildBlockedGrid();
  const start = (spawnLayer.objects || []).find(o => o.name === 'player_spawn');
  const sx = Math.floor(start.x / 16), sy = Math.floor(start.y / 16);
  const seen = Array.from({ length: map.height }, () => new Array(map.width).fill(false));
  const queue = [[sx, sy]];
  seen[sy][sx] = true;
  let reachable = 1;
  while (queue.length) {
    const [x, y] = queue.pop();
    for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) {
      const nx = x + dx, ny = y + dy;
      if (nx < 0 || ny < 0 || nx >= map.width || ny >= map.height) continue;
      if (seen[ny][nx] || blocked[ny][nx]) continue;
      seen[ny][nx] = true;
      reachable++;
      queue.push([nx, ny]);
    }
  }
  const totalOpen = map.width * map.height - blocked.flat().filter(Boolean).length;
  const unreachable = totalOpen - reachable;
  if (unreachable > 0) fail(unreachable + ' open tile(s) are unreachable from player_spawn (isolated pocket behind collision)');
  else ok('all ' + reachable + ' open tiles are reachable from player_spawn (no sealed-off areas)');

  // every NPC spawn must sit on a reachable, non-collision tile
  let badNpc = 0;
  for (const npc of (npcLayer.objects || [])) {
    const nx = Math.floor(npc.x / 16), ny = Math.floor(npc.y / 16);
    if (blocked[ny][nx]) { fail('NPC spawn "' + npc.name + '" sits on a collision tile'); badNpc++; }
    else if (!seen[ny][nx]) { fail('NPC spawn "' + npc.name + '" is not reachable from player_spawn'); badNpc++; }
  }
  if (badNpc === 0) ok('all NPC/animal spawns sit on open, reachable tiles');
}

console.log('');
console.log(failures === 0 ? ('VALID: 0 failures, ' + warnings + ' warning(s)') : (failures + ' FAILURE(S), ' + warnings + ' warning(s)'));
process.exit(failures === 0 ? 0 : 1);
