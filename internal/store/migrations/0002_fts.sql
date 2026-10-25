-- 0002: message search via FTS5 (WL-2.7). Never edit after ship.
CREATE VIRTUAL TABLE messages_fts USING fts5(body, content='messages', content_rowid='rowid', tokenize='unicode61');
INSERT INTO messages_fts(rowid, body) SELECT rowid, body FROM messages;
CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, body) VALUES (new.rowid, new.body);
END;
CREATE TRIGGER messages_ad AFTER DELETE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, body) VALUES('delete', old.rowid, old.body);
END;
CREATE TRIGGER messages_au AFTER UPDATE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, body) VALUES('delete', old.rowid, old.body);
  INSERT INTO messages_fts(rowid, body) VALUES (new.rowid, new.body);
END;
