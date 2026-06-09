<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="fixed inset-0 z-[140] flex items-center justify-center bg-slate-950/45 px-4 py-6 backdrop-blur-sm"
      @click.self="emit('update:modelValue', false)"
    >
      <section class="w-full max-w-3xl overflow-hidden rounded-[2rem] border border-base-200 bg-base-100 shadow-2xl">
        <div class="border-b border-base-200 px-6 py-5">
          <div class="text-xs font-black uppercase tracking-[0.22em] opacity-40">{{ eyebrow }}</div>
          <h3 class="mt-2 text-2xl font-black tracking-tight">{{ title }}</h3>
          <p v-if="message" class="mt-2 text-sm leading-6 opacity-70">{{ message }}</p>
        </div>

        <form class="px-6 py-5" @submit.prevent="submit">
          <label class="form-control gap-3">
            <span class="text-sm font-bold">{{ label }}</span>
            <textarea
              v-model="draft"
              class="textarea textarea-bordered min-h-48 rounded-2xl text-base leading-relaxed"
              :placeholder="placeholder"
            />
          </label>

          <div class="mt-6 flex flex-wrap items-center justify-end gap-3">
            <button type="button" class="btn btn-ghost rounded-2xl" @click="emit('update:modelValue', false)">Отмена</button>
            <button type="submit" class="btn btn-primary rounded-2xl px-6 font-black text-white" :disabled="loading || !draft.trim()">
              <span v-if="loading" class="loading loading-spinner"></span>
              <span v-else>{{ confirmLabel }}</span>
            </button>
          </div>
        </form>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';

const props = withDefaults(defineProps<{
  modelValue: boolean;
  title: string;
  initialValue?: string;
  label?: string;
  message?: string;
  placeholder?: string;
  confirmLabel?: string;
  eyebrow?: string;
  loading?: boolean;
}>(), {
  initialValue: '',
  label: 'Текст',
  message: '',
  placeholder: '',
  confirmLabel: 'Сохранить',
  eyebrow: 'Редактирование',
  loading: false,
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'submit', value: string): void;
}>();

const draft = ref('');

watch(
  () => [props.modelValue, props.initialValue],
  () => {
    if (props.modelValue) {
      draft.value = props.initialValue || '';
    }
  },
  { immediate: true },
);

const submit = () => {
  emit('submit', draft.value.trim());
};
</script>
