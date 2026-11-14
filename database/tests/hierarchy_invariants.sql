DO $$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['employee_locations','employee_location_history','business_invitations','hierarchy_receipts'] LOOP
 IF NOT(SELECT relrowsecurity FROM pg_class WHERE oid=('legui_private.'||n)::regclass) OR has_table_privilege('anon','legui_private.'||n,'SELECT') OR has_table_privilege('authenticated','legui_private.'||n,'SELECT') THEN RAISE EXCEPTION 'Hierarchy table not private: %',n;END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM public.employees e WHERE e.tenant_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM legui_private.employee_locations a WHERE a.employee_id=e.id AND a.tenant_id=e.tenant_id)) THEN RAISE EXCEPTION 'Legacy employee home assignment missing';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='business_location_capacity') OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='collaborator_capacity') OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='team_invitation_capacity') THEN RAISE EXCEPTION 'Capacity guards missing';END IF;
END $$;
