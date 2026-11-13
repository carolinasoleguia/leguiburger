-- The migration runner rolls this entire verification back, including all fixtures.
DO $$
DECLARE b uuid:=gen_random_uuid(); other_b uuid:=gen_random_uuid(); t1 uuid:=gen_random_uuid();t2 uuid:=gen_random_uuid();t3 uuid:=gen_random_uuid();
 p uuid:=gen_random_uuid();draft uuid:=gen_random_uuid();price uuid:=gen_random_uuid();a uuid:=gen_random_uuid();s uuid:=gen_random_uuid();
 e uuid; shared_employee uuid; invitation uuid:=gen_random_uuid();actor uuid:=gen_random_uuid();i integer; n numeric;capacity record;
BEGIN
 INSERT INTO public.brands(id,name,base_location_limit,base_collaborator_limit) VALUES(b,'__membership_fixture',5,100),(other_b,'__membership_other',1,5);
 INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(t1,b,'__membership_1');
 INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(actor,actor::text||'@example.invalid','fixture','admin',b,true);
 INSERT INTO legui_private.billing_plans(id,code,name) VALUES(p,'fixture_'||replace(p::text,'-',''),'Fixture Premium');
 INSERT INTO legui_private.billing_plan_prices(id,plan_id,currency,amount,interval_months,version,billing_unit,maximum_collaborators,membership_tier,is_active)
 VALUES(draft,p,'ARS',0,1,1,'brand',5,'premium',false);
 BEGIN
  UPDATE legui_private.billing_plan_prices SET is_active=true WHERE id=draft;
  RAISE EXCEPTION 'Unpriced draft became purchasable';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  UPDATE legui_private.billing_plan_prices SET amount=1000 WHERE id=draft;
  RAISE EXCEPTION 'Price terms were mutated';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  INSERT INTO legui_private.billing_plan_prices(plan_id,currency,amount,interval_months,version,billing_unit,membership_tier)
  VALUES(p,'ARS',1000,1,99,'brand','standard');
  RAISE EXCEPTION 'Null limits bypassed standard terms';
 EXCEPTION WHEN check_violation THEN NULL; END;
 INSERT INTO legui_private.billing_plan_prices(id,plan_id,currency,amount,interval_months,version,billing_unit,maximum_collaborators,membership_tier,extra_location_amount)
 VALUES(price,p,'ARS',1000,1,2,'brand',5,'premium',250);
 INSERT INTO legui_private.billing_accounts(id,brand_id,billing_unit,currency) VALUES(a,b,'brand','ARS');
 INSERT INTO legui_private.subscriptions(id,account_id,plan_price_id,currency,interval_months,billing_unit,contracted_amount,purchased_locations,environment,status,current_period_start,current_period_end)
 VALUES(s,a,price,'ARS',1,'brand',1,3,'production','active',now()-interval '1 day',now()+interval '29 days');
 SELECT contracted_amount INTO n FROM legui_private.subscriptions WHERE id=s;
 IF n<>1500 THEN RAISE EXCEPTION 'Expected base plus two extra stores, got %',n; END IF;
 SELECT * INTO capacity FROM legui_private.business_capacity(b);
 IF capacity.locations<>3 OR capacity.collaborators<>5 OR capacity.source<>'subscription_per_location' THEN RAISE EXCEPTION 'Wrong commercial capacity'; END IF;
 INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(t2,b,'__membership_2'),(t3,b,'__membership_3');
 BEGIN
  INSERT INTO public.tenants(brand_id,subdomain) VALUES(b,'__membership_4');
  RAISE EXCEPTION 'Unpurchased location was created';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  UPDATE legui_private.subscriptions SET purchased_locations=4,contracted_amount=1750 WHERE id=s;
  RAISE EXCEPTION 'Changed a contracted price without agreement';
 EXCEPTION WHEN check_violation THEN NULL; END;
 -- Named staff may exceed the terminal quota: shifts do not reserve devices.
 FOR i IN 1..6 LOOP
  e:=gen_random_uuid();IF i=1 THEN shared_employee:=e;END IF;
  INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active)
   VALUES(e,t1,'Fixture','Member',e::text||'@example.invalid','fixture','cashier',true);
 END LOOP;
 FOR i IN 1..5 LOOP
  INSERT INTO legui_private.auth_sessions(principal_kind,principal_id,role,brand_id,tenant_id,credential_digest)
   VALUES('employee',shared_employee,'cashier',b,t1,'fixture');
 END LOOP;
 BEGIN
  INSERT INTO legui_private.auth_sessions(principal_kind,principal_id,role,brand_id,tenant_id,credential_digest)
   VALUES('employee',shared_employee,'cashier',b,t1,'fixture');
  RAISE EXCEPTION 'Sixth simultaneous terminal was admitted';
 EXCEPTION WHEN check_violation THEN NULL;END;
 INSERT INTO legui_private.employee_locations(employee_id,tenant_id) VALUES(shared_employee,t2);
 INSERT INTO legui_private.auth_sessions(principal_kind,principal_id,role,brand_id,tenant_id,credential_digest)
  VALUES('employee',shared_employee,'cashier',b,t2,'fixture');
 UPDATE legui_private.auth_sessions SET revoked_at=clock_timestamp() WHERE id=(SELECT id FROM legui_private.auth_sessions WHERE tenant_id=t1 AND revoked_at IS NULL LIMIT 1);
 INSERT INTO legui_private.auth_sessions(principal_kind,principal_id,role,brand_id,tenant_id,credential_digest)
  VALUES('employee',shared_employee,'cashier',b,t1,'fixture');
 IF has_function_privilege('anon','legui_private.membership_access(uuid)','EXECUTE') THEN RAISE EXCEPTION 'Public access to membership function'; END IF;

END $$;

DO $$
DECLARE b uuid:=gen_random_uuid();t uuid:=gen_random_uuid();u uuid:=gen_random_uuid();e uuid:=gen_random_uuid();a record;expiry timestamptz;
BEGIN
 INSERT INTO public.brands(id,name) VALUES(b,'__trial_fixture');
 INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(t,b,'__trial_fixture');
 INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(u,u::text||'@example.invalid','fixture','admin',b,true);
 SELECT * INTO a FROM legui_private.membership_access(b);
 IF a.state<>'trial' OR NOT a.operational OR a.premium OR a.locations<>1 OR a.terminals_per_location<>2 THEN RAISE EXCEPTION 'Wrong initial seven-day policy';END IF;
 SELECT expires_at INTO expiry FROM legui_private.membership_trials WHERE brand_id=b AND expires_at-started_at=interval '168 hours';
 IF expiry IS NULL THEN RAISE EXCEPTION 'Trial is not exactly seven days'; END IF;
 UPDATE public.users SET first_name='Fixture' WHERE id=u;
 IF (SELECT expires_at FROM legui_private.membership_trials WHERE brand_id=b)<>expiry THEN RAISE EXCEPTION 'Trial restarted';END IF;
 BEGIN
  UPDATE legui_private.membership_trials SET started_at=now() WHERE brand_id=b;
  RAISE EXCEPTION 'Trial history is mutable';
 EXCEPTION WHEN check_violation THEN NULL;END;
 INSERT INTO public.employees(id,tenant_id,first_name,last_name,email,password_hash,role,is_active)
 VALUES(e,t,'Fixture','Worker',e::text||'@example.invalid','fixture','cashier',true);
 INSERT INTO legui_private.auth_sessions(principal_kind,principal_id,role,brand_id,tenant_id,credential_digest)
 VALUES('employee',e,'cashier',b,t,'fixture'),('employee',e,'cashier',b,t,'fixture');
 BEGIN
  INSERT INTO legui_private.auth_sessions(principal_kind,principal_id,role,brand_id,tenant_id,credential_digest)
   VALUES('employee',e,'cashier',b,t,'fixture');
  RAISE EXCEPTION 'Third trial terminal was admitted';
 EXCEPTION WHEN check_violation THEN NULL;END;
 IF has_table_privilege('service_role','legui_private.membership_trials','SELECT,INSERT,UPDATE,DELETE') THEN RAISE EXCEPTION 'Direct trial access';END IF;
END $$;

DO $$
DECLARE b uuid:=gen_random_uuid();u uuid:=gen_random_uuid();a record;
BEGIN
 INSERT INTO public.brands(id,name) VALUES(b,'__expired_trial_fixture');
 INSERT INTO legui_private.membership_trials(brand_id,started_at) VALUES(b,now()-interval '8 days');
 INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES(u,u::text||'@example.invalid','fixture','admin',b,true);
 SELECT * INTO a FROM legui_private.membership_access(b);
 IF a.state<>'expired' OR a.operational OR a.premium THEN RAISE EXCEPTION 'Expired trial got access'; END IF;
END $$;
