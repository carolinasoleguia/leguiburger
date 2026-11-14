-- A current business contract has priority. If billing is per location, use the
-- latest current location contract as the business capacity policy. Quotas are
-- not added together when multiple invoices/accounts exist.
CREATE OR REPLACE FUNCTION legui_private.business_capacity(b uuid) RETURNS TABLE(locations integer,collaborators integer,source text)
 LANGUAGE sql STABLE SET search_path='' AS $$
 WITH plan AS(SELECT p.maximum_locations,p.maximum_collaborators FROM legui_private.subscriptions s
 JOIN legui_private.billing_accounts a ON a.id=s.account_id AND a.closed_at IS NULL
 JOIN legui_private.billing_plan_prices p ON p.id=s.plan_price_id
 WHERE a.brand_id=b AND s.environment='production'
 AND ((s.status='active' AND s.current_period_start<=now() AND s.current_period_end>now()) OR (s.status='trialing' AND s.trial_ends_at>now()))
 ORDER BY (a.tenant_id IS NULL) DESC,s.started_at DESC,s.id LIMIT 1)
 SELECT p.maximum_locations,p.maximum_collaborators,'subscription'::text FROM plan p
 UNION ALL SELECT x.base_location_limit,x.base_collaborator_limit,'base'::text FROM public.brands x WHERE x.id=b AND NOT EXISTS(SELECT 1 FROM plan)
$$;
REVOKE ALL ON FUNCTION legui_private.business_capacity(uuid) FROM PUBLIC,anon,authenticated,service_role;
