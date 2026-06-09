<template>
  <div class="min-h-screen bg-base-200/30 px-5 py-10 lg:px-8">
    <div class="mx-auto max-w-4xl space-y-8">
      <button class="btn btn-ghost rounded-2xl font-bold" @click="$router.back()">&larr; Назад</button>

      <div v-if="loading" class="space-y-4">
        <div class="h-12 animate-pulse rounded-2xl bg-base-200"></div>
        <div class="h-96 animate-pulse rounded-[2rem] bg-base-200"></div>
      </div>

      <div v-else-if="signal" class="rounded-[2.5rem] border border-base-200 bg-base-100 p-8 shadow-xl">
        <div class="mb-6">
          <div class="inline-flex rounded-2xl bg-primary/10 px-4 py-3 text-xs font-black uppercase tracking-[0.22em] text-primary">
            Редактирование сигнала
          </div>
          <h1 class="mt-4 text-3xl font-black tracking-tight">Обновить сигнал</h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 opacity-65">
            После сохранения сигнал снова уйдет на проверку. Здесь можно спокойно исправить тип, срочность и описание без узких всплывающих окон.
          </p>
        </div>

        <form class="space-y-6" @submit.prevent="submit">
          <div class="grid gap-4 lg:grid-cols-2">
            <label class="form-control gap-2">
              <span class="text-sm font-bold">Тип сигнала</span>
              <select v-model="form.signal_type" class="select select-bordered h-14 rounded-2xl">
                <option value="demolition">Есть риск сноса</option>
                <option value="vandalism">Вандализм или повреждение</option>
                <option value="poor_condition">Плохое состояние памятника</option>
                <option value="trash">Захламление территории</option>
                <option value="unsafe_work">Подозрительные работы рядом</option>
                <option value="other">Другая проблема</option>
              </select>
            </label>

            <label class="form-control gap-2">
              <span class="text-sm font-bold">Срочность</span>
              <select v-model="form.urgency" class="select select-bordered h-14 rounded-2xl">
                <option value="low">Низкая</option>
                <option value="medium">Средняя</option>
                <option value="high">Критическая</option>
              </select>
            </label>
          </div>

          <label class="form-control gap-2">
            <span class="text-sm font-bold">Описание</span>
            <textarea
              v-model="form.description"
              class="textarea textarea-bordered min-h-56 rounded-[1.75rem] text-base leading-relaxed"
              placeholder="Опишите, что изменилось или что важно уточнить"
            />
          </label>

          <div class="flex justify-end gap-3 border-t border-base-200 pt-5">
            <button type="button" class="btn btn-ghost rounded-2xl" @click="$router.back()">Отмена</button>
            <button class="btn btn-primary rounded-2xl px-6 font-black text-white" :disabled="saving || !form.description.trim()">
              <span v-if="saving" class="loading loading-spinner"></span>
              <span v-else>Сохранить изменения</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import { useToast } from '../composables/useToast';

const route = useRoute();
const router = useRouter();
const toast = useToast();

const loading = ref(true);
const saving = ref(false);
const signal = ref<any | null>(null);
const form = reactive({
  signal_type: 'poor_condition',
  urgency: 'medium',
  description: '',
});

const loadSignal = async () => {
  loading.value = true;
  try {
    const { data } = await api.get(`/signals/${route.params.id}`);
    signal.value = data.signal;
    form.signal_type = data.signal.signal_type;
    form.urgency = data.signal.urgency;
    form.description = data.signal.description || '';
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось загрузить сигнал');
    await router.push('/profile');
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  void loadSignal();
});

const submit = async () => {
  if (!signal.value) return;
  saving.value = true;
  try {
    await api.put(`/signals/${signal.value.id}`, {
      signal_type: form.signal_type,
      urgency: form.urgency,
      description: form.description.trim(),
    });
    toast.success('Сигнал обновлен и отправлен на повторную проверку');
    await router.push('/profile');
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось обновить сигнал');
  } finally {
    saving.value = false;
  }
};
</script>
