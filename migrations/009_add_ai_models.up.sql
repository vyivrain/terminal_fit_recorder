CREATE TABLE ai_models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider TEXT NOT NULL CHECK (provider IN ('opencode', 'codex', 'claude')),
    model_id TEXT NOT NULL,
    display_name TEXT NOT NULL UNIQUE,
    is_selected INTEGER NOT NULL DEFAULT 0 CHECK (is_selected IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, model_id)
);

INSERT INTO ai_models (provider, model_id, display_name, is_selected) VALUES
    ('opencode', 'opencode-go/glm-5.3', 'GLM 5.3 · OpenCode', 1),
    ('codex', 'gpt-5.6-sol', 'GPT-5.6 Sol · Codex', 0),
    ('claude', 'opus', 'Claude Opus · Latest', 0);

CREATE UNIQUE INDEX ai_models_one_selected
ON ai_models(is_selected)
WHERE is_selected = 1;
