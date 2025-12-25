package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/chicks-net/quilt-shop-proximity/geocode"
	_ "modernc.org/sqlite"
)

const (
	quiltShopsPDF = "virginia-quilt-shops.pdf"
	pdfURL        = "https://vcq.org/wp-content/uploads/2025/03/2025_3-V1.0-Quilt-Shop-List.pdf"
	dbPath        = "quilt_shops.db"
)

// QuiltShop represents a quilt shop entry
type QuiltShop struct {
	Name    string
	Address string
	City    string
	Phone   string
	Email   string
	Website string
}

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

	// Download PDF if it doesn't exist
	if _, err := os.Stat(quiltShopsPDF); os.IsNotExist(err) {
		log.Println("Downloading Virginia quilt shops PDF...")
		if err := downloadPDF(); err != nil {
			log.Fatalf("Error downloading PDF: %v", err)
		}
	}

	// Parse the PDF
	log.Println("Parsing quilt shops from PDF...")
	shops, err := parseQuiltShopsPDF()
	if err != nil {
		log.Fatalf("Error parsing PDF: %v", err)
	}

	log.Printf("Found %d quilt shops\n", len(shops))

	// Create database and store data
	log.Println("Creating SQLite database...")
	if err := createDatabase(shops); err != nil {
		log.Fatalf("Error creating database: %v", err)
	}

	log.Printf("Successfully created %s with %d quilt shops\n", dbPath, len(shops))
}

// downloadPDF downloads the PDF file from the URL
func downloadPDF() error {
	resp, err := http.Get(pdfURL)
	if err != nil {
		return fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(quiltShopsPDF)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// parseQuiltShopsPDF extracts text from the PDF using pdftotext and parses shop information
func parseQuiltShopsPDF() ([]QuiltShop, error) {
	// Use pdftotext command line tool for better line break preservation
	cmd := exec.Command("pdftotext", quiltShopsPDF, "-")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run pdftotext: %w (make sure pdftotext is installed)", err)
	}

	// Parse the extracted text
	return parseShopsFromText(out.String()), nil
}

// parseShopsFromText parses shop entries from the extracted text
func parseShopsFromText(text string) []QuiltShop {
	var shops []QuiltShop

	// Regular expressions for pattern matching
	phoneRegex := regexp.MustCompile(`^\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	websiteRegex := regexp.MustCompile(`^(?:www\.|https?://)`)
	cityStateZipRegex := regexp.MustCompile(`^(.+),\s*VA\s+\d{5,6}`)

	// Skip patterns
	skipPatterns := []string{
		"Quilt Shops",
		"2025-V1.0",
	}

	// Not city names - common words in descriptions that might look like cities
	notCityNames := map[string]bool{
		"Closed Sunday":    true,
		"Events":           true,
		"Hours":            true,
		"Classes":          true,
		"Services":         true,
		"Machines":         true,
		"Founded":          true,
		"Located":          true,
		"Open":             true,
		"Spreading":        true,
		"Emily Isaman":     true,
		"Owner":            true,
		"Becky Garriner":   true,
		"Louann Gram":      true,
		"Authorized":       true,
	}

	// State machine states
	const (
		lookingForCity = iota
		expectingShopName
		collectingAddress
		collectingContactInfo
	)

	scanner := bufio.NewScanner(strings.NewReader(text))
	state := lookingForCity
	var currentCity string
	var currentShop *QuiltShop
	var addressLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip headers
		skip := false
		for _, pattern := range skipPatterns {
			if line == pattern {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Check if this is city, state, zip - this marks end of address
		if cityStateZipRegex.MatchString(line) {
			if currentShop != nil && len(addressLines) > 0 {
				currentShop.Address = strings.Join(addressLines, ", ")
				addressLines = nil
				state = collectingContactInfo
			}
			continue
		}

		// Check if this is a phone number
		if phoneRegex.MatchString(line) {
			if currentShop != nil && currentShop.Phone == "" {
				currentShop.Phone = line
			}
			continue
		}

		// Check if this is an email
		if emailRegex.MatchString(line) {
			if currentShop != nil && currentShop.Email == "" {
				currentShop.Email = line
			}
			continue
		}

		// Check if this is a website
		if websiteRegex.MatchString(line) {
			if currentShop != nil && currentShop.Website == "" {
				currentShop.Website = line
			}
			continue
		}

		// State machine logic
		words := strings.Fields(line)
		isShortTitleCase := len(words) <= 3 && len(words) > 0 &&
			!strings.Contains(line, ",") &&
			!regexp.MustCompile(`\d`).MatchString(line) &&
			!strings.Contains(strings.ToLower(line), "suite") &&
			!strings.Contains(strings.ToLower(line), "shopping") &&
			len(line) > 0 && line[0] >= 'A' && line[0] <= 'Z'

		switch state {
		case lookingForCity, collectingContactInfo:
			// We're looking for a city header or finished with a shop
			if isShortTitleCase && !notCityNames[line] {
				// Save previous shop if exists
				if currentShop != nil && currentShop.Name != "" && currentShop.City != "" {
					shops = append(shops, *currentShop)
				}

				currentCity = line
				currentShop = nil
				addressLines = nil
				state = expectingShopName
			} else if state == collectingContactInfo {
				// This is extra info after contact info, ignore it
				// Stay in collectingContactInfo state
			}

		case expectingShopName:
			// The next line after a city header must be the shop name
			currentShop = &QuiltShop{
				Name: line,
				City: currentCity,
			}
			state = collectingAddress

		case collectingAddress:
			// Collect address lines until we hit city,state,zip (handled above)
			addressLines = append(addressLines, line)
		}
	}

	// Don't forget the last shop
	if currentShop != nil && currentShop.Name != "" && currentShop.City != "" {
		if len(addressLines) > 0 {
			currentShop.Address = strings.Join(addressLines, ", ")
		}
		shops = append(shops, *currentShop)
	}

	return shops
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
		website TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_city ON quilt_shops(city);
	CREATE INDEX IF NOT EXISTS idx_name ON quilt_shops(name);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Insert data
	insertSQL := `INSERT INTO quilt_shops (name, address, city, phone, email, website) VALUES (?, ?, ?, ?, ?, ?)`
	stmt, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, shop := range shops {
		if _, err := stmt.Exec(shop.Name, shop.Address, shop.City, shop.Phone, shop.Email, shop.Website); err != nil {
			log.Printf("Warning: failed to insert shop %s: %v", shop.Name, err)
		}
	}

	return nil
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

		// Skip if no address
		if shop.Address == "" {
			log.Printf("       ⚠ Skipping - no address on file")
			// Still update the attempted timestamp
			db.Exec("UPDATE quilt_shops SET geocode_attempted_at = ? WHERE id = ?", time.Now(), shop.ID)
			continue
		}

		// Normalize VA address - remove trailing commas and add city, state
		fullAddress := strings.TrimSpace(shop.Address)
		fullAddress = strings.TrimSuffix(fullAddress, ",")
		fullAddress = fmt.Sprintf("%s, %s, VA", fullAddress, shop.City)

		log.Printf("       %s", fullAddress)

		// Geocode the address
		result := geocode.GeocodeAddress(fullAddress)

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
