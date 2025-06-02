CREATE TABLE IF NOT EXISTS repos (
    id BIGINT PRIMARY KEY,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    description TEXT NOT NULL,
    stars BIGINT NOT NULL,
    forks BIGINT NOT NULL,
    archived BOOLEAN NOT NULL,
    private BOOLEAN NOT NULL,
    is_fork BOOLEAN NOT NULL,
    language TEXT NOT NULL,
    size_mb REAL NOT NULL,
    updated_at TEXT NOT NULL,
    pushed_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    html_url TEXT NOT NULL,
    topics TEXT NOT NULL,
    visibility TEXT NOT NULL,
    license TEXT NOT NULL,
    open_issues BIGINT NOT NULL,
    languages_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dockerfiles (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES repos(id),
    full_name TEXT NOT NULL,
    path TEXT NOT NULL,
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS repo_languages (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES repos(id),
    language TEXT NOT NULL,
    bytes BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS dependency_files (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES repos(id),
    path TEXT NOT NULL,
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ci_configs (
    id SERIAL PRIMARY KEY,
    repo_id BIGINT NOT NULL REFERENCES repos(id),
    path TEXT NOT NULL,
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS readmes (
    repo_id BIGINT PRIMARY KEY REFERENCES repos(id),
    content TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS repo_security_features (
    repo_id BIGINT PRIMARY KEY REFERENCES repos(id),
    has_security_md BOOLEAN NOT NULL DEFAULT FALSE,
    has_dependabot BOOLEAN NOT NULL DEFAULT FALSE,
    has_codeql BOOLEAN NOT NULL DEFAULT FALSE
);
