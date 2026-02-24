package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"ldlcscraper.com/models"
)

func InitDatabase(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title text,
		price text,
		link TEXT UNIQUE,
		pic TEXT,
		description TEXT,
		stock TEXT,
		category TEXT,
		sub_category TEXT,
		source TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
	`)

	if err != nil {
		return nil, err
	}

	return db, nil
}

func InsertProduct(db *sql.DB, p models.Product) error {
	_, err := db.Exec(
		`INSERT OR IGNORE INTO products (title, price, link, pic, description, stock, category, sub_category, source) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Title, p.Price, p.Link, p.Pic, p.Desc, p.Stock, p.Category, p.SubCategory, p.Source,
	)

	return err
}
