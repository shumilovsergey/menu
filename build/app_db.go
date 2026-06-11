package main

// app_db.go — app-specific database migrations.
// Add your tables here. initDB() calls appMigrate() after the core users table.

func appMigrate() error {
	// Example — uncomment and edit for your app:
	// _, err := db.Exec(`CREATE TABLE IF NOT EXISTS items (
	// 	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	// 	user_id    INTEGER NOT NULL REFERENCES users(id),
	// 	name       TEXT    NOT NULL,
	// 	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	// )`)
	// return err
	return nil
}
