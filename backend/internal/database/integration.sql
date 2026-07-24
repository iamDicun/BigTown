ALTER TABLE maps
ADD COLUMN layer_names VARCHAR(500),
ADD COLUMN above_layer_name VARCHAR(80),
ADD COLUMN collision_layer_name VARCHAR(80);