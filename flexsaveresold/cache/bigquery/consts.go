package bq

const withGetKeyFromSystemLabels = `
CREATE TEMP FUNCTION getKeyFromSystemLabels(labels ARRAY<STRUCT<key STRING, value STRING>>, key_lookup STRING)
RETURNS STRING AS (
  (SELECT l.value FROM UNNEST(labels) l WHERE l.key = key_lookup LIMIT 1)
);
`
