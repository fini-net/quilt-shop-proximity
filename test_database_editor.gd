@tool
extends EditorScript

func _run():
	test_database_queries()

func test_database_queries():
	# Create SQLite instance
	var db = SQLite.new()
	
	# Open the production database
	var db_path = "res://data/quilt_shops.db"
	db.path = db_path
	db.open_db()
	
	# Test Query 1: Count total shops
	db.query("SELECT COUNT(*) as total FROM quilt_shops;")
	print("Total shops: ", db.query_result)
	
	# Test Query 2: Get shops by state
	db.query("SELECT state, COUNT(*) as count FROM quilt_shops GROUP BY state;")
	print("Shops by state: ", db.query_result)
	
	# Test Query 3: Get a few sample shops with coordinates
	db.query("SELECT name, city, state, latitude, longitude FROM quilt_shops LIMIT 5;")
	print("Sample shops: ", db.query_result)
	
	# Test Query 4: Get shops in a specific city (e.g., Berkeley)
	db.query("SELECT name, address, phone, website FROM quilt_shops WHERE city = 'Berkeley';")
	print("Berkeley shops: ", db.query_result)
	
	db.close_db()
	print("Database tests completed!")
