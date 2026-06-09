<template>
  <div class="mx-auto max-w-7xl space-y-10 p-6 lg:p-8">
    <header class="flex flex-col gap-6 border-b border-base-200 pb-8 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <div class="mb-3 inline-flex rounded-2xl bg-secondary/10 px-4 py-3 text-sm font-bold uppercase tracking-wider text-secondary">
          Раздел защиты памятников
        </div>
        <h1 class="text-4xl font-black tracking-tight lg:text-5xl">Сигналы угроз</h1>
        <p class="mt-3 max-w-3xl text-base font-medium opacity-60 lg:text-lg">
          Здесь собираются сообщения о рисках для памятников. Активные сигналы показываются отдельно по своему региону и по остальным регионам.
        </p>
      </div>

      <button
        class="btn btn-primary btn-lg rounded-2xl px-8 font-black shadow-xl shadow-primary/20"
        @click="showCreateSignalModal"
      >
        <PlusIcon class="mr-2 h-5 w-5" />
        Сообщить об угрозе
      </button>
    </header>

    <div class="grid gap-4 md:grid-cols-3">
      <div v-for="stat in stats" :key="stat.label" class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-md">
        <div class="text-xs font-black uppercase tracking-widest opacity-40">{{ stat.label }}</div>
        <div class="mt-2 text-4xl font-black">{{ stat.value }}</div>
      </div>
    </div>

    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div class="tabs tabs-boxed inline-flex rounded-2xl border border-base-200 bg-base-100/80 p-1.5">
        <button
          class="tab tab-lg rounded-xl px-6 font-bold"
          :class="{ 'tab-active bg-secondary text-secondary-content': signalsFilter === 'confirmed' }"
          @click="signalsFilter = 'confirmed'"
        >
          Активные
        </button>
        <button
          class="tab tab-lg rounded-xl px-6 font-bold"
          :class="{ 'tab-active bg-secondary text-secondary-content': signalsFilter === 'resolved' }"
          @click="signalsFilter = 'resolved'"
        >
          Завершенные
        </button>
      </div>

      <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
        <select v-model="selectedOtherRegion" class="select select-bordered rounded-2xl min-w-72">
          <option value="">Все остальные регионы</option>
          <option v-for="region in filteredRegionOptions" :key="region" :value="region">
            {{ region }}
          </option>
        </select>
      </div>
    </div>

    <template v-if="loading">
      <div class="grid gap-6 lg:grid-cols-2">
        <div v-for="index in 4" :key="index" class="h-64 rounded-[2.5rem] bg-base-200 animate-pulse"></div>
      </div>
    </template>

    <template v-else>
      <section v-if="showMyRegionBlock" class="space-y-5">
        <div class="flex items-end justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black tracking-tight">В моем регионе</h2>
            <p class="mt-1 text-sm font-medium opacity-55">{{ currentRegion }}</p>
          </div>
          <div class="badge badge-outline rounded-xl px-4 py-3 font-bold">
            {{ myRegionSignals.length }}
          </div>
        </div>

        <div v-if="myRegionSignals.length" class="grid gap-6 lg:grid-cols-2">
          <article v-for="signal in myRegionSignals" :key="signal.id" class="signal-card">
            <div class="group relative flex h-full flex-col justify-between overflow-hidden rounded-[3rem] border border-base-200 bg-base-100 p-8 shadow-xl transition-all duration-300 hover:-translate-y-1 hover:shadow-2xl">
              <div>
                <div class="absolute right-8 top-8 rounded-full px-4 py-2 text-center text-[11px] font-black uppercase tracking-wider shadow-lg" :class="urgencyBadgeClass(signal)">
                  {{ urgencyLabel(signal) }}
                </div>
                <div class="mb-7 pr-32">
                  <div class="mb-2 text-sm font-black uppercase tracking-[0.2em] opacity-45">{{ signalTypeLabel(signal.signal_type) }}</div>
                  <h3 class="text-2xl font-black tracking-tight">{{ signal.monument_name || signal.region || 'Неизвестный объект' }}</h3>
                  <div v-if="signal.region" class="mt-2 text-sm font-medium opacity-55">{{ signal.region }}</div>
                </div>
                <div v-if="signal.thumbnail" class="mb-6 overflow-hidden rounded-2xl">
                  <img :src="'/' + signal.thumbnail" class="h-52 w-full object-cover" />
                </div>
                <FormattedText :text="signal.description" class="line-clamp-4 text-base font-medium leading-relaxed opacity-75" />
              </div>
              <div class="mt-8 flex items-center justify-between gap-4 border-t border-base-200 pt-6">
                <div class="flex items-center gap-3">
                  <button class="btn btn-ghost btn-circle hover:bg-primary/10" :disabled="pendingSupportIds.has(signal.id)" @click="supportSignal(signal)">
                    <HeartIcon class="h-5 w-5 transition-all" :class="signal.is_supported ? 'fill-current text-primary scale-110' : 'opacity-40'" />
                  </button>
                  <span class="text-sm font-bold opacity-50">{{ signal.support_count || 0 }} поддержали</span>
                </div>
                <button class="btn btn-ghost btn-sm rounded-xl font-black opacity-50 hover:bg-base-200 hover:opacity-100" @click="openSignal(signal)">
                  Подробнее
                  <ArrowRightIcon class="ml-2 h-4 w-4" />
                </button>
              </div>
            </div>
          </article>
        </div>
        <div v-else class="rounded-[2rem] border border-dashed border-base-300 bg-base-100 p-8 text-center text-sm font-bold opacity-55">
          В выбранном разделе пока нет сигналов по текущему региону.
        </div>
      </section>

      <section class="space-y-5">
        <div class="flex items-end justify-between gap-4">
          <div>
            <h2 class="text-2xl font-black tracking-tight">Другие регионы</h2>
          </div>
          <div class="badge badge-outline rounded-xl px-4 py-3 font-bold">
            {{ otherSignals.length }}
          </div>
        </div>

        <div v-if="otherSignals.length" class="grid gap-6 lg:grid-cols-2">
          <article v-for="signal in otherSignals" :key="signal.id" class="signal-card">
            <div class="group relative flex h-full flex-col justify-between overflow-hidden rounded-[3rem] border border-base-200 bg-base-100 p-8 shadow-xl transition-all duration-300 hover:-translate-y-1 hover:shadow-2xl">
              <div>
                <div class="absolute right-8 top-8 rounded-full px-4 py-2 text-center text-[11px] font-black uppercase tracking-wider shadow-lg" :class="urgencyBadgeClass(signal)">
                  {{ urgencyLabel(signal) }}
                </div>
                <div class="mb-7 pr-32">
                  <div class="mb-2 text-sm font-black uppercase tracking-[0.2em] opacity-45">{{ signalTypeLabel(signal.signal_type) }}</div>
                  <h3 class="text-2xl font-black tracking-tight">{{ signal.monument_name || signal.region || 'Неизвестный объект' }}</h3>
                  <div v-if="signal.region" class="mt-2 text-sm font-medium opacity-55">{{ signal.region }}</div>
                </div>
                <div v-if="signal.thumbnail" class="mb-6 overflow-hidden rounded-2xl">
                  <img :src="'/' + signal.thumbnail" class="h-52 w-full object-cover" />
                </div>
                <FormattedText :text="signal.description" class="line-clamp-4 text-base font-medium leading-relaxed opacity-75" />
              </div>
              <div class="mt-8 flex items-center justify-between gap-4 border-t border-base-200 pt-6">
                <div class="flex items-center gap-3">
                  <button class="btn btn-ghost btn-circle hover:bg-primary/10" :disabled="pendingSupportIds.has(signal.id)" @click="supportSignal(signal)">
                    <HeartIcon class="h-5 w-5 transition-all" :class="signal.is_supported ? 'fill-current text-primary scale-110' : 'opacity-40'" />
                  </button>
                  <span class="text-sm font-bold opacity-50">{{ signal.support_count || 0 }} поддержали</span>
                </div>
                <button class="btn btn-ghost btn-sm rounded-xl font-black opacity-50 hover:bg-base-200 hover:opacity-100" @click="openSignal(signal)">
                  Подробнее
                  <ArrowRightIcon class="ml-2 h-4 w-4" />
                </button>
              </div>
            </div>
          </article>
        </div>
        <div v-else class="rounded-[2rem] border border-dashed border-base-300 bg-base-100 p-8 text-center text-sm font-bold opacity-55">
          По выбранному фильтру сигналы не найдены.
        </div>
      </section>
    </template>

    <dialog id="create_signal_modal" class="modal modal-bottom bg-base-300/30 backdrop-blur-xl sm:modal-middle">
      <div class="modal-box flex max-h-[90vh] max-w-none flex-col rounded-[3rem] border border-base-200 p-0 shadow-2xl sm:w-[min(94vw,920px)]">
        <button type="button" class="btn btn-circle btn-sm absolute right-5 top-5 z-20 bg-base-100/85 border-base-300 hover:bg-base-200" @click="closeCreateSignal">
          <XIcon class="h-4 w-4" />
        </button>

        <div class="flex-grow overflow-y-auto p-10">
          <div class="mb-6 inline-flex rounded-2xl bg-secondary/10 p-4 text-secondary">
            <ShieldAlertIcon class="h-10 w-10" />
          </div>
          <h3 class="mb-3 text-3xl font-black tracking-tight">Новый сигнал</h3>
          <p class="mb-8 font-medium opacity-60">Опишите проблему и при необходимости выберите памятник прямо на карте.</p>

          <form class="space-y-7" @submit.prevent="submitSignal">
            <div class="form-control gap-3">
              <label class="label font-bold opacity-70">Объект</label>
              <input
                v-model="newSignal.monument_name"
                type="text"
                class="input h-14 w-full rounded-2xl border-2 border-transparent bg-base-200 px-5 font-medium focus:border-secondary focus:outline-none"
                :class="badFields.monument_name ? 'border-error' : ''"
                placeholder="Название памятника"
                :disabled="!!newSignal.monument_id"
              />
              <span v-if="badFields.monument_name" class="text-sm font-bold text-error">{{ fieldMessage('monument_name', badFields.monument_name) }}</span>
            </div>

            <div class="space-y-4 rounded-[2rem] border border-base-200 bg-base-200/40 p-4">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div class="font-black">Привязка к памятнику на карте</div>
                  <div class="text-sm opacity-60">Можно выбрать существующую точку и не заполнять название вручную.</div>
                </div>
                <div class="flex gap-2">
                  <button type="button" class="btn btn-outline rounded-2xl" @click="toggleMonumentPicker">
                    {{ monumentPickerOpen ? 'Скрыть карту' : 'Выбрать на карте' }}
                  </button>
                  <button v-if="newSignal.monument_id" type="button" class="btn btn-ghost rounded-2xl" @click="clearSelectedMonument">
                    Очистить
                  </button>
                </div>
              </div>

              <div v-if="newSignal.monument_id" class="rounded-2xl border border-secondary/20 bg-secondary/10 px-4 py-3 text-sm font-semibold text-secondary">
                Выбран памятник: {{ newSignal.monument_name }}
              </div>

              <div v-if="monumentPickerOpen" class="space-y-3">
                <div ref="monumentPickerRef" class="h-72 w-full overflow-hidden rounded-[1.5rem] border border-base-200"></div>
                <div class="text-sm opacity-55">Кликните по маркеру памятника. После выбора поля обновятся автоматически.</div>
              </div>
            </div>

            <div class="form-control gap-3">
              <label class="label font-bold opacity-70">Что произошло</label>
              <select v-model="newSignal.signal_type" class="select h-14 rounded-2xl border-2 border-transparent bg-base-200 font-medium focus:border-secondary focus:outline-none">
                <option value="demolition">Есть риск сноса</option>
                <option value="vandalism">Вандализм или повреждение</option>
                <option value="poor_condition">Плохое состояние памятника</option>
                <option value="trash">Захламление территории</option>
                <option value="unsafe_work">Подозрительные работы рядом</option>
                <option value="other">Другая проблема</option>
              </select>
            </div>

            <div class="form-control gap-3">
              <label class="label font-bold opacity-70">Срочность</label>
              <select v-model="newSignal.urgency" class="select h-14 rounded-2xl border-2 border-transparent bg-base-200 font-medium focus:border-secondary focus:outline-none">
                <option value="low">Низкая</option>
                <option value="medium">Средняя</option>
                <option value="high">Критическая</option>
              </select>
            </div>

            <div class="form-control gap-3">
              <label class="label font-bold opacity-70">Описание ситуации</label>
              <RichTextEditor v-model="newSignal.description" :invalid="!!badFields.description" placeholder="Опишите, что происходит, как давно это замечено и почему ситуация опасна..." />
              <span v-if="badFields.description" class="text-sm font-bold text-error">{{ fieldMessage('description', badFields.description) }}</span>
            </div>

            <div class="form-control gap-3">
              <label class="label font-bold opacity-70">Фотографии подтверждения</label>
              <input type="file" multiple class="file-input file-input-bordered file-input-secondary h-14 w-full rounded-2xl border-none bg-base-200" accept="image/*" @change="handleFiles" />
              <span v-if="badFields.photos" class="text-sm font-bold text-error">{{ fieldMessage('photos', badFields.photos) }}</span>
              <div class="mt-4 flex gap-3 overflow-x-auto pb-2">
                <div v-for="(file, index) in previewUrls" :key="index" class="relative shrink-0">
                  <img :src="file" class="h-20 w-20 rounded-xl object-cover shadow-md" :class="badPhotos.includes(index) ? 'ring-4 ring-error' : ''" />
                  <button type="button" class="btn btn-circle btn-xs btn-error absolute -right-1.5 -top-1.5 z-10 shadow-md" @click="removeFile(index)">
                    <XIcon class="h-3 w-3" />
                  </button>
                  <div v-if="badPhotos.includes(index)" class="absolute inset-x-0 bottom-0 rounded-b-xl bg-error py-0.5 text-center text-[8px] font-bold text-white">AI</div>
                </div>
              </div>
            </div>

            <div v-if="aiWarnings.length > 0" class="alert border-none bg-warning/10 text-warning">
              <div class="flex flex-col gap-3">
                <div class="flex items-center gap-2 font-bold">
                  <AlertTriangleIcon class="h-5 w-5" />
                  AI-предупреждение
                </div>
                <ul class="list-disc pl-5 text-sm font-medium">
                  <li v-for="warning in aiWarnings" :key="warning">{{ warning }}</li>
                </ul>
                <label class="flex items-center gap-3 text-sm font-bold">
                  <input v-model="contentAck" type="checkbox" class="checkbox checkbox-warning rounded-lg" />
                  Подтверждается, что контент не нарушает правила
                </label>
              </div>
            </div>

            <div v-if="error" class="alert border-none bg-error/10 text-error">
              <AlertCircleIcon class="h-5 w-5" />
              <span class="text-sm font-bold">{{ error }}</span>
            </div>

            <div class="modal-action gap-3">
              <button type="button" class="btn btn-ghost h-14 flex-grow rounded-2xl font-bold" @click="closeCreateSignal">Отмена</button>
              <button type="submit" class="btn btn-secondary h-14 flex-[2] rounded-2xl font-black text-white shadow-xl shadow-secondary/20" :disabled="submitting">
                <span v-if="submitting" class="loading loading-spinner"></span>
                <span v-else>Отправить сигнал</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import {
  AlertCircleIcon,
  AlertTriangleIcon,
  ArrowRightIcon,
  HeartIcon,
  PlusIcon,
  ShieldAlertIcon,
  XIcon,
} from 'lucide-vue-next';
import api from '../api';
import RichTextEditor from '../components/RichTextEditor.vue';
import FormattedText from '../components/FormattedText.vue';
import { useAuthStore } from '../store/auth';
import { useToast } from '../composables/useToast';
import { applyValidationResult, buildFilesFingerprint, buildTextFingerprint, createValidationState } from '../composables/useContentValidation';
import { normalizeRichTextInput } from '../utils/richText';
import { useRegions } from '../composables/useRegions';

type SignalListItem = {
  id: string;
  monument_id?: string;
  monument_name?: string;
  region?: string;
  signal_type: string;
  urgency: 'low' | 'medium' | 'high';
  description: string;
  support_count?: number;
  is_supported?: boolean;
  thumbnail?: string;
  status: string;
};

const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
const toast = useToast();
const { regions, fetchRegions } = useRegions();

const loading = ref(false);
const submitting = ref(false);
const error = ref('');
const aiWarnings = ref<string[]>([]);
const contentAck = ref(false);
const previewUrls = ref<string[]>([]);
const selectedFiles = ref<File[]>([]);
const badFields = reactive<Record<string, string>>({});
const badPhotos = ref<number[]>([]);
const pendingSupportIds = reactive(new Set<string>());
const monumentPickerOpen = ref(false);
const monumentPickerRef = ref<HTMLElement | null>(null);
const monumentPickerMap = shallowRef<maplibregl.Map | null>(null);
const signalsFilter = ref<'confirmed' | 'resolved'>('confirmed');
const selectedOtherRegion = ref('');
const myRegionSignals = ref<SignalListItem[]>([]);
const otherSignals = ref<SignalListItem[]>([]);
const validation = createValidationState();

let validateTimer: number | undefined;

const currentRegion = computed(() => auth.user?.region?.trim() || '');
const showMyRegionBlock = computed(() => currentRegion.value.length > 0);

const filteredRegionOptions = computed(() => {
  return regions.value.filter((region) => region !== currentRegion.value);
});

const stats = ref([
  { label: 'Всего сигналов', value: '0' },
  { label: 'Устранено угроз', value: '0' },
  { label: 'Пользователей', value: '0' },
]);

const newSignal = reactive({
  monument_id: (route.query.monument_id as string) || null,
  monument_name: '',
  signal_type: 'poor_condition',
  urgency: 'medium',
  description: '',
  lon: 37.6173,
  lat: 55.7558,
});

const warningMap: Record<string, string> = {
  invalid_input: 'Проверьте обязательные поля формы',
  text_flagged: 'Текст может содержать недопустимый контент',
  image_flagged: 'Одно или несколько изображений могут быть недопустимыми',
  image_filter_unavailable: 'Сервис проверки изображений временно недоступен',
  text_filter_unavailable: 'Сервис проверки текста временно недоступен',
};

const fieldMessage = (field: string, value: string) => {
  const fieldMaps: Record<string, Record<string, string>> = {
    monument_name: { required: 'Это поле обязательно для заполнения' },
    description: {
      required: 'Это поле обязательно для заполнения',
      flagged: 'Описание не прошло AI-модерацию',
    },
    photos: {
      min_1: 'Добавьте хотя бы одну фотографию',
      max_10: 'Можно загрузить не более 10 фотографий',
    },
  };
  return fieldMaps[field]?.[value] || value;
};

const signalTypeLabel = (value: string) => ({
  demolition: 'Риск сноса',
  vandalism: 'Вандализм или повреждение',
  poor_condition: 'Плохое состояние памятника',
  trash: 'Захламление территории',
  unsafe_work: 'Подозрительные работы рядом',
  other: 'Другая проблема',
  neglect: 'Плохое состояние памятника',
  damage: 'Вандализм или повреждение',
  reconstruction: 'Подозрительные работы рядом',
}[value] || value);

const urgencyBadgeClass = (signal: SignalListItem) => {
  if (signal.status === 'pending') return 'bg-neutral text-neutral-content';
  if (signal.urgency === 'high') return 'bg-error text-error-content';
  if (signal.urgency === 'medium') return 'bg-warning text-warning-content';
  return 'bg-info text-info-content';
};

const urgencyLabel = (signal: SignalListItem) => {
  if (signal.status === 'pending') return 'Оценка модератора';
  if (signal.urgency === 'high') return 'Критическая';
  if (signal.urgency === 'medium') return 'Средняя';
  return 'Низкая';
};

const buildSignalParams = (scope: 'mine' | 'others') => {
  const params: Record<string, string> = { status: signalsFilter.value };
  if (scope === 'mine' && currentRegion.value) {
    params.region = currentRegion.value;
  }
  if (scope === 'others') {
    if (selectedOtherRegion.value) {
      params.region = selectedOtherRegion.value;
    } else if (currentRegion.value) {
      params.exclude_region = currentRegion.value;
    }
  }
  return params;
};

const fetchSignals = async () => {
  loading.value = true;
  try {
    const requests: Promise<any>[] = [];
    if (showMyRegionBlock.value) {
      requests.push(api.get('/signals', { params: buildSignalParams('mine') }));
    }
    requests.push(api.get('/signals', { params: buildSignalParams('others') }));

    const responses = await Promise.all(requests);
    if (showMyRegionBlock.value) {
      myRegionSignals.value = responses[0].data?.items || [];
      otherSignals.value = responses[1].data?.items || [];
    } else {
      myRegionSignals.value = [];
      otherSignals.value = responses[0].data?.items || [];
    }
  } finally {
    loading.value = false;
  }
};

const fetchStats = async () => {
  try {
    const { data } = await api.get('/stats');
    stats.value[0].value = String(data.signals?.total || 0);
    stats.value[1].value = String(data.signals?.resolved || 0);
    stats.value[2].value = String(data.users?.total || 0);
  } catch (fetchError) {
    console.error('Failed to fetch signal stats', fetchError);
  }
};

const openSignal = (signal: SignalListItem) => {
  router.push({ name: 'signal-detail', params: { id: signal.id } });
};

const supportSignal = async (signal: SignalListItem) => {
  if (!auth.isAuthenticated) {
    router.push('/login');
    return;
  }
  if (pendingSupportIds.has(signal.id)) return;

  pendingSupportIds.add(signal.id);
  try {
    if (signal.is_supported) {
      await api.delete(`/signals/${signal.id}/support`);
      signal.is_supported = false;
      signal.support_count = Math.max(0, (signal.support_count || 0) - 1);
      toast.success('Поддержка снята');
    } else {
      await api.post(`/signals/${signal.id}/support`);
      signal.is_supported = true;
      signal.support_count = (signal.support_count || 0) + 1;
      toast.success('Сигнал поддержан');
    }
  } catch {
    toast.warning('Не удалось обновить поддержку сигнала');
  } finally {
    pendingSupportIds.delete(signal.id);
  }
};

const syncValidationState = () => {
  aiWarnings.value = [...validation.warnings];
  Object.keys(badFields).forEach((key) => delete badFields[key]);
  Object.assign(badFields, validation.badFields);
  badPhotos.value = [...validation.badPhotos];
};

const currentFingerprint = () => {
  return [
    buildTextFingerprint(normalizeRichTextInput(newSignal.description)),
    buildFilesFingerprint(selectedFiles.value),
    newSignal.monument_id || '',
    newSignal.monument_name.trim(),
  ].join('::');
};

const markValidationDirty = () => {
  validation.isDirtyAfterValidation = true;
  if (contentAck.value) contentAck.value = false;
  scheduleValidation();
};

const validateSignal = async () => {
  const fingerprint = currentFingerprint();
  if (!fingerprint || fingerprint === validation.lastValidatedFingerprint) return;
  validation.isValidating = true;
  try {
    const formData = new FormData();
    if (newSignal.monument_id) formData.append('monument_id', newSignal.monument_id);
    formData.append('monument_name', newSignal.monument_name);
    formData.append('signal_type', newSignal.signal_type);
    formData.append('urgency', newSignal.urgency);
    formData.append('description', normalizeRichTextInput(newSignal.description));
    formData.append('lon', String(newSignal.lon));
    formData.append('lat', String(newSignal.lat));
    selectedFiles.value.forEach((file) => formData.append('photos', file));
    const { data } = await api.post('/signals/validate', formData);
    applyValidationResult(validation, data, warningMap);
    validation.lastValidatedFingerprint = fingerprint;
    validation.isDirtyAfterValidation = false;
    syncValidationState();
  } finally {
    validation.isValidating = false;
  }
};

const scheduleValidation = () => {
  if (validateTimer) window.clearTimeout(validateTimer);
  validateTimer = window.setTimeout(validateSignal, selectedFiles.value.length > 0 ? 400 : 700);
};

const handleFiles = (event: Event) => {
  const files = (event.target as HTMLInputElement).files;
  if (!files) return;
  for (let index = 0; index < files.length; index += 1) {
    if (selectedFiles.value.length >= 10) break;
    const file = files[index];
    selectedFiles.value.push(file);
    previewUrls.value.push(URL.createObjectURL(file));
  }
  markValidationDirty();
};

const removeFile = (index: number) => {
  selectedFiles.value.splice(index, 1);
  URL.revokeObjectURL(previewUrls.value[index]);
  previewUrls.value.splice(index, 1);
  markValidationDirty();
};

const showCreateSignalModal = () => {
  if (!auth.isAuthenticated) {
    router.push('/login');
    return;
  }
  (document.getElementById('create_signal_modal') as HTMLDialogElement).showModal();
};

const closeCreateSignal = () => {
  monumentPickerOpen.value = false;
  destroyMonumentPicker();
  (document.getElementById('create_signal_modal') as HTMLDialogElement).close();
};

const clearSelectedMonument = () => {
  newSignal.monument_id = null;
  newSignal.monument_name = '';
};

const destroyMonumentPicker = () => {
  if (monumentPickerMap.value) {
    monumentPickerMap.value.remove();
    monumentPickerMap.value = null;
  }
};

const initMonumentPicker = async () => {
  await nextTick();
  if (!monumentPickerRef.value || monumentPickerMap.value) return;

  const mapInstance = new maplibregl.Map({
    container: monumentPickerRef.value,
    style: {
      version: 8,
      sources: {
        'yandex-tiles': {
          type: 'raster',
          tiles: ['https://core-renderer-tiles.maps.yandex.net/tiles?l=map&x={x}&y={y}&z={z}&scale=1&lang=ru_RU'],
          tileSize: 256,
        },
      },
      layers: [{ id: 'yandex-layer', type: 'raster', source: 'yandex-tiles' }],
    },
    center: [newSignal.lon, newSignal.lat],
    zoom: 10,
  });

  monumentPickerMap.value = mapInstance;
  mapInstance.on('load', () => {
    mapInstance.addSource('monuments', {
      type: 'vector',
      tiles: [`${window.location.origin}/api/v1/tiles/monuments/{z}/{x}/{y}.mvt`],
      minzoom: 0,
      maxzoom: 14,
    });
    mapInstance.addLayer({
      id: 'picker-monuments-glow',
      type: 'circle',
      source: 'monuments',
      'source-layer': 'monuments',
      paint: {
        'circle-radius': ['interpolate', ['linear'], ['zoom'], 5, 10, 10, 18, 15, 24],
        'circle-color': '#4f46e5',
        'circle-opacity': 0.2,
        'circle-blur': 0.4,
      },
    });
    mapInstance.addLayer({
      id: 'picker-monuments',
      type: 'circle',
      source: 'monuments',
      'source-layer': 'monuments',
      paint: {
        'circle-radius': ['interpolate', ['linear'], ['zoom'], 5, 5, 10, 8, 15, 11],
        'circle-color': '#4f46e5',
        'circle-stroke-width': 2,
        'circle-stroke-color': '#ffffff',
      },
    });
    mapInstance.on('mouseenter', 'picker-monuments', () => {
      mapInstance.getCanvas().style.cursor = 'pointer';
    });
    mapInstance.on('mouseleave', 'picker-monuments', () => {
      mapInstance.getCanvas().style.cursor = '';
    });
    mapInstance.on('click', 'picker-monuments', async (event) => {
      const feature = event.features?.[0];
      const id = feature?.properties?.id;
      if (!id) return;
      try {
        const { data } = await api.get(`/monuments/${id}`);
        newSignal.monument_id = data.monument.id;
        newSignal.monument_name = data.monument.name;
        newSignal.lon = Number(data.monument.lon || event.lngLat.lng);
        newSignal.lat = Number(data.monument.lat || event.lngLat.lat);
        monumentPickerOpen.value = false;
        destroyMonumentPicker();
        toast.success(`Памятник «${data.monument.name}» выбран`);
      } catch {
        toast.error('Не удалось выбрать памятник');
      }
    });
  });
};

const toggleMonumentPicker = async () => {
  monumentPickerOpen.value = !monumentPickerOpen.value;
  if (monumentPickerOpen.value) await initMonumentPicker();
  else destroyMonumentPicker();
};

const submitSignal = async () => {
  submitting.value = true;
  error.value = '';

  const formData = new FormData();
  if (newSignal.monument_id) formData.append('monument_id', newSignal.monument_id);
  formData.append('monument_name', newSignal.monument_name);
  formData.append('signal_type', newSignal.signal_type);
  formData.append('urgency', newSignal.urgency);
  formData.append('description', normalizeRichTextInput(newSignal.description));
  formData.append('lon', String(newSignal.lon));
  formData.append('lat', String(newSignal.lat));
  formData.append('content_ack', String(contentAck.value));
  selectedFiles.value.forEach((file) => formData.append('photos', file));

  try {
    if (validation.isDirtyAfterValidation) await validateSignal();
    await api.post('/signals', formData);
    closeCreateSignal();
    toast.success('Сигнал отправлен на модерацию');
    Object.keys(badFields).forEach((key) => delete badFields[key]);
    badPhotos.value = [];
    aiWarnings.value = [];
    validation.lastValidatedFingerprint = '';
    validation.isDirtyAfterValidation = false;
    contentAck.value = false;
    selectedFiles.value = [];
    previewUrls.value = [];
    newSignal.description = '';
    await fetchSignals();
  } catch (submitError: any) {
    if (submitError.response?.status === 422) {
      const responseData = submitError.response.data;
      applyValidationResult(validation, {
        requires_ack: !!responseData.data?.requires_ack,
        reasons: responseData.data?.reasons || [],
        fields: responseData.fields || {},
      }, warningMap);
      syncValidationState();
    } else {
      error.value = submitError.response?.data?.message || 'Ошибка при сохранении';
    }
  } finally {
    submitting.value = false;
  }
};

watch(signalsFilter, () => {
  void fetchSignals();
});

watch(selectedOtherRegion, () => {
  void fetchSignals();
});

watch(() => [newSignal.description, newSignal.monument_name, newSignal.monument_id], () => {
  markValidationDirty();
}, { deep: true });

onMounted(async () => {
  await Promise.all([fetchRegions(true), fetchSignals(), fetchStats()]);
  if (route.query.monument_id) {
    try {
      const { data } = await api.get(`/monuments/${route.query.monument_id}`);
      newSignal.monument_name = data.monument.name;
      newSignal.lon = Number(data.monument.lon || newSignal.lon);
      newSignal.lat = Number(data.monument.lat || newSignal.lat);
    } catch (fetchError) {
      console.error('Failed to fetch monument name', fetchError);
    }
    showCreateSignalModal();
  }
});

onBeforeUnmount(() => {
  destroyMonumentPicker();
  if (validateTimer) window.clearTimeout(validateTimer);
});
</script>

<style scoped>
.signal-card {
  min-height: 100%;
}
</style>
