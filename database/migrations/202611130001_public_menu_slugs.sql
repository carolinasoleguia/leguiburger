-- Stable public menu addresses. Original IDs and commercial names remain unchanged.
ALTER TABLE public.brands ADD COLUMN slug text;
ALTER TABLE public.tenants ADD COLUMN slug text;
CREATE UNIQUE INDEX brands_public_slug_key ON public.brands(slug);
CREATE UNIQUE INDEX tenants_brand_public_slug_key ON public.tenants(brand_id,slug);
CREATE FUNCTION legui_private.menu_slug(value text,fallback text) RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $$
 SELECT coalesce(nullif(trim(both '-' FROM left(regexp_replace(translate(lower(value),'áàäâãåéèëêíìïîóòöôõúùüûñç','aaaaaaeeeeiiiiooooouuuunc'),'[^a-z0-9]+','-','g'),64)),''),fallback)
$$;
CREATE FUNCTION legui_private.assign_menu_slug() RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog AS $$
DECLARE base text; candidate text; n integer:=1; taken boolean;
BEGIN
 IF TG_OP='UPDATE' AND OLD.slug IS NOT NULL THEN
  IF NEW.slug IS DISTINCT FROM OLD.slug THEN RAISE EXCEPTION 'Public menu address is immutable' USING ERRCODE='23514'; END IF;
  RETURN NEW;
 END IF;
 IF TG_TABLE_NAME='brands' THEN base:=legui_private.menu_slug(NEW.name,'marca');
 ELSE base:=legui_private.menu_slug(coalesce(nullif(NEW.display_name,''),NEW.subdomain),'local'); END IF;
 -- Serialize allocation within each namespace; unique indexes also guard concurrent writes.
 IF TG_TABLE_NAME='brands' THEN PERFORM pg_advisory_xact_lock(hashtextextended('menu-slug:brands',0));
 ELSE PERFORM pg_advisory_xact_lock(hashtextextended('menu-slug:tenants:'||NEW.brand_id::text,0)); END IF;
 LOOP
  candidate:=base||CASE WHEN n=1 THEN '' ELSE '-'||n::text END;
  IF TG_TABLE_NAME='brands' THEN SELECT EXISTS(SELECT 1 FROM public.brands WHERE slug=candidate) INTO taken;
  ELSE SELECT EXISTS(SELECT 1 FROM public.tenants WHERE brand_id=NEW.brand_id AND slug=candidate) INTO taken; END IF;
  EXIT WHEN NOT taken; n:=n+1;
 END LOOP;
 NEW.slug:=candidate; RETURN NEW;
END $$;
CREATE TRIGGER assign_public_menu_slug BEFORE INSERT OR UPDATE OF slug ON public.brands FOR EACH ROW EXECUTE FUNCTION legui_private.assign_menu_slug();
CREATE TRIGGER assign_public_menu_slug BEFORE INSERT OR UPDATE OF slug ON public.tenants FOR EACH ROW EXECUTE FUNCTION legui_private.assign_menu_slug();
-- Deterministic assignment for existing records, including unpublished menus.
DO $$ DECLARE row_id uuid; BEGIN
 FOR row_id IN SELECT id FROM public.brands ORDER BY created_at,id LOOP UPDATE public.brands SET slug=slug WHERE id=row_id; END LOOP;
 FOR row_id IN SELECT id FROM public.tenants ORDER BY created_at,id LOOP UPDATE public.tenants SET slug=slug WHERE id=row_id; END LOOP;
END $$;
ALTER TABLE public.brands ALTER COLUMN slug SET NOT NULL,ADD CONSTRAINT brands_public_slug_format CHECK(slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug)<=80);
ALTER TABLE public.tenants ALTER COLUMN slug SET NOT NULL,ADD CONSTRAINT tenants_public_slug_format CHECK(slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug)<=80);
REVOKE ALL ON FUNCTION legui_private.menu_slug(text,text),legui_private.assign_menu_slug() FROM PUBLIC,anon,authenticated,service_role;
