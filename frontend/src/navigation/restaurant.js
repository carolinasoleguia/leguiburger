const item = (label, to, icon) => ({ label, to, icon });
export function restaurantWorkspace() {
  return {
    label: "Mi restaurante",
    accountLabel: "Administrador",
    locationSwitcher: false,
    home: item("Resumen", "/admin", "grid"),
    groups: [
      {
        key: "operation",
        label: "Operación",
        icon: "receipt",
        items: [
          item("Caja y pedidos", "/orders", "receipt"),
          item("Turnos de caja", "/cash", "card"),
          item("Cocina", "/kitchen", "flame"),
          item("Stock e insumos", "/inventory", "box"),
          item("Gastos", "/expenses", "receipt"),
          item("Producción", "/admin/production", "flame"),
        ],
      },
      {
        key: "menu",
        label: "Carta y atención",
        icon: "burger",
        items: [
          item("Catálogo", "/admin/catalog", "burger"),
          item("Disponibilidad de carta", "/menu-availability", "burger"),
          item("Combos y opciones", "/product-options", "burger"),
          item("Atención del local", "/store-operations", "clock"),
        ],
      },
      {
        key: "customers",
        label: "Clientes y promociones",
        icon: "users",
        items: [
          item("Clientes", "/customers", "users"),
          item("Cupones y compradores", "/marketing", "users"),
          item("Campañas", "/campaigns", "send"),
        ],
      },
      {
        key: "business",
        label: "Mi negocio y equipo",
        icon: "store",
        items: [
          item("Mi restaurante", "/restaurants", "burger"),
          item("Locales", "/locations", "store"),
          item("Equipo", "/team", "users"),
          item("Invitaciones", "/team/invitations", "users"),
          item("Cobros", "/payments", "card"),
          item("Preparar mi negocio", "/onboarding", "store"),
        ],
      },
      {
        key: "account",
        label: "Cuenta y actividad",
        icon: "shield",
        items: [
          item("Seguridad", "/account/security", "shield"),
          item("Actividad", "/audit", "activity"),
          item("Novedades", "/notifications", "clock"),
          item("Soporte y continuidad", "/care", "support"),
        ],
      },
    ],
  };
}
