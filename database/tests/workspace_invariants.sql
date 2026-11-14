DO $$ DECLARE name text; BEGIN
 FOREACH name IN ARRAY ARRAY['order_links','notifications','onboarding_completions','worker_heartbeats','email_changes'] LOOP
  IF NOT (SELECT relrowsecurity FROM pg_class WHERE oid=('legui_private.'||name)::regclass) THEN RAISE EXCEPTION 'RLS missing: %',name; END IF;
  IF has_table_privilege('anon','legui_private.'||name,'SELECT') OR has_table_privilege('authenticated','legui_private.'||name,'SELECT') OR has_table_privilege('service_role','legui_private.'||name,'SELECT') THEN RAISE EXCEPTION 'Private data exposed: %',name; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='legui_private' AND table_name='mail_outbox' AND column_name='lease_until') THEN RAISE EXCEPTION 'Mail delivery lease missing'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='order_link_tenant_order_fk' AND conrelid='legui_private.order_links'::regclass) THEN RAISE EXCEPTION 'Capability tenant guard missing'; END IF;
END $$;
