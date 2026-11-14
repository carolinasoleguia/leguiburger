import { computed, reactive } from "vue";
import { getJSON } from "../services/api.js";
import { session } from "../services/session.js";

const state = reactive({ account: "", access: null });
export function useMembership() {
  const access = computed(() =>
    state.account === session.user?.id ? state.access : null,
  );
  return {
    access,
    premium: computed(
      () =>
        ["owner", "super_admin"].includes(session.user?.role) ||
        access.value?.premium !== false,
    ),
    async refresh() {
      const account = session.user?.id;
      const data = await getJSON("/membership");
      if (session.user?.id === account) {
        state.account = account;
        state.access = data;
      }
      return data;
    },
  };
}
