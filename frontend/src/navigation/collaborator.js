const item = (label, to, icon) => ({ label, to, icon });
export function collaboratorWorkspace(role, tenant) {
  const counter = role === "cashier";
  return {
    label: "Mi espacio de trabajo",
    accountLabel: "Equipo",
    locationSwitcher: true,
    home: item("Mi espacio", `/tenant/${tenant}/employee`, "grid"),
    groups: [
      {
        key: "operation",
        label: "Operación",
        icon: "receipt",
        items: [
          ...(counter
            ? [
                item("Caja y pedidos", "/orders", "receipt"),
                item("Turnos de caja", "/cash", "card"),
                item("Disponibilidad de carta", "/menu-availability", "burger"),
                item("Clientes", "/customers", "users"),
                item("Cobros", "/payments", "card"),
              ]
            : []),
          item("Cocina", "/kitchen", "flame"),
          item("Stock e insumos", "/inventory", "box"),
        ],
      },
      {
        key: "account",
        label: "Mi cuenta",
        icon: "shield",
        items: [
          item("Seguridad", "/account/security", "shield"),
          item("Novedades", "/notifications", "clock"),
          item("Soporte y continuidad", "/care", "support"),
        ],
      },
    ],
  };
}
