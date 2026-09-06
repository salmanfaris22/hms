-- Inventory categories (drugs)
CREATE TABLE IF NOT EXISTS inv_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Manufacturers
CREATE TABLE IF NOT EXISTS inv_manufacturers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Drugs / consumables
CREATE TABLE IF NOT EXISTS inv_drugs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id       UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    generic_name    TEXT NOT NULL DEFAULT '',
    category_id     UUID REFERENCES inv_categories(id) ON DELETE SET NULL,
    strength        TEXT NOT NULL DEFAULT '',
    item_code       TEXT NOT NULL DEFAULT '',
    manufacturer_id UUID REFERENCES inv_manufacturers(id) ON DELETE SET NULL,
    instruction     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Stock batches per drug
CREATE TABLE IF NOT EXISTS inv_stock (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id     UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    drug_id       UUID NOT NULL REFERENCES inv_drugs(id) ON DELETE CASCADE,
    batch_code    TEXT NOT NULL DEFAULT '',
    expiry_date   DATE,
    qty_available INT NOT NULL DEFAULT 0,
    purchase_rate NUMERIC(12,2) NOT NULL DEFAULT 0,
    mrp           NUMERIC(12,2) NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Sales invoices
CREATE TABLE IF NOT EXISTS inv_sales (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id       UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    invoice_no      TEXT NOT NULL,
    customer_name   TEXT NOT NULL DEFAULT '',
    opd_id          TEXT NOT NULL DEFAULT '',
    prescribed_by   TEXT NOT NULL DEFAULT '',
    stock_point     TEXT NOT NULL DEFAULT 'Main Store',
    payment_method  TEXT NOT NULL DEFAULT 'Cash',
    subtotal        NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_pct    NUMERIC(5,2) NOT NULL DEFAULT 0,
    gst_enabled     BOOLEAN NOT NULL DEFAULT false,
    round_off       NUMERIC(8,2) NOT NULL DEFAULT 0,
    shipping        NUMERIC(8,2) NOT NULL DEFAULT 0,
    grand_total     NUMERIC(12,2) NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'Paid',
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inv_sale_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id     UUID NOT NULL REFERENCES inv_sales(id) ON DELETE CASCADE,
    drug_id     UUID REFERENCES inv_drugs(id) ON DELETE SET NULL,
    drug_name   TEXT NOT NULL DEFAULT '',
    batch_code  TEXT NOT NULL DEFAULT '',
    expiry_date DATE,
    unit        TEXT NOT NULL DEFAULT 'Tablet',
    qty         INT NOT NULL DEFAULT 1,
    unit_price  NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount    NUMERIC(5,2) NOT NULL DEFAULT 0,
    tax         NUMERIC(5,2) NOT NULL DEFAULT 0,
    total       NUMERIC(12,2) NOT NULL DEFAULT 0
);

-- Dispense / stock issue records
CREATE TABLE IF NOT EXISTS inv_dispenses (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id      UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    dispense_no    TEXT NOT NULL,
    patient_name   TEXT NOT NULL DEFAULT '',
    issued_to_dept TEXT NOT NULL DEFAULT '',
    prescribed_by  TEXT NOT NULL DEFAULT '',
    issued_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    notes          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'Dispensed',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inv_dispense_items (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dispense_id  UUID NOT NULL REFERENCES inv_dispenses(id) ON DELETE CASCADE,
    drug_id      UUID REFERENCES inv_drugs(id) ON DELETE SET NULL,
    drug_name    TEXT NOT NULL DEFAULT '',
    unit         TEXT NOT NULL DEFAULT 'Tablet',
    qty_prescribed INT NOT NULL DEFAULT 0,
    qty_total    INT NOT NULL DEFAULT 0
);

-- Assets
CREATE TABLE IF NOT EXISTS inv_assets (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id        UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    asset_no         TEXT NOT NULL DEFAULT '',
    name             TEXT NOT NULL,
    manufacturer     TEXT NOT NULL DEFAULT '',
    serial_number    TEXT NOT NULL DEFAULT '',
    model            TEXT NOT NULL DEFAULT '',
    location         TEXT NOT NULL DEFAULT '',
    purchase_date    DATE,
    current_value    NUMERIC(12,2) NOT NULL DEFAULT 0,
    purchase_cost    NUMERIC(12,2) NOT NULL DEFAULT 0,
    warranty_expiry  DATE,
    category         TEXT NOT NULL DEFAULT '',
    department       TEXT NOT NULL DEFAULT '',
    condition        TEXT NOT NULL DEFAULT 'Good',
    status           TEXT NOT NULL DEFAULT 'Active',
    notes            TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Purchase orders
CREATE TABLE IF NOT EXISTS inv_purchases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id       UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    purchase_id     TEXT NOT NULL,
    invoice_no      TEXT NOT NULL DEFAULT '',
    supplier_name   TEXT NOT NULL DEFAULT '',
    stock_point     TEXT NOT NULL DEFAULT 'Main Store',
    payment_method  TEXT NOT NULL DEFAULT 'Cash',
    subtotal        NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_pct    NUMERIC(5,2) NOT NULL DEFAULT 0,
    gst_enabled     BOOLEAN NOT NULL DEFAULT false,
    round_off       NUMERIC(8,2) NOT NULL DEFAULT 0,
    shipping        NUMERIC(8,2) NOT NULL DEFAULT 0,
    grand_total     NUMERIC(12,2) NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS inv_purchase_items (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id   UUID NOT NULL REFERENCES inv_purchases(id) ON DELETE CASCADE,
    drug_id       UUID REFERENCES inv_drugs(id) ON DELETE SET NULL,
    drug_name     TEXT NOT NULL DEFAULT '',
    batch_code    TEXT NOT NULL DEFAULT '',
    expiry_date   DATE,
    qty_purchased INT NOT NULL DEFAULT 0,
    qty_total     INT NOT NULL DEFAULT 0,
    purchase_rate NUMERIC(12,2) NOT NULL DEFAULT 0,
    gst_pct       NUMERIC(5,2) NOT NULL DEFAULT 0,
    free_qty      INT NOT NULL DEFAULT 0,
    mrp           NUMERIC(12,2) NOT NULL DEFAULT 0,
    pack_size     INT NOT NULL DEFAULT 1,
    discount_pct  NUMERIC(5,2) NOT NULL DEFAULT 0,
    amount        NUMERIC(12,2) NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_inv_drugs_clinic   ON inv_drugs(clinic_id);
CREATE INDEX IF NOT EXISTS idx_inv_stock_drug     ON inv_stock(drug_id);
CREATE INDEX IF NOT EXISTS idx_inv_sales_clinic   ON inv_sales(clinic_id);
CREATE INDEX IF NOT EXISTS idx_inv_dispenses_clinic ON inv_dispenses(clinic_id);
CREATE INDEX IF NOT EXISTS idx_inv_assets_clinic  ON inv_assets(clinic_id);
CREATE INDEX IF NOT EXISTS idx_inv_purchases_clinic ON inv_purchases(clinic_id);
