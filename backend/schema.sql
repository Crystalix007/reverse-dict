CREATE TABLE words (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    word TEXT NOT NULL,
    definition TEXT NOT NULL,
    example TEXT NOT NULL
);

CREATE TABLE embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    phrase TEXT NOT NULL,
    embedding BLOB NOT NULL,
    UNIQUE (phrase)
);

CREATE TABLE word_embeddings (
    word_id INTEGER NOT NULL,
    embedding_id INTEGER NOT NULL,
    PRIMARY KEY (word_id, embedding_id),
    FOREIGN KEY (word_id) REFERENCES words (id),
    FOREIGN KEY (embedding_id) REFERENCES embeddings (id)
);
