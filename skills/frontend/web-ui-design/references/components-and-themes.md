# Componentes, estilo y temas

Localizar componentes reales en `apps/web/src/components` y sus consumidores.
Los nombres de este documento son relativos a la raíz del repo.

- Conservar Server/Client Components según la necesidad real de interacción.
  No extender el límite cliente a una pantalla entera por compartir estilos.
- Definir props por comportamiento visible y variantes usadas; mantener
  elementos nativos, handlers y atributos accesibles. Componer con children
  cuando ayude, sin forzar componentes compound o polimorfismo en todos los casos.
- Mantener tipos TypeScript existentes. Una nueva prop debe conservar defaults
  y consumidores; no copiar patterns de Vue, Svelte o versiones antiguas de React.
- Revisar `apps/web/src/app/globals.css`, fonts, iconos y estilos adyacentes.
  Reusar escalas de espaciado y jerarquía; añadir tokens solo si expresan una
  decisión repetida, no un catálogo abstracto para cada valor.
- `next-themes` controla `class="dark"`; Tailwind usa el custom variant `dark`
  de globals.css. Conservar la relación entre clase, variables y preferencia
  del sistema al corregir temas; no introducir un segundo proveedor de tema.
- Comprobar texto, superficies, estados disabled/focus y errores en ambos
  temas. El color puede reforzar un estado, pero debe existir texto o semántica.
- Usar Lucide/iconos del repo con nombre accesible si son la única etiqueta;
  marcar decoración como tal. No mezclar sistemas de iconos sin motivo.

Verificar consumidores y cambios visuales en las pantallas afectadas. No
reemplazar el estilo completo, añadir dependencias de variantes/CSS-in-JS
ni exigir un design system nuevo para una mejora localizada.
