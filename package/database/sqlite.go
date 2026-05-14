package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitSQLite(path string) {
	var err error
	DB, err = sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("failed to open SQLite database: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("failed to ping SQLite database: %v", err)
	}

	createTables := `
	CREATE TABLE IF NOT EXISTS mangas (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		author TEXT,
		artist TEXT,
		genres TEXT,               -- stored as JSON array string
		chapter_count INTEGER,
		volume_count INTEGER,
		published_year INTEGER,
		status TEXT,
		popularity INTEGER,
		ranking INTEGER,
		cover_url TEXT,
		description TEXT
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
 		username TEXT UNIQUE NOT NULL,
 		password_hash TEXT NOT NULL,
 		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		role TEXT DEFAULT 'user'
	);
	CREATE TABLE IF NOT EXISTS reading_list (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	user_id TEXT NOT NULL,
    	manga_id TEXT NOT NULL,
    	current_chapter INTEGER,
		volume INTEGER,
		notes TEXT,
    	status TEXT CHECK(status IN ('reading', 'completed', 'plan_to_read')),
    	last_updated TEXT,
    	FOREIGN KEY (user_id) REFERENCES users(id),
    	FOREIGN KEY (manga_id) REFERENCES mangas(id)
	);
	CREATE TABLE IF NOT EXISTS notifications (
		user_id INTEGER NOT NULL PRIMARY KEY,
		client_udp_addr TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
		);
	CREATE TABLE IF NOT EXISTS subscriptions (
		user_id INTEGER NOT NULL,
		manga_id TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (manga_id) REFERENCES mangas(id)
	);
	CREATE TABLE IF NOT EXISTS reading_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		manga_id TEXT NOT NULL,
		chapter INTEGER NOT NULL,
		date_read DATE NOT NULL,

		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (manga_id) REFERENCES mangas(id)
	);
	CREATE TABLE IF NOT EXISTS sync_state (
    user_id INTEGER PRIMARY KEY,
    last_synced_at DATETIME NOT NULL
);
	CREATE TABLE IF NOT EXISTS chat_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		message TEXT NOT NULL,
		type TEXT NOT NULL,
		online INTEGER,
		timestamp INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_chat_room_time
	ON chat_messages(room, timestamp DESC);


	`
	if _, err := DB.Exec(createTables); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	log.Println("✅ SQLite connected and table ready at", path)
}

func Close() {
	if DB != nil {
		DB.Close()
		log.Println("SQLite connection closed.")
	}
}
