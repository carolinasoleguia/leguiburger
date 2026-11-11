import { computed, ref } from "vue";
const media = window.matchMedia("(prefers-reduced-motion: reduce)");
const systemReduced = ref(media.matches);
const saved = localStorage.getItem("legui-motion");
const preference = ref(["full", "reduced"].includes(saved) ? saved : "auto");
document.documentElement.dataset.motion = preference.value;
media.addEventListener(
  "change",
  (event) => (systemReduced.value = event.matches),
);
export function useMotion() {
  const reduced = computed(
    () =>
      preference.value === "reduced" ||
      (preference.value === "auto" && systemReduced.value),
  );
  function toggle() {
    preference.value = reduced.value ? "full" : "reduced";
    document.documentElement.dataset.motion = preference.value;
    localStorage.setItem("legui-motion", preference.value);
  }
  return { reduced, toggle };
}
