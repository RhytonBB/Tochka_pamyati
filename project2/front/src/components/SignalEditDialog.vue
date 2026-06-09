<template>
  <dialog v-if="modelValue" class="modal modal-open">
    <div class="modal-box max-w-2xl rounded-[2rem]">
      <button class="btn btn-circle btn-sm absolute right-5 top-5" @click="emit('update:modelValue', false)">
        <XIcon class="w-4 h-4" />
      </button>

      <h3 class="text-2xl font-black mb-2">Редактирование сигнала</h3>
      <p class="opacity-60 mb-6">После сохранения сигнал снова отправится на проверку.</p>

      <form class="space-y-4" @submit.prevent="submit">
        <div class="form-control gap-2">
          <label class="label font-bold">Тип сигнала</label>
          <select v-model="form.signal_type" class="select select-bordered rounded-2xl">
            <option value="demolition">Есть риск сноса</option>
            <option value="vandalism">Вандализм или повреждение</option>
            <option value="poor_condition">Плохое состояние памятника</option>
            <option value="trash">Захламление территории</option>
            <option value="unsafe_work">Подозрительные работы рядом</option>
            <option value="other">Другая проблема</option>
          </select>
        </div>

        <div class="form-control gap-2">
          <label class="label font-bold">Срочность</label>
          <select v-model="form.urgency" class="select select-bordered rounded-2xl">
            <option value="low">Низкая</option>
            <option value="medium">Средняя</option>
            <option value="high">Критическая</option>
          </select>
        </div>

        <div class="form-control gap-2">
          <label class="label font-bold">Описание</label>
          <textarea
            v-model="form.description"
            class="textarea textarea-bordered min-h-40 rounded-2xl"
            placeholder="Опишите, что именно изменилось"
          />
        </div>

        <div class="modal-action">
          <button type="button" class="btn btn-ghost" @click="emit('update:modelValue', false)">Отмена</button>
          <button class="btn btn-primary" :disabled="saving">
            <span v-if="saving" class="loading loading-spinner"></span>
            <span v-else>Сохранить</span>
          </button>
        </div>
      </form>
    </div>
  </dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue';
import { XIcon } from 'lucide-vue-next';

const props = defineProps<{
  modelValue: boolean;
  signal: {
    signal_type: string;
    urgency: string;
    description: string;
  } | null;
  saving?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'submit', payload: { signal_type: string; urgency: string; description: string }): void;
}>();

const form = reactive({
  signal_type: 'poor_condition',
  urgency: 'medium',
  description: '',
});

watch(() => props.signal, (signal) => {
  form.signal_type = signal?.signal_type || 'poor_condition';
  form.urgency = signal?.urgency || 'medium';
  form.description = signal?.description || '';
}, { immediate: true });

const submit = () => {
  emit('submit', {
    signal_type: form.signal_type,
    urgency: form.urgency,
    description: form.description.trim(),
  });
};
</script>
