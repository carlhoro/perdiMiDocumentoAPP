package db

const schema = `
CREATE TABLE IF NOT EXISTS users (
 id INTEGER PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
 token_hash TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS documents (
 id INTEGER PRIMARY KEY, number TEXT NOT NULL, type TEXT NOT NULL, owner_name TEXT NOT NULL,
 address TEXT NOT NULL, reported_by INTEGER NOT NULL REFERENCES users(id), created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS documents_number ON documents(number);
CREATE TABLE IF NOT EXISTS subscriptions (
 user_id INTEGER NOT NULL REFERENCES users(id), number TEXT NOT NULL, PRIMARY KEY(user_id, number)
);
CREATE INDEX IF NOT EXISTS subscriptions_number ON subscriptions(number);
CREATE TABLE IF NOT EXISTS notifications (
 id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id),
 document_id INTEGER NOT NULL REFERENCES documents(id), created_at INTEGER NOT NULL,
 UNIQUE(user_id, document_id)
);
CREATE TABLE IF NOT EXISTS logs (
 id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id), document_number TEXT NOT NULL,
 document_id INTEGER REFERENCES documents(id), operation TEXT NOT NULL, created_at INTEGER NOT NULL
);
`
