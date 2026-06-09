<template>
  <div class="min-h-screen bg-base-100">
    <div class="max-w-6xl mx-auto p-6 space-y-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <button @click="goBack" class="btn btn-ghost">
          <ArrowLeftIcon class="w-5 h-5 mr-2" />
          Назад к очереди
        </button>
        <div class="badge badge-lg px-6 py-5 rounded-2xl font-black text-base uppercase tracking-wider" :class="typeClass">{{ typeLabel }}</div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-20">
        <span class="loading loading-spinner loading-lg"></span>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="alert alert-error">
        {{ error }}
      </div>

      <!-- Content -->
      <div v-else-if="item" class="space-y-6">
        <!-- AI Warnings -->
        <div v-if="parsedAIFlags.length > 0" class="p-6 bg-error/10 text-error border border-error/20 rounded-3xl">
          <h3 class="font-black text-xl flex items-center gap-2 mb-4">
            <AlertTriangleIcon class="w-7 h-7" /> ВНИМАНИЕ ИИ
          </h3>
          <ul class="list-disc pl-6 space-y-2 font-bold">
            <li v-for="(warning, idx) in parsedAIFlags" :key="idx">{{ warning }}</li>
          </ul>
        </div>

        <!-- Title & Author -->
        <div class="bg-base-200 p-8 rounded-3xl">
          <h1 class="text-3xl font-black mb-4">{{ itemTitle }}</h1>
          <div v-if="isEditRequest" class="mb-4 inline-flex items-center rounded-full bg-amber-100 px-4 py-2 text-xs font-black uppercase tracking-[0.22em] text-amber-900">
            Заявка на редактирование
          </div>
          <div v-if="item.monument_name" class="text-lg opacity-70 mb-2">
            Памятник: <span class="font-bold">{{ item.monument_name }}</span>
          </div>
          <div class="flex items-center gap-4">
            <div class="avatar placeholder">
              <div class="bg-neutral text-neutral-content rounded-xl w-10 h-10 font-bold">
                {{ item.author_name?.[0] || 'U' }}
              </div>
            </div>
            <div>
              <div class="font-bold">{{ item.author_name || 'Аноним' }}</div>
              <div class="text-sm opacity-50">{{ new Date(item.created_at).toLocaleString() }}</div>
            </div>
          </div>
        </div>

        <!-- Description -->
        <div v-if="itemDescription && !isEditRequest" class="bg-base-200 p-8 rounded-3xl">
          <h3 class="font-black text-sm mb-4 opacity-50 uppercase tracking-wider">Описание</h3>
          <FormattedText class="text-base leading-relaxed" :text="itemDescription" />
        </div>

        <div v-if="type === 'reports'" class="bg-base-200 p-8 rounded-3xl space-y-5">
          <div class="p-5 rounded-2xl bg-warning/10 border border-warning/20">
            <div class="font-black text-xs uppercase tracking-wider opacity-60 mb-2">Рекомендуемое действие</div>
            <div class="font-bold">{{ recommendedAction }}</div>
          </div>

          <div class="rounded-2xl bg-base-100 border border-base-300 p-5 space-y-3">
            <div class="font-black text-sm uppercase tracking-wider opacity-50">Данные объекта</div>
            <div class="grid gap-3 md:grid-cols-2">
              <div>
                <div class="text-xs opacity-50 uppercase tracking-wider mb-1">Название</div>
                <div class="font-bold">{{ reportSnapshot.name || reportSnapshot.monument_name || 'Без названия' }}</div>
              </div>
              <div>
                <div class="text-xs opacity-50 uppercase tracking-wider mb-1">Автор</div>
                <div class="font-bold">{{ reportSnapshot.author_name || 'Не указан' }}</div>
              </div>
              <div v-if="reportSnapshot.status">
                <div class="text-xs opacity-50 uppercase tracking-wider mb-1">Статус</div>
                <div class="font-bold">{{ reportSnapshot.status }}</div>
              </div>
              <div v-if="reportSnapshot.lat !== undefined && reportSnapshot.lon !== undefined">
                <div class="text-xs opacity-50 uppercase tracking-wider mb-1">Координаты</div>
                <div class="font-bold">{{ Number(reportSnapshot.lat).toFixed(6) }}, {{ Number(reportSnapshot.lon).toFixed(6) }}</div>
              </div>
              <div v-if="reportSnapshot.linked_posts_count !== undefined">
                <div class="text-xs opacity-50 uppercase tracking-wider mb-1">Связанные посты</div>
                <div class="font-bold">{{ reportSnapshot.linked_posts_count }}</div>
              </div>
            </div>
            <router-link
              v-if="reportEntityLink"
              :to="reportEntityLink"
              class="btn btn-sm btn-outline rounded-xl font-bold mt-2"
            >
              Открыть объект
            </router-link>
          </div>

          <template v-if="item.votes?.length">
            <h3 class="font-black text-sm mb-1 opacity-50 uppercase tracking-wider">Голоса жалобы</h3>
            <div v-for="vote in item.votes" :key="vote.id" class="p-4 rounded-2xl bg-base-100 border border-base-300">
              <div class="flex items-center justify-between gap-3 mb-2">
                <div class="font-bold">{{ voteReporterLabel(vote) }}</div>
                <div class="text-xs opacity-50 font-bold">{{ new Date(vote.created_at).toLocaleString() }}</div>
              </div>
              <div class="text-xs uppercase tracking-wider opacity-45 mb-2">{{ reasonLabel(vote.reason_code) }}</div>
              <p v-if="vote.comment" class="text-sm opacity-75">{{ vote.comment }}</p>
              <p v-else class="text-sm opacity-40">Комментарий не добавлен</p>
            </div>
          </template>
        </div>

        <!-- Diff for edits -->
        <div v-if="reviewChanges.length" class="bg-base-200 p-8 rounded-3xl space-y-5">
          <h3 class="font-black text-sm mb-1 opacity-50 uppercase tracking-wider">Что изменится</h3>
          <div v-for="change in reviewChanges" :key="change.field" class="rounded-2xl bg-base-300 p-5">
            <div class="font-black uppercase opacity-45 text-[11px] mb-3">{{ change.fieldLabel }}</div>
            <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
              <div class="rounded-xl bg-error/10 px-4 py-3">
                <div class="text-[10px] font-black uppercase tracking-wider opacity-55 mb-1">Было</div>
                <FormattedText :text="change.oldValue" class="text-sm leading-relaxed opacity-80" />
              </div>
              <div class="rounded-xl bg-success/10 px-4 py-3">
                <div class="text-[10px] font-black uppercase tracking-wider opacity-55 mb-1">Будет</div>
                <FormattedText :text="change.newValue" class="text-sm font-bold leading-relaxed" />
              </div>
            </div>
          </div>
        </div>

        <div v-if="photoReview.old.length || photoReview.new.length" class="bg-base-200 p-8 rounded-3xl space-y-5">
          <h3 class="font-black text-sm mb-1 opacity-50 uppercase tracking-wider">Изменения фотографий</h3>
          <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <div class="rounded-2xl bg-base-300 p-5">
              <div class="mb-3 text-[10px] font-black uppercase tracking-wider text-rose-700">Было</div>
              <div v-if="photoReview.old.length" class="grid grid-cols-2 gap-3 md:grid-cols-3">
                <div v-for="photo in photoReview.old" :key="photo.key" class="space-y-2">
                  <img :src="normalizeImagePath(photo.path)" class="aspect-square w-full rounded-2xl object-cover ring-2 ring-rose-200" />
                  <div class="text-xs font-semibold text-rose-700">{{ photo.label }}</div>
                </div>
              </div>
              <div v-else class="text-sm opacity-45">Фотографий до изменения не было.</div>
            </div>
            <div class="rounded-2xl bg-base-300 p-5">
              <div class="mb-3 text-[10px] font-black uppercase tracking-wider text-emerald-700">Стало</div>
              <div v-if="photoReview.new.length" class="grid grid-cols-2 gap-3 md:grid-cols-3">
                <div v-for="photo in photoReview.new" :key="photo.key" class="space-y-2">
                  <img :src="normalizeImagePath(photo.path)" class="aspect-square w-full rounded-2xl object-cover ring-2 ring-emerald-200" />
                  <div class="text-xs font-semibold text-emerald-700">{{ photo.label }}</div>
                </div>
              </div>
              <div v-else class="text-sm opacity-45">После изменения фотографий не осталось.</div>
            </div>
          </div>
        </div>

        <div v-if="mapCenter" class="bg-base-200 p-8 rounded-3xl">
          <h3 class="font-black text-sm mb-4 opacity-50 uppercase flex items-center gap-2 tracking-wider">
            <MapIcon class="w-5 h-5" /> Координаты
          </h3>
          <div class="mt-4 grid gap-2 md:grid-cols-2">
            <div v-if="oldCoords" class="rounded-2xl bg-base-100 px-4 py-3">
              <div class="text-[10px] font-black uppercase tracking-wider text-rose-700 mb-1">Было</div>
              <div class="font-mono font-bold">{{ oldCoords.lat.toFixed(6) }}, {{ oldCoords.lon.toFixed(6) }}</div>
            </div>
            <div v-if="newCoords" class="rounded-2xl bg-base-100 px-4 py-3">
              <div class="text-[10px] font-black uppercase tracking-wider text-emerald-700 mb-1">{{ oldCoords ? 'Стало' : 'Текущая точка' }}</div>
              <div class="font-mono font-bold">{{ newCoords.lat.toFixed(6) }}, {{ newCoords.lon.toFixed(6) }}</div>
            </div>
          </div>
          <div id="mod_map" class="mt-4 w-full h-64 rounded-2xl border border-base-300"></div>
        </div>

        <!-- Photos -->
        <div v-if="photos.length > 0" class="bg-base-200 p-8 rounded-3xl">
          <h3 class="font-black text-sm mb-4 opacity-50 uppercase flex items-center gap-2 tracking-wider">
            <ImageIcon class="w-5 h-5" /> Фотографии ({{ photos.length }})
          </h3>
          <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            <div 
              v-for="(photo, idx) in photos" 
              :key="idx"
              class="relative cursor-pointer group rounded-2xl overflow-hidden shadow-md aspect-square"
              @click="openGallery(Number(idx))"
            >
              <img :src="normalizeImagePath(photo.preview_path || photo.thumbnail_path || photo.file_path)" class="w-full h-full object-cover" />
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/30 flex items-center justify-center transition-colors">
                <span class="text-white opacity-0 group-hover:opacity-100 font-bold bg-black/60 px-4 py-2 rounded-xl backdrop-blur-sm">
                  Увеличить
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Actions -->
        <div v-if="type === 'signals'" class="bg-base-200 p-6 rounded-3xl">
          <h3 class="font-black text-sm mb-4 opacity-50 uppercase tracking-wider">Уровень срочности сигнала</h3>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <button type="button" class="btn rounded-2xl font-black" :class="selectedUrgency === 'low' ? 'btn-info text-white' : 'btn-ghost bg-base-100'" @click="selectedUrgency = 'low'">Низкая</button>
            <button type="button" class="btn rounded-2xl font-black" :class="selectedUrgency === 'medium' ? 'btn-warning text-white' : 'btn-ghost bg-base-100'" @click="selectedUrgency = 'medium'">Средняя</button>
            <button type="button" class="btn rounded-2xl font-black" :class="selectedUrgency === 'high' ? 'btn-error text-white' : 'btn-ghost bg-base-100'" @click="selectedUrgency = 'high'">Критическая</button>
          </div>
          <p class="text-xs font-bold opacity-45 mt-3">Для сигнала нужно обязательно указать срочность перед подтверждением.</p>
        </div>

        <div class="flex items-center justify-end gap-4 pt-6 border-t border-base-300">
          <button @click="reject" class="btn btn-outline border-error text-error hover:bg-error hover:text-white px-10 h-16 rounded-2xl font-black text-lg">
            <XIcon class="w-6 h-6 mr-2" />
            Отклонить
          </button>
          <button @click="approve" class="btn btn-success px-10 h-16 rounded-2xl font-black text-white text-lg shadow-xl shadow-success/20">
            <CheckIcon class="w-6 h-6 mr-2" />
            {{ type === 'reports' && item?.category === 'integrity' ? 'Подтвердить' : 'Одобрить' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Photo Gallery -->
    <PhotoGallery
      :photos="galleryPhotos"
      :initial-index="galleryIdx"
      v-model="galleryOpen"
      :show-captions="false"
    />

    <!-- Reject Modal -->
    <dialog id="reject_modal" class="modal modal-bottom sm:modal-middle bg-base-300/30 backdrop-blur-xl">
      <div class="modal-box w-[min(94vw,860px)] max-w-none rounded-[3rem] p-10 border border-base-200 shadow-2xl relative">
        <button
          type="button"
          @click="closeRejectModal"
          class="btn btn-circle btn-sm absolute top-5 right-5 z-20 bg-base-100/85 border-base-300 hover:bg-base-200"
          aria-label="Закрыть форму"
        >
          <XIcon class="w-4 h-4" />
        </button>
        <h3 class="text-3xl font-black tracking-tight mb-4">Причина отклонения</h3>
        <p class="opacity-60 font-medium mb-6">Укажите причину отклонения.</p>
        
        <div class="flex flex-wrap gap-2 mb-4">
          <button
            v-for="preset in rejectPresets"
            :key="preset"
            type="button"
            @click="rejectComment = preset"
            class="badge badge-error badge-outline hover:bg-primary/10 hover:border-primary hover:text-primary cursor-pointer py-3 h-auto font-bold"
          >
            {{ preset }}
          </button>
        </div>

        <textarea 
          v-model="rejectComment"
          class="textarea textarea-bordered w-full h-40 rounded-3xl bg-base-200 border-none focus:ring-4 focus:ring-error/10 text-lg font-medium p-6" 
          placeholder="Причина..."
        ></textarea>
        
        <div class="modal-action gap-3 mt-6">
          <form method="dialog" class="flex-grow">
            <button class="btn btn-ghost w-full h-14 rounded-2xl font-bold">Отмена</button>
          </form>
          <button @click="confirmReject" class="btn btn-error flex-[2] h-14 rounded-2xl font-black text-white">
            Подтвердить
          </button>
        </div>
      </div>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import { useToast } from '../composables/useToast';
import PhotoGallery from '../components/PhotoGallery.vue';
import FormattedText from '../components/FormattedText.vue';
import { 
  ArrowLeftIcon, AlertTriangleIcon, ImageIcon, MapIcon,
  XIcon, CheckIcon 
} from 'lucide-vue-next';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';

const route = useRoute();
const router = useRouter();
const toast = useToast();

const loading = ref(true);
const error = ref('');
const item = ref<any>(null);
const type = computed(() => route.params.type as string);
const id = computed(() => route.params.id as string);

const typeLabel = computed(() => {
  const labels: Record<string, string> = {
    monuments: 'Новая точка',
    posts: 'Новый пост',
    signals: 'Сигнал угрозы',
    edits: 'Правка',
    reports: 'Жалоба',
  };
  return labels[type.value] || 'Заявка';
});

const typeClass = computed(() => {
  const classes: Record<string, string> = {
    monuments: 'badge-primary',
    posts: 'badge-secondary',
    signals: 'badge-error',
    edits: 'badge-warning',
    reports: 'badge-error',
  };
  return classes[type.value] || 'badge-primary';
});

const itemTitle = computed(() => {
  if (!item.value) return '';
  if (type.value === 'reports') {
    return `${entityTypeLabel(item.value.entity_type)}: ${reasonLabel(item.value.reason_code)}`;
  }
  if (type.value === 'edits') {
    return item.value.title || item.value.monument_name || 'Заявка на редактирование';
  }
  if (type.value === 'signals') {
    return item.value.signal_type ? 'Сигнал: ' + signalTypeLabel(item.value.signal_type) : 'Сигнал угрозы';
  }
  return item.value.name || item.value.monument_name || item.value.title || 'Без названия';
});

const signalTypeLabel = (value: string) => {
  const labels: Record<string, string> = {
    demolition: 'Риск сноса',
    vandalism: 'Вандализм или повреждение',
    poor_condition: 'Плохое состояние памятника',
    trash: 'Захламление территории',
    unsafe_work: 'Подозрительные работы рядом',
    other: 'Другая угроза',
    neglect: 'Плохое состояние памятника',
    damage: 'Вандализм или повреждение',
    reconstruction: 'Подозрительные работы рядом',
  };
  return labels[value] || value;
};

const itemDescription = computed(() => {
  if (!item.value) return '';
  if (type.value === 'reports') {
    const snapshot = item.value.entity_snapshot || {};
    const direct = typeof snapshot.description === 'string' ? snapshot.description.trim() : '';
    if (direct) return direct;
    const content = typeof snapshot.content === 'string' ? snapshot.content.trim() : '';
    if (content) return content;
    return `Жалоб: ${item.value.distinct_reporters_count || item.value.reports_count || 0}`;
  }
  const direct = typeof item.value.description === 'string' ? item.value.description.trim() : '';
  if (direct) return direct;
  const fromProps = typeof item.value.properties?.description === 'string' ? item.value.properties.description.trim() : '';
  if (fromProps) return fromProps;
  if (Array.isArray(item.value.posts)) {
    const firstWithText = item.value.posts.find((p: any) => typeof p?.description === 'string' && p.description.trim() !== '');
    if (firstWithText?.description) return String(firstWithText.description).trim();
  }
  return '';
});

const parsedAIFlags = computed(() => {
  if (!item.value?.ai_flags) return [];
  if (type.value === 'reports' && item.value?.entity_snapshot?.ai_flags) {
    const flags = item.value.entity_snapshot.ai_flags;
    const warnings: string[] = [];
    if (flags.toxic_text) warnings.push('Токсичный текст');
    if (flags.image_filter?.error) warnings.push('Ошибка модерации изображения');
    if (flags.high_risk) warnings.push('Высокий риск');
    return warnings;
  }
  const flags = item.value.ai_flags;
  const warnings: string[] = [];
  if (flags.toxic_text) warnings.push('Токсичный текст');
  if (flags.image_filter?.error) warnings.push('Ошибка модерации изображения');
  if (flags.high_risk) warnings.push('Высокий риск');
  return warnings;
});

const photos = computed(() => {
  if (!item.value) return [];
  if (type.value === 'reports') {
    const snapshot = item.value.entity_snapshot || {};
    if (snapshot.thumbnail || snapshot.preview) {
      return [{
        preview_path: snapshot.preview || snapshot.thumbnail,
        thumbnail_path: snapshot.thumbnail || snapshot.preview,
      }];
    }
    return [];
  }
  if (item.value.photos && Array.isArray(item.value.photos) && item.value.photos.length > 0) {
    return item.value.photos;
  }
  if (item.value.posts && Array.isArray(item.value.posts)) {
    const fromPosts = item.value.posts.flatMap((post: any) => (Array.isArray(post.photos) ? post.photos : []));
    if (fromPosts.length > 0) return fromPosts;
  }
  if (item.value.thumbnail) return [{ preview_path: item.value.thumbnail }];
  if (item.value.thumbnail_path) return [{ preview_path: item.value.thumbnail_path }];
  return [];
});

const reportSnapshot = computed(() => (type.value === 'reports' ? (item.value?.entity_snapshot || {}) : {}));
const reportEntityLink = computed(() => {
  if (type.value !== 'reports' || !item.value) return '';
  if (item.value.entity_type === 'monument') return `/monument/${item.value.entity_id}`;
  if (item.value.entity_type === 'signal') return `/signal/${item.value.entity_id}`;
  if (reportSnapshot.value?.monument_id) return `/monument/${reportSnapshot.value.monument_id}`;
  if (reportSnapshot.value?.signal_id) return `/signal/${reportSnapshot.value.signal_id}`;
  return '';
});

const galleryOpen = ref(false);
const galleryIdx = ref(0);

const normalizeImagePath = (path?: string) => {
  if (!path) return '';
  return path.startsWith('/') ? path : `/${path}`;
};

const fieldLabel = (value: string) => {
  const labels: Record<string, string> = {
    name: 'Название',
    description: 'Описание',
    lat: 'Широта',
    lon: 'Долгота',
    photo_removed: 'Удаляемые фотографии',
    photo_added: 'Добавляемые фотографии',
  };
  return labels[value] || value;
};

const parseCoords = (raw?: string) => {
  if (!raw) return null;
  const parts = raw.split(',').map((part) => Number(part.trim()));
  if (parts.length !== 2 || parts.some((part) => Number.isNaN(part))) return null;
  return { lat: parts[0], lon: parts[1] };
};

const galleryPhotos = computed(() => {
  return photos.value.map((p: any) => ({
    ...p,
    file_path: p.file_path ? normalizeImagePath(p.file_path) : undefined,
    preview_path: p.preview_path ? normalizeImagePath(p.preview_path) : undefined,
    thumbnail_path: p.thumbnail_path ? normalizeImagePath(p.thumbnail_path) : undefined,
  }));
});

const openGallery = (idx: number) => {
  galleryIdx.value = idx;
  galleryOpen.value = true;
};

const selectedUrgency = ref<'low' | 'medium' | 'high' | ''>('');
const isEditRequest = computed(() => {
  if (!item.value) return false;
  return type.value === 'edits' || !!item.value.is_edit_request || reviewChanges.value.length > 0;
});
const normalizeEditPayload = (payload: any) => {
  if (!payload || typeof payload !== 'object') return payload;
  if (payload.edit && typeof payload.edit === 'object') {
    return { ...payload.edit, ...payload };
  }
  if (payload.item && typeof payload.item === 'object') {
    return { ...payload.item, ...payload };
  }
  return payload;
};
const auditSource = computed(() => {
  if (!item.value) return [];
  if (type.value === 'edits') {
    if (Array.isArray(item.value.related) && item.value.related.length > 0) return item.value.related;
    if (item.value.entry) return [item.value.entry];
  }
  if (Array.isArray(item.value.audit) && item.value.audit.length > 0) return item.value.audit;
  return [];
});
const reviewChanges = computed(() => {
  if (!item.value) return [];
  const normalize = (value: unknown) => {
    if (value === undefined || value === null || value === '') return '(пусто)';
    return String(value);
  };
  if (item.value.diff && typeof item.value.diff === 'object') {
    return Object.entries(item.value.diff).map(([field, change]: [string, any]) => ({
      field,
      fieldLabel: field,
      oldValue: normalize(change?.old),
      newValue: normalize(change?.new),
    }));
  }
  if (auditSource.value.length > 0) {
    return auditSource.value
      .filter((entry: any) => (entry.status === 'pending' || type.value === 'edits') && !String(entry.field_name || '').startsWith('photo_'))
      .map((entry: any) => ({
        field: entry.field_name || 'field',
        fieldLabel: fieldLabel(entry.field_name || 'field'),
        oldValue: normalize(entry.old_value),
        newValue: normalize(entry.new_value),
      }));
  }
  if (item.value.field_name || item.value.old_value !== undefined || item.value.new_value !== undefined) {
    return [{
      field: item.value.field_name || 'field',
      fieldLabel: fieldLabel(item.value.field_name || 'field'),
      oldValue: normalize(item.value.old_value),
      newValue: normalize(item.value.new_value),
    }];
  }
  return [];
});
const photoReview = computed(() => {
  const oldPhotos: Array<{ key: string; path: string; label: string }> = [];
  const newPhotos: Array<{ key: string; path: string; label: string }> = [];
  for (const entry of auditSource.value) {
    const field = String(entry.field_name || '');
    if (field === 'photo_removed' && entry.old_value) {
      oldPhotos.push({
        key: `${entry.id}-old`,
        path: String(entry.old_value),
        label: 'Будет удалено',
      });
    }
    if (field === 'photo_added' && entry.new_value) {
      newPhotos.push({
        key: `${entry.id}-new`,
        path: String(entry.new_value),
        label: 'Будет добавлено',
      });
    }
  }
  return { old: oldPhotos, new: newPhotos };
});
const oldCoords = computed<{ lat: number; lon: number } | null>(() => {
  const latEntry = reviewChanges.value.find((entry: any) => entry.field === 'lat');
  const lonEntry = reviewChanges.value.find((entry: any) => entry.field === 'lon');
  const baseLat = typeof item.value?.lat === 'number' ? item.value.lat : null;
  const baseLon = typeof item.value?.lon === 'number' ? item.value.lon : null;
  if (latEntry || lonEntry) {
    const lat = latEntry ? Number(latEntry.oldValue) : baseLat;
    const lon = lonEntry ? Number(lonEntry.oldValue) : baseLon;
    if (lat !== null && lon !== null && !Number.isNaN(lat) && !Number.isNaN(lon)) return { lat, lon };
  }
  if (typeof item.value?.entry?.field_name === 'string' && item.value.entry.field_name === 'lat_lon' && item.value.entry.old_value) {
    const parsed = parseCoords(item.value.entry.old_value);
    if (parsed) return parsed;
  }
  return null;
});
const newCoords = computed<{ lat: number; lon: number } | null>(() => {
  const latEntry = reviewChanges.value.find((entry: any) => entry.field === 'lat');
  const lonEntry = reviewChanges.value.find((entry: any) => entry.field === 'lon');
  const baseLat = typeof item.value?.lat === 'number' ? item.value.lat : null;
  const baseLon = typeof item.value?.lon === 'number' ? item.value.lon : null;
  if (latEntry || lonEntry) {
    const lat = latEntry ? Number(latEntry.newValue) : baseLat;
    const lon = lonEntry ? Number(lonEntry.newValue) : baseLon;
    if (lat !== null && lon !== null && !Number.isNaN(lat) && !Number.isNaN(lon)) return { lat, lon };
  }
  if (typeof item.value?.lat === 'number' && typeof item.value?.lon === 'number') {
    return { lat: item.value.lat, lon: item.value.lon };
  }
  if (typeof reportSnapshot.value?.lat === 'number' && typeof reportSnapshot.value?.lon === 'number') {
    return { lat: reportSnapshot.value.lat, lon: reportSnapshot.value.lon };
  }
  if (typeof item.value?.lat === 'number' && typeof item.value?.lon === 'number') {
    return { lat: item.value.lat, lon: item.value.lon };
  }
  if (typeof item.value?.entry?.field_name === 'string' && item.value.entry.field_name === 'lat_lon' && item.value.entry.new_value) {
    return parseCoords(item.value.entry.new_value);
  }
  return null;
});
const mapCenter = computed<{ lat: number; lon: number } | null>(() => newCoords.value || oldCoords.value);
const recommendedAction = computed(() => {
  if (type.value !== 'reports' || !item.value) return '';
  if (item.value.entity_type === 'monument' && item.value.reason_code === 'wrong_coords') {
    return item.value.suggested_fix?.suggested_lon !== undefined ? 'Принять предложенные координаты' : 'Проверить координаты вручную';
  }
  if (item.value.entity_type === 'monument' && item.value.reason_code === 'wrong_name') {
    return item.value.suggested_fix?.suggested_title ? 'Принять предложенное название' : 'Проверить название памятника';
  }
  if (item.value.entity_type === 'photo') {
    return 'Скрыть только фотографию, если жалоба подтверждается';
  }
  if (item.value.category === 'abuse') {
    return 'Скрыть проблемный контент и уведомить автора';
  }
  return 'Проверить данные и при необходимости применить исправление';
});

const goBack = () => {
  router.push('/moderation');
};

const voteReporterLabel = (vote: any) => {
  if (vote?.reporter_name && String(vote.reporter_name).trim() !== '') {
    return `Жалоба от ${vote.reporter_name}`;
  }
  return 'Жалоба от пользователя';
};

const approve = async () => {
  if (type.value === 'signals' && !selectedUrgency.value) {
    toast.warning('Для сигнала укажите срочность перед подтверждением');
    return;
  }
  try {
    const payload: Record<string, any> = { action: 'approve' };
    if (type.value === 'reports' && item.value?.category === 'integrity') {
      payload.action = item.value?.suggested_fix && Object.keys(item.value.suggested_fix).length > 0 ? 'apply_fix' : 'approve';
    }
    if (type.value === 'signals') {
      payload.urgency = selectedUrgency.value;
    }
    await api.post(`/moderation/${type.value}/${id.value}/action`, payload);
    toast.success('Одобрено');
    router.push('/moderation');
  } catch (err: any) {
    toast.error(err.message || 'Ошибка');
  }
};

const rejectComment = ref('');
const rejectPresets = computed(() => {
  if (type.value === 'reports' && item.value) {
    const reason = item.value.reason_code;
    const entityType = item.value.entity_type;
    if (reason === 'wrong_coords') {
      return [
        'Координаты в жалобе не подтверждены: фактическая точка указана верно.',
        'Предложенная точка некорректна: требуется дополнительная проверка на местности.',
        'Недостаточно данных для изменения координат.'
      ];
    }
    if (reason === 'wrong_name') {
      return [
        'Текущее название подтверждено по доступным источникам.',
        'Предложенное название не подтверждено.',
        'Недостаточно данных для переименования объекта.'
      ];
    }
    if (reason === 'duplicate') {
      return [
        'Совпадение с указанным объектом не подтверждено.',
        'Это разные объекты, объединение не требуется.',
        'Недостаточно оснований считать объект дубликатом.'
      ];
    }
    if (entityType === 'photo') {
      return [
        'Фото соответствует объекту и правилам публикации.',
        'Нарушений в фото не выявлено.',
        'Оснований для скрытия фото недостаточно.'
      ];
    }
    if (entityType === 'comment') {
      return [
        'Комментарий не нарушает правила сообщества.',
        'Признаков оскорбления или спама не обнаружено.',
        'Оснований для удаления комментария недостаточно.'
      ];
    }
    if (item.value.category === 'abuse') {
      return [
        'Нарушение не подтверждено после проверки.',
        'Контент не содержит признаков спама или оскорблений.',
        'Недостаточно доказательств для применения санкций.'
      ];
    }
    return [
      'Жалоба не подтвердилась после проверки.',
      'Данные объекта корректны, изменения не требуются.',
      'Недостаточно доказательств для подтверждения жалобы.'
    ];
  }

  if (type.value === 'signals') {
    return [
      'Угроза не подтверждена по приложенным материалам.',
      'Недостаточно данных для подтверждения сигнала.',
      'Сигнал не соответствует критериям публикации.'
    ];
  }
  if (type.value === 'posts') {
    return [
      'Пост не соответствует правилам публикации.',
      'Содержание поста недостаточно информативно.',
      'Недостаточно подтверждений фактов в публикации.'
    ];
  }
  if (type.value === 'monuments') {
    return [
      'Новые данные о точке не подтверждены.',
      'Недостаточно материалов для публикации объекта.',
      'Точка дублирует существующую запись.'
    ];
  }
  return [
    'Заявка не соответствует правилам публикации.',
    'Недостаточно данных для модерации.',
    'Требуется доработка и повторная отправка.'
  ];
});

const reject = () => {
  const modal = document.getElementById('reject_modal') as HTMLDialogElement;
  modal?.showModal();
};

const closeRejectModal = () => {
  const modal = document.getElementById('reject_modal') as HTMLDialogElement;
  modal?.close();
};

const confirmReject = async () => {
  if (!rejectComment.value.trim()) {
    toast.warning('Укажите комментарий для автора, чтобы он понимал, что исправить');
    return;
  }
  try {
    const payload: Record<string, any> = { action: 'reject' };
    if (type.value === 'signals') {
      payload.official_response = rejectComment.value;
    } else {
      if (type.value === 'reports') {
        payload.moderator_comment = rejectComment.value;
      } else {
        payload.comment = rejectComment.value;
      }
    }
    await api.post(`/moderation/${type.value}/${id.value}/action`, payload);
    toast.success('Отклонено');
    router.push('/moderation');
  } catch (err: any) {
    toast.error(err.message || 'Ошибка');
  }
};

let map: maplibregl.Map | null = null;

const initMap = () => {
  if (!mapCenter.value || map) return;
  
  setTimeout(() => {
    map = new maplibregl.Map({
      container: 'mod_map',
      style: {
        version: 8,
        sources: {
          'yandex-tiles': {
            type: 'raster',
            tiles: ['https://core-renderer-tiles.maps.yandex.net/tiles?l=map&x={x}&y={y}&z={z}&scale=1&lang=ru_RU'],
            tileSize: 256,
          }
        },
        layers: [{ id: 'yandex-layer', type: 'raster', source: 'yandex-tiles' }]
      },
      center: [mapCenter.value!.lon, mapCenter.value!.lat],
      zoom: 15,
    });
    if (oldCoords.value) {
      const oldPopup = new maplibregl.Popup({ closeButton: false, closeOnClick: false, offset: 20 })
        .setText('Было');
      new maplibregl.Marker({ color: '#ef4444' })
        .setLngLat([oldCoords.value.lon, oldCoords.value.lat])
        .setPopup(oldPopup)
        .addTo(map)
        .togglePopup();
    }
    if (newCoords.value) {
      const newPopup = new maplibregl.Popup({ closeButton: false, closeOnClick: false, offset: 20 })
        .setText(oldCoords.value ? 'Стало' : 'Текущая точка');
      new maplibregl.Marker({ color: oldCoords.value ? '#16a34a' : '#16a34a' })
        .setLngLat([newCoords.value.lon, newCoords.value.lat])
        .setPopup(newPopup)
        .addTo(map)
        .togglePopup();
    }
  }, 100);
};

onMounted(async () => {
  try {
    let url = '';
    let data: any = {};
    
    if (type.value === 'monuments') {
      url = `/monuments/${id.value}`;
      const res = await api.get(url);
      data = { ...res.data.monument, photos: res.data.photos, posts: res.data.posts, audit: res.data.audit || [] };
    } else if (type.value === 'posts') {
      url = `/moderation/posts/${id.value}`;
      const res = await api.get(url);
      data = { ...res.data.post, photos: res.data.photos, audit: res.data.audit || [] };
    } else if (type.value === 'signals') {
      url = `/signals/${id.value}`;
      const res = await api.get(url);
      data = { ...res.data.signal, photos: res.data.photos };
    } else if (type.value === 'reports') {
      url = `/moderation/reports/${id.value}`;
      const res = await api.get(url);
      data = res.data;
    } else if (type.value === 'edits') {
      url = `/moderation/edits/${id.value}`;
      const res = await api.get(url);
      data = normalizeEditPayload(res.data);
    } else {
      url = `/moderation/${type.value}/${id.value}`;
      const res = await api.get(url);
      data = res.data;
    }
    
    item.value = data;
    if (type.value === 'signals' && data.status !== 'pending' && ['low', 'medium', 'high'].includes(data.urgency)) {
      selectedUrgency.value = data.urgency;
    }
    loading.value = false;
    
    if (mapCenter.value) {
      setTimeout(initMap, 200);
    }
  } catch (err: any) {
    error.value = err.message || 'Ошибка загрузки';
    loading.value = false;
  }
});

const entityTypeLabel = (value: string) => {
  const labels: Record<string, string> = {
    monument: 'Памятник',
    post: 'Пост',
    photo: 'Фото',
    signal: 'Сигнал',
    comment: 'Комментарий',
  };
  return labels[value] || value;
};

const reasonLabel = (value: string) => {
  const labels: Record<string, string> = {
    wrong_name: 'неверное название',
    wrong_coords: 'неверные координаты',
    duplicate: 'дубликат',
    wrong_photo: 'нерелевантные фото',
    fake_object: 'несуществующий объект',
    offensive_object: 'оскорбительный объект',
    false_info: 'ложная информация',
    offensive: 'оскорбительный контент',
    spam: 'спам',
    flood: 'флуд',
    irrelevant_to_monument: 'не относится к памятнику',
    not_relevant: 'не относится к объекту',
    low_quality: 'плохое или нечитаемое фото',
    private_data: 'личные данные на фото',
    false_threat: 'ложная угроза',
    manipulation: 'манипуляция или ввод в заблуждение',
    offtopic: 'не по теме',
    harassment: 'травля или преследование',
    provocation: 'провокация',
    other: 'другое',
  };
  return labels[value] || 'другая причина';
};
</script>
