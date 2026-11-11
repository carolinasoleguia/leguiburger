// Shared behavior for both the original screens and the workspace screens.
// Draft values remain in their component on failed requests; secrets are never
// persisted in localStorage or copied to telemetry.
import { confirmAction } from "../services/confirmation.js";
const forms = new Map();
let sequence = 0;
let submitting = null;
export function formFingerprint(form) {
  return JSON.stringify(
    [...form.querySelectorAll("input,select,textarea")]
      .filter((el) => !["submit", "button", "hidden"].includes(el.type))
      .map((el) => [
        el.name || el.id,
        el.type === "checkbox" || el.type === "radio"
          ? el.checked
          : el.type === "file"
            ? [...el.files].map((f) => [f.name, f.size, f.lastModified])
            : el.value,
      ]),
  );
}
export function formRequest(form) {
  form = submitting || form;
  return form && forms.has(form)
    ? {
        form,
        fingerprint: formFingerprint(form),
        revision: forms.get(form).revision,
      }
    : null;
}
export function formSaved(request) {
  if (
    !request ||
    !forms.has(request.form) ||
    formFingerprint(request.form) !== request.fingerprint
  )
    return;
  forms.get(request.form).baseline = request.fingerprint;
  setTimeout(() => {
    const state = forms.get(request.form);
    if (state && state.revision === request.revision) {
      state.baseline = formFingerprint(request.form);
      state.edited = false;
    }
  }, 0);
}
function dirty(form) {
  const state = forms.get(form);
  return state?.edited && formFingerprint(form) !== state.baseline;
}
function message() {
  return "Tenés cambios sin guardar. ¿Querés salir y descartarlos?";
}
export function installFormCare(app, router) {
  app.directive("form-care", {
    mounted(form) {
      const state = {
        baseline: formFingerprint(form),
        edited: false,
        revision: 0,
        errors: new Map(),
      };
      forms.set(form, state);
      for (const field of form.querySelectorAll("input,select,textarea")) {
        const label = field.closest(".input-group")?.querySelector("label");
        if (label && !label.contains(field) && !label.htmlFor) {
          if (!field.id) field.id = `form-field-${++sequence}`;
          label.htmlFor = field.id;
        }
      }
      const submit = () => {
        submitting = form;
        queueMicrotask(() => {
          if (submitting === form) submitting = null;
        });
      };
      form.addEventListener("submit", submit, true);
      const input = () => {
        if (!state.edited) state.edited = true;
        state.revision++;
      };
      const validate = (event) => {
        const field = event.target;
        if (!field.matches("input,select,textarea")) return;
        let hint = state.errors.get(field);
        if (field.validity.valid) {
          field.removeAttribute("aria-invalid");
          if (hint) {
            hint.remove();
            state.errors.delete(field);
            field.setAttribute(
              "aria-describedby",
              field.dataset.careDescription || "",
            );
          }
          return;
        }
        if (!hint) {
          hint = document.createElement("small");
          hint.className = "field-error";
          hint.id = `field-error-${++sequence}`;
          field.insertAdjacentElement("afterend", hint);
          state.errors.set(field, hint);
          field.dataset.careDescription =
            field.getAttribute("aria-describedby") || "";
          field.setAttribute(
            "aria-describedby",
            `${field.dataset.careDescription} ${hint.id}`.trim(),
          );
        }
        field.setAttribute("aria-invalid", "true");
        hint.textContent = field.validationMessage;
      };
      const blur = (e) => {
        if (state.edited) validate(e);
      };
      form.addEventListener("input", input);
      form.addEventListener("change", input);
      form.addEventListener("invalid", validate, true);
      form.addEventListener("focusout", blur);
      const clear = (event) => {
        const field = event.target;
        if (field.validity?.valid && state.errors.has(field)) {
          state.errors.get(field).remove();
          state.errors.delete(field);
          field.removeAttribute("aria-invalid");
          field.setAttribute(
            "aria-describedby",
            field.dataset.careDescription || "",
          );
        }
      };
      form.addEventListener("input", clear);
      state.cleanup = () => {
        form.removeEventListener("submit", submit, true);
        form.removeEventListener("input", input);
        form.removeEventListener("input", clear);
        form.removeEventListener("change", input);
        form.removeEventListener("invalid", validate, true);
        form.removeEventListener("focusout", blur);
      };
      // Vue's initial models may settle during the first render.
      queueMicrotask(() => {
        if (!state.edited) state.baseline = formFingerprint(form);
      });
    },
    updated(form) {
      const state = forms.get(form);
      if (state && !state.edited) state.baseline = formFingerprint(form);
    },
    unmounted(form) {
      forms.get(form)?.cleanup();
      forms.delete(form);
    },
  });
  app.directive("dialog-care", {
    mounted(dialog) {
      const previous = document.activeElement;
      dialog.setAttribute("role", "dialog");
      dialog.setAttribute("aria-modal", "true");
      dialog.tabIndex = -1;
      const title = dialog.querySelector("h1,h2,h3");
      if (title) {
        if (!title.id) title.id = `dialog-title-${++sequence}`;
        dialog.setAttribute("aria-labelledby", title.id);
      }
      const focusables = () =>
        [
          ...dialog.querySelectorAll(
            'button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled),a[href],[tabindex="0"]',
          ),
        ].filter((el) => el.getClientRects().length);
      let replay = false;
      const closing = (event) => {
        if (!replay && [...dialog.querySelectorAll("form")].some(dirty)) {
          event.preventDefault();
          event.stopImmediatePropagation();
          const target = event.target.closest("button") || event.target;
          confirmAction(message(), "Cambios sin guardar").then((yes) => {
            if (yes && target.isConnected) {
              replay = true;
              target.click();
              replay = false;
            }
          });
        }
      };
      const click = (event) => {
        const button = event.target.closest("button");
        if (
          button &&
          (/cancelar|cerrar/i.test(button.textContent) ||
            button.classList.contains("modal-close"))
        )
          closing(event);
      };
      const keydown = (event) => {
        if (event.key === "Tab") {
          const items = focusables(),
            first = items[0],
            last = items.at(-1);
          if (!first) {
            event.preventDefault();
            dialog.focus();
          } else if (
            event.shiftKey &&
            (document.activeElement === first ||
              document.activeElement === dialog)
          ) {
            event.preventDefault();
            last.focus();
          } else if (!event.shiftKey && document.activeElement === last) {
            event.preventDefault();
            first.focus();
          }
        } else if (event.key === "Escape") {
          event.preventDefault();
          dialog.querySelector(".modal-close")?.click();
        }
      };
      dialog.addEventListener("click", click, true);
      dialog.addEventListener("keydown", keydown);
      const backdrop = (event) => {
        if (event.target === dialog.parentElement) closing(event);
      };
      dialog.parentElement?.addEventListener("click", backdrop, true);
      requestAnimationFrame(() => (focusables()[0] || dialog).focus());
      dialog._careCleanup = () => {
        dialog.removeEventListener("click", click, true);
        dialog.removeEventListener("keydown", keydown);
        dialog.parentElement?.removeEventListener("click", backdrop, true);
        if (previous?.isConnected) previous.focus();
      };
    },
    unmounted(dialog) {
      dialog._careCleanup?.();
    },
  });
  router.beforeEach((to, from) => {
    if (!from.meta.requiresAuth || to.path === "/login") return true;
    return (
      ![...forms.keys()].some(dirty) ||
      confirmAction(message(), "Cambios sin guardar")
    );
  });
  window.addEventListener("beforeunload", (event) => {
    if ([...forms.keys()].some(dirty)) {
      event.preventDefault();
      event.returnValue = "";
    }
  });
}
