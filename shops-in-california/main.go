package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chicks-net/quilt-shop-proximity/geocode"
	_ "modernc.org/sqlite"
)

const (
	quiltShopsURL = "https://ronatheribbiter.com/quilt-shops-california/"
	dbPath        = "quilt_shops.db"
)

// QuiltShop represents a quilt shop entry
type QuiltShop struct {
	Name    string
	Address string
	City    string
	Phone   string
	Email   string
}

// seenShops tracks shops we've already added to prevent duplicates
var seenShops = make(map[string]bool)

func main() {
	// Check for geocode command
	if len(os.Args) > 1 && os.Args[1] == "geocode" {
		log.Println("Starting geocoding process...")
		if err := geocodeShops(); err != nil {
			log.Fatalf("Error geocoding shops: %v", err)
		}
		log.Println("Geocoding complete!")
		return
	}

	// Fetch the webpage
	log.Println("Fetching quilt shops data...")
	shops, err := fetchQuiltShops()
	if err != nil {
		log.Fatalf("Error fetching quilt shops: %v", err)
	}

	log.Printf("Found %d quilt shops\n", len(shops))

	// Create database and store data
	log.Println("Creating SQLite database...")
	if err := createDatabase(shops); err != nil {
		log.Fatalf("Error creating database: %v", err)
	}

	log.Printf("Successfully created %s with %d quilt shops\n", dbPath, len(shops))
}

// fetchQuiltShops scrapes the quilt shops from the website
func fetchQuiltShops() ([]QuiltShop, error) {
	resp, err := http.Get(quiltShopsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var shops []QuiltShop

	// Skip list - common non-shop strings to ignore
	skipStrings := map[string]bool{
		"click here":                         true,
		"related posts":                      true,
		"quilt shop lists":                   true,
		"list of quilt shows":                true,
		"quilt shops":                        true,
		"find a quilt shop":                  true,
		"california":                         true,
		"big quilter's bucket list":          true,
		"planning your next quilting adventure": true,
		"travel tips for your next road trip": true,
		"create a realistic road trip budget": true,
		"traveling quilters group":           true,
		"facebook":                           true,
		"more on the blog":                   true,
		"from the e-store":                   true,
		"quilt shop lists in the us":         true,
	}

	// Track cities - find all h3 headers and process shops after each one
	doc.Find("h3").Each(func(i int, h3 *goquery.Selection) {
		cityText := strings.TrimSpace(strings.ToLower(h3.Text()))

		// Get following siblings until we hit the next h3
		// Multiple divs may contain shops for the same city
		h3.NextAll().EachWithBreak(func(j int, sibling *goquery.Selection) bool {
			// Check if THIS element is an h3 (next city header)
			if goquery.NodeName(sibling) == "h3" {
				return false // Stop iteration
			}

			// Check if this element contains an h3 child (next city section)
			if sibling.Find("h3").Length() > 0 {
				return false // Stop iteration
			}

			// Process all pre.wp-block-verse within this sibling
			sibling.Find("pre.wp-block-verse").Each(func(k int, pre *goquery.Selection) {
				pre.Find("strong").Each(func(l int, strong *goquery.Selection) {
					shopName := strings.TrimSpace(strong.Text())

					if shopName == "" || skipStrings[strings.ToLower(shopName)] {
						return
					}

					shop := parseShopFromPre(pre.Text(), shopName, cityText)
					shopKey := strings.ToLower(shop.Name) + "|" + strings.ToLower(shop.City)

					if shop.Name != "" && (shop.Address != "" || shop.Phone != "") && !seenShops[shopKey] {
						seenShops[shopKey] = true
						shops = append(shops, shop)
					}
				})
			})

			// Continue to next sibling
			return true
		})
	})

	return shops, nil
}

// parseShopFromPre parses a shop entry from a pre block's text content
func parseShopFromPre(preText, shopName, city string) QuiltShop {
	shop := QuiltShop{
		Name: shopName,
		City: city,
	}

	// Split the text into lines
	lines := strings.Split(preText, "\n")

	// Find the line with the shop name, then get the next few lines
	foundShop := false
	linesAfterShop := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Found the shop name
		if strings.Contains(line, shopName) {
			foundShop = true
			continue
		}

		if !foundShop {
			continue
		}

		linesAfterShop++

		// Stop after processing 4 lines after the shop name (increased from 3)
		// Format is typically: address, phone, email, [optional website/notes]
		if linesAfterShop > 4 {
			break
		}

		// Classify this line - order matters!
		// Check email first (most specific pattern)
		if isEmail(line) {
			shop.Email = line
		} else if isPhone(line) {
			shop.Phone = line
		} else if shop.Address == "" {
			// First non-email, non-phone line is the address
			shop.Address = line
		}
		// Ignore subsequent lines that don't match known patterns
	}

	return shop
}

// isEmail checks if a string looks like an email address
func isEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

// isPhone checks if a string looks like a phone number
func isPhone(s string) bool {
	// Remove common phone number characters
	cleaned := strings.ReplaceAll(s, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "+", "") // International prefix

	// Phone number must be ONLY digits after cleaning
	// Valid lengths: 10 (US), 11 (with country code)
	if len(cleaned) < 10 || len(cleaned) > 11 {
		return false
	}

	// Every character must be a digit
	for _, c := range cleaned {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

// createDatabase creates the SQLite database and populates it with shop data
func createDatabase(shops []QuiltShop) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Create table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS quilt_shops (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		address TEXT,
		city TEXT NOT NULL,
		phone TEXT,
		email TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_city ON quilt_shops(city);
	CREATE INDEX IF NOT EXISTS idx_name ON quilt_shops(name);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Insert data
	insertSQL := `INSERT INTO quilt_shops (name, address, city, phone, email) VALUES (?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, shop := range shops {
		if _, err := stmt.Exec(shop.Name, shop.Address, shop.City, shop.Phone, shop.Email); err != nil {
			log.Printf("Warning: failed to insert shop %s: %v", shop.Name, err)
		}
	}

	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// geocodeShops adds GPS coordinates to shops in the database
func geocodeShops() error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Apply schema migration
	// SQLite doesn't support IF NOT EXISTS with ALTER TABLE, so we try to add columns
	// and ignore errors if they already exist
	db.Exec("ALTER TABLE quilt_shops ADD COLUMN latitude REAL")
	db.Exec("ALTER TABLE quilt_shops ADD COLUMN longitude REAL")
	db.Exec("ALTER TABLE quilt_shops ADD COLUMN geocode_attempted_at DATETIME")

	// Create index for coordinates
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_coordinates ON quilt_shops(latitude, longitude)"); err != nil {
		log.Printf("Warning: failed to create index: %v", err)
	}

	// Query shops that need geocoding
	rows, err := db.Query(`
		SELECT id, name, address, city
		FROM quilt_shops
		WHERE latitude IS NULL
		ORDER BY id
	`)
	if err != nil {
		return fmt.Errorf("failed to query shops: %w", err)
	}
	defer rows.Close()

	// Collect shops to geocode
	type shopToGeocode struct {
		ID      int
		Name    string
		Address string
		City    string
	}
	var shops []shopToGeocode
	for rows.Next() {
		var shop shopToGeocode
		if err := rows.Scan(&shop.ID, &shop.Name, &shop.Address, &shop.City); err != nil {
			log.Printf("Warning: failed to scan shop: %v", err)
			continue
		}
		shops = append(shops, shop)
	}

	if len(shops) == 0 {
		log.Println("No shops need geocoding. All done!")
		return nil
	}

	log.Printf("Geocoding %d shops...\n", len(shops))

	// Geocode each shop
	for i, shop := range shops {
		log.Printf("[%d/%d] %s", i+1, len(shops), shop.Name)
		log.Printf("       %s", shop.Address)

		// Skip if no address
		if shop.Address == "" {
			log.Printf("       ⚠ Skipping - no address on file")
			// Still update the attempted timestamp
			db.Exec("UPDATE quilt_shops SET geocode_attempted_at = ? WHERE id = ?", time.Now(), shop.ID)
			continue
		}

		// Geocode the address
		result := geocode.GeocodeAddress(shop.Address)

		if result.Error != nil {
			log.Printf("       ✗ Failed: %v", result.Error)
			// Update attempted timestamp
			db.Exec("UPDATE quilt_shops SET geocode_attempted_at = ? WHERE id = ?", time.Now(), shop.ID)
			continue
		}

		// Update database with coordinates
		_, err := db.Exec(`
			UPDATE quilt_shops
			SET latitude = ?, longitude = ?, geocode_attempted_at = ?
			WHERE id = ?
		`, result.Coords.Latitude, result.Coords.Longitude, time.Now(), shop.ID)

		if err != nil {
			log.Printf("       ✗ Failed to update database: %v", err)
		} else {
			log.Printf("       ✓ %.4f, %.4f", result.Coords.Latitude, result.Coords.Longitude)
		}
	}

	return nil
}
