-- supported_model_scopes is Antigravity-only metadata. Clear historical
-- default values from all other platforms so user-facing subscription plans
-- do not advertise Claude/Gemini/Imagen capabilities by accident.
ALTER TABLE groups
ALTER COLUMN supported_model_scopes SET DEFAULT '[]'::jsonb;

UPDATE groups
SET supported_model_scopes = '[]'::jsonb
WHERE COALESCE(platform, '') <> 'antigravity'
  AND supported_model_scopes <> '[]'::jsonb;
