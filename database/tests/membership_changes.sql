DO $$
BEGIN
 IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid='legui_private.membership_changes'::regclass) THEN RAISE EXCEPTION 'Membership change quotes need RLS';END IF;
 IF has_table_privilege('anon','legui_private.membership_changes','SELECT,INSERT,UPDATE,DELETE')
 OR has_table_privilege('authenticated','legui_private.membership_changes','SELECT,INSERT,UPDATE,DELETE')
 OR has_table_privilege('service_role','legui_private.membership_changes','SELECT,INSERT,UPDATE,DELETE') THEN
  RAISE EXCEPTION 'Browser/API database roles must not access membership changes';
 END IF;
END $$;
