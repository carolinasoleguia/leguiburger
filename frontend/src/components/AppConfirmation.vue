<script setup>
import {
  pendingConfirmation as pending,
  resolveConfirmation,
} from "../services/confirmation.js";
</script>
<template>
  <Teleport to="body"
    ><Transition name="modal"
      ><div
        v-if="pending"
        class="modal-overlay confirmation-overlay"
        @click.self="resolveConfirmation(false)"
      >
        <section v-dialog-care class="modal-card confirmation-card">
          <h2>{{ pending.title }}</h2>
          <p>{{ pending.message }}</p>
          <div class="operation-actions">
            <button
              type="button"
              class="btn-secondary modal-close-action"
              @click="resolveConfirmation(false)"
            >
              Cancelar</button
            ><button
              type="button"
              class="btn-primary"
              @click="resolveConfirmation(true)"
            >
              Confirmar
            </button>
          </div>
          <button
            type="button"
            class="modal-close"
            aria-label="Cerrar confirmación"
            @click="resolveConfirmation(false)"
          >
            ×
          </button>
        </section>
      </div></Transition
    ></Teleport
  >
</template>
