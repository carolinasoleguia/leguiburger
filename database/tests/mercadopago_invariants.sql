DO $$ DECLARE item record;count_tables integer:=0;BEGIN
 FOR item IN SELECT c.oid,c.relname,c.relrowsecurity FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='legui_private' AND c.relkind='r' AND c.relname LIKE 'mp_%' LOOP
  count_tables:=count_tables+1;
  IF NOT item.relrowsecurity OR has_table_privilege('anon',item.oid,'SELECT') OR has_table_privilege('authenticated',item.oid,'SELECT') OR has_table_privilege('service_role',item.oid,'SELECT') THEN
   RAISE EXCEPTION 'Provider state is exposed: %',item.relname;
  END IF;
 END LOOP;
 IF count_tables<>11 THEN RAISE EXCEPTION 'Incomplete Mercado Pago schema';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='mp_observations_immutable' AND tgenabled<>'D') OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='mp_allocations_immutable' AND tgenabled<>'D') OR NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgname='order_mp_refund_evidence' AND tgenabled<>'D') THEN
  RAISE EXCEPTION 'Missing provider evidence guards';
 END IF;
END $$;
