-- 0001: initial schema per plan §6. Never edit after ship.
CREATE TABLE chats(
  jid TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  avatar_path TEXT, unread INTEGER NOT NULL DEFAULT 0,
  mentions INTEGER NOT NULL DEFAULT 0,
  archived INTEGER NOT NULL DEFAULT 0, pinned INTEGER NOT NULL DEFAULT 0,
  muted_until INTEGER NOT NULL DEFAULT 0,
  ephemeral_secs INTEGER NOT NULL DEFAULT 0,
  last_msg_id TEXT, last_ts INTEGER NOT NULL DEFAULT 0,
  draft TEXT NOT NULL DEFAULT '');
CREATE TABLE messages(
  chat_jid TEXT NOT NULL REFERENCES chats(jid),
  id TEXT NOT NULL,
  sender_jid TEXT NOT NULL, is_from_me INTEGER NOT NULL,
  ts INTEGER NOT NULL,
  kind TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  quoted_id TEXT,
  media_path TEXT, media_mime TEXT, media_size INTEGER,
  media_sha256 BLOB, thumb_path TEXT, waveform BLOB,
  state TEXT NOT NULL,
  edited INTEGER NOT NULL DEFAULT 0, starred INTEGER NOT NULL DEFAULT 0,
  view_once INTEGER NOT NULL DEFAULT 0,
  raw_proto BLOB,
  PRIMARY KEY(chat_jid,id));
CREATE INDEX idx_msg_chat_ts ON messages(chat_jid, ts);
CREATE TABLE reactions(
  chat_jid TEXT, msg_id TEXT, sender_jid TEXT, emoji TEXT, ts INTEGER,
  PRIMARY KEY(chat_jid,msg_id,sender_jid));
CREATE TABLE contacts(
  jid TEXT PRIMARY KEY, lid TEXT, phone TEXT,
  push_name TEXT NOT NULL DEFAULT '', contact_name TEXT NOT NULL DEFAULT '',
  about TEXT NOT NULL DEFAULT '', avatar_path TEXT,
  blocked INTEGER NOT NULL DEFAULT 0, is_business INTEGER NOT NULL DEFAULT 0);
CREATE TABLE group_members(
  group_jid TEXT, user_jid TEXT, role TEXT NOT NULL,
  PRIMARY KEY(group_jid,user_jid));
CREATE TABLE polls(
  chat_jid TEXT, msg_id TEXT, question TEXT, options_json TEXT,
  selectable INTEGER, PRIMARY KEY(chat_jid,msg_id));
CREATE TABLE poll_votes(
  chat_jid TEXT, msg_id TEXT, voter_jid TEXT, options_json TEXT, ts INTEGER,
  PRIMARY KEY(chat_jid,msg_id,voter_jid));
CREATE TABLE calls(
  id TEXT PRIMARY KEY, chat_jid TEXT, direction TEXT,
  kind TEXT,
  started_ts INTEGER, duration_s INTEGER, outcome TEXT);
CREATE TABLE downloads(
  chat_jid TEXT, msg_id TEXT, state TEXT, bytes_done INTEGER, bytes_total INTEGER,
  PRIMARY KEY(chat_jid,msg_id));
CREATE TABLE kv(key TEXT PRIMARY KEY, value TEXT);
