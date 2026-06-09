<template>
  <Teleport to="body">
    <div
      v-if="modelValue"
      class="fixed inset-0 z-[140] flex items-center justify-center bg-slate-950/45 px-4 py-6 backdrop-blur-sm"
      @click.self="emit('update:modelValue', false)"
    >
      <div class="flex w-full justify-center">
        <section class="w-[min(92vw,44rem)] min-w-[22rem] shrink-0 overflow-hidden rounded-[2rem] border border-base-200 bg-base-100 shadow-2xl">
        <div class="border-b border-base-200 px-6 py-5">
          <div class="text-xs font-black uppercase tracking-[0.22em] opacity-40">{{ eyebrow }}</div>
          <h3 class="mt-2 text-2xl font-black tracking-tight">{{ title }}</h3>
          <p v-if="message" class="mt-2 max-w-none whitespace-normal break-words text-sm leading-6 opacity-70">{{ message }}</p>
        </div>

        <div v-if="details.length" class="space-y-3 px-6 py-5">
          <div
            v-for="detail in details"
            :key="detail"
            class="rounded-2xl bg-base-200/70 px-4 py-3 text-sm font-medium leading-6 whitespace-normal break-words"
          >
            {{ detail }}
          </div>
        </div>

        <div class="flex flex-wrap items-center justify-end gap-3 border-t border-base-200 px-6 py-5">
          <button type="button" class="btn btn-ghost rounded-2xl" @click="emit('update:modelValue', false)">Отмена</button>
          <button
            type="button"
            class="btn rounded-2xl px-6 font-black"
            :class="destructive ? 'btn-error text-white' : 'btn-primary text-white'"
            :disabled="loading"
            @click="emit('confirm')"
          >
            <span v-if="loading" class="loading loading-spinner"></span>
            <span v-else>{{ confirmLabel }}</span>
          </button>
        </div>
        </section>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  modelValue: boolean;
  title: string;
  message?: string;
  confirmLabel?: string;
  eyebrow?: string;
  destructive?: boolean;
  loading?: boolean;
  details?: string[];
}>(), {
  message: '',
  confirmLabel: 'Подтвердить',
  eyebrow: 'Подтверждение действия',
  destructive: false,
  loading: false,
  details: () => [],
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'confirm'): void;
}>();
</script>
