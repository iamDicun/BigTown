'use strict';
const fs = require('fs');
const path = require('path');

const MAP_W = 250;
const MAP_H = 175;
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
  { name: 'Cliff_Tile', source: '../Tiles/Cliff_Tile.tsj', firstgid: 209, tilecount: 18 },
  { name: 'Path_Tile', source: '../Tiles/Path_Tile.tsj', firstgid: 227, tilecount: 18 },
  { name: 'FarmLand_Tile', source: '../Tiles/FarmLand_Tile.tsj', firstgid: 245, tilecount: 9 },
  { name: 'Beach_Tile', source: '../Tiles/Beach_Tile.tsj', firstgid: 254, tilecount: 15 },
];

function gidFor(name, localIndex) {
  const ts = TILESETS.find(t => t.name === name);
  if (!ts) throw new Error('unknown tileset ' + name);
  if (localIndex < 0 || localIndex >= ts.tilecount) throw new Error(name + ' index out of range: ' + localIndex);
  return ts.firstgid + localIndex;
}

// GIDs
const GID_GRASS = gidFor('Grass_Middle', 0);
const GID_PATH_MID = gidFor('Path_Middle', 0);
const GID_WATER_FILL = gidFor('Water_Middle', 0);
const GID_CHEST = gidFor('Chest', 0);

// Water_Tile
const WATER = {
  TL: gidFor('Water_Tile', 0), T: gidFor('Water_Tile', 1), TR: gidFor('Water_Tile', 2),
  L: gidFor('Water_Tile', 3), C: GID_WATER_FILL, R: gidFor('Water_Tile', 5),
  BL: gidFor('Water_Tile', 6), B: gidFor('Water_Tile', 7), BR: gidFor('Water_Tile', 8),
  TL_IN: gidFor('Water_Tile', 13), TR_IN: gidFor('Water_Tile', 12),
  BL_IN: gidFor('Water_Tile', 10), BR_IN: gidFor('Water_Tile', 9),
};

// Cliff_Tile
const CLIFF = {
  TL: gidFor('Cliff_Tile', 0), T: gidFor('Cliff_Tile', 1), TR: gidFor('Cliff_Tile', 2),
  L: gidFor('Cliff_Tile', 3), C: 0, R: gidFor('Cliff_Tile', 5),
  BL: gidFor('Cliff_Tile', 6), B: gidFor('Cliff_Tile', 7), BR: gidFor('Cliff_Tile', 8),
  TL_IN: gidFor('Cliff_Tile', 13), TR_IN: gidFor('Cliff_Tile', 12),
  BL_IN: gidFor('Cliff_Tile', 10), BR_IN: gidFor('Cliff_Tile', 9),
};

// Beach_Tile
const BEACH = {
  TL: gidFor('Beach_Tile', 12), T: gidFor('Beach_Tile', 11), TR: gidFor('Beach_Tile', 10),
  L: gidFor('Beach_Tile', 7), C: GID_WATER_FILL, R: gidFor('Beach_Tile', 5),
  BL: gidFor('Beach_Tile', 2), B: gidFor('Beach_Tile', 1), BR: gidFor('Beach_Tile', 0),
  SAND_C: gidFor('Beach_Tile', 6),
};

// Path_Tile
const PATH = {
  TL: gidFor('Path_Tile', 0), T: gidFor('Path_Tile', 1), TR: gidFor('Path_Tile', 2),
  L: gidFor('Path_Tile', 3), C: gidFor('Path_Tile', 4), R: gidFor('Path_Tile', 5),
  BL: gidFor('Path_Tile', 6), B: gidFor('Path_Tile', 7), BR: gidFor('Path_Tile', 8),
  TL_IN: gidFor('Path_Tile', 13), TR_IN: gidFor('Path_Tile', 12),
  BL_IN: gidFor('Path_Tile', 10), BR_IN: gidFor('Path_Tile', 9),
};

// FarmLand_Tile
const FARMLAND = {
  TL: gidFor('FarmLand_Tile', 0), T: gidFor('FarmLand_Tile', 1), TR: gidFor('FarmLand_Tile', 2),
  L: gidFor('FarmLand_Tile', 3), C: gidFor('FarmLand_Tile', 4), R: gidFor('FarmLand_Tile', 5),
  BL: gidFor('FarmLand_Tile', 6), B: gidFor('FarmLand_Tile', 7), BR: gidFor('FarmLand_Tile', 8),
  TL_IN: gidFor('FarmLand_Tile', 4), TR_IN: gidFor('FarmLand_Tile', 4),
  BL_IN: gidFor('FarmLand_Tile', 4), BR_IN: gidFor('FarmLand_Tile', 4),
};

// Fences
const FENCE = {
  POST: gidFor('Fences', 0),
  H_LEFT: gidFor('Fences', 1),
  H_MID: gidFor('Fences', 2),
  H_RIGHT: gidFor('Fences', 3),
};

// Decor
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

function setTile(grid, x, y, gid) {
  if (x >= 0 && x < MAP_W && y >= 0 && y < MAP_H) {
    grid[y][x] = gid;
  }
}

function fillRect(grid, x, y, w, h, gid) {
  for (let dy = 0; dy < h; dy++) {
    for (let dx = 0; dx < w; dx++) {
      setTile(grid, x + dx, y + dy, gid);
    }
  }
}

// Grid setup
const ground = blankGrid(MAP_W, MAP_H);
const decoBelow = blankGrid(MAP_W, MAP_H);
const objects = blankGrid(MAP_W, MAP_H);
const decoAbove = blankGrid(MAP_W, MAP_H);
fillRect(ground, 0, 0, MAP_W, MAP_H, GID_GRASS);

const collisionObjects = [];
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

// ---------------------------------------------------------------------
// Initialize masks for organic layout
// ---------------------------------------------------------------------
function booleanMask() {
  return Array.from({ length: MAP_H }, () => new Array(MAP_W).fill(false));
}

const lakeMask = booleanMask();      // grass border water
const seaMask = booleanMask();       // sand border water (beach)
const hillMask = booleanMask();      // raised plateaus
const pathMask = booleanMask();      // roads
const farmlandMask = booleanMask();  // crop fields
const walkableMask = booleanMask();  // tracks allowed zones for paths/spawns

function addCircleToMask(mask, cx, cy, r) {
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      const dx = x - cx;
      const dy = y - cy;
      if (dx * dx + dy * dy <= r * r) {
        mask[y][x] = true;
      }
    }
  }
}

// ---------------------------------------------------------------------
// Define Organic Shapes
// ---------------------------------------------------------------------

// 1. Sandy Beach Lake (seaMask) - Large Central-West Lake
addCircleToMask(seaMask, 55, 65, 15);
addCircleToMask(seaMask, 42, 58, 12);
addCircleToMask(seaMask, 68, 62, 9);
addCircleToMask(seaMask, 58, 52, 11);

// 2. Freshwater Pond 1 - Northeast Pond
addCircleToMask(lakeMask, 180, 30, 8);
addCircleToMask(lakeMask, 172, 33, 6);

// 3. Freshwater Pond 2 - Southeast Swamp/Pond
addCircleToMask(lakeMask, 195, 125, 10);
addCircleToMask(lakeMask, 205, 120, 8);
addCircleToMask(lakeMask, 188, 128, 6);

// 4. Northwest Hill
addCircleToMask(hillMask, 25, 22, 14);
addCircleToMask(hillMask, 38, 20, 10);
addCircleToMask(hillMask, 20, 30, 8);

// 5. Southwest Hill
addCircleToMask(hillMask, 40, 130, 13);
addCircleToMask(hillMask, 30, 135, 9);
addCircleToMask(hillMask, 50, 125, 8);

// 6. East Hill
addCircleToMask(hillMask, 215, 65, 15);
addCircleToMask(hillMask, 205, 75, 11);
addCircleToMask(hillMask, 225, 58, 9);

// 7. Farmlands
function addRectToMask(mask, x, y, w, h) {
  for (let dy = 0; dy < h; dy++) {
    for (let dx = 0; dx < w; dx++) {
      const gx = x + dx, gy = y + dy;
      if (gx >= 0 && gx < MAP_W && gy >= 0 && gy < MAP_H) {
        mask[gy][gx] = true;
      }
    }
  }
}
// Rounded, rustic gardens
addRectToMask(farmlandMask, 90, 24, 10, 9);
addRectToMask(farmlandMask, 175, 84, 11, 9);

// ---------------------------------------------------------------------
// Define House Locations and door coordinates
// ---------------------------------------------------------------------
const houses = [
  { name: 'Mayor House', x: 120, y: 130, doorX: 122, doorY: 137 },
  { name: 'Fisher House', x: 35, y: 80, doorX: 37, doorY: 87 },
  { name: 'North Farmer House', x: 80, y: 30, doorX: 82, doorY: 37 },
  { name: 'East Farmer House', x: 160, y: 90, doorX: 162, doorY: 97 },
  { name: 'Hilltop House', x: 25, y: 20, doorX: 27, doorY: 27 }, // on NW Hill
];

// ---------------------------------------------------------------------
// Path Mask Drawing (Winding Roads)
// ---------------------------------------------------------------------
function drawPathLine(x0, y0, x1, y1, r = 1.5) {
  const steps = Math.max(Math.abs(x1 - x0), Math.abs(y1 - y0)) * 2;
  for (let i = 0; i <= steps; i++) {
    const t = steps === 0 ? 0 : i / steps;
    const cx = x0 + (x1 - x0) * t;
    const cy = y0 + (y1 - y0) * t;
    // Set mask inside radius
    for (let y = Math.floor(cy - r); y <= Math.ceil(cy + r); y++) {
      for (let x = Math.floor(cx - r); x <= Math.ceil(cx + r); x++) {
        if (x >= 0 && x < MAP_W && y >= 0 && y < MAP_H) {
          const dx = x - cx;
          const dy = y - cy;
          if (dx * dx + dy * dy <= r * r) {
            pathMask[y][x] = true;
          }
        }
      }
    }
  }
}

// Spawn Plaza (clean, wide road block)
for (let y = 157; y <= 163; y++) {
  for (let x = 122; x <= 128; x++) {
    pathMask[y][x] = true;
  }
}

// Main road segments
drawPathLine(125, 163, 125, 174, 1.5); // down to entrance
drawPathLine(125, 157, 125, 137, 1.5); // to mayor house door area
drawPathLine(122, 137, 125, 137, 1.0); // connect to mayor door

// Main path going north from Mayor House to main highway at y=105
drawPathLine(125, 137, 125, 105, 1.5);

// Winding main highway (y=105) going West and East
// West road: winds around the lake to the fisher's house
drawPathLine(125, 105, 105, 107, 1.5);
drawPathLine(105, 107, 85, 102, 1.5);
drawPathLine(85, 102, 65, 105, 1.5);
drawPathLine(65, 105, 48, 100, 1.5);
drawPathLine(48, 100, 37, 95, 1.5);
drawPathLine(37, 95, 37, 87, 1.0); // directly into fisher's house door!

// East road: winds to the east farmer and east hill
drawPathLine(125, 105, 145, 103, 1.5);
drawPathLine(145, 103, 162, 105, 1.5);
drawPathLine(162, 105, 162, 97, 1.0); // to east farmer door
drawPathLine(162, 105, 185, 100, 1.5);
drawPathLine(185, 100, 198, 88, 1.5);
drawPathLine(198, 88, 198, 70, 1.2); // to East Hill ramp entrance!

// North road: connects mayor house / highway to north farmland and NW hill
drawPathLine(125, 105, 125, 60, 1.5);
drawPathLine(125, 60, 110, 45, 1.5);
drawPathLine(110, 45, 95, 38, 1.5);
drawPathLine(95, 38, 82, 37, 1.0); // to north farmer door
drawPathLine(82, 37, 70, 28, 1.5);
drawPathLine(70, 28, 52, 23, 1.5);
drawPathLine(52, 23, 40, 23, 1.2); // to NW Hill ramp!
// Path inside NW hill
drawPathLine(38, 23, 27, 23, 1.0);
drawPathLine(27, 23, 27, 27, 1.0); // to hilltop house door

// Southwest Hill path: connects highway to SW Hill ramp
drawPathLine(45, 100, 45, 116, 1.2);

// East Hill hilltop path: winds inside East Hill
drawPathLine(198, 70, 215, 70, 1.2);

// Force paths to avoid water
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (lakeMask[y][x] || seaMask[y][x]) {
      pathMask[y][x] = false;
    }
  }
}

// ---------------------------------------------------------------------
// Autotiling Resolution Algorithms
// ---------------------------------------------------------------------
function getAutotileGid(mask, x, y, tileset) {
  const u = y > 0 && mask[y - 1][x];
  const d = y < MAP_H - 1 && mask[y + 1][x];
  const l = x > 0 && mask[y][x - 1];
  const r = x < MAP_W - 1 && mask[y][x + 1];

  // Corner matches
  if (!u && d && !l && r) return tileset.TL;
  if (!u && d && l && !r) return tileset.TR;
  if (u && !d && !l && r) return tileset.BL;
  if (u && !d && l && !r) return tileset.BR;

  // Edge matches
  if (!u && d) return tileset.T;
  if (u && !d) return tileset.B;
  if (!l && r) return tileset.L;
  if (l && !r) return tileset.R;
  // Inner corner matches
  if (u && d && l && r) {
    const ul = y > 0 && x > 0 && mask[y - 1][x - 1];
    const ur = y > 0 && x < MAP_W - 1 && mask[y - 1][x + 1];
    const bl = y < MAP_H - 1 && x > 0 && mask[y + 1][x - 1];
    const br = y < MAP_H - 1 && x < MAP_W - 1 && mask[y + 1][x + 1];

    if (!ul && tileset.TL_IN) return tileset.TL_IN;
    if (!ur && tileset.TR_IN) return tileset.TR_IN;
    if (!bl && tileset.BL_IN) return tileset.BL_IN;
    if (!br && tileset.BR_IN) return tileset.BR_IN;
  }

  // Default is center
  return tileset.C;
}

// ---------------------------------------------------------------------
// Paint ground and borders
// ---------------------------------------------------------------------

// 1. Paint Sandy Lakes (seaMask) and Sand Beaches
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    // If inside seaMask
    if (seaMask[y][x]) {
      const gid = getAutotileGid(seaMask, x, y, BEACH);
      setTile(ground, x, y, gid);
    }
  }
}
// Add sandy buffer around the sea mask
const sandMask = booleanMask();
for (let y = 1; y < MAP_H - 1; y++) {
  for (let x = 1; x < MAP_W - 1; x++) {
    if (!seaMask[y][x]) {
      // Check if neighbors touch seaMask
      let touchesSea = false;
      for (let dy = -2; dy <= 2; dy++) {
        for (let dx = -2; dx <= 2; dx++) {
          const gx = x + dx, gy = y + dy;
          if (gx >= 0 && gx < MAP_W && gy >= 0 && gy < MAP_H && seaMask[gy][gx]) {
            touchesSea = true;
            break;
          }
        }
        if (touchesSea) break;
      }
      if (touchesSea && !lakeMask[y][x] && !hillMask[y][x] && !pathMask[y][x]) {
        sandMask[y][x] = true;
      }
    }
  }
}
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (sandMask[y][x]) {
      setTile(ground, x, y, BEACH.SAND_C);
    }
  }
}

// 2. Paint Freshwater Lakes (lakeMask) - Green grass border
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (lakeMask[y][x]) {
      const gid = getAutotileGid(lakeMask, x, y, WATER);
      setTile(ground, x, y, gid);
    }
  }
}

// Add collisions for all water tiles
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (lakeMask[y][x] || seaMask[y][x]) {
      collisionObjects.push(rectObject('water_' + x + '_' + y, 'wall', x, y, 1, 1, null));
    }
  }
}


// 3. Paint Farmlands (farmlandMask) - Rounded garden borders
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (farmlandMask[y][x] && !lakeMask[y][x] && !seaMask[y][x] && !pathMask[y][x]) {
      const gid = getAutotileGid(farmlandMask, x, y, FARMLAND);
      setTile(ground, x, y, gid);

      // Scatter flowers/decor inside the farm plots
      if (gid === FARMLAND.C && Math.random() < 0.25) {
        const flowerGids = [DECOR.TULIP_RED, DECOR.TULIP_YELLOW, DECOR.BLOOM_RED, DECOR.BLOOM_YELLOW, DECOR.WHITE_FLOWER_1, DECOR.WHITE_FLOWER_2];
        const randomFlower = flowerGids[Math.floor(Math.random() * flowerGids.length)];
        setTile(decoBelow, x, y, randomFlower);
      }
    }
  }
}

// Place farm decorative fences
const farmFences = [
  { y: 35, x0: 118, x1: 132 },
  { y: 129, x0: 44, x1: 56 },
  { y: 129, x0: 189, x1: 201 }
];
farmFences.forEach(f => {
  for (let x = f.x0; x <= f.x1; x++) {
    if (ground[f.y][x] === GID_GRASS && objects[f.y][x] === 0 && !hillMask[f.y][x]) {
      let gid = FENCE.H_MID;
      if (x === f.x0) gid = FENCE.H_LEFT;
      else if (x === f.x1) gid = FENCE.H_RIGHT;
      setTile(objects, x, f.y, gid);
      collisionObjects.push(rectObject('farm_fence_' + x + '_' + f.y, 'wall', x, f.y, 1, 1, null));
    }
  }
});

// 4. Paint Autotiled Paths (pathMask)
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (pathMask[y][x]) {
      const gid = getAutotileGid(pathMask, x, y, PATH);
      setTile(ground, x, y, gid);
    }
  }
}

// ---------------------------------------------------------------------
// Paint Hills / Cliffs (Corrected vertical elevation and organic shapes)
// ---------------------------------------------------------------------

// For hill compilation, we determine borders using autotiling
const cliffTiles = blankGrid(MAP_W, MAP_H);

for (let y = 1; y < MAP_H - 1; y++) {
  for (let x = 1; x < MAP_W - 1; x++) {
    if (hillMask[y][x]) {
      // Hill top is walkable grass
      if (!pathMask[y][x]) {
        setTile(ground, x, y, GID_GRASS);
      }

      // Check if it is a border tile
      const u = hillMask[y - 1][x];
      const d = hillMask[y + 1][x];
      const l = hillMask[y][x - 1];
      const r = hillMask[y][x + 1];
      const isBorder = !u || !d || !l || !r;

      if (isBorder) {
        // Ramps definition: carve cliff gaps automatically near paths
        let isRamp = false;
        for (let dy = -2; dy <= 2; dy++) {
          for (let dx = -2; dx <= 2; dx++) {
            const gx = x + dx, gy = y + dy;
            if (gx >= 0 && gx < MAP_W && gy >= 0 && gy < MAP_H && pathMask[gy][gx]) {
              isRamp = true;
              break;
            }
          }
          if (isRamp) break;
        }

        if (isRamp) continue;

        // Pass 1: standard autotile corner/edge matching for hill top
        const gid = getAutotileGid(hillMask, x, y, CLIFF);
        if (gid !== 0) {
          setTile(cliffTiles, x, y, gid);
        }
      }
    }
  }
}

// Pass 2: Add 1x1 collision objects for all border cliff tiles
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    const tile = cliffTiles[y][x];
    if (tile !== 0) {
      collisionObjects.push(rectObject('cliff_wall_' + x + '_' + y, 'wall', x, y, 1, 1, null));
    }
  }
}

// Write compiled cliff tiles into the objects layer
for (let y = 0; y < MAP_H; y++) {
  for (let x = 0; x < MAP_W; x++) {
    if (cliffTiles[y][x] !== 0) {
      setTile(objects, x, y, cliffTiles[y][x]);
    }
  }
}

// ---------------------------------------------------------------------
// Paint Houses
// ---------------------------------------------------------------------
for (const house of houses) {
  for (let row = 0; row < HOUSE_H; row++) {
    for (let col = 0; col < HOUSE_W; col++) {
      const gid = houseGid(row * HOUSE_W + col);
      setTile(objects, house.x + col, house.y + row, gid);
    }
  }
  // House footprint collision boxes
  collisionObjects.push(rectObject('house_upper_' + house.x, 'wall', house.x, house.y, HOUSE_W, HOUSE_H - 1, null));
  collisionObjects.push(rectObject('house_base_l_' + house.x, 'wall', house.x, house.y + HOUSE_H - 1, 2, 1, null));
  collisionObjects.push(rectObject('house_base_r_' + house.x, 'wall', house.doorX + 1, house.y + HOUSE_H - 1, HOUSE_W - 3, 1, null));
}

// ---------------------------------------------------------------------
// Paint Fences
// ---------------------------------------------------------------------
const FENCE_ROW = 173;
function placeFenceRun(x0, x1, row) {
  for (let x = x0; x <= x1; x++) {
    let gid;
    if (x === x0) gid = FENCE.H_LEFT;
    else if (x === x1) gid = FENCE.H_RIGHT;
    else gid = FENCE.H_MID;
    setTile(objects, x, row, gid);
  }
  collisionObjects.push(rectObject('fence_wall_' + x0, 'wall', x0, row, x1 - x0 + 1, 1, null));
}

// Bottom border gate entrance fence run
placeFenceRun(15, 122, FENCE_ROW);
placeFenceRun(128, 235, FENCE_ROW);
setTile(objects, 14, FENCE_ROW, FENCE.POST);
setTile(objects, 236, FENCE_ROW, FENCE.POST);
collisionObjects.push(rectObject('fence_post_l', 'wall', 14, FENCE_ROW, 1, 1, null));
collisionObjects.push(rectObject('fence_post_r', 'wall', 236, FENCE_ROW, 1, 1, null));

// ---------------------------------------------------------------------
// Trees & Forest Generation (Oak_Tree and Oak_Tree_Small)
// ---------------------------------------------------------------------
const treeCollisions = [];
function placeBigOakTree(originX, originY) {
  for (let row = 0; row < OAKTREE_H; row++) {
    for (let col = 0; col < OAKTREE_W; col++) {
      const gid = oakTreeGid(row * OAKTREE_W + col);
      const gx = originX + col, gy = originY + row;
      if (row <= 2) setTile(decoAbove, gx, gy, gid);
      else setTile(objects, gx, gy, gid);
    }
  }
  treeCollisions.push({ x: originX + 1, y: originY + 3, w: 2, h: 2 });
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
  treeCollisions.push({ x: originX, y: originY + 2, w: 2, h: 1 });
}

function touchesCliffOrHillBorder(x, y) {
  // Check cliff tiles and water tiles
  for (let dy = -1; dy <= 1; dy++) {
    for (let dx = -1; dx <= 1; dx++) {
      const gx = x + dx, gy = y + dy;
      if (gx >= 0 && gx < MAP_W && gy >= 0 && gy < MAP_H) {
        if (cliffTiles[gy][gx] !== 0) return true;
        if (lakeMask[gy][gx] || seaMask[gy][gx]) return true;
      }
    }
  }
  // Check hill transition in a 2-tile radius
  for (let dy = -2; dy <= 2; dy++) {
    for (let dx = -2; dx <= 2; dx++) {
      const gx = x + dx, gy = y + dy;
      if (gx >= 0 && gx < MAP_W && gy >= 0 && gy < MAP_H) {
        if (hillMask[y][x] !== hillMask[gy][gx]) return true;
      }
    }
  }
  return false;
}

// Organic forest layout helper
function fillForestOrganic(x0, y0, w, h, density = 0.3) {
  for (let y = y0; y < y0 + h - 5; y += 4) {
    for (let x = x0; x < x0 + w - 4; x += 3) {
      // Add slight noise to coordinates to make it organic and less grid-aligned!
      const nx = x + Math.floor(Math.sin(y) * 1.5);
      const ny = y + Math.floor(Math.cos(x) * 1.5);

      if (Math.random() < density) {
        let clean = true;
        for (let dy = 0; dy < 5; dy++) {
          for (let dx = 0; dx < 4; dx++) {
            const gx = nx + dx, gy = ny + dy;
            if (gx >= MAP_W || gy >= MAP_H || gx < 0 || gy < 0) { clean = false; break; }
            if (touchesCliffOrHillBorder(gx, gy) ||
                ground[gy][gx] === GID_PATH_MID || 
                (ground[gy][gx] >= PATH.TL && ground[gy][gx] <= PATH.BR) || 
                ground[gy][gx] === GID_WATER_FILL || 
                (ground[gy][gx] >= WATER.TL && ground[gy][gx] <= WATER.BR) ||
                (ground[gy][gx] >= BEACH.TL && ground[gy][gx] <= BEACH.BR) ||
                objects[gy][gx] !== 0) {
              clean = false;
              break;
            }
          }
          if (!clean) break;
        }
        if (clean) {
          if (Math.random() < 0.6) {
            placeBigOakTree(nx, ny);
          } else {
            placeSmallOakTree(nx, ny, Math.random() < 0.5 ? 'A' : 'B');
          }
        }
      }
    }
  }
}

// 1. Organic edge forests (natural border boundaries)
fillForestOrganic(0, 0, 10, MAP_H, 0.95);
fillForestOrganic(MAP_W - 10, 0, 10, MAP_H, 0.95);
fillForestOrganic(0, 0, MAP_W, 10, 0.95);
fillForestOrganic(0, 165, 120, 10, 0.95);
fillForestOrganic(130, 165, 120, 10, 0.95);

// Add thick collision walls over borders for physics efficiency
collisionObjects.push(rectObject('border_forest_left', 'wall', 0, 0, 13, MAP_H, null));
collisionObjects.push(rectObject('border_forest_right', 'wall', MAP_W - 13, 0, 13, MAP_H, null));
collisionObjects.push(rectObject('border_forest_top', 'wall', 0, 0, MAP_W, 12, null));
collisionObjects.push(rectObject('border_forest_tl_corner', 'wall', 12, 12, 4, 5, null));
collisionObjects.push(rectObject('border_forest_bottom_l', 'wall', 0, 165, 120, 10, null));
collisionObjects.push(rectObject('border_forest_bottom_r', 'wall', 130, 165, 120, 10, null));

// 2. Hide hill back slopes (north edges) under dense trees to complete the visual elevation!
for (let y = 1; y < MAP_H - 5; y++) {
  for (let x = 1; x < MAP_W - 5; x++) {
    // If we are just north of a hill top edge
    if (hillMask[y + 1][x] && !hillMask[y][x]) {
      // Place a tree here if clean and has space (avoid overlapping trees)
      let spaceClean = true;
      for (let dx = -1; dx <= 2; dx++) {
        for (let dy = 0; dy <= 2; dy++) {
          const gx = x + dx, gy = y + dy;
          if (gx >= 0 && gx < MAP_W && gy >= 0 && gy < MAP_H) {
            if (objects[gy][gx] !== 0) {
              spaceClean = false;
              break;
            }
          }
        }
        if (!spaceClean) break;
      }

      if (spaceClean && ground[y][x] !== GID_WATER_FILL && ground[y][x] !== GID_PATH_MID) {
        if (Math.random() < 0.8) {
          placeSmallOakTree(x, y, Math.random() < 0.5 ? 'A' : 'B');
        }
      }
    }
  }
}

// 3. Scatter forest clusters and clearings in open nature fields
fillForestOrganic(10, 10, 230, 150, 0.2);

// Place tree collisions for map interior trees
treeCollisions.forEach((r, i) => {
  if (r.x > 9 && r.x < MAP_W - 11 && r.y > 9 && r.y < MAP_H - 11) {
    collisionObjects.push(rectObject('tree_' + i, 'obstacle', r.x, r.y, r.w, r.h, null));
  }
});

// ---------------------------------------------------------------------
// Props, Clutter, and Lamps
// ---------------------------------------------------------------------
const propCollisions = [];
function placeProp(x, y, gid, layer) {
  setTile(layer, x, y, gid);
  propCollisions.push({ x, y, w: 1, h: 1 });
}

function placeLamp(x, y) {
  setTile(objects, x, y - 2, DECOR.LAMP_TOP);
  setTile(objects, x, y - 1, DECOR.LAMP_MID);
  setTile(objects, x, y, DECOR.LAMP_BASE);
  propCollisions.push({ x, y, w: 1, h: 1 });
}

// Place lamps along winding roads
const lampLocations = [
  { x: 121, y: 156 }, { x: 129, y: 156 },
  { x: 121, y: 125 }, { x: 129, y: 125 },
  { x: 121, y: 105 }, { x: 129, y: 105 },
  { x: 100, y: 94 }, { x: 150, y: 94 },
  { x: 60, y: 94 }, { x: 190, y: 94 },
  { x: 121, y: 60 }, { x: 129, y: 60 },
  { x: 100, y: 24 }, { x: 150, y: 24 },
];
for (const loc of lampLocations) {
  if (objects[loc.y][loc.x] === 0 && ground[loc.y][loc.x] === GID_GRASS) {
    placeLamp(loc.x, loc.y);
  }
}

// Place hidden exploration chests
const chestPositions = [
  { x: 30, y: 12 },    // Northwest hill corner
  { x: 225, y: 60 },   // East hill hidden spot
  { x: 50, y: 130 },   // Southwest hill corner
  { x: 205, y: 135 },  // Southeast swamp island
  { x: 32, y: 48 },    // Island/bank on Central Lake
];
chestPositions.forEach(pos => {
  if (objects[pos.y][pos.x] === 0) {
    placeProp(pos.x, pos.y, GID_CHEST, objects);
  }
});

// Scatter rocks and stumps organically
for (let y = 10; y < MAP_H - 10; y += 8) {
  for (let x = 10; x < MAP_W - 10; x += 12) {
    if (ground[y][x] === GID_GRASS && objects[y][x] === 0) {
      const rand = Math.random();
      if (rand < 0.1) {
        placeProp(x, y, DECOR.ROCK_CLUSTER, objects);
      } else if (rand < 0.2) {
        placeProp(x, y, DECOR.STUMP, objects);
      } else if (rand < 0.28) {
        placeProp(x, y, DECOR.LOG, objects);
      }
    }
  }
}

propCollisions.forEach((r, i) => {
  collisionObjects.push(rectObject('prop_' + i, 'obstacle', r.x, r.y, r.w, r.h, null));
});

// ---------------------------------------------------------------------
// Walkable Clutter: Flowers, pots, grass sprigs
// ---------------------------------------------------------------------
for (let y = 10; y < MAP_H - 10; y += 4) {
  for (let x = 10; x < MAP_W - 10; x += 5) {
    if (ground[y][x] === GID_GRASS && objects[y][x] === 0 && decoBelow[y][x] === 0) {
      if (Math.random() < 0.15) {
        const rand = Math.random();
        let gid;
        if (rand < 0.2) gid = DECOR.TULIP_RED;
        else if (rand < 0.4) gid = DECOR.TULIP_YELLOW;
        else if (rand < 0.6) gid = DECOR.WHITE_FLOWER_1;
        else if (rand < 0.8) gid = DECOR.WHITE_FLOWER_2;
        else gid = DECOR.GRASS_SPRIG_1;
        setTile(decoBelow, x, y, gid);
      }
    }
  }
}

// Decorate house porches
for (const house of houses) {
  setTile(decoBelow, house.x - 1, house.y + HOUSE_H - 1, DECOR.POT_RED);
  setTile(decoBelow, house.x + HOUSE_W, house.y + HOUSE_H - 1, DECOR.POT_YELLOW);
}

// ---------------------------------------------------------------------
// SpawnPoints & NPCSpawns (Validated Walkable Zones)
// ---------------------------------------------------------------------
const spawnPointObjects = [
  pointObject('player_spawn', 'spawn', 125, 160, [{ name: 'facing', type: 'string', value: 'north' }]),
];

const npcSpawnObjects = [];

function addNpcSpawn(name, type, tx, ty, properties) {
  const walkable = findWalkableTileNear(tx, ty);
  npcSpawnObjects.push(pointObject(name, type, walkable.x, walkable.y, properties));
}

function findWalkableTileNear(tx, ty) {
  function isBlocked(x, y) {
    if (x < 10 || x >= MAP_W - 10 || y < 10 || y >= MAP_H - 10) return true;
    if (ground[y][x] === GID_WATER_FILL || 
        (ground[y][x] >= WATER.TL && ground[y][x] <= WATER.BR) ||
        (ground[y][x] >= BEACH.TL && ground[y][x] <= BEACH.BR)) return true; // water is blocked
    if (objects[y][x] !== 0) return true;
    for (const obj of collisionObjects) {
      const ox0 = Math.floor(obj.x / TS);
      const oy0 = Math.floor(obj.y / TS);
      const ox1 = Math.ceil((obj.x + obj.width) / TS);
      const oy1 = Math.ceil((obj.y + obj.height) / TS);
      if (x >= ox0 && x < ox1 && y >= oy0 && y < oy1) return true;
    }
    return false;
  }

  for (let r = 0; r < 25; r++) {
    for (let dx = -r; dx <= r; dx++) {
      for (let dy = -r; dy <= r; dy++) {
        if (Math.abs(dx) === r || Math.abs(dy) === r) {
          const nx = tx + dx, ny = ty + dy;
          if (nx >= 0 && nx < MAP_W && ny >= 0 && ny < MAP_H && !isBlocked(nx, ny)) {
            return { x: nx, y: ny };
          }
        }
      }
    }
  }
  return { x: tx, y: ty };
}

// Place 35 NPC Spawns
// Villagers
addNpcSpawn('npc_spawn_mayor', 'villager', 121, 138, [{ name: 'role', type: 'string', value: 'villager' }]);
addNpcSpawn('npc_spawn_fisher1', 'villager', 32, 85, [{ name: 'role', type: 'string', value: 'fisher' }]);
addNpcSpawn('npc_spawn_fisher2', 'villager', 66, 55, [{ name: 'role', type: 'string', value: 'fisher' }]);
addNpcSpawn('npc_spawn_farmer1', 'villager', 82, 38, [{ name: 'role', type: 'string', value: 'villager' }]);
addNpcSpawn('npc_spawn_farmer2', 'villager', 162, 98, [{ name: 'role', type: 'string', value: 'villager' }]);
addNpcSpawn('npc_spawn_hermit', 'villager', 27, 28, [{ name: 'role', type: 'string', value: 'villager' }]);

// Animals near farmlands
addNpcSpawn('npc_spawn_sheep1', 'animal', 85, 28, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_sheep2', 'animal', 86, 26, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_chicken1', 'animal', 103, 27, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_chicken2', 'animal', 104, 29, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_cow1', 'animal', 170, 88, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_cow2', 'animal', 171, 91, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_pig1', 'animal', 158, 86, [{ name: 'species', type: 'string', value: 'pig' }]);
addNpcSpawn('npc_spawn_pig2', 'animal', 157, 89, [{ name: 'species', type: 'string', value: 'pig' }]);

// Nature meadows
addNpcSpawn('npc_spawn_sheep_nature1', 'animal', 60, 20, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_sheep_nature2', 'animal', 150, 18, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_sheep_nature3', 'animal', 220, 25, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_sheep_nature4', 'animal', 230, 45, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_chicken_nature1', 'animal', 50, 110, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_chicken_nature2', 'animal', 105, 115, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_chicken_nature3', 'animal', 150, 110, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_chicken_nature4', 'animal', 70, 140, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_chicken_nature5', 'animal', 180, 150, [{ name: 'species', type: 'string', value: 'chicken' }]);
addNpcSpawn('npc_spawn_cow_nature1', 'animal', 110, 60, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_cow_nature2', 'animal', 145, 65, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_cow_nature3', 'animal', 85, 75, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_cow_nature4', 'animal', 100, 80, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_pig_nature1', 'animal', 75, 120, [{ name: 'species', type: 'string', value: 'pig' }]);
addNpcSpawn('npc_spawn_pig_nature2', 'animal', 170, 125, [{ name: 'species', type: 'string', value: 'pig' }]);
addNpcSpawn('npc_spawn_pig_nature3', 'animal', 100, 145, [{ name: 'species', type: 'string', value: 'pig' }]);
addNpcSpawn('npc_spawn_pig_nature4', 'animal', 150, 140, [{ name: 'species', type: 'string', value: 'pig' }]);

// Hilltops
addNpcSpawn('npc_spawn_hill_sheep1', 'animal', 35, 18, [{ name: 'species', type: 'string', value: 'sheep' }]);
addNpcSpawn('npc_spawn_hill_pig1', 'animal', 225, 75, [{ name: 'species', type: 'string', value: 'pig' }]);
addNpcSpawn('npc_spawn_hill_cow1', 'animal', 45, 130, [{ name: 'species', type: 'string', value: 'cow' }]);
addNpcSpawn('npc_spawn_hill_chicken1', 'animal', 50, 125, [{ name: 'species', type: 'string', value: 'chicken' }]);

// ---------------------------------------------------------------------
// Assemble Map JSON
// ---------------------------------------------------------------------
function tileLayer(name, grid, id) {
  const data = [];
  for (let y = 0; y < MAP_H; y++) {
    for (let x = 0; x < MAP_W; x++) {
      data.push(grid[y][x]);
    }
  }
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
