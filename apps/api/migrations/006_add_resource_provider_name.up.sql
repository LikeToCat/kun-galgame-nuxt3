-- Adds galgame_resource.provider_name to cache the human-readable provider
-- labels (e.g. "百度网盘", "OneDrive") at write time. The legacy `provider`
-- text[] column is retained — it stores the coarser 10-key taxonomy used by
-- the galgame list filter (gr.provider && ?). The two columns are kept in
-- sync by the resource service on Create/Update.
ALTER TABLE galgame_resource
    ADD COLUMN IF NOT EXISTS provider_name jsonb NOT NULL DEFAULT '[]'::jsonb;
