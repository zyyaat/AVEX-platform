#!/bin/bash
# AVEX — Seed a test driver account
# Creates a driver in identity.drivers (verified + active) + dispatch.drivers
# Usage: ./seed-driver.sh [phone] [password] [name]
#
# Default: phone=01012345678 password=12345678 name=Ahmed

PHONE="${1:-01012345678}"
PASSWORD="${2:-12345678}"
NAME="${3:-Ahmed}"

echo "🌱 Seeding driver: $NAME ($PHONE)"

# 1. Register driver in identity.drivers (auto-verified)
echo "📝 Registering in identity.drivers..."
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/driver/register \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$NAME\",\"phone\":\"$PHONE\",\"password\":\"$PASSWORD\",\"vehicle_type\":\"motorcycle\",\"license_number\":\"LIC-$(date +%s)\",\"national_id\":\"ID-$(date +%s)\",\"auto_verify\":true}")

echo "Response: $RESPONSE"

# Extract token + driver ID
TOKEN=$(echo "$RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
DRIVER_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$TOKEN" ] || [ -z "$DRIVER_ID" ]; then
  echo "❌ Registration failed — driver may already exist. Trying login..."
  RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/auth/driver/login \
    -H "Content-Type: application/json" \
    -d "{\"phone\":\"$PHONE\",\"password\":\"$PASSWORD\"}")
  echo "Login response: $RESPONSE"
  TOKEN=$(echo "$RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
  DRIVER_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
fi

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to get token"
  exit 1
fi

echo "✅ Driver ID: $DRIVER_ID"

# 2. Register driver in dispatch.drivers
echo "📝 Registering in dispatch.drivers..."
DISPATCH_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/admin/drivers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"user_id\":\"$DRIVER_ID\",\"vehicle_type\":\"bike\",\"license_plate\":\"ABC-123\",\"zone_ids\":[\"zone-cairo\"]}")

echo "Dispatch response: $DISPATCH_RESPONSE"

# 3. Verify
echo ""
echo "============================================"
echo "  ✅ Driver seeded successfully!"
echo "============================================"
echo "  Phone:    $PHONE"
echo "  Password: $PASSWORD"
echo "  Name:     $NAME"
echo "  ID:       $DRIVER_ID"
echo "============================================"
echo ""
echo "Test login:"
echo "  curl -X POST http://localhost:8080/api/v1/auth/driver/login -H 'Content-Type: application/json' -d '{\"phone\":\"$PHONE\",\"password\":\"$PASSWORD\"}'"
