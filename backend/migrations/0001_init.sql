CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS rules (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    start_url TEXT NOT NULL,
    item_selector TEXT NOT NULL DEFAULT '',
    link_selector TEXT NOT NULL DEFAULT '',
    fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    respect_robots BOOLEAN NOT NULL DEFAULT TRUE,
    qps DOUBLE PRECISION NOT NULL DEFAULT 2,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    rule_id BIGINT NOT NULL REFERENCES rules(id),
    seed_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    max_depth INTEGER NOT NULL DEFAULT 1,
    concurrency INTEGER NOT NULL DEFAULT 2,
    status TEXT NOT NULL DEFAULT 'created',
    stats JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_rule_id ON tasks(rule_id);

CREATE TABLE IF NOT EXISTS crawl_results (
    id BIGSERIAL PRIMARY KEY,
    task_id BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_results_task_id ON crawl_results(task_id);
CREATE INDEX IF NOT EXISTS idx_results_created_at ON crawl_results(created_at);

CREATE TABLE IF NOT EXISTS worker_nodes (
    id TEXT PRIMARY KEY,
    role TEXT NOT NULL DEFAULT 'worker',
    cpu DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_mb DOUBLE PRECISION NOT NULL DEFAULT 0,
    pages_per_min DOUBLE PRECISION NOT NULL DEFAULT 0,
    fail_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'unknown',
    last_seen TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    action TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at);
