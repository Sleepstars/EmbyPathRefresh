CREATE TABLE IF NOT EXISTS file_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_path TEXT NOT NULL,
    target_path TEXT NOT NULL,
    modified_time DATETIME NOT NULL,
    processed_time DATETIME,
    delete_scheduled DATETIME,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_records_status ON file_records(status);
CREATE INDEX IF NOT EXISTS idx_file_records_source_path ON file_records(source_path);
CREATE UNIQUE INDEX IF NOT EXISTS idx_file_records_source_path_unique ON file_records(source_path) WHERE status != 'deleted';
