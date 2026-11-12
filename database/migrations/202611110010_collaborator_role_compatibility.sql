-- The legacy check predates the kitchen role and duplicates the stricter
-- employees_collaborator_roles check installed by the identity migration.
-- Keep the strict check (including for all new/updated rows); preserve legacy rows.
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='public.employees'::regclass AND conname='employees_collaborator_roles') THEN
  RAISE EXCEPTION 'The collaborator-only role guard is required';
 END IF;
END $$;
ALTER TABLE public.employees DROP CONSTRAINT IF EXISTS chk_employee_role;
