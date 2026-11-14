import { ref } from "vue";
export const pendingRequests = ref(0);
export const pendingNavigation = ref(false);
const buttons = new WeakMap();
let clicked = null;
export function beginActivity(form) {
  pendingRequests.value++;
  const button =
    clicked || form?.querySelector('button[type="submit"],button:not([type])');
  if (button) {
    const entry = buttons.get(button) || {
      count: 0,
      aria: button.getAttribute("aria-busy"),
    };
    entry.count++;
    buttons.set(button, entry);
    button.classList.add("is-pending");
    button.setAttribute("aria-busy", "true");
  }
  let done = false;
  return () => {
    if (done) return;
    done = true;
    pendingRequests.value = Math.max(0, pendingRequests.value - 1);
    if (button) {
      const entry = buttons.get(button);
      if (entry && --entry.count === 0) {
        button.classList.remove("is-pending");
        if (entry.aria === null) button.removeAttribute("aria-busy");
        else button.setAttribute("aria-busy", entry.aria);
        buttons.delete(button);
      }
    }
  };
}
export function installActivity(router) {
  document.addEventListener(
    "click",
    (event) => {
      clicked = event.target.closest?.("button") || null;
      queueMicrotask(() => {
        clicked = null;
      });
    },
    true,
  );
  router.beforeEach(() => {
    pendingNavigation.value = true;
  });
  router.afterEach(() => {
    pendingNavigation.value = false;
  });
  router.onError(() => {
    pendingNavigation.value = false;
  });
}
