DO $$ DECLARE n text; BEGIN
 FOREACH n IN ARRAY ARRAY['support_tickets','support_messages','care_receipts','business_backups','business_closures','runtime_errors','staged_product_images'] LOOP
 IF NOT(SELECT relrowsecurity FROM pg_class WHERE oid=('legui_private.'||n)::regclass)
 OR has_table_privilege('anon','legui_private.'||n,'SELECT')
 OR has_table_privilege('authenticated','legui_private.'||n,'SELECT')
 OR has_table_privilege('service_role','legui_private.'||n,'SELECT') THEN RAISE EXCEPTION 'Care table exposed: %',n; END IF;
 END LOOP;
 IF NOT EXISTS(SELECT 1 FROM storage.buckets WHERE id='business-backups' AND NOT public) THEN RAISE EXCEPTION 'Backup bucket not private'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_policy WHERE polrelid='storage.objects'::regclass AND polname='leguiburger_private_backups_api_only' AND NOT polpermissive) THEN RAISE EXCEPTION 'Backup storage isolation missing';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='guard_closed_business') OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='guard_staged_image') THEN RAISE EXCEPTION 'Care integrity guard missing';END IF;
END $$;
