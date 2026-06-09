<template>
  <div class="p-8 max-w-5xl mx-auto space-y-8">
    <button class="btn btn-ghost rounded-2xl font-bold" @click="$router.back()">&larr; Назад</button>

    <div v-if="loading" class="space-y-4">
      <div class="h-12 bg-base-200 rounded-2xl animate-pulse"></div>
      <div class="h-96 bg-base-200 rounded-[2rem] animate-pulse"></div>
    </div>

    <div v-else-if="error" class="alert alert-error rounded-2xl">{{ error }}</div>

    <div v-else class="bg-base-100 rounded-[2.5rem] border border-base-200 shadow-xl p-8 space-y-6">
      <div>
        <div class="inline-flex bg-primary/10 p-3 rounded-2xl text-primary mb-4 font-bold tracking-tight text-sm uppercase">
          Повторная отправка
        </div>
        <h1 class="text-3xl font-black tracking-tight">{{ editorTitle }}</h1>
        <p class="opacity-60 font-medium">Исправьте замечания, и заявка снова уйдет на модерацию.</p>
      </div>

      <div v-if="moderationComment" class="bg-error/10 border border-error/10 rounded-2xl p-5">
        <div class="text-xs font-black uppercase tracking-widest text-error mb-2">Комментарий модератора</div>
        <p class="font-bold text-error/80">{{ moderationComment }}</p>
      </div>

      <div v-if="type === 'monument'" class="form-control w-full gap-3">
        <label class="label font-bold opacity-70">Название</label>
        <input v-model="form.name" class="input w-full rounded-2xl h-14 bg-base-200 border-2 font-medium transition-colors focus:outline-none focus:ring-4 focus:ring-primary/10" :class="validation.badFields.name ? 'border-error' : 'border-transparent focus:border-primary'" />
        <span v-if="validation.badFields.name" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('name', validation.badFields.name) }}</span>
      </div>

      <div v-if="type === 'monument'" class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="form-control w-full gap-3">
          <label class="label font-bold opacity-70">Долгота</label>
          <input v-model.number="form.lon" type="number" step="any" class="input w-full rounded-2xl h-14 bg-base-200 border-2 font-medium transition-colors focus:outline-none focus:ring-4 focus:ring-primary/10" :class="validation.badFields.lon ? 'border-error' : 'border-transparent focus:border-primary'" />
          <span v-if="validation.badFields.lon" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('lon', validation.badFields.lon) }}</span>
        </div>
        <div class="form-control w-full gap-3">
          <label class="label font-bold opacity-70">Широта</label>
          <input v-model.number="form.lat" type="number" step="any" class="input w-full rounded-2xl h-14 bg-base-200 border-2 font-medium transition-colors focus:outline-none focus:ring-4 focus:ring-primary/10" :class="validation.badFields.lat ? 'border-error' : 'border-transparent focus:border-primary'" />
          <span v-if="validation.badFields.lat" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('lat', validation.badFields.lat) }}</span>
        </div>
      </div>
      <div v-if="type === 'monument'" class="space-y-3">
        <div class="flex items-center justify-between gap-3">
          <label class="font-bold opacity-70">Положение на карте</label>
          <span class="text-xs font-semibold opacity-50">Нажмите на карту, чтобы выбрать точку</span>
        </div>
        <div ref="coordsMapRef" class="h-72 w-full overflow-hidden rounded-[1.5rem] border border-base-200"></div>
      </div>

      <div class="form-control w-full gap-3">
        <label class="label font-bold opacity-70">Описание</label>
        <RichTextEditor v-model="form.description" :invalid="!!validation.badFields.description" />
        <span v-if="validation.badFields.description" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('description', validation.badFields.description) }}</span>
      </div>

      <div class="space-y-3">
        <div class="font-bold opacity-70">Текущие фотографии</div>
        <div class="flex flex-wrap gap-3">
          <div v-for="photo in existingPhotos" :key="photo.id" class="relative">
            <img :src="normalizePath(photo.thumbnail_path || photo.preview_path)" class="w-24 h-24 object-cover rounded-2xl" :class="validation.badExistingPhotoIds.includes(photo.id) ? 'ring-4 ring-error' : ''" />
            <button class="btn btn-circle btn-xs btn-error absolute -top-2 -right-2" @click.prevent="togglePhotoRemoval(photo.id)">
              <span v-if="removedPhotoIds.has(photo.id)">+</span>
              <span v-else>&times;</span>
            </button>
            <div v-if="removedPhotoIds.has(photo.id)" class="absolute inset-0 bg-black/50 rounded-2xl grid place-items-center text-white text-xs font-black">Удалится</div>
          </div>
        </div>
      </div>

      <div class="form-control w-full gap-3">
        <label class="label font-bold opacity-70">Новые фотографии</label>
        <input type="file" multiple accept="image/*" class="file-input file-input-bordered rounded-2xl h-14 bg-base-200 border-none" @change="handleFiles" />
        <span v-if="validation.badFields.photos" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('photos', validation.badFields.photos) }}</span>
        <div class="flex flex-wrap gap-3">
          <div v-for="(url, idx) in previewUrls" :key="url" class="relative">
            <img :src="url" class="w-24 h-24 object-cover rounded-2xl" :class="validation.badPhotos.includes(idx) ? 'ring-4 ring-error' : ''" />
            <button class="btn btn-circle btn-xs btn-error absolute -top-2 -right-2" @click.prevent="removeFile(idx)">&times;</button>
          </div>
        </div>
      </div>

      <div v-if="validation.warnings.length > 0" class="alert alert-warning rounded-2xl bg-warning/10 border-none flex flex-col items-start">
        <div class="font-bold">AI-предупреждение</div>
        <ul class="list-disc list-inside text-sm font-medium opacity-80">
          <li v-for="warning in validation.warnings" :key="warning">{{ warning }}</li>
        </ul>
        <label class="flex items-center gap-3 pt-2">
          <input v-model="contentAck" type="checkbox" class="checkbox checkbox-warning rounded-lg" />
          <span class="text-sm font-bold">Я уверен, что исправления корректны</span>
        </label>
      </div>

      <div v-if="validation.duplicates.length > 0" class="bg-base-200 rounded-2xl p-5">
        <div class="font-bold mb-2">Похожие памятники рядом</div>
        <div v-for="item in validation.duplicates" :key="item.id" class="text-sm opacity-70">
          {{ item.name }} — {{ Math.round(item.dist) }} м
        </div>
      </div>

      <div class="flex justify-end gap-3">
        <button class="btn btn-ghost rounded-2xl font-bold" @click="$router.back()">Отмена</button>
        <button class="btn btn-primary rounded-2xl font-black" :disabled="submitting" @click="submit">
          <span v-if="submitting" class="loading loading-spinner"></span>
          <span v-else>Отправить снова</span>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import { useToast } from '../composables/useToast';
import { applyValidationResult, buildFilesFingerprint, buildTextFingerprint, createValidationState } from '../composables/useContentValidation';
import RichTextEditor from '../components/RichTextEditor.vue';
import { normalizeRichTextInput } from '../utils/richText';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';

const route = useRoute();
const router = useRouter();
const toast = useToast();

const type = computed(() => String(route.params.type));
const id = computed(() => String(route.params.id));
const loading = ref(true);
const submitting = ref(false);
const error = ref('');
const contentAck = ref(false);
const previewUrls = ref<string[]>([]);
const selectedFiles = ref<File[]>([]);
const removedPhotoIds = reactive(new Set<string>());
const existingPhotos = ref<any[]>([]);
const validation = createValidationState();
let validateTimer: number | undefined;
const coordsMapRef = ref<HTMLElement | null>(null);
const coordsMap = shallowRef<maplibregl.Map | null>(null);
const coordsMarker = shallowRef<maplibregl.Marker | null>(null);

const form = reactive({
  name: '',
  lon: 0,
  lat: 0,
  description: '',
  monumentId: '',
});

const editorTitle = computed(() => type.value === 'monument' ? 'Исправить заявку на памятник' : 'Исправить отклоненный пост');
const moderationComment = ref('');

const warningMap: Record<string, string> = {
  text_flagged: 'Текст содержит недопустимый контент',
  image_flagged: 'Некоторые фотографии не прошли проверку',
  possible_duplicate: 'Похоже, такой памятник уже есть рядом',
  image_filter_unavailable: 'Сервис проверки изображений временно недоступен',
  text_filter_unavailable: 'Сервис проверки текста временно недоступен',
};

warningMap.invalid_input = 'Проверьте обязательные поля формы';

const fieldMessage = (field: string, value: string) => {
  const fieldMaps: Record<string, Record<string, string>> = {
    name: {
      required: 'Это поле обязательно для заполнения',
      possible_duplicate: 'Похоже, такой памятник уже существует рядом',
    },
    description: {
      required: 'Это поле обязательно для заполнения',
      flagged: 'Описание не прошло AI-модерацию',
    },
    lon: { invalid: 'Укажите корректную долготу' },
    lat: { invalid: 'Укажите корректную широту' },
    photos: {
      min_1: 'Добавьте хотя бы одну фотографию',
      max_10: 'Можно загрузить не более 10 фотографий',
    },
  };
  if (value === 'required_or_photos') return 'Нужно добавить текст поста или фотографии';
  return fieldMaps[field]?.[value] || 'Проверьте заполнение поля';
};

const normalizePath = (path?: string) => {
  if (!path) return '';
  return path.startsWith('/') ? path : `/${path}`;
};

const loadData = async () => {
  loading.value = true;
  error.value = '';
  try {
    if (type.value === 'post') {
      const { data } = await api.get('/me/posts');
      const item = (data.items || []).find((entry: any) => entry.id === id.value);
      if (!item) throw new Error('Пост не найден');
      form.description = item.description || '';
      form.monumentId = item.monument_id;
      moderationComment.value = item.moderation_comment || '';
      existingPhotos.value = item.photos || [];
    } else {
      const { data } = await api.get('/me/monuments');
      const item = (data.items || []).find((entry: any) => entry.id === id.value);
      if (!item) throw new Error('Памятник не найден');
      form.name = item.name || '';
      form.lon = item.lon;
      form.lat = item.lat;
      form.description = item.properties?.description || '';
      moderationComment.value = item.moderation_comment || '';
      existingPhotos.value = item.photos || [];
    }
  } catch (err: any) {
    error.value = err.message || 'Не удалось загрузить данные';
  } finally {
    loading.value = false;
    if (type.value === 'monument') {
      await nextTick();
      initCoordsMap();
    }
  }
};

const initCoordsMap = () => {
  if (type.value !== 'monument' || !coordsMapRef.value || coordsMap.value) return;
  const map = new maplibregl.Map({
    container: coordsMapRef.value,
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
    center: [form.lon || 37.6173, form.lat || 55.7558],
    zoom: 13,
  });
  coordsMap.value = map;
  coordsMarker.value = new maplibregl.Marker({ color: '#2563eb' })
    .setLngLat([form.lon || 37.6173, form.lat || 55.7558])
    .addTo(map);

  map.on('click', (e) => {
    form.lon = Number(e.lngLat.lng.toFixed(6));
    form.lat = Number(e.lngLat.lat.toFixed(6));
    coordsMarker.value?.setLngLat([form.lon, form.lat]);
  });
};

const currentFingerprint = () => [
  form.name.trim(),
  form.lon,
  form.lat,
  buildTextFingerprint(normalizeRichTextInput(form.description)),
  buildFilesFingerprint(selectedFiles.value),
  Array.from(removedPhotoIds).sort().join('|'),
].join('::');

const scheduleValidation = () => {
  if (validateTimer) window.clearTimeout(validateTimer);
  validateTimer = window.setTimeout(() => validate(), selectedFiles.value.length > 0 ? 400 : 700);
};

const markDirty = () => {
  validation.isDirtyAfterValidation = true;
  if (contentAck.value) contentAck.value = false;
  scheduleValidation();
};

watch(() => [form.name, form.lon, form.lat, form.description], () => markDirty(), { deep: true });
watch(() => [form.lon, form.lat], () => {
  if (coordsMap.value) {
    coordsMarker.value?.setLngLat([form.lon, form.lat]);
  }
});

const validate = async () => {
  const fingerprint = currentFingerprint();
  if (!fingerprint || fingerprint === validation.lastValidatedFingerprint) return;
  validation.isValidating = true;
  try {
    const formData = new FormData();
    formData.append('description', normalizeRichTextInput(form.description));
    Array.from(removedPhotoIds).forEach((photoId) => formData.append('remove_photo_ids', photoId));
    selectedFiles.value.forEach((file) => formData.append('photos', file));
    let data;
    if (type.value === 'post') {
      const response = await api.post(`/posts/${id.value}/validate`, formData);
      data = response.data;
    } else {
      formData.append('name', form.name);
      formData.append('lon', String(form.lon));
      formData.append('lat', String(form.lat));
      const response = await api.post(`/monuments/${id.value}/validate`, formData);
      data = response.data;
    }
    applyValidationResult(validation, data, warningMap);
    validation.lastValidatedFingerprint = fingerprint;
    validation.isDirtyAfterValidation = false;
  } catch (err) {
    console.error('Validation failed', err);
  } finally {
    validation.isValidating = false;
  }
};

const handleFiles = (event: Event) => {
  const files = (event.target as HTMLInputElement).files;
  if (!files) return;
  for (let i = 0; i < files.length; i += 1) {
    const file = files[i];
    selectedFiles.value.push(file);
    previewUrls.value.push(URL.createObjectURL(file));
  }
  markDirty();
};

const removeFile = (idx: number) => {
  selectedFiles.value.splice(idx, 1);
  URL.revokeObjectURL(previewUrls.value[idx]);
  previewUrls.value.splice(idx, 1);
  markDirty();
};

const togglePhotoRemoval = (photoId: string) => {
  if (removedPhotoIds.has(photoId)) {
    removedPhotoIds.delete(photoId);
  } else {
    removedPhotoIds.add(photoId);
  }
  markDirty();
};

const submit = async () => {
  submitting.value = true;
  try {
    if (validation.isDirtyAfterValidation) {
      await validate();
    }
    const formData = new FormData();
    formData.append('description', normalizeRichTextInput(form.description));
    formData.append('content_ack', String(contentAck.value));
    Array.from(removedPhotoIds).forEach((photoId) => formData.append('remove_photo_ids', photoId));
    selectedFiles.value.forEach((file) => formData.append('photos', file));

    if (type.value === 'post') {
      await api.put(`/monuments/${form.monumentId}/posts/${id.value}`, formData);
      toast.success('Пост снова отправлен на модерацию');
      await router.push('/profile');
      return;
    }

    formData.append('name', form.name);
    formData.append('lon', String(form.lon));
    formData.append('lat', String(form.lat));
    await api.put(`/monuments/${id.value}`, formData);
    toast.success('Заявка снова отправлена на модерацию');
    await router.push('/profile');
  } catch (err: any) {
    if (err.response?.status === 422) {
      applyValidationResult(validation, {
        requires_ack: !!err.response.data?.data?.requires_ack,
        reasons: err.response.data.data?.reasons || [],
        fields: err.response.data.fields || {},
      }, warningMap);
    } else {
      toast.error(err.response?.data?.message || 'Не удалось отправить исправления');
    }
  } finally {
    submitting.value = false;
  }
};

onMounted(loadData);
onBeforeUnmount(() => {
  if (coordsMap.value) {
    coordsMap.value.remove();
    coordsMap.value = null;
  }
});
</script>
