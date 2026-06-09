<template>
  <dialog v-if="modelValue" class="modal modal-open">
    <div class="modal-box max-w-2xl rounded-[2rem]">
      <button class="btn btn-circle btn-sm absolute right-5 top-5" @click="emit('update:modelValue', false)">
        <XIcon class="w-4 h-4" />
      </button>

      <h3 class="text-2xl font-black mb-2">{{ resolved ? 'Завершение сигнала' : 'Повторное открытие сигнала' }}</h3>
      <p class="opacity-60 mb-6">
        {{ resolved ? 'Нужно подтвердить итог и, при частичном решении, кратко пояснить результат.' : 'Сигнал снова станет активным и вернется в раздел защиты.' }}
      </p>

      <form class="space-y-4" @submit.prevent="submit">
        <template v-if="resolved">
          <div class="grid gap-3">
            <label v-for="option in options" :key="option.value" class="flex cursor-pointer items-start gap-3 rounded-2xl border border-base-200 p-4 hover:border-primary/30 hover:bg-primary/5">
              <input v-model="resolutionKind" type="radio" class="radio radio-primary mt-1" :value="option.value" />
              <div>
                <div class="font-black">{{ option.label }}</div>
                <div class="text-sm opacity-60">{{ option.description }}</div>
              </div>
            </label>
          </div>

          <div v-if="resolutionKind === 'partial'" class="form-control gap-2">
            <label class="label font-bold">Что удалось сделать и что осталось?</label>
            <textarea
              v-model="resolutionComment"
              class="textarea textarea-bordered min-h-32 rounded-2xl"
              placeholder="Кратко опишите частичный результат"
            />
          </div>
        </template>

        <div class="modal-action">
          <button type="button" class="btn btn-ghost" @click="emit('update:modelValue', false)">Отмена</button>
          <button class="btn btn-primary" :disabled="saving">
            <span v-if="saving" class="loading loading-spinner"></span>
            <span v-else>{{ resolved ? 'Подтвердить' : 'Открыть снова' }}</span>
          </button>
        </div>
      </form>
    </div>
  </dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { XIcon } from 'lucide-vue-next';

const props = withDefaults(defineProps<{
  modelValue: boolean;
  resolved?: boolean;
  saving?: boolean;
}>(), {
  resolved: true,
  saving: false,
});

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'submit', payload: { resolved: boolean; resolution_kind?: string; resolution_comment?: string }): void;
}>();

const options = [
  { value: 'successful', label: 'Устранено результативно', description: 'Проблема решена, сигнал можно закрыть.' },
  { value: 'partial', label: 'Частично решено', description: 'Удалось исправить часть проблемы, но не все.' },
  { value: 'unsuccessful', label: 'Закрыто без результата', description: 'Сигнал закрывается, но заметного результата нет.' },
];

const resolutionKind = ref<'successful' | 'partial' | 'unsuccessful'>('successful');
const resolutionComment = ref('');

watch(() => props.modelValue, (opened) => {
  if (opened) {
    resolutionKind.value = 'successful';
    resolutionComment.value = '';
  }
});

const submit = () => {
  emit('submit', {
    resolved: props.resolved,
    resolution_kind: props.resolved ? resolutionKind.value : undefined,
    resolution_comment: props.resolved ? resolutionComment.value.trim() : undefined,
  });
};
</script>
