-- Executed in a savepoint by cmd/migrate and always rolled back, including fixtures/helpers.
CREATE FUNCTION legui_private.test_assert(ok boolean,label text) RETURNS void LANGUAGE plpgsql AS $$
BEGIN IF ok IS DISTINCT FROM true THEN RAISE EXCEPTION 'Assertion failed: %',label; END IF; END $$;
CREATE FUNCTION legui_private.test_rejected(statement text,expected_state text,label text) RETURNS void LANGUAGE plpgsql AS $$
DECLARE rejected boolean:=false;
BEGIN
 BEGIN EXECUTE statement;
 EXCEPTION WHEN OTHERS THEN
  IF SQLSTATE<>expected_state THEN RAISE EXCEPTION 'Unexpected SQLSTATE for %: % (%)',label,SQLSTATE,SQLERRM; END IF;
  rejected:=true;
 END;
 PERFORM legui_private.test_assert(rejected,label);
END $$;

DO $$
<<fixture>>
DECLARE
 b1 uuid:=gen_random_uuid();b2 uuid:=gen_random_uuid();t1 uuid:=gen_random_uuid();t2 uuid:=gen_random_uuid();
 p1 uuid:=gen_random_uuid();customer2 uuid:=gen_random_uuid();o1 uuid:=gen_random_uuid();d1 uuid:=gen_random_uuid();
 owner_id uuid:=gen_random_uuid();manager_id uuid:=gen_random_uuid();charge1 uuid:=gen_random_uuid();
 plan_id uuid:=gen_random_uuid();price_id uuid:=gen_random_uuid();account_id uuid:=gen_random_uuid();
 subscription_id uuid:=gen_random_uuid();invoice_id uuid:=gen_random_uuid();billing_charge uuid:=gen_random_uuid();
 cost_supply uuid:=gen_random_uuid();n bigint;row record;
BEGIN
 INSERT INTO public.brands(id,name,data_classification) VALUES(b1,'__migration_business_a','test'),(b2,'__migration_business_b','test');
 INSERT INTO public.tenants(id,brand_id,subdomain) VALUES(t1,b1,'__migration_a'),(t2,b2,'__migration_b');
 INSERT INTO public.users(id,email,password_hash,role,brand_id,is_active) VALUES
  (owner_id,owner_id::text||'@example.invalid','not-a-login','owner',NULL,true),
  (manager_id,manager_id::text||'@example.invalid','not-a-login','admin',b1,true);
 INSERT INTO public.products(id,brand_id,name,base_price) VALUES(p1,b1,'__migration_burger',100);
 INSERT INTO public.tenant_products(tenant_id,product_id,current_stock) VALUES(t1,p1,5);
 PERFORM legui_private.test_assert((SELECT count(*)=1 FROM public.inventory_movements WHERE tenant_id=t1 AND item_id=p1 AND balance_before=0 AND balance_after=5),'opening inventory capture');
 PERFORM legui_private.test_assert((SELECT price=100 FROM public.price_history WHERE tenant_id=t1 AND product_id=p1 AND end_date IS NULL),'initial effective price');
 UPDATE public.products SET base_price=110 WHERE id=p1;
 PERFORM legui_private.test_assert((SELECT price=110 FROM public.price_history WHERE tenant_id=t1 AND product_id=p1 AND end_date IS NULL),'base price propagation');
 UPDATE public.tenant_products SET price_override=90 WHERE tenant_id=t1 AND product_id=p1;
 UPDATE public.products SET base_price=120 WHERE id=p1;
 PERFORM legui_private.test_assert((SELECT price=90 FROM public.price_history WHERE tenant_id=t1 AND product_id=p1 AND end_date IS NULL),'local override remains independent');
 PERFORM set_config('leguiburger.operation_key','fixture-stock-operation',true);
 PERFORM set_config('leguiburger.request_digest',repeat('a',64),true);
 PERFORM set_config('leguiburger.inventory_reason','sale',true);
 UPDATE public.tenant_products SET current_stock=current_stock-1 WHERE tenant_id=t1 AND product_id=p1;
 PERFORM legui_private.test_rejected(format('UPDATE public.tenant_products SET current_stock=current_stock-1 WHERE tenant_id=%L AND product_id=%L',t1,p1),'23505','duplicate stock operation rolls back balance');
 PERFORM legui_private.test_assert((SELECT current_stock=4 FROM public.tenant_products WHERE tenant_id=t1 AND product_id=p1),'stock deduplication');
 PERFORM set_config('leguiburger.operation_key','',true);PERFORM set_config('leguiburger.request_digest','',true);PERFORM set_config('leguiburger.inventory_reason','',true);
 PERFORM set_config('leguiburger.expected_stock_version','0',true);
 PERFORM legui_private.test_rejected(format('UPDATE public.tenant_products SET current_stock=3 WHERE tenant_id=%L AND product_id=%L',t1,p1),'40001','stale stock version');
 PERFORM set_config('leguiburger.expected_stock_version','',true);
 INSERT INTO public.supplies(id,tenant_id,name,current_wholesale_cost,current_stock,measurement_unit) VALUES(cost_supply,t1,'__migration_supply',10,5,'kg');
 UPDATE public.supplies SET current_wholesale_cost=12 WHERE id=cost_supply;
 PERFORM legui_private.test_assert((SELECT count(*)=2 FROM public.supply_cost_history WHERE supply_id=cost_supply),'cost history captures changes');
 PERFORM legui_private.test_rejected(format('DELETE FROM public.inventory_movements WHERE tenant_id=%L',t1),'23514','inventory ledger immutable');

 INSERT INTO public.customers(id,tenant_id,first_name,last_name,email) VALUES(customer2,t2,'Test','Fixture',customer2::text||'@example.invalid');
 PERFORM legui_private.test_rejected(format('INSERT INTO public.orders(tenant_id,customer_id,order_status_id,payment_status_id,subtotal) VALUES(%L,%L,1,1,100)',t1,customer2),'23503','cross-tenant customer rejected');
 INSERT INTO public.orders(id,tenant_id,order_status_id,payment_status_id,subtotal) VALUES(o1,t1,1,1,100);
 UPDATE public.orders SET order_status_id=2 WHERE id=o1;
 PERFORM legui_private.test_assert((SELECT count(*)=2 FROM public.order_status_events WHERE order_id=o1),'order lifecycle automatically recorded');
 INSERT INTO public.order_details(id,tenant_id,order_id,product_id,product_name,quantity,list_price_unit,final_price_unit,frozen_cost_unit) VALUES(d1,t1,o1,p1,'Fixture',1,100,100,40);
 INSERT INTO public.sales(tenant_id,order_id,order_detail_id,product_id,product_name,quantity,frozen_cost_unit,total_list_price,total_net_revenue) VALUES(t1,o1,d1,p1,'Fixture',1,40,100,100);
 PERFORM legui_private.test_rejected(format('INSERT INTO public.sales(tenant_id,order_id,order_detail_id,product_id,product_name,quantity,frozen_cost_unit,total_list_price,total_net_revenue) VALUES(%L,%L,%L,%L,''Fixture'',1,40,100,100)',t1,o1,d1,p1),'23505','sale posting once per line');
 INSERT INTO public.order_payments(id,tenant_id,order_id,currency,kind,amount,status,method,settled_at,idempotency_key,request_digest)
 VALUES(charge1,t1,o1,'ARS','charge',60,'succeeded','cash',now(),'charge-one',repeat('a',64));
 PERFORM legui_private.test_assert((SELECT total_paid=60 AND payment_status_id=2 FROM public.orders WHERE id=o1),'partial order payment');
 INSERT INTO public.order_payments(tenant_id,order_id,currency,kind,amount,status,method,settled_at,idempotency_key,request_digest)
 VALUES(t1,o1,'ARS','charge',40,'succeeded','cash',now(),'charge-two',repeat('b',64));
 PERFORM legui_private.test_assert((SELECT total_paid=100 AND payment_status_id=3 FROM public.orders WHERE id=o1),'full order payment');
 PERFORM legui_private.test_rejected(format('INSERT INTO public.order_payments(tenant_id,order_id,currency,kind,amount,status,method,settled_at,idempotency_key,request_digest) VALUES(%L,%L,''ARS'',''charge'',1,''succeeded'',''cash'',now(),''overcharge'',%L)',t1,o1,repeat('c',64)),'23514','overpayment rejected');
 INSERT INTO public.order_payments(tenant_id,order_id,currency,kind,original_payment_id,amount,status,method,settled_at,idempotency_key,request_digest)
 VALUES(t1,o1,'ARS','refund',charge1,20,'succeeded','cash',now(),'refund-one',repeat('d',64));
 PERFORM legui_private.test_assert((SELECT total_paid=80 AND payment_status_id=5 FROM public.orders WHERE id=o1),'refund reconciles order balance');
 PERFORM legui_private.test_rejected(format('UPDATE public.orders SET total_paid=999 WHERE id=%L',o1),'23514','paid total cannot be fabricated');
 PERFORM legui_private.test_rejected(format('UPDATE public.order_payments SET amount=1 WHERE id=%L',charge1),'23514','successful order payment immutable');
 PERFORM legui_private.test_rejected(format('DELETE FROM public.order_payments WHERE id=%L',charge1),'23514','payment history cannot be erased');
 PERFORM legui_private.test_rejected(format('INSERT INTO public.order_payments(tenant_id,order_id,currency,environment,kind,amount,status,method,idempotency_key,request_digest) VALUES(%L,%L,''ARS'',''sandbox'',''charge'',1,''pending'',''cash'',''wrong-environment'',%L)',t1,o1,repeat('a',64)),'23503','sandbox payment cannot attach to production order');
 PERFORM legui_private.test_rejected(format('INSERT INTO public.order_payments(tenant_id,order_id,currency,kind,amount,status,method,idempotency_key,request_digest) VALUES(%L,%L,''ARS'',''charge'',''NaN'',''pending'',''cash'',''nan'',%L)',t1,o1,repeat('a',64)),'23514','nonfinite money rejected');

 INSERT INTO legui_private.billing_plans(id,code,name) VALUES(plan_id,'migration_fixture_plan','Fixture plan');
 INSERT INTO legui_private.billing_plan_prices(id,plan_id,currency,amount,interval_months,version,billing_unit) VALUES(price_id,plan_id,'ARS',100,1,1,'brand');
 PERFORM legui_private.test_rejected(format('UPDATE legui_private.billing_plan_prices SET amount=200 WHERE id=%L',price_id),'23514','price version immutable');
 INSERT INTO legui_private.billing_accounts(id,brand_id,billing_unit,currency) VALUES(account_id,b1,'brand','ARS');
 PERFORM legui_private.test_rejected(format('INSERT INTO legui_private.billing_accounts(brand_id,tenant_id,billing_unit) VALUES(%L,%L,''tenant'')',b1,t2),'23503','billing account tenant belongs to its brand');
 INSERT INTO legui_private.subscriptions(id,account_id,plan_price_id,currency,interval_months,billing_unit,contracted_amount,environment,status,current_period_start,current_period_end)
 VALUES(subscription_id,account_id,price_id,'ARS',1,'brand',100,'production','active',now(),now()+interval '1 month');
 UPDATE legui_private.subscriptions SET cancel_at_period_end=true WHERE id=subscription_id;
 PERFORM legui_private.test_assert((SELECT version=2 FROM legui_private.subscriptions WHERE id=subscription_id),'subscription version increments');
 PERFORM legui_private.test_assert((SELECT count(*)=2 FROM legui_private.subscription_events e WHERE e.subscription_id=fixture.subscription_id AND e.account_id=fixture.account_id),'subscription change history');
 INSERT INTO legui_private.billing_invoices(id,account_id,subscription_id,environment,currency,amount_total,status,period_start,period_end,issued_at,due_at,idempotency_key,request_digest)
 VALUES(invoice_id,account_id,subscription_id,'production','ARS',100,'open',now(),now()+interval '1 month',now(),now()+interval '7 days','invoice-one',repeat('a',64));
 INSERT INTO legui_private.billing_payments(id,account_id,invoice_id,environment,currency,kind,amount,status,method,settled_at,idempotency_key,request_digest)
 VALUES(billing_charge,account_id,invoice_id,'production','ARS','charge',100,'succeeded','transfer',now(),'billing-charge-one',repeat('b',64));
 PERFORM legui_private.test_assert((SELECT amount_paid=100 AND balance_due=0 AND status='paid' FROM legui_private.billing_invoices WHERE id=invoice_id),'membership charge reconciliation');
 PERFORM legui_private.test_rejected(format('UPDATE legui_private.billing_invoices SET amount_total=120 WHERE id=%L',invoice_id),'23514','issued invoice terms immutable');
 PERFORM legui_private.test_rejected(format('UPDATE legui_private.billing_payments SET amount=80 WHERE id=%L',billing_charge),'23514','successful membership payment immutable');
 INSERT INTO legui_private.billing_credit_notes(account_id,invoice_id,environment,currency,amount,reason,idempotency_key,request_digest)
 VALUES(account_id,invoice_id,'production','ARS',50,'Fixture cancellation','credit-one',repeat('c',64));
 INSERT INTO legui_private.billing_payments(account_id,invoice_id,environment,currency,kind,original_payment_id,amount,status,method,settled_at,idempotency_key,request_digest)
 VALUES(account_id,invoice_id,'production','ARS','refund',billing_charge,50,'succeeded','transfer',now(),'billing-refund-one',repeat('d',64));
 PERFORM legui_private.test_assert((SELECT amount_paid=50 AND credit_total=50 AND balance_due=0 FROM legui_private.billing_invoices WHERE id=invoice_id),'credit plus refund does not invent debt');
 INSERT INTO legui_private.billing_webhook_events(provider,environment,external_event_id,event_type,payload_digest) VALUES('fixture-provider','production','event-one','payment.updated',repeat('e',64));
 PERFORM legui_private.test_rejected('INSERT INTO legui_private.billing_webhook_events(provider,environment,external_event_id,event_type,payload_digest) VALUES(''fixture-provider'',''production'',''event-one'',''payment.updated'',repeat(''e'',64))','23505','webhook deduplication');
 PERFORM legui_private.test_rejected(format('INSERT INTO public.activity_events(brand_id,tenant_id,event_name,source) VALUES(%L,%L,''catalog.opened'',''server'')',b1,t2),'23503','activity cannot cross businesses');
 PERFORM legui_private.test_rejected(format('INSERT INTO legui_private.platform_admin_events(actor_user_id,event_name,operation_key,request_digest) VALUES(%L,''billing.changed'',''fixture-op'',%L)',manager_id,repeat('f',64)),'42501','restaurant manager cannot act as platform administrator');

 FOR row IN SELECT schemaname,tablename FROM pg_catalog.pg_tables WHERE schemaname='legui_private' OR (schemaname='public' AND tablename IN ('order_payments','order_status_events','inventory_movements','operating_expenses','activity_events')) LOOP
  PERFORM legui_private.test_assert(NOT has_table_privilege('anon',format('%I.%I',row.schemaname,row.tablename),'SELECT,INSERT,UPDATE,DELETE'),'no anonymous access to '||row.tablename);
  PERFORM legui_private.test_assert(NOT has_table_privilege('authenticated',format('%I.%I',row.schemaname,row.tablename),'SELECT,INSERT,UPDATE,DELETE'),'no direct browser access to '||row.tablename);
 END LOOP;
END $$;
