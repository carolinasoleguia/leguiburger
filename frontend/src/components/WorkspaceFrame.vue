<script setup>
import DashboardShell from "./DashboardShell.vue";
import PageHeading from "./PageHeading.vue";
import StableContent from "./StableContent.vue";
defineProps({
  workspace: Object,
  title: String,
  description: String,
  local: { type: Boolean, default: true },
});
</script>
<template>
  <DashboardShell :role="workspace.shellRole.value"
    ><section class="operations-page">
      <PageHeading :title="title" :description="description">
        <button
          class="btn-secondary"
          :disabled="workspace.loading.value || workspace.busy.value"
          @click="workspace.refresh"
        >
          Actualizar
        </button>
      </PageHeading>
      <div v-if="local && workspace.manager.value" class="workspace-filters">
        <label
          >Local<select
            v-model="workspace.tenant.value"
            :disabled="workspace.busy.value"
          >
            <option
              v-for="item in workspace.tenants.value"
              :key="item.id"
              :value="item.id"
            >
              {{ item.name || item.subdomain }}
            </option>
          </select></label
        ><span class="workspace-local-hint">{{
          workspace.loading.value
            ? "Consultando tus locales…"
            : !workspace.tenants.value.length
              ? "Todavía no hay locales activos."
              : "Datos del local seleccionado."
        }}</span>
      </div>
      <div v-if="workspace.error.value" class="error-msg" role="alert">
        {{ workspace.error.value }}
        <button
          class="text-button"
          :disabled="workspace.busy.value"
          @click="workspace.refresh"
        >
          Reintentar
        </button>
      </div>
      <p v-if="workspace.notice.value" class="status-notice" role="status">
        {{ workspace.notice.value }}
      </p>
      <StableContent
        :loading="workspace.loading.value"
        :busy="workspace.busy.value"
        ><slot /></StableContent
      ><slot name="footer" /></section
  ></DashboardShell>
</template>
