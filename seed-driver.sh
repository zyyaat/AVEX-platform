#!/bin/bash
# =============================================================================
# AVEX — Seed admin + driver accounts
# =============================================================================
# Creates:
#   1. Admin user (for managing the platform)
#   2. Driver in identity.drivers (verified + active)
#   3. Driver in dispatch.drivers (for delivery operations)
#
# Usage:
#   ./seed-driver.sh [driver_phone] [driver_password] [driver_name]
#
# Defaults:
#   Admin:   phone=01000000000  password=admin123
#   Driver:  phone=01012345678  password=12345678  name=Ahmed
# =============================================================================

set -e

DRIVER_PHONE="${1:-01012345678}"
DRIVER_PASSWORD="${2:-12345678}"
DRIVER_NAME="${3:-Ahmed}"

ADMIN_PHONE="01000000000"
ADMIN_PASSWORD="admin123"

API="http://localhost:8080/api/v1"

echo "🌱 AVEX — Seeding admin + driver accounts"
echo "============================================"

# -----------------------------------------------------------------------------
# 1. Create / Login Admin
# -----------------------------------------------------------------------------
echo ""
echo "📝 Step 1: Admin account..."
ADMIN_RESPONSE=$(curl -s -X POST "$API/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"AVEX Admin\",\"phone\":\"$ADMIN_PHONE\",\"password\":\"$ADMIN_PASSWORD\",\"email\":\"admin@avex.com\"}" 2>/dev/null || echo "")

ADMIN_TOKEN=$(echo "$ADMIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$ADMIN_TOKEN" ]; then
  echo "   Admin already exists, logging in..."
  ADMIN_RESPONSE=$(curl -s -X POST "$API/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"phone\":\"$ADMIN_PHONE\",\"password\":\"$ADMIN_PASSWORD\"}")
  ADMIN_TOKEN=$(echo "$ADMIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi

if [ -z "$ADMIN_TOKEN" ]; then
  echo "❌ Failed to create/login admin"
  echo "   Response: $ADMIN_RESPONSE"
  exit 1
fi

# Promote to admin (set is_admin=true via direct DB if needed)
# The register endpoint may not set is_admin. We'll try the admin endpoints
# and if they fail, we know the user needs to be promoted.
echo "   ✅ Admin token obtained"

# -----------------------------------------------------------------------------
# 2. Register / Login Driver in identity.drivers
# -----------------------------------------------------------------------------
echo ""
echo "📝 Step 2: Driver account (identity.drivers)..."
DRIVER_RESPONSE=$(curl -s -X POST "$API/auth/driver/register" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$DRIVER_NAME\",\"phone\":\"$DRIVER_PHONE\",\"password\":\"$DRIVER_PASSWORD\",\"vehicle_type\":\"motorcycle\",\"license_number\":\"LIC-$(date +%s)\",\"national_id\":\"ID-$(date +%s)\",\"auto_verify\":true}" 2>/dev/null || echo "")

DRIVER_TOKEN=$(echo "$DRIVER_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
DRIVER_ID=$(echo "$DRIVER_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$DRIVER_TOKEN" ] || [ -z "$DRIVER_ID" ]; then
  echo "   Driver already exists, logging in..."
  DRIVER_RESPONSE=$(curl -s -X POST "$API/auth/driver/login" \
    -H "Content-Type: application/json" \
    -d "{\"phone\":\"$DRIVER_PHONE\",\"password\":\"$DRIVER_PASSWORD\"}")
  DRIVER_TOKEN=$(echo "$DRIVER_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
  DRIVER_ID=$(echo "$DRIVER_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
fi

if [ -z "$DRIVER_TOKEN" ] || [ -z "$DRIVER_ID" ]; then
  echo "❌ Failed to create/login driver"
  echo "   Response: $DRIVER_RESPONSE"
  exit 1
fi

echo "   ✅ Driver ID: $DRIVER_ID"

# -----------------------------------------------------------------------------
# 3. Register Driver in dispatch.drivers (using admin token)
# -----------------------------------------------------------------------------
echo ""
echo "📝 Step 3: Dispatch driver (dispatch.drivers)..."
DISPATCH_RESPONSE=$(curl -s -X POST "$API/admin/drivers" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -d "{\"user_id\":\"$DRIVER_ID\",\"vehicle_type\":\"bike\",\"license_plate\":\"ABC-$(date +%s | tail -c 4)\",\"zone_ids\":[\"zone-cairo\"]}")

DISPATCH_ID=$(echo "$DISPATCH_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$DISPATCH_ID" ]; then
  echo "   ✅ Dispatch driver ID: $DISPATCH_ID"
else
  echo "   ⚠️  Dispatch registration failed (may already exist)"
  echo "   Response: $DISPATCH_RESPONSE"
  echo "   Trying with driver token instead..."
  DISPATCH_RESPONSE=$(curl -s -X POST "$API/admin/drivers" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $DRIVER_TOKEN" \
    -d "{\"user_id\":\"$DRIVER_ID\",\"vehicle_type\":\"bike\",\"license_plate\":\"ABC-$(date +%s | tail -c 4)\",\"zone_ids\":[\"zone-cairo\"]}")
  DISPATCH_ID=$(echo "$DISPATCH_RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
  if [ -n "$DISPATCH_ID" ]; then
    echo "   ✅ Dispatch driver ID: $DISPATCH_ID (via driver token)"
  else
    echo "   ⚠️  Response: $DISPATCH_RESPONSE"
  fi
fi

# -----------------------------------------------------------------------------
# 4. Verify
# -----------------------------------------------------------------------------
echo ""
echo "============================================"
echo "  ✅ Seeding complete!"
echo "============================================"
echo ""
echo "  Admin Account:"
echo "    Phone:    $ADMIN_PHONE"
echo "    Password: $ADMIN_PASSWORD"
echo ""
echo "  Driver Account:"
echo "    Phone:    $DRIVER_PHONE"
echo "    Password: $DRIVER_PASSWORD"
echo "    Name:     $DRIVER_NAME"
echo "    ID:       $DRIVER_ID"
echo ""
echo "  Test driver login:"
echo "    curl -X POST $API/auth/driver/login -H 'Content-Type: application/json' -d '{\"phone\":\"$DRIVER_PHONE\",\"password\":\"$DRIVER_PASSWORD\"}'"
echo ""
echo "  Test dispatch driver:"
echo "    curl -s \"$API/drivers?user_id=$DRIVER_ID\" -H 'Authorization: Bearer <token>'"
echo "============================================"
