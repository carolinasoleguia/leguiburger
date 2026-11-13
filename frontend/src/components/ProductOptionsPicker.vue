<script setup>
import { computed, useId } from "vue";
import { money } from "../composables/useWorkspace.js";
import { optionGroups } from "../composables/productOptions.js";
const props = defineProps({
  product: Object,
  extras: { type: Array, default: () => [] },
  modelValue: { type: Array, default: () => [] },
});
const emit = defineEmits(["update:modelValue"]),
  id = useId();
const groups = computed(() => optionGroups(props.product, props.extras));
const quantity = (item) =>
  props.modelValue.find((x) => x.id === item)?.quantity || 0;
const count = (g) =>
  props.modelValue
    .filter((x) => g.options.some((o) => o.id === x.id))
    .reduce((n, x) => n + x.quantity, 0);
function select(group, option, event) {
  let next = props.modelValue.filter((x) => x.id !== option.id);
  if (group.max === 1)
    next = next.filter((x) => !group.options.some((o) => o.id === x.id));
  if (event.target.checked) next.push({ id: option.id, quantity: 1 });
  emit("update:modelValue", next);
}
function clearGroup(group) {
  emit(
    "update:modelValue",
    props.modelValue.filter((x) => !group.options.some((o) => o.id === x.id)),
  );
}
function amount(group, option, event) {
  const max = Math.min(5, group.max - count(group) + quantity(option.id));
  const q = Math.max(1, Math.min(max, Number(event.target.value) || 1));
  emit(
    "update:modelValue",
    props.modelValue.map((x) =>
      x.id === option.id ? { ...x, quantity: q } : x,
    ),
  );
}
</script>
<template>
  <div v-if="groups.length" class="product-options-picker">
    <fieldset v-for="g in groups" :key="g.id" class="option-group">
      <legend>{{ g.name }}</legend>
      <p class="option-group-help">
        <template v-if="g.id === 'legacy'"
          >Hasta 10 extras, con hasta 5 unidades de cada uno.</template
        >
        <template v-else
          >{{
            g.min === g.max
              ? "Elegí " + g.min
              : "Elegí entre " + g.min + " y " + g.max
          }}
          por unidad · {{ g.min > 0 ? "Obligatorio" : "Opcional" }}</template
        >
        <span>{{ count(g) }}/{{ g.max }}</span>
      </p>
      <p v-if="!g.options.length" class="option-group-help">
        No hay opciones disponibles en este grupo.
      </p>
      <button
        v-if="g.min === 0 && count(g) > 0"
        type="button"
        class="text-button"
        @click="clearGroup(g)"
      >
        Quitar selección de {{ g.name }}
      </button>
      <div v-for="o in g.options" :key="o.id" class="product-option-row">
        <label class="product-option-choice buyer-extra"
          ><input
            :type="g.max === 1 ? 'radio' : 'checkbox'"
            :name="id + '-' + g.id"
            :value="o.id"
            :checked="quantity(o.id) > 0"
            :disabled="
              !o.available ||
              (!quantity(o.id) &&
                g.max !== 1 &&
                (count(g) >= g.max || modelValue.length >= 10))
            "
            @change="select(g, o, $event)"
          /><span
            >{{ o.name
            }}<small v-if="!o.available">No disponible por ahora</small></span
          ><strong>{{
            Number(o.price) === 0 ? "Incluido" : "+ " + money(o.price)
          }}</strong></label
        >
        <label
          v-if="quantity(o.id) > 0 && g.max > 1"
          class="product-option-quantity"
          >Unidades de {{ o.name
          }}<input
            :value="quantity(o.id)"
            type="number"
            min="1"
            :max="Math.min(5, g.max - count(g) + quantity(o.id))"
            @input="amount(g, o, $event)"
        /></label>
      </div>
    </fieldset>
  </div>
</template>
<style scoped>
.product-options-picker {
  display: grid;
  gap: 1.25rem;
  min-width: 0;
}
.option-group {
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 1rem;
  min-width: 0;
}
.option-group legend {
  padding: 0 0.35rem;
  font-weight: 700;
}
.option-group-help {
  color: var(--text-muted);
  font-size: 0.82rem;
  line-height: 1.5;
  margin: 0.2rem 0 0.75rem;
}
.option-group-help span {
  float: right;
}
.product-option-choice {
  display: flex !important;
  align-items: center;
  gap: 0.7rem;
  min-height: 48px;
  font-size: 0.9rem;
}
.product-option-choice input {
  width: 20px !important;
  height: 20px !important;
  min-height: 20px !important;
  padding: 0 !important;
  flex-shrink: 0;
  accent-color: var(--accent);
}
.product-option-choice span {
  flex: 1;
}
.product-option-choice small {
  display: block;
  color: var(--text-muted);
}
.product-option-choice strong {
  font-size: 0.82rem;
  white-space: nowrap;
}
.product-option-quantity {
  display: flex !important;
  align-items: center;
  gap: 1rem;
  font-size: 0.8rem !important;
  justify-content: space-between;
  margin: 0.25rem 0 0.75rem;
}
.product-option-quantity input {
  max-width: 78px;
  min-height: 42px;
  color: var(--text);
  background: var(--field);
  border: 1px solid var(--control-border);
  border-radius: 8px;
  padding: 0.5rem;
}
</style>
