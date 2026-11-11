package operations

import (
	"context"
	"gorm.io/gorm"
)

// Durable source identities make periodic scans restartable without duplicate inbox entries.
// Managers see financial alerts; staff only receive preparation and stock alerts.
func GenerateNotifications(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Exec(`WITH recipients AS (
 SELECT 'user' kind,u.id,u.role,t.id tenant_id FROM public.users u JOIN public.tenants t ON t.brand_id=u.brand_id AND t.active WHERE u.is_active AND u.role='admin'
 UNION ALL SELECT 'employee',e.id,e.role,e.tenant_id FROM public.employees e JOIN public.tenants t ON t.id=e.tenant_id AND t.active WHERE e.is_active AND e.role IN ('employee','cashier','kitchen')
 ), alerts AS(
 SELECT tp.tenant_id,'stock:'||tp.tenant_id||':'||tp.product_id||':'||tp.stock_version event_key,'Queda poco stock' title,p.name||' llegó al mínimo configurado.' message,'/inventory' destination,false financial FROM public.tenant_products tp JOIN public.products p ON p.id=tp.product_id WHERE tp.is_active AND tp.track_stock AND tp.current_stock<=tp.minimum_stock
 UNION ALL SELECT s.tenant_id,'supply:'||s.id||':'||s.stock_version,'Revisá tus insumos',s.name||' llegó al mínimo configurado.','/inventory',false FROM public.supplies s WHERE s.is_active AND s.current_stock<=s.minimum_stock
 UNION ALL SELECT o.tenant_id,'order:'||o.id||':'||o.version,'Pedido listo para preparar','Hay un pedido aceptado esperando en cocina.','/kitchen',false FROM public.orders o JOIN public.order_statuses_lookup st ON st.id=o.order_status_id WHERE st.code='accepted' AND o.payment_environment='production' AND o.created_at>clock_timestamp()-interval '1 day'
 UNION ALL SELECT x.tenant_id,'payment:'||x.id,'Cobro confirmado','Un pago fue confirmado y aplicado al pedido.','/orders',true FROM public.order_payments x WHERE x.environment='production' AND x.status='succeeded' AND x.kind='charge' AND x.created_at>clock_timestamp()-interval '7 days'
 ) INSERT INTO legui_private.notifications(principal_kind,principal_id,event_key,title,message,destination,brand_id,tenant_id)
 SELECT DISTINCT r.kind,r.id,a.event_key,a.title,a.message,a.destination,t.brand_id,t.id FROM recipients r JOIN alerts a ON a.tenant_id=r.tenant_id JOIN public.tenants t ON t.id=a.tenant_id WHERE NOT a.financial OR r.role='admin' ON CONFLICT DO NOTHING;
 INSERT INTO legui_private.notifications(principal_kind,principal_id,event_key,title,message,destination,brand_id,tenant_id)
 SELECT 'user',u.id,'membership:'||s.id||':'||s.version,'Revisá tu membresía','La membresía registra un pago pendiente o su renovación está cancelada.','/payments',a.brand_id,a.tenant_id
 FROM public.users u JOIN legui_private.billing_accounts a ON a.brand_id=u.brand_id JOIN legui_private.subscriptions s ON s.account_id=a.id
 WHERE u.is_active AND u.role='admin' AND s.environment='production' AND (s.status='past_due' OR s.cancel_at_period_end) ON CONFLICT DO NOTHING;
 INSERT INTO legui_private.notifications(principal_kind,principal_id,event_key,title,message,destination,brand_id,tenant_id)
 SELECT 'user',a.invited_by,'invitation:'||a.id,'Invitación enviada','Podés seguir el estado del acceso de tu equipo.','/team/invitations',t.brand_id,a.tenant_id
 FROM legui_private.auth_action_tokens a JOIN public.users u ON u.id=a.invited_by AND u.is_active JOIN public.tenants t ON t.id=a.tenant_id WHERE a.purpose='team_invitation' AND a.created_at>clock_timestamp()-interval '7 days' ON CONFLICT DO NOTHING;
 INSERT INTO legui_private.notifications(principal_kind,principal_id,event_key,title,message,destination,brand_id,tenant_id)
 SELECT 'user',u.id,'business:'||b.id,'Un nuevo negocio se sumó','Revisá la marca y clasificá su uso para mantener correctas las métricas.','/owner/brands',b.id,NULL
 FROM public.users u CROSS JOIN public.brands b WHERE u.is_active AND u.role IN ('owner','super_admin') AND b.created_at>clock_timestamp()-interval '7 days' ON CONFLICT DO NOTHING;`).Error
}
