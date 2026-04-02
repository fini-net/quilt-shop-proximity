extends Node

## DatabaseManager Unit Tests
##
## Test suite for the DatabaseManager singleton.
## Run this by attaching to a Node in a scene and running the scene.
##
## Test Coverage:
## - Haversine distance formula accuracy and edge cases
## - Database query methods (get_all_shops, get_database_version)
## - Proximity search functionality and sorting
## - Error handling with invalid inputs

var _test_count: int = 0
var _passed_count: int = 0
var _failed_count: int = 0


func _ready() -> void:
	"""Run all tests on scene start."""
	print("\n" + "=".repeat(60))
	print("DatabaseManager Unit Tests")
	print("=".repeat(60))

	run_all_tests()

	print("\n" + "=".repeat(60))
	print("Test Results: %d passed, %d failed out of %d tests" % [_passed_count, _failed_count, _test_count])
	print("=".repeat(60) + "\n")

	if _failed_count == 0:
		print("✓ ALL TESTS PASSED!")
	else:
		push_error("Some tests failed - see output above")


func run_all_tests() -> void:
	"""Execute all test suites."""
	test_haversine_distance()
	test_get_all_shops()
	test_get_database_version()
	test_get_shops_near()
	test_error_handling()


# Test Assertion Helper
func assert_true(condition: bool, message: String) -> void:
	"""Assert that condition is true."""
	_test_count += 1
	if condition:
		_passed_count += 1
		print("  ✓ " + message)
	else:
		_failed_count += 1
		print("  ✗ FAILED: " + message)
		push_error("Test failed: " + message)


func assert_approx_equal(actual: float, expected: float, tolerance: float, message: String) -> void:
	"""Assert that actual is approximately equal to expected within tolerance."""
	var diff = abs(actual - expected)
	assert_true(diff <= tolerance, message + " (expected: %.2f, got: %.2f, diff: %.2f)" % [expected, actual, diff])


# Test Suites

func test_haversine_distance() -> void:
	"""Test Haversine distance formula implementation."""
	print("\n--- Testing haversine_distance ---")

	# Test 1: Known distance - Berkeley, CA to San Francisco, CA
	# Berkeley: 37.8716°N, 122.2727°W
	# San Francisco: 37.7749°N, 122.4194°W
	# Expected distance: ~12.4 miles
	var berkeley_lat = 37.8716
	var berkeley_lon = -122.2727
	var sf_lat = 37.7749
	var sf_lon = -122.4194

	var distance = DatabaseManager.haversine_distance(
		berkeley_lat, berkeley_lon, sf_lat, sf_lon
	)

	assert_approx_equal(distance, 12.4, 0.5,
		"Berkeley to SF distance should be ~12.4 miles")

	# Test 2: Same point should give 0 distance
	var zero_distance = DatabaseManager.haversine_distance(
		berkeley_lat, berkeley_lon, berkeley_lat, berkeley_lon
	)

	assert_approx_equal(zero_distance, 0.0, 0.001,
		"Distance from point to itself should be 0")

	# Test 3: Known distance - Los Angeles to San Diego
	# Los Angeles: 34.0522°N, 118.2437°W
	# San Diego: 32.7157°N, 117.1611°W
	# Expected distance: ~120 miles
	var la_lat = 34.0522
	var la_lon = -118.2437
	var sd_lat = 32.7157
	var sd_lon = -117.1611

	var la_to_sd = DatabaseManager.haversine_distance(
		la_lat, la_lon, sd_lat, sd_lon
	)

	assert_approx_equal(la_to_sd, 120.0, 5.0,
		"Los Angeles to San Diego distance should be ~120 miles")

	# Test 4: Symmetry - distance(A,B) should equal distance(B,A)
	var ab = DatabaseManager.haversine_distance(berkeley_lat, berkeley_lon, sf_lat, sf_lon)
	var ba = DatabaseManager.haversine_distance(sf_lat, sf_lon, berkeley_lat, berkeley_lon)

	assert_approx_equal(ab, ba, 0.001,
		"Distance should be symmetric (A to B == B to A)")


func test_get_all_shops() -> void:
	"""Test get_all_shops database query."""
	print("\n--- Testing get_all_shops ---")

	var shops = DatabaseManager.get_all_shops()

	# Test 1: Should return shops
	assert_true(shops.size() > 0, "Should have shops in database")

	# Test 2: Should have exactly 60 shops (as per production database)
	assert_true(shops.size() == 60, "Should have exactly 60 shops")

	# Test 3: Verify structure of first shop
	if shops.size() > 0:
		var shop = shops[0]
		assert_true(shop.has("id"), "Shop should have 'id' field")
		assert_true(shop.has("name"), "Shop should have 'name' field")
		assert_true(shop.has("address"), "Shop should have 'address' field")
		assert_true(shop.has("city"), "Shop should have 'city' field")
		assert_true(shop.has("state"), "Shop should have 'state' field")
		assert_true(shop.has("latitude"), "Shop should have 'latitude' field")
		assert_true(shop.has("longitude"), "Shop should have 'longitude' field")
		assert_true(shop.has("phone"), "Shop should have 'phone' field")
		assert_true(shop.has("email"), "Shop should have 'email' field")
		assert_true(shop.has("website"), "Shop should have 'website' field")

	# Test 4: All shops should have valid coordinates
	var valid_coords = true
	for shop in shops:
		if not (is_finite(shop.latitude) and is_finite(shop.longitude)):
			valid_coords = false
			break
	assert_true(valid_coords, "All shops should have valid latitude and longitude")

	# Test 5: States should be CA or VA
	var valid_states = true
	for shop in shops:
		if shop.state != "CA" and shop.state != "VA":
			valid_states = false
			break
	assert_true(valid_states, "All shops should be in CA or VA")


func test_get_database_version() -> void:
	"""Test get_database_version method."""
	print("\n--- Testing get_database_version ---")

	var version = DatabaseManager.get_database_version()

	assert_true(version == "1.0.0",
		"Database version should be '1.0.0', got: '" + version + "'")


func test_get_shops_near() -> void:
	"""Test proximity search functionality."""
	print("\n--- Testing get_shops_near ---")

	# Test 1: Find shops near Berkeley, CA (20 mile radius)
	var berkeley_lat = 37.8716
	var berkeley_lon = -122.2727
	var nearby = DatabaseManager.get_shops_near(berkeley_lat, berkeley_lon, 20.0)

	assert_true(nearby.size() > 0, "Should find shops near Berkeley within 20 miles")

	# Test 2: All results should have distance field
	var all_have_distance = true
	for shop in nearby:
		if not shop.has("distance"):
			all_have_distance = false
			break
	assert_true(all_have_distance, "All results should include 'distance' field")

	# Test 3: All shops should be within the specified radius
	var all_within_radius = true
	for shop in nearby:
		if shop.distance > 20.0:
			all_within_radius = false
			print("    Found shop outside radius: %s at %.2f miles" % [shop.name, shop.distance])
			break
	assert_true(all_within_radius, "All shops should be within 20 mile radius")

	# Test 4: Results should be sorted by distance (ascending)
	var is_sorted = true
	for i in range(nearby.size() - 1):
		if nearby[i].distance > nearby[i + 1].distance:
			is_sorted = false
			break
	assert_true(is_sorted, "Results should be sorted by distance (nearest first)")

	# Test 5: Smaller radius should find fewer or equal shops
	var close_shops = DatabaseManager.get_shops_near(berkeley_lat, berkeley_lon, 5.0)
	assert_true(close_shops.size() <= nearby.size(),
		"5 mile radius should find fewer or equal shops than 20 mile radius")

	# Test 6: All close shops should be in the larger result set
	var all_close_in_nearby = true
	for close_shop in close_shops:
		var found = false
		for nearby_shop in nearby:
			if close_shop.id == nearby_shop.id:
				found = true
				break
		if not found:
			all_close_in_nearby = false
			break
	assert_true(all_close_in_nearby,
		"All shops within 5 miles should also be within 20 miles")

	# Test 7: Middle of Pacific Ocean should find no shops
	var ocean_lat = 30.0
	var ocean_lon = -140.0
	var ocean_shops = DatabaseManager.get_shops_near(ocean_lat, ocean_lon, 100.0)
	assert_true(ocean_shops.size() == 0,
		"Middle of Pacific Ocean should have no nearby shops")

	# Test 8: Very large radius should find all shops
	var all_shops_count = DatabaseManager.get_all_shops().size()
	var large_radius = DatabaseManager.get_shops_near(37.0, -119.0, 1000.0)
	assert_true(large_radius.size() == all_shops_count,
		"Very large radius (1000 miles) should find all %d shops" % all_shops_count)

	# Test 9: Zero radius should find only shops at exact location (none expected)
	var zero_radius = DatabaseManager.get_shops_near(berkeley_lat, berkeley_lon, 0.0)
	assert_true(zero_radius.size() == 0,
		"Zero radius should find no shops (no shop exactly at test coordinates)")


func test_error_handling() -> void:
	"""Test error handling with invalid inputs."""
	print("\n--- Testing error handling ---")

	# Test 1: Invalid coordinates (NaN) should not crash
	var result = DatabaseManager.get_shops_near(NAN, NAN, 10.0)
	assert_true(result.is_empty(),
		"Invalid coordinates (NaN) should return empty array")

	# Test 2: Negative radius should return empty array
	result = DatabaseManager.get_shops_near(37.0, -122.0, -10.0)
	assert_true(result.is_empty(),
		"Negative radius should return empty array")

	# Test 3: Infinite coordinates should not crash
	result = DatabaseManager.get_shops_near(INF, -INF, 10.0)
	assert_true(result.is_empty(),
		"Infinite coordinates should return empty array")

	# Test 4: Haversine with NaN should return 0
	var distance = DatabaseManager.haversine_distance(NAN, 0.0, 0.0, 0.0)
	assert_true(distance == 0.0,
		"Haversine with NaN should return 0.0")

	# Test 5: Haversine with infinite values should return 0
	distance = DatabaseManager.haversine_distance(INF, 0.0, 0.0, 0.0)
	assert_true(distance == 0.0,
		"Haversine with INF should return 0.0")

	# Test 6: Very large valid coordinates (should work)
	result = DatabaseManager.get_shops_near(89.0, 179.0, 10.0)
	assert_true(result is Array,
		"Very large valid coordinates should return array (even if empty)")
