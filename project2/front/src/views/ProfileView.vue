<template>
  <div class="min-h-screen bg-base-200/30 pb-20">
    <div class="mx-auto max-w-7xl space-y-8 px-5 pt-10 lg:px-8">
      <section class="flex flex-col gap-6 rounded-[2rem] border border-base-200 bg-base-100 p-8 shadow-xl md:flex-row md:items-center">
        <div class="avatar placeholder">
          <div class="h-24 w-24 rounded-[1.5rem] bg-primary text-4xl font-black text-primary-content">
            {{ auth.user?.username?.[0]?.toUpperCase() }}
          </div>
        </div>
        <div class="flex-1">
          <div class="mb-2 flex flex-wrap items-center gap-3">
            <h1 class="text-4xl font-black tracking-tight">{{ auth.user?.username }}</h1>
            <span class="badge badge-primary badge-outline font-bold">{{ translateRole(auth.user?.role_name) }}</span>
          </div>
          <p class="font-medium opacity-60">{{ auth.user?.email }}</p>
          <p v-if="auth.restrictionSummary && auth.restrictionSummary.status !== 'active'" class="mt-4 rounded-2xl border border-warning/20 bg-warning/10 px-4 py-3 text-sm font-semibold">
            {{ auth.restrictionSummary.message }}
          </p>
        </div>
        <button class="btn btn-ghost btn-circle bg-base-200/70" @click="showSettings = true">
          <SettingsIcon class="h-5 w-5" />
        </button>
      </section>

      <section v-if="trustSummary" class="rounded-[2rem] border border-slate-200 bg-white p-6 shadow-xl">
        <div class="grid gap-6 xl:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.85fr)]">
          <div class="space-y-5">
            <div class="inline-flex rounded-full bg-slate-900 px-4 py-2 text-xs font-black uppercase tracking-[0.22em] text-white">
              Уровень доверия
            </div>
            <div>
              <h2 class="text-3xl font-black tracking-tight text-slate-900">{{ trustSummary.label }}</h2>
              <p class="mt-2 max-w-2xl text-base leading-relaxed text-slate-600">{{ trustSummary.message }}</p>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <div class="h-28 overflow-hidden rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3">
                <div class="text-[11px] font-black uppercase tracking-[0.2em] text-slate-400">Текущий счет</div>
                <div class="mt-2 text-3xl font-black text-slate-900">{{ trustSummary.score }}</div>
              </div>
              <div class="h-28 overflow-hidden rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3">
                <div class="text-[11px] font-black uppercase tracking-[0.2em] text-slate-400">Следующий уровень</div>
                <div class="mt-2 truncate text-lg font-black text-slate-900">
                  {{ trustSummary.next_level_label || 'Максимальный уровень' }}
                </div>
                <div v-if="trustSummary.next_level_score !== undefined && trustSummary.next_level_score !== null" class="mt-1 text-sm text-slate-500">
                  Нужно дойти до {{ trustSummary.next_level_score }}
                </div>
              </div>
            </div>

            <div class="rounded-[1.75rem] border border-slate-200 bg-slate-50 p-5">
              <div class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                <div>
                  <div class="text-xs font-black uppercase tracking-[0.2em] text-slate-400">Шкала доверия</div>
                  <div class="mt-2 text-lg font-bold text-slate-900">Текущая позиция отмечена маркером</div>
                </div>
                <div class="text-sm text-slate-500">Диапазон: от -15 до 15</div>
              </div>

              <div class="mt-7">
                <div class="relative h-6 rounded-full bg-slate-100 p-1 shadow-inner">
                  <div class="grid h-full grid-cols-4 gap-1 overflow-hidden rounded-full">
                    <div v-for="segment in trustSegments" :key="segment.key" :class="segment.bgClass" class="h-full rounded-full"></div>
                  </div>
                  <div class="absolute top-1/2 z-10 -translate-y-1/2 -translate-x-1/2" :style="{ left: `${trustMarkerPercent}%` }">
                    <div class="flex flex-col items-center">
                      <div class="mb-2 rounded-full bg-slate-900 px-3 py-1 text-xs font-black text-white shadow-lg">{{ trustSummary.score }}</div>
                      <div class="h-5 w-5 rounded-full border-[3px] border-white bg-slate-900 shadow-lg"></div>
                    </div>
                  </div>
                </div>

                <div class="mt-5 grid grid-cols-2 gap-3 md:grid-cols-4">
                  <div v-for="segment in trustSegments" :key="segment.key + '-label'" class="space-y-1">
                    <div class="text-sm font-black text-slate-900">{{ segment.label }}</div>
                    <div class="text-xs text-slate-500">{{ segment.range }}</div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="space-y-5">
            <div class="rounded-[1.75rem] bg-slate-900 p-6 text-white shadow-2xl">
              <div class="text-xs font-black uppercase tracking-[0.22em] text-white/55">Что действует сейчас</div>
              <div v-if="trustSummary.restrictions.length" class="mt-4 grid gap-3">
                <div v-for="restriction in trustSummary.restrictions" :key="restriction" class="overflow-hidden rounded-2xl bg-white/8 px-4 py-3 text-sm leading-relaxed">
                  {{ restriction }}
                </div>
              </div>
              <div v-else class="mt-4 rounded-2xl bg-emerald-400/15 px-4 py-3 text-sm text-emerald-100">
                Дополнительных ограничений нет.
              </div>
            </div>

            <div class="rounded-[1.75rem] border border-base-200 bg-base-100 p-6">
              <div class="flex items-center justify-between gap-3">
                <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Последние изменения</div>
                <span class="text-xs font-bold opacity-40">{{ trustSummary.recent_events.length }} событий</span>
              </div>
              <div v-if="trustSummary.recent_events.length" class="mt-4 space-y-3">
                <div v-for="event in trustSummary.recent_events" :key="event.id" class="flex gap-4 overflow-hidden rounded-2xl bg-slate-50 px-4 py-3">
                  <div class="mt-1 flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl font-black" :class="event.delta >= 0 ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'">
                    {{ event.delta > 0 ? `+${event.delta}` : event.delta }}
                  </div>
                  <div class="min-w-0 flex-1 overflow-hidden">
                    <div class="truncate font-semibold leading-snug text-slate-900">{{ event.comment || 'Изменение уровня доверия' }}</div>
                    <div class="mt-1 text-xs text-slate-500">{{ formatTrustEventDate(event.created_at) }}</div>
                  </div>
                </div>
              </div>
              <div v-else class="mt-4 rounded-2xl bg-slate-50 px-4 py-3 text-sm text-slate-500">
                Изменений уровня доверия пока не было.
              </div>
            </div>
          </div>
        </div>
      </section>

      <div class="tabs tabs-boxed inline-flex rounded-2xl border border-base-200 bg-base-100/70 p-1">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab tab-lg rounded-xl px-8 font-bold"
          :class="{ 'tab-active bg-primary text-primary-content': activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          {{ tab.name }}
        </button>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <template v-if="loading">
          <div v-for="index in 4" :key="index" class="h-56 animate-pulse rounded-[2rem] bg-base-200"></div>
        </template>

        <template v-else-if="items.length === 0">
          <div class="col-span-full rounded-[2rem] border border-dashed border-base-300 bg-base-100 p-12 text-center opacity-70">
            Здесь пока пусто
          </div>
        </template>

        <article
          v-for="item in items"
          :key="item.id"
          class="flex h-[34rem] flex-col gap-5 overflow-hidden rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-lg"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0">
              <div class="mb-2 flex items-center gap-2">
                <span class="badge badge-outline font-bold">{{ typeLabel }}</span>
                <span v-if="item.is_archived" class="badge badge-info badge-outline font-bold">В архиве</span>
                <span v-if="item.monument_is_orphaned" class="badge badge-secondary badge-outline font-bold">Точка без автора</span>
                <span v-if="item.deleted_at" class="badge badge-error badge-outline font-bold">Удалено</span>
                <span v-else-if="item.is_hidden" class="badge badge-warning badge-outline font-bold">Скрыто</span>
              </div>
              <h3 class="line-clamp-2 text-2xl font-black tracking-tight">{{ itemTitle(item) }}</h3>
              <p class="mt-2 truncate text-xs uppercase tracking-widest opacity-40">{{ itemMeta(item) }}</p>
            </div>
            <span v-if="item.status" class="badge h-8 shrink-0 rounded-full px-4 text-xs font-black tracking-wide" :class="statusClass(item.status)">
              {{ getStatusLabel(item.status) }}
            </span>
          </div>

          <div class="min-h-0 flex-1 overflow-hidden">
            <FormattedText v-if="itemDescription(item)" :text="itemDescription(item)" class="line-clamp-[14] leading-relaxed opacity-80" />
            <p v-else class="opacity-40">Описание не добавлено</p>
          </div>

          <div v-if="item.photos?.length" class="flex gap-2 overflow-x-auto">
            <img
              v-for="(photo, index) in item.photos"
              :key="photo.id"
              :src="'/' + photo.thumbnail_path"
              class="h-20 w-20 cursor-pointer rounded-xl object-cover"
              @click="openGallery(item, Number(index))"
            />
          </div>
          <img
            v-else-if="item.thumbnail"
            :src="'/' + item.thumbnail"
            class="h-20 w-20 cursor-pointer rounded-xl object-cover"
            @click="openGallery(item, 0)"
          />

          <div v-if="item.status === 'resolved' && item.resolution_kind" class="rounded-2xl border border-success/20 bg-success/10 px-4 py-3 text-sm">
            <div class="font-black text-success">{{ resolutionLabel(item.resolution_kind) }}</div>
            <div v-if="item.resolution_comment" class="mt-1 opacity-75">{{ item.resolution_comment }}</div>
          </div>

          <div v-if="item.moderation_comment" class="rounded-2xl border border-error/20 bg-error/10 px-4 py-3 text-sm">
            {{ item.moderation_comment }}
          </div>

          <div class="flex flex-wrap items-center justify-between gap-3 border-t border-base-200 pt-4">
            <span class="text-xs font-bold opacity-40">{{ new Date(item.created_at).toLocaleString() }}</span>
            <div class="flex flex-wrap gap-2">
              <button v-if="activeTab !== 'comments'" class="btn btn-ghost btn-sm font-bold text-primary" @click="editItem(item)">
                <Edit3Icon class="mr-1 h-4 w-4" /> Изменить
              </button>
              <button v-if="activeTab === 'signals'" class="btn btn-ghost btn-sm font-bold text-success" @click="router.push(`/signal-resolve/${item.id}`)">
                <Edit3Icon class="mr-1 h-4 w-4" /> {{ item.status === 'resolved' ? 'Открыть снова' : 'Завершить' }}
              </button>
              <button v-if="activeTab === 'comments'" class="btn btn-ghost btn-sm font-bold text-primary" @click="editItem(item)">
                <Edit3Icon class="mr-1 h-4 w-4" /> Изменить
              </button>
              <button v-if="activeTab === 'comments'" class="btn btn-ghost btn-sm font-bold text-error" @click="deleteComment(item)">
                <Trash2Icon class="mr-1 h-4 w-4" /> Удалить
              </button>
              <button v-if="activeTab === 'monuments'" class="btn btn-ghost btn-sm font-bold text-error" @click="deleteMonument(item)">
                <Trash2Icon class="mr-1 h-4 w-4" /> Удалить
              </button>
              <button v-if="activeTab === 'posts'" class="btn btn-ghost btn-sm font-bold text-error" @click="deletePost(item)">
                <Trash2Icon class="mr-1 h-4 w-4" /> Удалить
              </button>
              <button v-if="activeTab === 'posts' && item.is_archived" class="btn btn-ghost btn-sm font-bold text-success" @click="restoreArchivedPost(item, true)">
                <Edit3Icon class="mr-1 h-4 w-4" /> Вернуть на карту
              </button>
              <button v-if="activeTab === 'posts' && item.is_archived && item.restore_decision_status !== 'declined'" class="btn btn-ghost btn-sm font-bold" @click="restoreArchivedPost(item, false)">
                <Edit3Icon class="mr-1 h-4 w-4" /> Оставить в архиве
              </button>
              <button v-if="activeTab === 'signals'" class="btn btn-ghost btn-sm font-bold text-error" @click="deleteSignal(item)">
                <Trash2Icon class="mr-1 h-4 w-4" /> Удалить
              </button>
              <button class="btn btn-ghost btn-sm font-bold" @click="viewItem(item)">Подробнее</button>
            </div>
          </div>
        </article>
      </div>
    </div>

    <div v-if="showSettings" class="fixed inset-0 z-50 bg-slate-950/45 backdrop-blur-sm" @click.self="showSettings = false">
      <div class="mx-auto flex min-h-full max-w-7xl items-center justify-center px-6 py-8">
        <section class="relative flex max-h-[90vh] w-full max-w-5xl flex-col overflow-hidden rounded-[2rem] border border-base-200 bg-base-100 shadow-2xl">
          <button class="btn btn-circle btn-sm absolute right-5 top-5 z-10 bg-base-100/90 shadow-md" @click="showSettings = false">
            <XIcon class="h-4 w-4" />
          </button>
          <div class="overflow-y-auto px-8 py-8">
            <div class="mb-8">
              <div class="text-xs font-black uppercase tracking-[0.22em] opacity-40">Настройки профиля</div>
              <h3 class="mt-2 text-3xl font-black">Личные данные и уведомления</h3>
            </div>

            <form class="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]" @submit.prevent="saveSettings">
              <section class="space-y-5 rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-sm">
                <div class="text-lg font-black">Основные данные</div>
                <div class="form-control gap-2">
                  <label class="label font-bold">Никнейм</label>
                  <input v-model="settings.username" class="input input-bordered h-14 rounded-2xl text-base font-medium" />
                </div>
                <div class="form-control gap-2">
                  <label class="label font-bold">Регион</label>
                  <RegionSelector v-model="settings.region" />
                </div>
              </section>

              <section class="space-y-5 rounded-[1.75rem] border border-base-200 bg-base-200/50 p-6 shadow-sm">
                <div class="text-lg font-black">Уведомления</div>
                <label class="flex items-start justify-between gap-4 rounded-2xl bg-base-100 px-4 py-4">
                  <span class="font-semibold">О статусе постов</span>
                  <input v-model="settings.notification_settings.post_status" type="checkbox" class="toggle toggle-primary" />
                </label>
                <label class="flex items-start justify-between gap-4 rounded-2xl bg-base-100 px-4 py-4">
                  <span class="font-semibold">О новых сигналах в регионе</span>
                  <input v-model="settings.notification_settings.new_signal_city" type="checkbox" class="toggle toggle-primary" />
                </label>
              </section>

              <section class="space-y-5 rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-sm xl:col-span-2">
                <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                  <div>
                    <div class="text-lg font-black">Смена пароля</div>
                    <p class="mt-1 text-sm opacity-60">Пароль меняется через код на электронной почте. Текущий пароль здесь не требуется.</p>
                  </div>
                  <button type="button" class="btn btn-primary rounded-2xl px-6 font-bold" @click="startPasswordRecovery">
                    Восстановить пароль по коду
                  </button>
                </div>
                <div class="rounded-2xl border border-base-200 bg-base-200/50 px-4 py-4 text-sm leading-6 opacity-75">
                  1. На электронную почту отправляется код подтверждения.
                  <br />
                  2. Код вводится на странице входа.
                  <br />
                  3. После подтверждения можно задать новый пароль.
                </div>
              </section>

              <div class="xl:col-span-2 flex items-center justify-end gap-3 border-t border-base-200 pt-5">
                <button type="button" class="btn btn-ghost" @click="showSettings = false">Отмена</button>
                <button class="btn btn-primary rounded-2xl px-6 font-black" :disabled="saving">
                  <span v-if="saving" class="loading loading-spinner"></span>
                  <span v-else>Сохранить</span>
                </button>
              </div>
            </form>
          </div>
        </section>
      </div>
    </div>

    <PhotoGallery v-model="galleryOpen" :photos="galleryPhotos" :initial-index="galleryIdx" :show-captions="true" />
    <TextEditDialog
      v-model="commentEditOpen"
      title="Редактирование комментария"
      message="Измененный комментарий снова пройдет проверку и при необходимости будет скрыт до модерации."
      label="Новый текст комментария"
      placeholder="Введите обновленный текст"
      :initial-value="commentEditItem?.content || ''"
      :loading="commentEditPending"
      @submit="submitCommentEdit"
    />
    <ConfirmActionDialog
      v-model="confirmOpen"
      :title="confirmState.title"
      :message="confirmState.message"
      :details="confirmState.details"
      :confirm-label="confirmState.confirmLabel"
      :loading="confirmPending"
      :destructive="confirmState.destructive"
      @confirm="runConfirmedAction"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { Edit3Icon, SettingsIcon, Trash2Icon, XIcon } from 'lucide-vue-next';
import api from '../api';
import PhotoGallery from '../components/PhotoGallery.vue';
import FormattedText from '../components/FormattedText.vue';
import RegionSelector from '../components/RegionSelector.vue';
import ConfirmActionDialog from '../components/ConfirmActionDialog.vue';
import TextEditDialog from '../components/TextEditDialog.vue';
import { useToast } from '../composables/useToast';
import { useAuthStore } from '../store/auth';

const auth = useAuthStore();
const router = useRouter();
const toast = useToast();

const activeTab = ref('monuments');
const tabs = [
  { id: 'monuments', name: 'Мои памятники' },
  { id: 'posts', name: 'Мои посты' },
  { id: 'signals', name: 'Мои сигналы' },
  { id: 'comments', name: 'Мои комментарии' },
];

const items = ref<any[]>([]);
const loading = ref(false);
const showSettings = ref(false);
const saving = ref(false);
const galleryOpen = ref(false);
const galleryIdx = ref(0);
const selectedItem = ref<any>(null);
const commentEditOpen = ref(false);
const commentEditPending = ref(false);
const commentEditItem = ref<any | null>(null);
const confirmOpen = ref(false);
const confirmPending = ref(false);
const confirmAction = ref<null | (() => Promise<void>)>(null);
const confirmState = reactive({
  title: 'Подтверждение действия',
  message: '',
  confirmLabel: 'Подтвердить',
  destructive: false,
  details: [] as string[],
});

const settings = reactive({
  username: auth.user?.username || '',
  city: auth.user?.city || '',
  region: auth.user?.region || '',
  notification_settings: {
    post_status: auth.user?.notification_settings?.post_status ?? true,
    new_signal_city: auth.user?.notification_settings?.new_signal_city ?? true,
  },
});

const trustSummary = computed(() => auth.user?.trust_summary || null);

const trustSegments = [
  { key: 'restricted', label: 'Ограниченный', range: '-10 и ниже', bgClass: 'bg-rose-500' },
  { key: 'risky', label: 'Пониженный', range: '-1...-9', bgClass: 'bg-amber-400' },
  { key: 'standard', label: 'Стандартный', range: '0...9', bgClass: 'bg-sky-500' },
  { key: 'trusted', label: 'Высокий', range: '10 и выше', bgClass: 'bg-emerald-500' },
];

const trustMarkerPercent = computed(() => {
  const score = trustSummary.value?.score ?? 0;
  const min = -15;
  const max = 15;
  const clamped = Math.min(max, Math.max(min, score));
  return ((clamped - min) / (max - min)) * 100;
});

const typeLabel = computed(() => {
  if (activeTab.value === 'monuments') return 'Памятник';
  if (activeTab.value === 'posts') return 'Пост';
  if (activeTab.value === 'signals') return 'Сигнал';
  return 'Комментарий';
});

const normalizeImagePath = (path?: string) => {
  if (!path) return undefined;
  return path.startsWith('/') ? path : `/${path}`;
};

const galleryPhotos = computed(() => {
  if (!selectedItem.value) return [];
  const photos = [...(selectedItem.value.photos || [])];
  if (photos.length === 0 && selectedItem.value.thumbnail) {
    photos.push({ thumbnail_path: selectedItem.value.thumbnail, preview_path: selectedItem.value.thumbnail });
  }
  return photos.map((photo: any) => ({
    ...photo,
    file_path: normalizeImagePath(photo.file_path),
    preview_path: normalizeImagePath(photo.preview_path || photo.file_path || photo.thumbnail_path),
    thumbnail_path: normalizeImagePath(photo.thumbnail_path || photo.preview_path || photo.file_path),
    caption: selectedItem.value?.description || selectedItem.value?.content,
  }));
});

const openGallery = (item: any, index: number) => {
  selectedItem.value = item;
  galleryIdx.value = index;
  galleryOpen.value = true;
};

const fetchItems = async () => {
  loading.value = true;
  try {
    const { data } = await api.get(`/me/${activeTab.value}`);
    items.value = data.items || [];
  } catch (error) {
    console.error(error);
    items.value = [];
  } finally {
    loading.value = false;
  }
};

onMounted(fetchItems);
watch(activeTab, fetchItems);

const getStatusLabel = (status: string) => ({
  approved: 'Одобрено',
  confirmed: 'Подтверждено',
  pending: 'На проверке',
  rejected: 'Отклонено',
  resolved: 'Решено',
}[status] || status);

const statusClass = (status: string) => {
  if (status === 'pending') return 'border-0 bg-amber-400 text-amber-950';
  if (status === 'rejected') return 'badge-error';
  if (status === 'approved' || status === 'confirmed' || status === 'resolved') return 'badge-success';
  return 'badge-outline';
};

const itemTitle = (item: any) => {
  if (activeTab.value === 'monuments') return item.name || 'Без названия';
  if (activeTab.value === 'posts') return item.monument_name ? `Пост к памятнику «${item.monument_name}»` : 'Пост';
  if (activeTab.value === 'signals') return item.monument_name || item.region || 'Сигнал';
  return 'Комментарий к сигналу';
};

const itemDescription = (item: any) => {
  if (activeTab.value === 'monuments') return item.properties?.description || '';
  if (activeTab.value === 'comments') return item.content || '';
  return item.description || '';
};

const itemMeta = (item: any) => {
  if (activeTab.value === 'monuments') {
    const parts = [];
    if (item.region) parts.push(item.region);
    if (typeof item.lat === 'number' && typeof item.lon === 'number') parts.push(`${item.lat.toFixed(4)}, ${item.lon.toFixed(4)}`);
    return parts.join(' • ');
  }
  if (activeTab.value === 'posts') return 'Пользовательский пост';
  if (activeTab.value === 'comments') return item.deleted_at ? 'Удалено автором' : item.is_hidden ? 'Скрыто' : 'Комментарий автора';
  return [item.region, item.signal_type].filter(Boolean).join(' • ') || 'Сигнал угрозы';
};

const resolutionLabel = (value?: string) => ({
  successful: 'Устранено результативно',
  partial: 'Частично решено',
  unsuccessful: 'Закрыто без результата',
}[value || ''] || 'Итог не указан');

const saveSettings = async () => {
  saving.value = true;
  try {
    await api.put('/profile', settings);
    await auth.fetchMe();
    showSettings.value = false;
    toast.success('Настройки сохранены');
  } catch {
    toast.error('Не удалось сохранить настройки');
  } finally {
    saving.value = false;
  }
};

const startPasswordRecovery = async () => {
  showSettings.value = false;
  await router.push({ name: 'login', query: { forgot: 'true', email: auth.user?.email || '' } });
};

const viewItem = (item: any) => {
  if (activeTab.value === 'monuments') return router.push(`/monument/${item.id}`);
  if (activeTab.value === 'posts') return router.push(`/monument/${item.monument_id}#post-${item.id}`);
  if (activeTab.value === 'comments') return router.push(`/signal/${item.signal_id}`);
  return router.push(`/signal/${item.id}`);
};

const openConfirmDialog = (options: {
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  details?: string[];
  action: () => Promise<void>;
}) => {
  confirmState.title = options.title;
  confirmState.message = options.message;
  confirmState.confirmLabel = options.confirmLabel;
  confirmState.destructive = !!options.destructive;
  confirmState.details = options.details || [];
  confirmAction.value = options.action;
  confirmOpen.value = true;
};

const runConfirmedAction = async () => {
  if (!confirmAction.value) return;
  confirmPending.value = true;
  try {
    await confirmAction.value();
    confirmOpen.value = false;
  } finally {
    confirmPending.value = false;
    confirmAction.value = null;
  }
};

const editItem = async (item: any) => {
  if (activeTab.value === 'monuments') {
    router.push(`/submission-edit/monument/${item.id}`);
    return;
  }
  if (activeTab.value === 'posts') {
    router.push(`/submission-edit/post/${item.id}`);
    return;
  }
  if (activeTab.value === 'signals') {
    router.push(`/signal-edit/${item.id}`);
    return;
  }
  if (activeTab.value !== 'comments') return;
  commentEditItem.value = item;
  commentEditOpen.value = true;
};

const submitCommentEdit = async (value: string) => {
  if (!commentEditItem.value) return;
  const nextValue = value.trim();
  if (!nextValue || nextValue === commentEditItem.value.content) {
    commentEditOpen.value = false;
    return;
  }
  commentEditPending.value = true;
  try {
    const { data } = await api.put(`/signals/${commentEditItem.value.signal_id}/comments/${commentEditItem.value.id}`, { content: nextValue });
    commentEditItem.value.content = nextValue;
    commentEditItem.value.edited_at = new Date().toISOString();
    commentEditItem.value.is_hidden = !!data?.is_hidden;
    commentEditOpen.value = false;
    toast.success(commentEditItem.value.is_hidden ? 'Комментарий обновлен и скрыт до проверки' : 'Комментарий обновлен');
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось обновить комментарий');
  } finally {
    commentEditPending.value = false;
  }
};

const deletePost = (item: any) => {
  openConfirmDialog({
    title: 'Удаление поста',
    message: 'Пост будет удален вместе с его фотографиями.',
    confirmLabel: 'Удалить пост',
    destructive: true,
    details: ['Действие затрагивает только выбранный пост.'],
    action: async () => {
      await api.delete(`/posts/${item.id}`);
      items.value = items.value.filter((entry) => entry.id !== item.id);
      toast.success('Пост удален');
    },
  });
};

const restoreArchivedPost = (item: any, publish: boolean) => {
  openConfirmDialog({
    title: publish ? 'Возврат поста на карту' : 'Оставить пост в архиве',
    message: publish
      ? 'Пост снова станет доступен в карточке точки и на карте.'
      : 'Пост останется только в архиве профиля и не будет показан публично.',
    confirmLabel: publish ? 'Вернуть пост' : 'Оставить в архиве',
    details: publish
      ? ['На карту вернется только этот пост.']
      : ['Пост сохранится в профиле и его можно будет опубликовать позже.'],
    action: async () => {
      await api.post(`/posts/${item.id}/restore`, { publish });
      toast.success(publish ? 'Пост снова опубликован.' : 'Пост оставлен в архиве.');
      await fetchItems();
    },
  });
};

const deleteMonument = (item: any) => {
  openConfirmDialog({
    title: 'Удаление точки',
    message: 'Точка будет удалена или переведена в безопасный режим без автора, если у нее есть материалы других пользователей.',
    confirmLabel: 'Удалить точку',
    destructive: true,
    details: ['Собственный пост автора точки и его фотографии будут удалены.'],
    action: async () => {
      await api.delete(`/monuments/${item.id}`);
      items.value = items.value.filter((entry) => entry.id !== item.id);
      toast.success('Точка удалена');
    },
  });
};

const deleteComment = (item: any) => {
  openConfirmDialog({
    title: 'Удаление комментария',
    message: 'Комментарий будет скрыт из обсуждения.',
    confirmLabel: 'Удалить комментарий',
    destructive: true,
    action: async () => {
      await api.delete(`/signals/${item.signal_id}/comments/${item.id}`);
      item.deleted_at = new Date().toISOString();
      toast.success('Комментарий удален');
    },
  });
};

const deleteSignal = (item: any) => {
  openConfirmDialog({
    title: 'Удаление сигнала',
    message: 'Сигнал будет удален вместе с фотографиями и связанными комментариями.',
    confirmLabel: 'Удалить сигнал',
    destructive: true,
    action: async () => {
      await api.delete(`/signals/${item.id}`);
      items.value = items.value.filter((entry) => entry.id !== item.id);
      toast.success('Сигнал удален');
    },
  });
};

const translateRole = (role?: string) => ({
  admin: 'Администратор',
  moderator: 'Модератор',
  user: 'Пользователь',
}[role || 'user'] || 'Пользователь');

const formatTrustEventDate = (value: string) => new Date(value).toLocaleString();
</script>
