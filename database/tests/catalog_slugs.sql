DO $$ DECLARE a uuid:=gen_random_uuid(); b uuid:=gen_random_uuid(); x uuid:=gen_random_uuid(); y uuid:=gen_random_uuid(); z uuid:=gen_random_uuid(); first_slug text; local_slug text; BEGIN
 INSERT INTO public.brands(id,name,base_location_limit) VALUES(a,'Prueba Ñandú Slugs',4),(b,'Prueba Ñandú Slugs',4);
 SELECT slug INTO first_slug FROM public.brands WHERE id=a;
 IF first_slug NOT LIKE 'prueba-nandu-slugs%' OR (SELECT slug FROM public.brands WHERE id=b)=first_slug THEN RAISE EXCEPTION 'Brand normalization/uniqueness failed: % / %',first_slug,current_setting('client_encoding')||'/'||current_setting('server_encoding');END IF;
 INSERT INTO public.tenants(id,brand_id,subdomain,display_name) VALUES(x,a,'slug-'||x::text,'Centro Ñandú'),(y,a,'slug-'||y::text,'Centro Ñandú'),(z,b,'slug-'||z::text,'Centro Ñandú');
 SELECT slug INTO local_slug FROM public.tenants WHERE id=x;
 IF local_slug<>'centro-nandu' OR (SELECT slug FROM public.tenants WHERE id=y)<>'centro-nandu-2' OR (SELECT slug FROM public.tenants WHERE id=z)<>local_slug THEN RAISE EXCEPTION 'Location uniqueness scope failed';END IF;
 UPDATE public.brands SET name='Otro nombre' WHERE id=a;UPDATE public.tenants SET display_name='Otra sucursal' WHERE id=x;
 IF (SELECT slug FROM public.brands WHERE id=a)<>first_slug OR (SELECT slug FROM public.tenants WHERE id=x)<>local_slug THEN RAISE EXCEPTION 'Rename broke public address';END IF;
 BEGIN UPDATE public.brands SET slug='changed' WHERE id=a;RAISE EXCEPTION 'Mutable slug';EXCEPTION WHEN check_violation THEN NULL;END;
END $$;
