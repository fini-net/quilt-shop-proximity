package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/ledongthuc/pdf"
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

// parseQuiltShopsPDF extracts text from the PDF and parses shop information
func parseQuiltShopsPDF() ([]QuiltShop, error) {
	f, r, err := pdf.Open(quiltShopsPDF)
	if err != nil {
		return nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var fullText strings.Builder
	totalPages := r.NumPage()

	// Extract text from all pages
	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			log.Printf("Warning: failed to extract text from page %d: %v", pageIndex, err)
			continue
		}

		fullText.WriteString(text)
		fullText.WriteString("\n")
	}

	// Parse the extracted text
	return parseShopsFromText(fullText.String()), nil
}

// parseShopsFromText parses shop entries from the extracted text
func parseShopsFromText(text string) []QuiltShop {
	var shops []QuiltShop

	// Regular expressions for pattern matching
	phoneRegex := regexp.MustCompile(`\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}(?:\s+extension\s+\d+)?`)
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	websiteRegex := regexp.MustCompile(`(?i)(?:www\.[^\s]+|https?://[^\s]+)`)
	cityStateZipRegex := regexp.MustCompile(`([A-Za-z\s]+),\s*VA\s+\d{5,6}`)

	// Known city names from the PDF - these appear before shop names
	knownCities := map[string]bool{
		"Alexandria": true, "Ashburn": true, "Brookneal": true, "Capron": true,
		"Charlottesville": true, "Chesapeake": true, "Clifton Forge": true,
		"Covington": true, "Crewe": true, "Culpeper": true, "Dayton": true,
		"Fairfax": true, "Fairfield": true, "Forest": true, "Front Royal": true,
		"Glen Allen": true, "Gloucester": true, "Hampton": true, "Harrisonburg": true,
		"Herndon": true, "Leesburg": true, "Lynchburg": true, "Manassas": true,
		"Martinsville": true, "Midlothian": true, "Newport News": true, "Norfolk": true,
		"Orange": true, "Petersburg": true, "Portsmouth": true, "Powhatan": true,
		"Purcellville": true, "Radford": true, "Richmond": true, "Roanoke": true,
		"Smithfield": true, "Springfield": true, "Staunton": true, "Sterling": true,
		"Suffolk": true, "Toano": true, "Vienna": true, "Vinton": true,
		"Virginia Beach": true, "Warrenton": true, "Waynesboro": true, "Williamsburg": true,
		"Winchester": true, "Woodbridge": true, "Yorktown": true,
	}

	// Split into tokens
	tokens := strings.Fields(text)

	var i int
	for i < len(tokens) {
		// Skip headers
		if tokens[i] == "Quilt" && i+1 < len(tokens) && tokens[i+1] == "Shops" {
			i += 2
			continue
		}
		if tokens[i] == "2025-V1.0" {
			i++
			continue
		}

		// Check if this is a city name
		cityName := ""
		tokensUsed := 0

		// Try one-word city
		if knownCities[tokens[i]] {
			cityName = tokens[i]
			tokensUsed = 1
		} else if i+1 < len(tokens) {
			// Try two-word city
			twoWord := tokens[i] + " " + tokens[i+1]
			if knownCities[twoWord] {
				cityName = twoWord
				tokensUsed = 2
			}
		}

		if cityName == "" {
			i++
			continue
		}

		i += tokensUsed

		// Now extract shop data
		shop := QuiltShop{City: cityName}

		// Collect tokens until we hit the next city name or city,state,zip pattern
		var shopTokens []string
		for i < len(tokens) {
			// Check if we've hit the next city
			if knownCities[tokens[i]] {
				break
			}
			if i+1 < len(tokens) && knownCities[tokens[i]+" "+tokens[i+1]] {
				break
			}

			shopTokens = append(shopTokens, tokens[i])
			i++

			// Check if we have enough tokens to form a complete shop
			if len(shopTokens) > 50 {
				break
			}
		}

		// Parse the shop tokens
		shopText := strings.Join(shopTokens, " ")

		// Extract city, state, zip to find where address ends
		cityStateZipMatches := cityStateZipRegex.FindStringIndex(shopText)
		if cityStateZipMatches == nil {
			continue // No valid address found
		}

		// Everything before city,state,zip is name + address
		beforeCityStateZip := shopText[:cityStateZipMatches[0]]
		afterCityStateZip := shopText[cityStateZipMatches[1]:]

		// Split name from address - shop name is typically the first part
		// until we hit something that looks like a street address
		nameParts := []string{}
		addressParts := []string{}
		foundAddress := false

		words := strings.Fields(beforeCityStateZip)
		for _, word := range words {
			// Check if this looks like start of address (number or common street indicators)
			if !foundAddress && (regexp.MustCompile(`^\d+`).MatchString(word) ||
				strings.HasSuffix(strings.ToLower(word), "shopping") ||
				strings.HasSuffix(strings.ToLower(word), "centre,")) {
				foundAddress = true
			}

			if foundAddress {
				addressParts = append(addressParts, word)
			} else {
				nameParts = append(nameParts, word)
			}
		}

		shop.Name = strings.TrimSpace(strings.Join(nameParts, " "))
		shop.Address = strings.TrimSpace(strings.Join(addressParts, " "))

		// Remove trailing comma from address if present
		shop.Address = strings.TrimSuffix(shop.Address, ",")

		// Extract phone, email, website from after city,state,zip
		phoneMatches := phoneRegex.FindAllString(afterCityStateZip, -1)
		if len(phoneMatches) > 0 {
			shop.Phone = phoneMatches[0]
		}

		emailMatches := emailRegex.FindAllString(afterCityStateZip, -1)
		if len(emailMatches) > 0 {
			shop.Email = emailMatches[0]
		}

		websiteMatches := websiteRegex.FindAllString(afterCityStateZip, -1)
		if len(websiteMatches) > 0 {
			shop.Website = websiteMatches[0]
		}

		// Only add if we have minimum required data
		if shop.Name != "" && (shop.Phone != "" || shop.Email != "" || shop.Website != "") {
			shops = append(shops, shop)
		}
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
