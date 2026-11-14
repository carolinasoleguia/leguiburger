package operations

import (
	"net/http"
)

func (h *Handler) restaurantAnalytics(r *http.Request, p Principal) (Result, error) {
	scope, err := h.businessScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	zone := "America/Argentina/Buenos_Aires"
	if scope.TenantID != "" {
		if err = h.repo.DB.WithContext(r.Context()).Raw(`SELECT timezone FROM public.tenants WHERE id=?`, scope.TenantID).Scan(&zone).Error; err != nil {
			return Result{}, err
		}
	}
	period, err := readPeriod(r, zone)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), restaurantAnalyticsSQL, scope.BrandID, scope.TenantID, period.From, period.Until, period.Timezone))
}
func (h *Handler) operationalAnalytics(r *http.Request, p Principal) (Result, error) {
	scope, err := h.tenantScope(r.Context(), p, r)
	if err != nil {
		return Result{}, err
	}
	zone := "America/Argentina/Buenos_Aires"
	if err = h.repo.DB.WithContext(r.Context()).Raw(`SELECT timezone FROM public.tenants WHERE id=?`, scope.TenantID).Scan(&zone).Error; err != nil {
		return Result{}, err
	}
	period, err := readPeriod(r, zone)
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), operationalAnalyticsSQL, scope.TenantID, period.From, period.Until, period.Timezone))
}
func (h *Handler) platformAnalytics(r *http.Request, p Principal) (Result, error) {
	if !p.Platform() {
		return Result{}, forbidden()
	}
	period, err := readPeriod(r, "America/Argentina/Buenos_Aires")
	if err != nil {
		return Result{}, err
	}
	return resultJSON(h.repo.JSON(r.Context(), platformAnalyticsSQL, period.From, period.Until, period.Timezone))
}

const restaurantAnalyticsSQL = `WITH params AS (
 SELECT ?::uuid brand_id,nullif(?,'')::uuid tenant_id,?::timestamptz starts,?::timestamptz ends,?::text tz
), locations AS(SELECT t.* FROM public.tenants t,params p WHERE t.brand_id=p.brand_id AND (p.tenant_id IS NULL OR t.id=p.tenant_id)),
 selected_orders AS(SELECT o.*,s.code status FROM public.orders o JOIN locations l ON l.id=o.tenant_id JOIN public.order_statuses_lookup s ON s.id=o.order_status_id,params p WHERE o.payment_environment='production' AND o.created_at>=p.starts AND o.created_at<p.ends),
 selected_sales AS(SELECT s.* FROM public.sales s JOIN locations l ON l.id=s.tenant_id,params p WHERE s.payment_environment='production' AND s.voided_at IS NULL AND s.sale_date>=p.starts AND s.sale_date<p.ends),
 money AS(SELECT currency,sum(total_net_revenue) sales,count(DISTINCT order_id) orders,count(*) FILTER(WHERE frozen_cost_unit IS NULL) missing_cost_lines,
 CASE WHEN count(*) FILTER(WHERE frozen_cost_unit IS NULL)>0 THEN NULL ELSE sum(total_net_revenue-quantity*frozen_cost_unit) END gross_margin FROM selected_sales GROUP BY currency),
 cash AS(SELECT x.currency,sum(CASE x.kind WHEN 'charge' THEN x.amount ELSE -x.amount END) collected,
 sum(x.amount) FILTER(WHERE x.kind='refund') refunds FROM public.order_payments x JOIN locations l ON l.id=x.tenant_id,params p WHERE x.environment='production' AND x.status='succeeded' AND x.settled_at>=p.starts AND x.settled_at<p.ends GROUP BY x.currency),
 costs AS(SELECT e.currency,sum(e.amount) amount FROM public.operating_expenses e JOIN locations l ON l.id=e.tenant_id,params p WHERE e.voided_at IS NULL AND e.incurred_at>=p.starts AND e.incurred_at<p.ends GROUP BY e.currency),
 dues AS(SELECT o.currency,sum(o.total_due-o.total_paid) outstanding FROM public.orders o JOIN locations l ON l.id=o.tenant_id JOIN public.order_statuses_lookup st ON st.id=o.order_status_id WHERE o.payment_environment='production' AND st.code NOT IN ('canceled','rejected') GROUP BY o.currency),
currencies AS(SELECT currency FROM money UNION SELECT currency FROM cash UNION SELECT currency FROM costs UNION SELECT currency FROM dues),
 daily AS(SELECT (s.sale_date AT TIME ZONE p.tz)::date AS day,s.currency,sum(s.total_net_revenue) total,count(DISTINCT s.order_id) orders FROM selected_sales s,params p GROUP BY 1,2),
 client_first AS(SELECT o.customer_id,min(o.created_at) first_order FROM public.orders o JOIN locations l ON l.id=o.tenant_id JOIN public.order_statuses_lookup st ON st.id=o.order_status_id WHERE o.customer_id IS NOT NULL AND o.payment_environment='production' AND st.code IN ('accepted','preparing','ready','dispatched','delivered') GROUP BY o.customer_id),
 clients AS(SELECT DISTINCT o.customer_id,f.first_order FROM selected_orders o JOIN client_first f ON f.customer_id=o.customer_id WHERE o.status IN ('accepted','preparing','ready','dispatched','delivered')),
 recent AS(SELECT o.id,o.tenant_id,o.created_at,o.currency,o.total_due,o.total_paid,o.status FROM selected_orders o ORDER BY o.created_at DESC,o.id DESC LIMIT 8)
SELECT jsonb_build_object(
 'scope',jsonb_build_object('brand_id',p.brand_id,'tenant_id',p.tenant_id),
 'period',jsonb_build_object('from',p.starts,'until_exclusive',p.ends,'timezone',p.tz),
 'metrics',jsonb_build_object('orders_created',(SELECT count(*) FROM selected_orders),'completed_orders',(SELECT count(*) FROM selected_orders WHERE status='delivered'),
 'canceled_orders',(SELECT count(*) FROM selected_orders WHERE status IN ('canceled','rejected')),
 'acceptance_rate',(SELECT 100.0*count(*) FILTER(WHERE status NOT IN ('pending','rejected','canceled'))/nullif(count(*) FILTER(WHERE status<>'pending'),0) FROM selected_orders),
 'new_customers',(SELECT count(*) FROM clients WHERE first_order>=p.starts),'returning_customers',(SELECT count(*) FROM clients WHERE first_order<p.starts),
 'has_financial_data',EXISTS(SELECT 1 FROM selected_sales) OR EXISTS(SELECT 1 FROM cash) OR EXISTS(SELECT 1 FROM costs)),
 'money',coalesce((SELECT jsonb_agg(jsonb_build_object('currency',c.currency,'sales',coalesce(m.sales,0),'sales_orders',coalesce(m.orders,0),
  'average_ticket',CASE WHEN m.orders>0 THEN round(m.sales/m.orders,2) ELSE NULL END,'collected',coalesce(x.collected,0),'refunds',coalesce(x.refunds,0),
  'outstanding',coalesce(debt.outstanding,0),'gross_margin',CASE WHEN m.orders IS NULL THEN 0 ELSE m.gross_margin END,'operating_expenses',coalesce(e.amount,0),'estimated_operating_result',(CASE WHEN m.orders IS NULL THEN 0 ELSE m.gross_margin END)-coalesce(e.amount,0),
  'missing_cost_lines',coalesce(m.missing_cost_lines,0)) ORDER BY c.currency) FROM currencies c LEFT JOIN money m USING(currency) LEFT JOIN cash x USING(currency) LEFT JOIN costs e USING(currency) LEFT JOIN dues debt USING(currency)),'[]'::jsonb),
 'daily_sales',coalesce((SELECT jsonb_agg(jsonb_build_object('day',calendar.day::date,'currency',c.currency,'total',coalesce(d.total,0),'orders',coalesce(d.orders,0)) ORDER BY calendar.day,c.currency)
  FROM generate_series((p.starts AT TIME ZONE p.tz)::date,((p.ends-interval '1 microsecond') AT TIME ZONE p.tz)::date,interval '1 day') calendar(day) CROSS JOIN (SELECT currency FROM money) c LEFT JOIN daily d ON d.day=calendar.day::date AND d.currency=c.currency),'[]'::jsonb),
 'order_states',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT status,count(*) count FROM selected_orders GROUP BY status ORDER BY status)v),'[]'::jsonb),
 'top_products',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT product_id,max(product_name) product_name,currency,sum(quantity) units,sum(total_net_revenue-total_shipping_amount) revenue FROM selected_sales GROUP BY product_id,currency ORDER BY sum(quantity) DESC,product_id LIMIT 10)v),'[]'::jsonb),
 'by_location',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT l.id tenant_id,l.subdomain,s.currency,count(DISTINCT s.order_id) orders,sum(s.total_net_revenue) sales FROM locations l JOIN selected_sales s ON s.tenant_id=l.id GROUP BY l.id,l.subdomain,s.currency ORDER BY sum(s.total_net_revenue) DESC)v),'[]'::jsonb),
 'by_payment_method',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT x.method,x.currency,sum(CASE x.kind WHEN 'charge' THEN x.amount ELSE -x.amount END) amount FROM public.order_payments x JOIN locations l ON l.id=x.tenant_id WHERE x.environment='production' AND x.status='succeeded' AND x.settled_at>=p.starts AND x.settled_at<p.ends GROUP BY x.method,x.currency)v),'[]'::jsonb),
 'recent_orders',coalesce((SELECT jsonb_agg(to_jsonb(recent)) FROM recent),'[]'::jsonb),
 'previous_period',jsonb_build_object('from',p.starts-(p.ends-p.starts),'until_exclusive',p.starts,'sales',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(
 SELECT s.currency,sum(s.total_net_revenue) amount,count(DISTINCT s.order_id) orders FROM public.sales s JOIN locations l ON l.id=s.tenant_id WHERE s.payment_environment='production' AND s.voided_at IS NULL AND s.sale_date>=p.starts-(p.ends-p.starts) AND s.sale_date<p.starts GROUP BY s.currency)v),'[]'::jsonb))
)::text AS payload FROM params p`

const operationalAnalyticsSQL = `WITH params AS(SELECT ?::uuid tenant_id,?::timestamptz starts,?::timestamptz ends,?::text tz)
SELECT jsonb_build_object('tenant_id',p.tenant_id,'period',jsonb_build_object('from',p.starts,'until_exclusive',p.ends,'timezone',p.tz),
 'production',jsonb_build_object('reports',count(r.id),'medallions_produced',coalesce(sum(r.medallions_produced),0),'breads_purchased',coalesce(sum(r.breads_purchased),0)),
 'daily_production',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT production_date,sum(medallions_produced) medallions_produced,sum(breads_purchased) breads_purchased FROM public.production_reports WHERE tenant_id=p.tenant_id AND production_date>=(p.starts AT TIME ZONE p.tz)::date AND production_date<(p.ends AT TIME ZONE p.tz)::date GROUP BY production_date ORDER BY production_date)v),'[]'::jsonb),
 'stock',jsonb_build_object('products',(SELECT count(*) FROM public.tenant_products WHERE tenant_id=p.tenant_id AND is_active),
 'low_stock_products',(SELECT count(*) FROM public.tenant_products WHERE tenant_id=p.tenant_id AND is_active AND track_stock AND current_stock<=minimum_stock),
 'unavailable_products',(SELECT count(*) FROM public.tenant_products WHERE tenant_id=p.tenant_id AND is_active AND NOT is_available)),
 'orders_waiting',(SELECT count(*) FROM public.orders o JOIN public.order_statuses_lookup s ON s.id=o.order_status_id WHERE o.tenant_id=p.tenant_id AND o.payment_environment='production' AND s.code IN ('pending','accepted','preparing','ready'))
)::text AS payload FROM params p LEFT JOIN public.production_reports r ON r.tenant_id=p.tenant_id AND r.production_date>=(p.starts AT TIME ZONE p.tz)::date AND r.production_date<(p.ends AT TIME ZONE p.tz)::date GROUP BY p.tenant_id,p.starts,p.ends,p.tz`

const platformAnalyticsSQL = `WITH params AS(SELECT ?::timestamptz starts,?::timestamptz ends,?::text tz),
 production_brands AS(SELECT id FROM public.brands WHERE data_classification='production'),
 accounts AS(SELECT a.* FROM legui_private.billing_accounts a JOIN production_brands b ON b.id=a.brand_id),
 subscriptions AS(SELECT s.* FROM legui_private.subscriptions s JOIN accounts a ON a.id=s.account_id WHERE s.environment='production'),
 cash AS(SELECT x.currency,sum(CASE x.kind WHEN 'charge' THEN x.amount ELSE -x.amount END) collected,sum(x.amount) FILTER(WHERE x.kind='refund') refunds FROM legui_private.billing_payments x JOIN accounts a ON a.id=x.account_id,params p WHERE x.environment='production' AND x.status='succeeded' AND x.settled_at>=p.starts AND x.settled_at<p.ends GROUP BY x.currency),
 recurring AS(SELECT s.currency,sum(s.contracted_amount*s.quantity/s.interval_months) mrr FROM subscriptions s JOIN accounts a ON a.id=s.account_id WHERE s.status='active' AND s.current_period_end>now() AND s.current_period_start<=now() AND a.closed_at IS NULL GROUP BY s.currency),
 debt AS(SELECT i.currency,sum(i.balance_due) outstanding,sum(i.balance_due) FILTER(WHERE i.due_at<now()) overdue FROM legui_private.billing_invoices i JOIN accounts a ON a.id=i.account_id WHERE i.environment='production' AND i.status IN ('open','uncollectible') GROUP BY i.currency),
 currencies AS(SELECT currency FROM cash UNION SELECT currency FROM recurring UNION SELECT currency FROM debt),
 active_start AS(SELECT s.id FROM subscriptions s,params p WHERE (SELECT e.status FROM legui_private.subscription_events e WHERE e.subscription_id=s.id AND e.occurred_at<p.starts ORDER BY e.occurred_at DESC,e.subscription_version DESC LIMIT 1)='active'),
 usage AS(SELECT DISTINCT e.brand_id FROM public.activity_events e JOIN production_brands b ON b.id=e.brand_id,params p WHERE e.environment='production' AND e.source='server' AND e.occurred_at>=p.starts AND e.occurred_at<p.ends)
SELECT jsonb_build_object('period',jsonb_build_object('from',p.starts,'until_exclusive',p.ends,'timezone',p.tz),
 'businesses',jsonb_build_object('registered',(SELECT count(*) FROM public.brands),'production',(SELECT count(*) FROM production_brands),
 'unclassified',(SELECT count(*) FROM public.brands WHERE data_classification='unclassified'),
 'registered_in_period',(SELECT count(*) FROM public.brands WHERE created_at>=p.starts AND created_at<p.ends),
 'production_registered_in_period',(SELECT count(*) FROM public.brands WHERE data_classification='production' AND created_at>=p.starts AND created_at<p.ends),
 'activated_in_period',(SELECT count(*) FROM public.brands WHERE data_classification='production' AND activated_at>=p.starts AND activated_at<p.ends),
 'used_in_period',(SELECT count(*) FROM usage),
 'active_last_day',(SELECT count(DISTINCT e.brand_id) FROM public.activity_events e JOIN production_brands b ON b.id=e.brand_id WHERE e.environment='production' AND e.source='server' AND e.occurred_at>=now()-interval '1 day' AND e.occurred_at<=now()),
 'active_last_week',(SELECT count(DISTINCT e.brand_id) FROM public.activity_events e JOIN production_brands b ON b.id=e.brand_id WHERE e.environment='production' AND e.source='server' AND e.occurred_at>=now()-interval '7 days' AND e.occurred_at<=now()),
 'active_last_month',(SELECT count(DISTINCT e.brand_id) FROM public.activity_events e JOIN production_brands b ON b.id=e.brand_id WHERE e.environment='production' AND e.source='server' AND e.occurred_at>=now()-interval '30 days' AND e.occurred_at<=now()),'paying',(SELECT count(DISTINCT a.brand_id) FROM accounts a JOIN subscriptions s ON s.account_id=a.id WHERE s.status='active' AND s.current_period_end>now() AND s.contracted_amount>0 AND EXISTS(SELECT 1 FROM legui_private.billing_payments x WHERE x.account_id=a.id AND x.environment='production' AND x.kind='charge' AND x.status='succeeded'))),
 'locations',jsonb_build_object('registered',(SELECT count(*) FROM public.tenants),'enabled',(SELECT count(*) FROM public.tenants WHERE active),'registered_in_period',(SELECT count(*) FROM public.tenants WHERE created_at>=p.starts AND created_at<p.ends)),
 'subscriptions',jsonb_build_object('active',(SELECT count(*) FROM subscriptions WHERE status='active' AND current_period_end>now()),'trialing',(SELECT count(*) FROM subscriptions WHERE status='trialing' AND trial_ends_at>now()),'past_due',(SELECT count(*) FROM subscriptions WHERE status='past_due'),
 'canceled_in_period',(SELECT count(*) FROM subscriptions WHERE ended_at>=p.starts AND ended_at<p.ends),
 'active_at_period_start',(SELECT count(*) FROM active_start),
 'churn_rate',(SELECT 100.0*count(*) FILTER(WHERE s.ended_at>=p.starts AND s.ended_at<p.ends)/nullif(count(*),0) FROM subscriptions s JOIN active_start a ON a.id=s.id)),
 'money',coalesce((SELECT jsonb_agg(jsonb_build_object('currency',c.currency,'collected',coalesce(x.collected,0),'refunds',coalesce(x.refunds,0),'active_mrr',coalesce(r.mrr,0),'active_arr',coalesce(r.mrr,0)*12,'outstanding',coalesce(d.outstanding,0),'overdue',coalesce(d.overdue,0)) ORDER BY c.currency) FROM currencies c LEFT JOIN cash x USING(currency) LEFT JOIN recurring r USING(currency) LEFT JOIN debt d USING(currency)),'[]'::jsonb),
 'daily_new_businesses',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT (created_at AT TIME ZONE p.tz)::date AS day,count(*) registered FROM public.brands WHERE created_at>=p.starts AND created_at<p.ends GROUP BY 1 ORDER BY 1)v),'[]'::jsonb),
 'activity_by_feature',coalesce((SELECT jsonb_agg(to_jsonb(v)) FROM(SELECT e.event_name,count(*) events,count(DISTINCT e.brand_id) businesses FROM public.activity_events e JOIN production_brands b ON b.id=e.brand_id WHERE e.environment='production' AND e.source='server' AND e.occurred_at>=p.starts AND e.occurred_at<p.ends GROUP BY e.event_name ORDER BY count(*) DESC LIMIT 20)v),'[]'::jsonb),
 'revenue_definition','Only production payments of businesses classified as production. Membership revenue is separate from restaurant sales.',
 'mrr_definition','Active, current contractual recurring amount normalized to months; annual amount / 12. Not cash collected.'
)::text AS payload FROM params p`
