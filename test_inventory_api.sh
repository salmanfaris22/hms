#!/bin/bash
# Test inventory API endpoints
# Run: bash test_inventory_api.sh

BASE="http://localhost:8080/api"
EMAIL="sui@gmail.com"
PASS="sui@gmail.com"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass=0; fail=0

check() {
  local label=$1 code=$2 expected=$3
  if [ "$code" -eq "$expected" ]; then
    echo -e "  ${GREEN}✓ PASS${NC} $label (HTTP $code)"
    ((pass++))
  else
    echo -e "  ${RED}✗ FAIL${NC} $label (expected $expected, got $code)"
    ((fail++))
  fi
}

echo ""
echo "═══════════════════════════════════════════════════"
echo "   HMS Inventory API Test Suite"
echo "   Login: $EMAIL / $PASS"
echo "═══════════════════════════════════════════════════"

# ─── Auth ─────────────────────────────────────────────────────────────────────
echo -e "\n${CYAN}[Auth]${NC}"
RESP=$(curl -s -X POST "$BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}")
check "POST /auth/login ($EMAIL)" "$CODE" 200

TOKEN=$(echo "$RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('token',''))" 2>/dev/null)
if [ -z "$TOKEN" ]; then
  echo -e "${RED}No token — aborting${NC}"
  exit 1
fi
echo "  Token: ${TOKEN:0:40}..."
AUTH="Authorization: Bearer $TOKEN"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/auth/me" -H "$AUTH")
check "GET /auth/me" "$CODE" 200

# ─── Clinics ──────────────────────────────────────────────────────────────────
echo -e "\n${CYAN}[Clinics]${NC}"
CLINICS=$(curl -s "$BASE/clinics" -H "$AUTH")
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/clinics" -H "$AUTH")
check "GET /clinics" "$CODE" 200

CLINIC_ID=$(echo "$CLINICS" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items = d.get('data',{}).get('clinics', d.get('data',{}).get('items',[]))
print(items[0]['id'] if items else '')
" 2>/dev/null)
CLINIC_NAME=$(echo "$CLINICS" | python3 -c "
import sys,json
d=json.load(sys.stdin)
items = d.get('data',{}).get('clinics', d.get('data',{}).get('items',[]))
print(items[0]['name'] if items else '')
" 2>/dev/null)

if [ -z "$CLINIC_ID" ]; then
  echo -e "  ${RED}No clinic found${NC}"
  exit 1
fi
echo "  Using clinic: '$CLINIC_NAME' ($CLINIC_ID)"
Q="clinicId=$CLINIC_ID"

# ─── Stock: Drug Summary ──────────────────────────────────────────────────────
echo -e "\n${CYAN}[Stock Management — Inventory]${NC}"
SUMMARY=$(curl -s "$BASE/inventory/drugs/summary?$Q" -H "$AUTH")
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/drugs/summary?$Q" -H "$AUTH")
check "GET /inventory/drugs/summary" "$CODE" 200
echo "  $(echo $SUMMARY | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(f\"  totalItems={d.get('totalItems',0)}  totalValue={d.get('totalValue',0)}  lowStock={d.get('lowStock',0)}\")" 2>/dev/null)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/drugs?$Q" -H "$AUTH")
check "GET /inventory/drugs" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/drugs?$Q&q=Amox" -H "$AUTH")
check "GET /inventory/drugs?q=Amox (search filter)" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/drugs?$Q&status=In+Stock" -H "$AUTH")
check "GET /inventory/drugs?status=In Stock" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/drugs/pos?$Q" -H "$AUTH")
check "GET /inventory/drugs/pos (POS catalog)" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/drugs/pos?$Q&q=Para" -H "$AUTH")
check "GET /inventory/drugs/pos?q=Para" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/stock?$Q" -H "$AUTH")
check "GET /inventory/stock" "$CODE" 200

# Categories
echo -e "\n${CYAN}[Categories & Manufacturers]${NC}"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/categories?$Q" -H "$AUTH")
check "GET /inventory/categories" "$CODE" 200

CAT_RESP=$(curl -s -X POST "$BASE/inventory/categories?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" -d '{"name":"Test Category XYZ"}')
CAT_CODE=$(echo "$CAT_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/categories" "$CAT_CODE" 201
CAT_ID=$(echo "$CAT_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
if [ -n "$CAT_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE/inventory/categories/$CAT_ID?$Q" -H "$AUTH")
  check "DELETE /inventory/categories/:id" "$CODE" 200
fi

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/manufacturers?$Q" -H "$AUTH")
check "GET /inventory/manufacturers" "$CODE" 200

MFR_RESP=$(curl -s -X POST "$BASE/inventory/manufacturers?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" -d '{"name":"Test Pharma XYZ"}')
MFR_ID=$(echo "$MFR_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
MFR_CODE=$(echo "$MFR_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/manufacturers" "$MFR_CODE" 201
if [ -n "$MFR_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE/inventory/manufacturers/$MFR_ID?$Q" -H "$AUTH")
  check "DELETE /inventory/manufacturers/:id" "$CODE" 200
fi

# Create / delete drug
DRUG_RESP=$(curl -s -X POST "$BASE/inventory/drugs?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"Test API Drug 100mg","genericName":"TestGeneric","strength":"100mg","itemCode":"TAPI001"}')
DRUG_ID=$(echo "$DRUG_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
DRUG_CODE=$(echo "$DRUG_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/drugs" "$DRUG_CODE" 201
if [ -n "$DRUG_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE/inventory/drugs/$DRUG_ID?$Q" -H "$AUTH")
  check "DELETE /inventory/drugs/:id" "$CODE" 200
fi

# ─── Stock Issue / Dispense ───────────────────────────────────────────────────
echo -e "\n${CYAN}[Stock Issue / Dispense]${NC}"
DISP_SUM=$(curl -s "$BASE/inventory/dispenses/summary?$Q" -H "$AUTH")
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/dispenses/summary?$Q" -H "$AUTH")
check "GET /inventory/dispenses/summary" "$CODE" 200
echo "  $(echo $DISP_SUM | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(f\"  totalDispensed={d.get('totalDispensed',0)}  pending={d.get('pending',0)}  totalUnitsOut={d.get('totalUnitsOut',0)}\")" 2>/dev/null)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/dispenses?$Q" -H "$AUTH")
check "GET /inventory/dispenses" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/dispenses?$Q&status=Dispensed" -H "$AUTH")
check "GET /inventory/dispenses?status=Dispensed" "$CODE" 200

DISP_RESP=$(curl -s -X POST "$BASE/inventory/dispenses?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"patientName":"API Test Patient","issuedToDept":"OPD","prescribedBy":"Dr. Test","issuedDate":"2024-12-01","notes":"Test","items":[{"drugId":"","drugName":"Paracetamol","unit":"Tablet","qtyPrescribed":10,"qtyTotal":10}]}')
DISP_ID=$(echo "$DISP_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
DISP_CODE=$(echo "$DISP_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/dispenses" "$DISP_CODE" 201

if [ -n "$DISP_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE/inventory/dispenses/$DISP_ID?$Q" \
    -H "$AUTH" -H "Content-Type: application/json" -d '{"status":"Pending"}')
  check "PATCH /inventory/dispenses/:id (status update)" "$CODE" 200
fi

# ─── Sales & POS ─────────────────────────────────────────────────────────────
echo -e "\n${CYAN}[Sales & POS]${NC}"
SALES_SUM=$(curl -s "$BASE/inventory/sales/summary?$Q" -H "$AUTH")
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/sales/summary?$Q" -H "$AUTH")
check "GET /inventory/sales/summary" "$CODE" 200
echo "  $(echo $SALES_SUM | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(f\"  todaySales={d.get('todaySales',0)}  weeklyRevenue={d.get('weeklyRevenue',0)}  totalTxns={d.get('totalTxns',0)}\")" 2>/dev/null)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/sales?$Q" -H "$AUTH")
check "GET /inventory/sales" "$CODE" 200

SALE_RESP=$(curl -s -X POST "$BASE/inventory/sales?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d "{\"clinicId\":\"$CLINIC_ID\",\"customerName\":\"API Test\",\"paymentMethod\":\"Cash\",\"subtotal\":25.50,\"grandTotal\":25.50,\"notes\":\"\",\"items\":[{\"drugId\":\"\",\"drugName\":\"Paracetamol 650mg\",\"batchCode\":\"B001\",\"unit\":\"Tablet\",\"qty\":2,\"unitPrice\":12.75,\"discount\":0,\"tax\":0,\"total\":25.50}]}")
SALE_ID=$(echo "$SALE_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
SALE_CODE=$(echo "$SALE_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/sales (Quick Sale)" "$SALE_CODE" 201

if [ -n "$SALE_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE/inventory/sales/$SALE_ID?$Q" -H "$AUTH")
  check "DELETE /inventory/sales/:id" "$CODE" 200
fi

# Purchases
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/purchases?$Q" -H "$AUTH")
check "GET /inventory/purchases" "$CODE" 200

PUR_RESP=$(curl -s -X POST "$BASE/inventory/purchases?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"invoiceNo":"SINV-TEST","supplierName":"Test Supplier Co","paymentMethod":"Cash","subtotal":1000,"grandTotal":1000,"notes":"","items":[{"drugId":"","drugName":"Amoxicillin 500mg","batchCode":"B999","qtyPurchased":50,"qtyTotal":50,"purchaseRate":8,"mrp":12.5,"gstPct":5,"freeQty":0,"packSize":10,"discountPct":0,"amount":400}]}')
PUR_ID=$(echo "$PUR_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
PUR_CODE=$(echo "$PUR_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/purchases" "$PUR_CODE" 201
if [ -n "$PUR_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE/inventory/purchases/$PUR_ID?$Q" -H "$AUTH")
  check "DELETE /inventory/purchases/:id" "$CODE" 200
fi

# ─── Asset Management ─────────────────────────────────────────────────────────
echo -e "\n${CYAN}[Asset Management]${NC}"
ASSET_SUM=$(curl -s "$BASE/inventory/assets/summary?$Q" -H "$AUTH")
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/assets/summary?$Q" -H "$AUTH")
check "GET /inventory/assets/summary" "$CODE" 200
echo "  $(echo $ASSET_SUM | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(f\"  totalAssets={d.get('totalAssets',0)}  active={d.get('activeAssets',0)}  bookValue={d.get('totalBookValue',0)}\")" 2>/dev/null)"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/assets?$Q" -H "$AUTH")
check "GET /inventory/assets" "$CODE" 200

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/inventory/assets?$Q&status=Active" -H "$AUTH")
check "GET /inventory/assets?status=Active" "$CODE" 200

ASSET_RESP=$(curl -s -X POST "$BASE/inventory/assets?$Q" \
  -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"API Test Bed","manufacturer":"Test Co","serialNumber":"SN-API-001","model":"T1","location":"Ward C","purchaseDate":"2024-01-15","currentValue":8000,"purchaseCost":12000,"category":"Furniture","department":"General Ward","condition":"Good","status":"Active","notes":"API test"}')
ASSET_ID=$(echo "$ASSET_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
ASSET_CODE=$(echo "$ASSET_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(201 if d.get('data',{}).get('id') else 400)" 2>/dev/null)
check "POST /inventory/assets" "$ASSET_CODE" 201

if [ -n "$ASSET_ID" ]; then
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PATCH "$BASE/inventory/assets/$ASSET_ID?$Q" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"name":"API Test Bed (updated)","manufacturer":"Test Co","serialNumber":"SN-API-001","model":"T1","location":"ICU","currentValue":7500,"purchaseCost":12000,"category":"Furniture","department":"ICU","condition":"Fair","status":"Maintenance","notes":"Moved to ICU"}')
  check "PATCH /inventory/assets/:id (update)" "$CODE" 200

  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE/inventory/assets/$ASSET_ID?$Q" -H "$AUTH")
  check "DELETE /inventory/assets/:id" "$CODE" 200
fi

# ─── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo "═══════════════════════════════════════════════════"
echo -e " Results: ${GREEN}$pass passed${NC}  ${RED}$fail failed${NC}  $(($pass + $fail)) total"
echo "═══════════════════════════════════════════════════"
[ "$fail" -eq 0 ] && echo -e " ${GREEN}ALL TESTS PASSED ✓${NC}" || echo -e " ${RED}$fail TEST(S) FAILED ✗${NC}"
echo ""
