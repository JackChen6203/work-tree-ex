#!/usr/bin/env bash
set -euo pipefail

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
REQUEST_ID="$(date +%s)"

echo "Seeding demo data via ${API_BASE_URL}"

trip_response="$(
  curl -sS -f \
    -X POST "${API_BASE_URL}/trips" \
    -H "Content-Type: application/json" \
    -H "Idempotency-Key: seed-trip-${REQUEST_ID}" \
    -d '{
      "name":"Demo Trip Taipei",
      "destinationText":"Taipei",
      "startDate":"2026-04-10",
      "endDate":"2026-04-12",
      "timezone":"Asia/Taipei",
      "currency":"TWD",
      "travelersCount":2
    }'
)"
trip_id="$(echo "${trip_response}" | jq -r '.data.id')"

if [[ -z "${trip_id}" || "${trip_id}" == "null" ]]; then
  echo "failed to parse trip id from response: ${trip_response}" >&2
  exit 1
fi

days_response="$(curl -sS -f "${API_BASE_URL}/trips/${trip_id}/days")"
day_id="$(echo "${days_response}" | jq -r '.data[0].dayId')"

if [[ -z "${day_id}" || "${day_id}" == "null" ]]; then
  echo "failed to parse day id from response: ${days_response}" >&2
  exit 1
fi

curl -sS -f \
  -X POST "${API_BASE_URL}/trips/${trip_id}/items" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: seed-item-${REQUEST_ID}" \
  -d "{
    \"dayId\":\"${day_id}\",
    \"title\":\"Taipei 101\",
    \"itemType\":\"place_visit\",
    \"allDay\":false,
    \"note\":\"Demo seed itinerary item\"
  }" >/dev/null

curl -sS -f \
  -X PUT "${API_BASE_URL}/trips/${trip_id}/budget" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: seed-budget-${REQUEST_ID}" \
  -d '{
    "totalBudget":18000,
    "currency":"TWD",
    "categories":[
      {"category":"food","plannedAmount":5000},
      {"category":"transit","plannedAmount":3000}
    ]
  }' >/dev/null

curl -sS -f \
  -X POST "${API_BASE_URL}/trips/${trip_id}/expenses" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: seed-expense-${REQUEST_ID}" \
  -d '{
    "category":"food",
    "amount":320,
    "currency":"TWD",
    "note":"Demo seed expense"
  }' >/dev/null

curl -sS -f \
  -X POST "${API_BASE_URL}/notifications/trigger" \
  -H "Content-Type: application/json" \
  -d "{
    \"eventType\":\"seed.completed\",
    \"resourceId\":\"${trip_id}\",
    \"title\":\"Demo seed completed\",
    \"body\":\"Local demo seed data has been created.\",
    \"link\":\"/trips/${trip_id}\",
    \"userId\":\"00000000-0000-0000-0000-000000000001\"
  }" >/dev/null

echo "Seed completed successfully."
echo "tripId=${trip_id}"
echo "dayId=${day_id}"
