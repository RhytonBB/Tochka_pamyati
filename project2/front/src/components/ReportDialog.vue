<template>
  <dialog ref="dialogRef" class="modal modal-bottom sm:modal-middle bg-base-300/30 backdrop-blur-xl">
    <div class="modal-box w-[min(94vw,860px)] max-w-none rounded-[2.5rem] p-0 border border-base-200 shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
      <div class="p-8 sm:p-10 space-y-6 overflow-y-auto">
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="inline-flex bg-error/10 text-error rounded-2xl px-4 py-2 font-black text-xs uppercase tracking-wider mb-4">
              Жалоба
            </div>
            <h3 class="text-3xl font-black tracking-tight">Пожаловаться на {{ entityLabel }}</h3>
            <p class="opacity-60 font-medium mt-2">{{ subjectLine }}</p>
          </div>
          <button type="button" class="btn btn-circle btn-sm" @click="close">
            <XIcon class="w-4 h-4" />
          </button>
        </div>

        <div class="space-y-3">
          <div class="font-black text-xs uppercase tracking-wider opacity-45">Причина</div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <button
              v-for="reason in reasons"
              :key="reason.code"
              type="button"
              class="text-left rounded-2xl px-5 py-4 border transition-all font-bold"
              :class="selectedReason === reason.code ? 'bg-primary text-primary-content border-primary shadow-lg shadow-primary/20' : 'bg-base-200 border-base-200 hover:border-primary/25 hover:bg-base-100'"
              @click="selectedReason = reason.code"
            >
              {{ reason.label }}
            </button>
          </div>
        </div>

        <div v-if="selectedReason === 'wrong_name'" class="space-y-2">
          <div class="font-black text-xs uppercase tracking-wider opacity-45">Правильное название</div>
          <input v-model.trim="suggestedTitle" type="text" class="input input-bordered w-full rounded-2xl h-14 bg-base-200 border-base-200" placeholder="Как должно называться" />
        </div>

        <div v-if="selectedReason === 'wrong_coords'" class="space-y-3">
          <div class="font-black text-xs uppercase tracking-wider opacity-45">Исправленные координаты</div>
          <div ref="mapContainerRef" class="w-full h-64 rounded-2xl overflow-hidden border border-base-200"></div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <input :value="suggestedLonText" type="text" class="input input-bordered w-full rounded-2xl h-14 bg-base-200 border-base-200 font-mono" placeholder="Долгота" readonly />
            <input :value="suggestedLatText" type="text" class="input input-bordered w-full rounded-2xl h-14 bg-base-200 border-base-200 font-mono" placeholder="Широта" readonly />
          </div>
          <p class="text-sm opacity-50 font-medium">Нажмите на карту, чтобы указать правильную точку.</p>
        </div>

        <div v-if="selectedReason === 'duplicate' && entityType === 'monument'" class="space-y-3">
          <div class="font-black text-xs uppercase tracking-wider opacity-45">Похожая точка</div>
          <input
            v-model.trim="duplicateQuery"
            type="text"
            class="input input-bordered w-full rounded-2xl h-14 bg-base-200 border-base-200"
            placeholder="Найдите уже существующий памятник"
            @input="debouncedLoadSuggestions"
          />
          <div v-if="duplicateSuggestions.length > 0" class="space-y-2 max-h-56 overflow-y-auto">
            <button
              v-for="item in duplicateSuggestions"
              :key="item.id"
              type="button"
              class="w-full text-left rounded-2xl p-4 border transition-all"
              :class="duplicateTargetId === item.id ? 'border-primary bg-primary/5' : 'border-base-200 bg-base-100 hover:border-primary/30'"
              @click="selectDuplicate(item)"
            >
              <div class="font-bold">{{ item.name }}</div>
              <div class="text-xs opacity-50 font-medium mt-1">{{ item.lat.toFixed(4) }}, {{ item.lon.toFixed(4) }}</div>
            </button>
          </div>
        </div>

        <div v-if="showComment" class="space-y-2">
          <div class="font-black text-xs uppercase tracking-wider opacity-45">
            {{ selectedReason === 'other' ? 'Комментарий' : 'Уточнение' }}
          </div>
          <textarea
            v-model.trim="comment"
            class="textarea textarea-bordered w-full h-32 rounded-2xl bg-base-200 border-base-200"
            :placeholder="selectedReason === 'other' ? 'Опишите проблему' : 'Коротко уточните, что именно не так'"
          ></textarea>
        </div>

        <div v-if="error" class="alert alert-error rounded-2xl">
          {{ error }}
        </div>

        <div class="sticky bottom-0 z-10 -mx-2 px-2 py-3 bg-base-100/95 backdrop-blur-md border-t border-base-200 flex gap-3">
          <button type="button" class="btn btn-ghost flex-1 h-14 rounded-2xl font-bold" @click="close">Отмена</button>
          <button type="button" class="btn btn-primary flex-[1.3] h-14 rounded-2xl font-black text-white" :disabled="submitting || !selectedReason" @click="submit">
            <span v-if="submitting" class="loading loading-spinner"></span>
            <span v-else>Отправить жалобу</span>
          </button>
        </div>
      </div>
    </div>
  </dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import { XIcon } from 'lucide-vue-next';
import api from '../api';

interface SuggestionItem {
  id: string;
  name: string;
  lat: number;
  lon: number;
}

const props = defineProps<{
  modelValue: boolean;
  entityType: 'monument' | 'post' | 'photo' | 'signal' | 'comment';
  entityId: string;
  subject?: string;
  duplicateSeed?: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'submitted'): void;
}>();

const dialogRef = ref<HTMLDialogElement | null>(null);
const selectedReason = ref('');
const comment = ref('');
const suggestedTitle = ref('');
const suggestedLon = ref<number | null>(null);
const suggestedLat = ref<number | null>(null);
const duplicateQuery = ref(props.duplicateSeed || '');
const duplicateSuggestions = ref<SuggestionItem[]>([]);
const duplicateTargetId = ref('');
const error = ref('');
const submitting = ref(false);
const mapContainerRef = ref<HTMLElement | null>(null);
let pickerMap: maplibregl.Map | null = null;
let pickerMarker: maplibregl.Marker | null = null;
let suggestionTimer: number | undefined;

const defaultLon = 37.6176;
const defaultLat = 55.7558;

const reasonMap: Record<string, Array<{ code: string; label: string }>> = {
  monument: [
    { code: 'wrong_name', label: 'Неверное название' },
    { code: 'wrong_coords', label: 'Неверные координаты' },
    { code: 'duplicate', label: 'Дубликат точки' },
    { code: 'wrong_photo', label: 'Нерелевантные фото' },
    { code: 'fake_object', label: 'Несуществующий объект' },
    { code: 'offensive_object', label: 'Оскорбительный объект' },
    { code: 'other', label: 'Другое' },
  ],
  post: [
    { code: 'false_info', label: 'Ложная информация' },
    { code: 'offensive', label: 'Оскорбительный контент' },
    { code: 'spam', label: 'Спам' },
    { code: 'flood', label: 'Флуд' },
    { code: 'duplicate', label: 'Дубликат' },
    { code: 'irrelevant_to_monument', label: 'Не относится к памятнику' },
    { code: 'other', label: 'Другое' },
  ],
  photo: [
    { code: 'not_relevant', label: 'Фото не относится к объекту' },
    { code: 'low_quality', label: 'Плохое или нечитаемое фото' },
    { code: 'offensive', label: 'Оскорбительное фото' },
    { code: 'duplicate', label: 'Дубликат' },
    { code: 'private_data', label: 'Личные данные на фото' },
    { code: 'other', label: 'Другое' },
  ],
  signal: [
    { code: 'false_threat', label: 'Ложная угроза' },
    { code: 'spam', label: 'Спам' },
    { code: 'offensive', label: 'Оскорбительный текст или фото' },
    { code: 'duplicate', label: 'Дубликат' },
    { code: 'manipulation', label: 'Манипуляция' },
    { code: 'other', label: 'Другое' },
  ],
  comment: [
    { code: 'offensive', label: 'Оскорбление' },
    { code: 'spam', label: 'Спам' },
    { code: 'offtopic', label: 'Флуд или не по теме' },
    { code: 'harassment', label: 'Травля' },
    { code: 'provocation', label: 'Провокация' },
    { code: 'other', label: 'Другое' },
  ],
};

const commentEnabledReasons = new Set([
  'other',
  'false_info',
  'offensive',
  'spam',
  'duplicate',
  'wrong_photo',
  'offtopic',
  'fake_object',
  'offensive_object',
  'flood',
  'irrelevant_to_monument',
  'low_quality',
  'private_data',
  'false_threat',
  'manipulation',
  'harassment',
  'provocation',
]);

const entityLabel = computed(() => ({
  monument: 'памятник',
  post: 'пост',
  photo: 'фотографию',
  signal: 'сигнал',
  comment: 'комментарий',
}[props.entityType]));

const subjectLine = computed(() => props.subject ? `Объект: ${props.subject}` : 'Выберите причину и при необходимости добавьте уточнение.');
const reasons = computed(() => reasonMap[props.entityType] || []);
const showComment = computed(() => commentEnabledReasons.has(selectedReason.value));
const suggestedLonText = computed(() => suggestedLon.value === null ? '' : suggestedLon.value.toFixed(6));
const suggestedLatText = computed(() => suggestedLat.value === null ? '' : suggestedLat.value.toFixed(6));

watch(() => props.modelValue, (open) => {
  if (open) dialogRef.value?.showModal();
  else if (dialogRef.value?.open) dialogRef.value.close();
});

const resetForm = () => {
  selectedReason.value = '';
  comment.value = '';
  suggestedTitle.value = '';
  suggestedLon.value = null;
  suggestedLat.value = null;
  duplicateQuery.value = props.duplicateSeed || '';
  duplicateSuggestions.value = [];
  duplicateTargetId.value = '';
  error.value = '';
  submitting.value = false;
};

watch(() => props.modelValue, (open) => {
  if (open) resetForm();
});

const close = () => {
  emit('update:modelValue', false);
  error.value = '';
};

const loadSuggestions = async () => {
  if (props.entityType !== 'monument' || selectedReason.value !== 'duplicate' || duplicateQuery.value.trim().length < 2) {
    duplicateSuggestions.value = [];
    return;
  }
  try {
    const { data } = await api.get('/search/suggest', { params: { q: duplicateQuery.value, limit: 5 } });
    duplicateSuggestions.value = Array.isArray(data) ? data : (data.items || []);
  } catch {
    duplicateSuggestions.value = [];
  }
};

const debouncedLoadSuggestions = () => {
  if (suggestionTimer) window.clearTimeout(suggestionTimer);
  suggestionTimer = window.setTimeout(loadSuggestions, 250);
};

const selectDuplicate = (item: SuggestionItem) => {
  duplicateTargetId.value = item.id;
  duplicateQuery.value = item.name;
};

const destroyMap = () => {
  if (pickerMarker) {
    pickerMarker.remove();
    pickerMarker = null;
  }
  if (pickerMap) {
    pickerMap.remove();
    pickerMap = null;
  }
};

const updateMarker = () => {
  if (!pickerMap || suggestedLon.value === null || suggestedLat.value === null) return;
  if (!pickerMarker) {
    pickerMarker = new maplibregl.Marker({ color: '#dc2626' }).setLngLat([suggestedLon.value, suggestedLat.value]).addTo(pickerMap);
  } else {
    pickerMarker.setLngLat([suggestedLon.value, suggestedLat.value]);
  }
};

const initPickerMap = async () => {
  if (!props.modelValue || selectedReason.value !== 'wrong_coords') return;
  await nextTick();
  if (!mapContainerRef.value || pickerMap) return;

  const centerLon = suggestedLon.value ?? defaultLon;
  const centerLat = suggestedLat.value ?? defaultLat;

  pickerMap = new maplibregl.Map({
    container: mapContainerRef.value,
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
    center: [centerLon, centerLat],
    zoom: 12,
  });

  pickerMap.on('click', (e) => {
    suggestedLon.value = Number(e.lngLat.lng.toFixed(6));
    suggestedLat.value = Number(e.lngLat.lat.toFixed(6));
    updateMarker();
  });

  pickerMap.on('load', () => {
    if (suggestedLon.value === null || suggestedLat.value === null) {
      suggestedLon.value = Number(centerLon.toFixed(6));
      suggestedLat.value = Number(centerLat.toFixed(6));
    }
    updateMarker();
  });
};

watch(() => [props.modelValue, selectedReason.value] as const, async ([open, reason]) => {
  if (!open) {
    destroyMap();
    return;
  }
  if (reason === 'wrong_coords') {
    await initPickerMap();
    return;
  }
  destroyMap();
});

onBeforeUnmount(() => {
  destroyMap();
});

const submit = async () => {
  error.value = '';
  if (!selectedReason.value) {
    error.value = 'Выберите причину жалобы.';
    return;
  }
  if (selectedReason.value === 'other' && !comment.value.trim()) {
    error.value = 'Для причины "Другое" нужен комментарий.';
    return;
  }
  if (selectedReason.value === 'wrong_name' && !suggestedTitle.value.trim()) {
    error.value = 'Укажите правильное название.';
    return;
  }
  if (selectedReason.value === 'wrong_coords' && (suggestedLon.value === null || suggestedLat.value === null)) {
    error.value = 'Укажите правильную точку на карте.';
    return;
  }
  if (selectedReason.value === 'duplicate' && props.entityType === 'monument' && !duplicateTargetId.value) {
    error.value = 'Выберите существующий памятник для объединения.';
    return;
  }

  submitting.value = true;
  try {
    await api.post('/reports', {
      entity_type: props.entityType,
      entity_id: props.entityId,
      reason_code: selectedReason.value,
      comment: comment.value || undefined,
      suggested_title: selectedReason.value === 'wrong_name' ? suggestedTitle.value || undefined : undefined,
      suggested_lon: selectedReason.value === 'wrong_coords' ? suggestedLon.value : undefined,
      suggested_lat: selectedReason.value === 'wrong_coords' ? suggestedLat.value : undefined,
      duplicate_target_id: selectedReason.value === 'duplicate' ? duplicateTargetId.value || undefined : undefined,
    });
    emit('submitted');
    close();
  } catch (err: any) {
    error.value = err?.response?.data?.message || 'Не удалось отправить жалобу.';
  } finally {
    submitting.value = false;
  }
};
</script>
