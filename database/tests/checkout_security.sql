DO $$ DECLARE table_name text; role_name text; BEGIN
 FOREACH table_name IN ARRAY ARRAY['abuse_subjects','abuse_incidents','abuse_clearances','abuse_verified_tokens'] LOOP
  IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid=('legui_private.'||table_name)::regclass) THEN RAISE EXCEPTION 'RLS missing';END IF;
  FOREACH role_name IN ARRAY ARRAY['anon','authenticated','service_role'] LOOP
   IF has_table_privilege(role_name,'legui_private.'||table_name,'SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'Direct API role has abuse data access';END IF;
  END LOOP;
 END LOOP;
END $$;
