DELETE FROM planner_footpaths
WHERE (from_stop_id = 'metro:81' AND to_stop_id = 'metro:500')
   OR (from_stop_id = 'metro:500' AND to_stop_id = 'metro:81');

DROP INDEX IF EXISTS idx_planner_footpaths_to_stop_id;
DROP INDEX IF EXISTS idx_planner_footpaths_from_stop_id;
DROP TABLE IF EXISTS planner_footpaths;
