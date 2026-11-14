-- Always executed under the migration runner's rollback-only savepoint.
DO $$
DECLARE brand uuid:=gen_random_uuid();tenant uuid:=gen_random_uuid();actor uuid:=gen_random_uuid();worker uuid:=gen_random_uuid();sid uuid:=gen_random_uuid();
BEGIN
 INSERT INTO public.brands(id,name) VALUES(brand,'__security_fixture');
 INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(tenant,brand,'__security_fixture');
 INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(actor,actor||'@example.invalid','fixture','admin',brand,true);
 INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active) VALUES(worker,tenant,'Fixture','Worker',worker||'@example.invalid','fixture','employee',true);
 BEGIN
  UPDATE public.employees SET role='owner' WHERE id=worker;
  RAISE EXCEPTION 'Employee platform role was accepted';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
 INSERT INTO legui_private.auth_sessions(id,principal_kind,principal_id,role,brand_id,credential_digest) VALUES(sid,'user',actor,'admin',brand,repeat('a',64));
 UPDATE public.users SET password_hash='new-fixture-password-hash' WHERE id=actor;
 IF EXISTS(SELECT 1 FROM legui_private.auth_sessions WHERE id=sid AND revoked_at IS NULL) THEN RAISE EXCEPTION 'Password change did not revoke sessions';END IF;
 IF has_table_privilege('anon','legui_private.auth_sessions','SELECT') OR has_table_privilege('authenticated','legui_private.auth_factors','SELECT') OR has_table_privilege('service_role','legui_private.auth_action_tokens','SELECT') THEN RAISE EXCEPTION 'Private identity state is exposed';END IF;
 INSERT INTO legui_private.security_events(actor_kind,actor_id,event_name)VALUES('user',actor,'fixture');
 BEGIN
  DELETE FROM legui_private.security_events WHERE actor_id=actor;
  RAISE EXCEPTION 'Audit deletion was accepted';
 EXCEPTION WHEN check_violation THEN NULL;
 END;
END $$;
