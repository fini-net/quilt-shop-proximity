class_name DatabaseManager
extends Node

## DatabaseManager Singleton
##
## Handles all database queries and proximity calculations for the quilt shop finder app.
## Accessible globally via autoload as DatabaseManager.
##
## Usage:
##   var shops = DatabaseManager.get_all_shops()
##   var nearby = DatabaseManager.get_shops_near(37.8716, -122.2727, 10.0)
##   var distance = DatabaseManager.haversine_distance(lat1, lon1, lat2, lon2)

# Signals
signal database_error(error_message: String)
signal database_ready()

# Constants
const EARTH_RADIUS_MILES = 3958.8  # Mean radius of Earth in miles
const EARTH_RADIUS_KM = 6371.0     # Mean radius of Earth in kilometers

# Private variables
var _db: SQLite = null
var _is_initialized: bool = false
var _db_path: String = "res://data/quilt_shops.db"


func _ready() -> void:
	"""Initialize database connection on autoload startup."""
	_initialize_database()


func _exit_tree() -> void:
	"""Clean up database connection on shutdown."""
	if _db != null:
		_db.close_db()
		_db = null
	_is_initialized = false


func _initialize_database() -> void:
	"""Open database connection and validate it's ready to use."""
	# Create SQLite instance
	_db = SQLite.new()
	_db.path = _db_path

	# Open database
	if not _db.open_db():
		push_error("DatabaseManager: Failed to open database at " + _db_path)
		database_error.emit("Failed to open database")
		return

	# Validate database by querying metadata
	_db.query("SELECT COUNT(*) as count FROM metadata;")

	if _db.query_result.is_empty():
		push_error("DatabaseManager: Database validation failed - metadata table not found")
		database_error.emit("Database validation failed")
		_db.close_db()
		_db = null
		return

	# Mark as initialized
	_is_initialized = true
	database_ready.emit()
	print("DatabaseManager: Initialized successfully")


func _ensure_database_ready() -> bool:
	"""Verify database is open and ready. Emit error signal if not."""
	if not _is_initialized:
		push_error("DatabaseManager: Database not initialized")
		database_error.emit("Database not initialized")
		return false

	if _db == null:
		push_error("DatabaseManager: Database connection is null")
		database_error.emit("Database connection is null")
		return false

	return true


func get_all_shops() -> Array:
	"""
	Get all quilt shops from the database.

	Returns:
		Array of dictionaries containing shop data with keys:
		id, name, address, city, state, phone, email, website,
		latitude, longitude, created_at, geocode_attempted_at

		Returns empty array if database is not ready or query fails.
	"""
	if not _ensure_database_ready():
		return []

	# Execute query
	_db.query("SELECT * FROM quilt_shops;")

	# Check for query errors
	if _db.query_result.is_empty():
		# Could be empty table or error - verify which
		_db.query("SELECT COUNT(*) as count FROM quilt_shops;")
		if _db.query_result.is_empty():
			push_error("DatabaseManager: Failed to query quilt_shops table")
			database_error.emit("Failed to query database")
			return []
		# Genuinely empty table
		return []

	return _db.query_result


func get_database_version() -> String:
	"""
	Get database version from metadata table.

	Returns:
		Version string (e.g., "1.0.0") or "unknown" if not found.
	"""
	if not _ensure_database_ready():
		return ""

	_db.query("SELECT value FROM metadata WHERE key = 'version';")

	if _db.query_result.is_empty():
		push_error("DatabaseManager: Version metadata not found")
		database_error.emit("Version metadata not found")
		return "unknown"

	return _db.query_result[0]["value"]


func haversine_distance(lat1: float, lon1: float, lat2: float, lon2: float) -> float:
	"""
	Calculate great-circle distance between two points on Earth using the Haversine formula.

	The Haversine formula:
	  a = sin²(Δφ/2) + cos(φ₁) · cos(φ₂) · sin²(Δλ/2)
	  c = 2 · atan2(√a, √(1−a))
	  d = R · c

	Where:
	  φ = latitude (in radians)
	  λ = longitude (in radians)
	  R = Earth's radius (3958.8 miles)

	Reference: https://www.movable-type.co.uk/scripts/latlong.html

	Args:
		lat1: Latitude of first point (degrees)
		lon1: Longitude of first point (degrees)
		lat2: Latitude of second point (degrees)
		lon2: Longitude of second point (degrees)

	Returns:
		Distance in miles. Returns 0.0 if any coordinate is NaN or infinite.
	"""
	# Validate inputs
	if not is_finite(lat1) or not is_finite(lon1) or not is_finite(lat2) or not is_finite(lon2):
		push_error("DatabaseManager: Invalid coordinates in haversine_distance")
		return 0.0

	# Convert degrees to radians
	var phi1 = deg_to_rad(lat1)
	var phi2 = deg_to_rad(lat2)
	var delta_phi = deg_to_rad(lat2 - lat1)
	var delta_lambda = deg_to_rad(lon2 - lon1)

	# Haversine formula
	var a = sin(delta_phi / 2.0) * sin(delta_phi / 2.0) + \
	        cos(phi1) * cos(phi2) * \
	        sin(delta_lambda / 2.0) * sin(delta_lambda / 2.0)

	var c = 2.0 * atan2(sqrt(a), sqrt(1.0 - a))

	# Distance in miles
	return EARTH_RADIUS_MILES * c


func get_shops_near(lat: float, lon: float, radius_miles: float) -> Array:
	"""
	Find quilt shops within a specified radius of given coordinates.

	Uses the Haversine formula to calculate accurate great-circle distances
	and filters shops within the specified radius. Results are sorted by
	distance in ascending order.

	Args:
		lat: Center point latitude (degrees)
		lon: Center point longitude (degrees)
		radius_miles: Search radius in miles

	Returns:
		Array of dictionaries with shop data plus a 'distance' field.
		Sorted by distance (nearest first).
		Returns empty array if database not ready or invalid coordinates.
	"""
	# Validate inputs
	if not is_finite(lat) or not is_finite(lon) or not is_finite(radius_miles):
		push_error("DatabaseManager: Invalid parameters in get_shops_near")
		return []

	if radius_miles < 0:
		push_error("DatabaseManager: Negative radius in get_shops_near")
		return []

	if not _ensure_database_ready():
		return []

	# Get all shops
	var all_shops = get_all_shops()
	if all_shops.is_empty():
		return []

	var nearby_shops = []

	# Calculate distance for each shop and filter by radius
	for shop in all_shops:
		var distance = haversine_distance(lat, lon, shop.latitude, shop.longitude)

		if distance <= radius_miles:
			# Create a copy of shop data with distance added
			var shop_with_distance = shop.duplicate()
			shop_with_distance["distance"] = distance
			nearby_shops.append(shop_with_distance)

	# Sort by distance (ascending - nearest first)
	nearby_shops.sort_custom(func(a, b): return a.distance < b.distance)

	return nearby_shops
