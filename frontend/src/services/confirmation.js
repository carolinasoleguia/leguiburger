import { shallowRef } from "vue";
export const pendingConfirmation = shallowRef(null);
const queue = [];
export function confirmAction(message, title = "Confirmar acción") {
  return new Promise((resolve) => {
    queue.push({ message, title, resolve });
    if (!pendingConfirmation.value) pendingConfirmation.value = queue.shift();
  });
}
export function resolveConfirmation(confirmed) {
  const item = pendingConfirmation.value;
  pendingConfirmation.value = null;
  item?.resolve(confirmed);
  queueMicrotask(() => {
    if (!pendingConfirmation.value)
      pendingConfirmation.value = queue.shift() || null;
  });
}
