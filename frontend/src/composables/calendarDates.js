const zone = "America/Argentina/Buenos_Aires";
export function todayKey() {
  return new Intl.DateTimeFormat("en-CA", { timeZone: zone }).format(
    new Date(),
  );
}
export function dateFor(key) {
  return new Date(`${key}T12:00:00Z`);
}
export function shiftDay(key, amount) {
  const date = dateFor(key);
  date.setUTCDate(date.getUTCDate() + amount);
  return date.toISOString().slice(0, 10);
}
export function datesBetween(from, to) {
  const count = Math.round((dateFor(to) - dateFor(from)) / 86400000) + 1;
  return Array.from({ length: Math.max(0, Math.min(count, 366)) }, (_, i) =>
    shiftDay(from, i),
  );
}
export function formatDay(key, options = { day: "numeric", month: "short" }) {
  return new Intl.DateTimeFormat("es-AR", {
    timeZone: zone,
    ...options,
  }).format(dateFor(key));
}
