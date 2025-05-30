CREATE TABLE IF NOT EXISTS repos (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    description TEXT NOT NULL,
    stars INTEGER NOT NULL,
    forks INTEGER NOT NULL,
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
    open_issues INTEGER NOT NULL,
    languages_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dockerfiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    full_name TEXT NOT NULL,
    content TEXT NOT NULL,
    FOREIGN KEY (repo_id) REFERENCES repos(id)
);

CREATE TABLE IF NOT EXISTS repo_languages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id INTEGER NOT NULL,
    language TEXT NOT NULL,
    bytes INTEGER NOT NULL,
    FOREIGN KEY (repo_id) REFERENCES repos(id)
);
