-- Add test_code column to pathology_tests if it doesn't exist (for existing tenant DBs)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'pathology_tests' AND column_name = 'test_code'
    ) THEN
        ALTER TABLE pathology_tests ADD COLUMN test_code TEXT NOT NULL DEFAULT '';
    END IF;
END $$;