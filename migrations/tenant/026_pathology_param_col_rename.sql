-- Older tenants were created with `parameter_id` on pathology_test_parameters,
-- while the code and migration 010 expect `param_id`. Harmonize idempotently.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'pathology_test_parameters' AND column_name = 'parameter_id'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'pathology_test_parameters' AND column_name = 'param_id'
    ) THEN
        ALTER TABLE pathology_test_parameters RENAME COLUMN parameter_id TO param_id;
    END IF;
END $$;

-- Ensure supporting index and unique constraint use the canonical name.
DROP INDEX IF EXISTS pathology_test_parameters_param_idx;
CREATE INDEX IF NOT EXISTS pathology_test_parameters_param_id_idx
    ON pathology_test_parameters (param_id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'pathology_test_parameters_test_id_param_id_key'
    ) THEN
        BEGIN
            ALTER TABLE pathology_test_parameters
                ADD CONSTRAINT pathology_test_parameters_test_id_param_id_key
                UNIQUE (test_id, param_id);
        EXCEPTION WHEN duplicate_table OR duplicate_object THEN
            NULL;
        END;
    END IF;
END $$;

-- Drop the old unique constraint name if it still lingers from the rename.
ALTER TABLE pathology_test_parameters
    DROP CONSTRAINT IF EXISTS pathology_test_parameters_test_id_parameter_id_key;
