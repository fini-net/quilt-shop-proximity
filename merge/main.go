package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

const (
	caDatabasePath     = "../shops-in-california/quilt_shops.db"
	vaDatabasePath     = "../shops-in-virginia/quilt_shops.db"
	mergedDatabasePath = "quilt_shops.db"
)

// Shop represents a quilt shop record
type Shop struct {
	Name               string
	Address            sql.NullString
	City               string
	Phone              sql.NullString
	Email              sql.NullString
	Website            sql.NullString
	Latitude           float64
	Longitude          float64
	CreatedAt          string
	GeocodeAttemptedAt sql.NullString
}

func main() {
	// Remove existing merged database if it exists
	if err := os.Remove(mergedDatabasePath); err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to remove existing database: %v", err)
	}

	// Create new merged database
	mergedDB, err := sql.Open("sqlite", mergedDatabasePath)
	if err != nil {
		log.Fatalf("Failed to create merged database: %v", err)
	}
	defer mergedDB.Close()

	// Create schema
	if err := createSchema(mergedDB); err != nil {
		log.Fatalf("Failed to create schema: %v", err)
	}

	// Merge California shops
	caCount, err := mergeStateShops(mergedDB, caDatabasePath, "CA")
	if err != nil {
		log.Fatalf("Failed to merge CA shops: %v", err)
	}
	fmt.Printf("✅ Merged %d California shops with coordinates\n", caCount)

	// Merge Virginia shops
	vaCount, err := mergeStateShops(mergedDB, vaDatabasePath, "VA")
	if err != nil {
		log.Fatalf("Failed to merge VA shops: %v", err)
	}
	fmt.Printf("✅ Merged %d Virginia shops with coordinates\n", vaCount)

	// Verify total count
	var totalCount int
	err = mergedDB.QueryRow("SELECT COUNT(*) FROM quilt_shops").Scan(&totalCount)
	if err != nil {
		log.Fatalf("Failed to count merged shops: %v", err)
	}
	fmt.Printf("✅ Total shops in merged database: %d\n", totalCount)

	// VACUUM to optimize database
	if _, err := mergedDB.Exec("VACUUM"); err != nil {
		log.Fatalf("Failed to VACUUM database: %v", err)
	}
	fmt.Println("✅ Database optimized with VACUUM")

	fmt.Printf("\n🎉 Successfully created merged database at: %s\n", mergedDatabasePath)
}

func createSchema(db *sql.DB) error {
	schema := `
		CREATE TABLE quilt_shops (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			address TEXT,
			city TEXT NOT NULL,
			state TEXT NOT NULL,
			phone TEXT,
			email TEXT,
			website TEXT,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			geocode_attempted_at DATETIME
		);

		CREATE INDEX idx_city ON quilt_shops(city);
		CREATE INDEX idx_state ON quilt_shops(state);
		CREATE INDEX idx_coordinates ON quilt_shops(latitude, longitude);
	`
	_, err := db.Exec(schema)
	return err
}

func mergeStateShops(mergedDB *sql.DB, sourcePath, state string) (int, error) {
	// Open source database
	sourceDB, err := sql.Open("sqlite", sourcePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open %s database: %w", state, err)
	}
	defer sourceDB.Close()

	// Check if website column exists
	var query string
	var hasWebsite bool
	err = sourceDB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('quilt_shops') WHERE name='website'").Scan(&hasWebsite)
	if err != nil {
		return 0, fmt.Errorf("failed to check schema: %w", err)
	}

	// Query shops with coordinates only
	if hasWebsite {
		query = `
			SELECT name, address, city, phone, email, website, latitude, longitude, created_at, geocode_attempted_at
			FROM quilt_shops
			WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		`
	} else {
		query = `
			SELECT name, address, city, phone, email, latitude, longitude, created_at, geocode_attempted_at
			FROM quilt_shops
			WHERE latitude IS NOT NULL AND longitude IS NOT NULL
		`
	}

	rows, err := sourceDB.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query %s shops: %w", state, err)
	}
	defer rows.Close()

	// Prepare insert statement
	insertStmt, err := mergedDB.Prepare(`
		INSERT INTO quilt_shops (name, address, city, state, phone, email, website, latitude, longitude, created_at, geocode_attempted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	// Insert shops
	count := 0
	for rows.Next() {
		var shop Shop
		var err error

		if hasWebsite {
			err = rows.Scan(
				&shop.Name,
				&shop.Address,
				&shop.City,
				&shop.Phone,
				&shop.Email,
				&shop.Website,
				&shop.Latitude,
				&shop.Longitude,
				&shop.CreatedAt,
				&shop.GeocodeAttemptedAt,
			)
		} else {
			err = rows.Scan(
				&shop.Name,
				&shop.Address,
				&shop.City,
				&shop.Phone,
				&shop.Email,
				&shop.Latitude,
				&shop.Longitude,
				&shop.CreatedAt,
				&shop.GeocodeAttemptedAt,
			)
		}

		if err != nil {
			return count, fmt.Errorf("failed to scan shop: %w", err)
		}

		_, err = insertStmt.Exec(
			shop.Name,
			shop.Address,
			shop.City,
			state,
			shop.Phone,
			shop.Email,
			shop.Website,
			shop.Latitude,
			shop.Longitude,
			shop.CreatedAt,
			shop.GeocodeAttemptedAt,
		)
		if err != nil {
			return count, fmt.Errorf("failed to insert shop: %w", err)
		}
		count++
	}

	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("error iterating %s shops: %w", state, err)
	}

	return count, nil
}
