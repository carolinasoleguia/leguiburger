export function optionGroups(product, extras = []) {
  return product?.options_configured
    ? product.option_groups || []
    : extras.length
      ? [
          {
            id: "legacy",
            name: "¿Le sumamos algo?",
            min: 0,
            max: 50,
            options: extras,
          },
        ]
      : [];
}
export function optionsComplete(product, selections, extras = []) {
  const groups = optionGroups(product, extras),
    known = new Set(groups.flatMap((g) => g.options.map((x) => x.id)));
  if (
    selections.length > 10 ||
    selections.some(
      (x) =>
        !known.has(x.id) ||
        !Number.isInteger(x.quantity) ||
        x.quantity < 1 ||
        x.quantity > 5,
    )
  )
    return false;
  return groups.every((g) => {
    const ids = new Set(g.options.map((x) => x.id)),
      n = selections
        .filter((x) => ids.has(x.id))
        .reduce((n, x) => n + x.quantity, 0);
    return n >= g.min && n <= g.max;
  });
}
export function selectedOptions(product, selections, extras = []) {
  const selected = new Map(selections.map((x) => [x.id, x.quantity]));
  return optionGroups(product, extras).flatMap((g) =>
    g.options
      .filter((x) => selected.has(x.id))
      .map((x) => ({
        ...x,
        quantity: selected.get(x.id),
        group_name: product?.options_configured ? g.name : "",
      })),
  );
}
function cents(value) {
  const [whole, fraction = ""] = String(value || "0").split(".");
  return BigInt(whole) * 100n + BigInt((fraction + "00").slice(0, 2));
}
export function optionsPrice(base, extras) {
  const value = extras.reduce(
    (n, x) => n + cents(x.price) * BigInt(x.quantity || 1),
    cents(base),
  );
  return `${value / 100n}.${String(value % 100n).padStart(2, "0")}`;
}
