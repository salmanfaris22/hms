-- Add missing columns to pathology_tests for existing tenant DBs
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'pathology_tests' AND column_name = 'test_type'
    ) THEN
        ALTER TABLE pathology_tests ADD COLUMN test_type TEXT NOT NULL DEFAULT '';
    END IF;
    
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'pathology_tests' AND column_name = 'description'
    ) THEN
        ALTER TABLE pathology_tests ADD COLUMN description TEXT NOT NULL DEFAULT '';
    END IF;
END $$;