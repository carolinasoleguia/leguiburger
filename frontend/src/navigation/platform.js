const item = (label, to, icon) => ({ label, to, icon });
export function platformWorkspace() {
  return {
    label: "Plataforma LeguiBurger",
    accountLabel: "Dueño de la app",
    locationSwitcher: false,
    home: item("Resumen de la app", "/owner", "grid"),
    groups: [
      {
        key: "operation",
        label: "Operación de locales",
        icon: "receipt",
        items: [
          item("Caja y pedidos", "/orders", "receipt"),
          item("Turnos de caja", "/cash", "card"),
          item("Cocina", "/kitchen", "flame"),
          item("Stock e insumos", "/inventory", "box"),
          item("Gastos", "/expenses", "receipt"),
        ],
      },
      {
        key: "menu",
        label: "Carta y atención",
        icon: "burger",
        items: [
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
        label: "Plataforma y negocios",
        icon: "store",
        items: [
          item("Restaurantes", "/restaurants", "burger"),
          item("Locales", "/locations", "store"),
          item("Cobros y membresías", "/payments", "card"),
          item("Invitaciones", "/team/invitations", "users"),
          item("Estado del sistema", "/platform/health", "clock"),
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
