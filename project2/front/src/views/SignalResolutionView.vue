<template>
  <div class="min-h-screen bg-base-200/30 px-5 py-10 lg:px-8">
    <div class="mx-auto max-w-4xl space-y-8">
      <button class="btn btn-ghost rounded-2xl font-bold" @click="$router.back()">&larr; Назад</button>

      <div v-if="loading" class="space-y-4">
        <div class="h-12 animate-pulse rounded-2xl bg-base-200"></div>
        <div class="h-80 animate-pulse rounded-[2rem] bg-base-200"></div>
      </div>

      <div v-else-if="signal" class="rounded-[2.5rem] border border-base-200 bg-base-100 p-8 shadow-xl">
        <div class="mb-6">
          <div class="inline-flex rounded-2xl bg-secondary/10 px-4 py-3 text-xs font-black uppercase tracking-[0.22em] text-secondary">
            Завершение сигнала
          </div>
          <h1 class="mt-4 text-3xl font-black tracking-tight">
            {{ signal.status === 'resolved' ? 'Вернуть сигнал в активные' : 'Отметить сигнал как завершенный' }}
          </h1>
          <p class="mt-2 max-w-2xl text-sm leading-6 opacity-65">
            Здесь фиксируется итог: проблема устранена, решена частично или закрыта без результата.
          </p>
        </div>

        <form class="space-y-5" @submit.prevent="submit">
          <template v-if="signal.status !== 'resolved'">
            <label v-for="option in options" :key="option.value" class="flex cursor-pointer items-start gap-4 rounded-[1.5rem] border border-base-200 p-5 hover:border-primary/30 hover:bg-primary/5">
              <input v-model="resolutionKind" type="radio" class="radio radio-primary mt-1" :value="option.value" />
              <div>
                <div class="font-black">{{ option.label }}</div>
                <div class="mt-1 text-sm leading-6 opacity-65">{{ option.description }}</div>
              </div>
            </label>

            <label v-if="resolutionKind === 'partial'" class="form-control gap-2">
              <span class="text-sm font-bold">Что удалось исправить и что осталось?</span>
              <textarea
                v-model="resolutionComment"
                class="textarea textarea-bordered min-h-40 rounded-[1.75rem]"
                placeholder="Кратко опишите, что уже решено, а что еще требует внимания"
              />
            </label>
          </template>

          <div v-else class="rounded-[1.5rem] border border-base-200 bg-base-200/50 px-5 py-4 text-sm leading-6">
            Сигнал снова станет активным и вернется в раздел защиты.
          </div>

          <div class="flex justify-end gap-3 border-t border-base-200 pt-5">
            <button type="button" class="btn btn-ghost rounded-2xl" @click="$router.back()">Отмена</button>
            <button class="btn btn-primary rounded-2xl px-6 font-black text-white" :disabled="saving || (signal.status !== 'resolved' && resolutionKind === 'partial' && !resolutionComment.trim())">
              <span v-if="saving" class="loading loading-spinner"></span>
              <span v-else>{{ signal.status === 'resolved' ? 'Открыть снова' : 'Подтвердить итог' }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import { useToast } from '../composables/useToast';

const route = useRoute();
const router = useRouter();
const toast = useToast();

const loading = ref(true);
const saving = ref(false);
const signal = ref<any | null>(null);
const resolutionKind = ref<'successful' | 'partial' | 'unsuccessful'>('successful');
const resolutionComment = ref('');

const options = [
  { value: 'successful', label: 'Устранено результативно', description: 'Проблема решена, сигнал можно закрыть.' },
  { value: 'partial', label: 'Частично решено', description: 'Удалось исправить только часть проблемы.' },
  { value: 'unsuccessful', label: 'Закрыто без результата', description: 'Сигнал закрывается, но заметного результата нет.' },
];

const loadSignal = async () => {
  loading.value = true;
  try {
    const { data } = await api.get(`/signals/${route.params.id}`);
    signal.value = data.signal;
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
    await api.post(`/signals/${signal.value.id}/status`, {
      resolved: signal.value.status !== 'resolved',
      resolution_kind: signal.value.status !== 'resolved' ? resolutionKind.value : undefined,
      resolution_comment: signal.value.status !== 'resolved' ? resolutionComment.value.trim() : undefined,
    });
    toast.success(signal.value.status === 'resolved' ? 'Сигнал снова открыт' : 'Сигнал перенесен в завершенные');
    await router.push('/profile');
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось изменить статус сигнала');
  } finally {
    saving.value = false;
  }
};
</script>
