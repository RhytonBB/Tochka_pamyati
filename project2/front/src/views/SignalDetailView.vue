<template>
  <div class="min-h-screen bg-base-200/30 pb-20 pt-10">
    <div class="mx-auto max-w-5xl space-y-8 px-4 sm:px-6 lg:px-8">
      <div v-if="loading" class="space-y-6">
        <div class="h-12 w-1/3 animate-pulse rounded-2xl bg-base-200"></div>
        <div class="h-64 animate-pulse rounded-3xl bg-base-200"></div>
      </div>

      <template v-else-if="detail">
        <button class="btn btn-ghost btn-sm font-bold opacity-60 hover:opacity-100" @click="$router.back()">
          &larr; Назад
        </button>

        <section class="rounded-[2.5rem] border border-base-200 bg-base-100 p-8 shadow-xl">
          <div class="mb-6 flex flex-wrap items-center gap-3">
            <div class="badge rounded-xl px-4 py-3 font-black uppercase tracking-wider" :class="detail.signal.status === 'pending' ? 'badge-neutral' : urgencyColor(detail.signal.urgency)">
              {{ detail.signal.status === 'pending' ? 'Срочность уточняется' : `Угроза: ${urgencyLabel(detail.signal.urgency)}` }}
            </div>
            <div class="badge badge-neutral rounded-xl py-3 text-[10px] font-bold uppercase tracking-wider">
              {{ formatStatus(detail.signal.status) }}
            </div>
            <div class="ml-auto text-sm font-bold opacity-40">
              {{ new Date(detail.signal.created_at).toLocaleString() }}
            </div>
            <div v-if="auth.isAuthenticated" class="dropdown dropdown-end">
              <button tabindex="0" class="btn btn-ghost btn-sm btn-circle">
                <MoreHorizontalIcon class="h-4 w-4 opacity-60" />
              </button>
              <ul tabindex="0" class="dropdown-content menu menu-sm z-[2] mt-2 w-48 rounded-2xl border border-base-200 bg-base-100 p-2 shadow-xl">
                <li><button @click="signalReportOpen = true">Пожаловаться</button></li>
              </ul>
            </div>
          </div>

          <h1 class="mb-4 text-3xl font-black tracking-tight">{{ signalTypeLabel(detail.signal.signal_type) }}</h1>
          <FormattedText :text="detail.signal.description" class="text-lg font-medium leading-relaxed opacity-80" />

          <div class="mt-6 flex items-center gap-3">
            <button class="btn btn-ghost btn-circle hover:bg-primary/10" :disabled="supportPending" @click="toggleSupport">
              <svg viewBox="0 0 24 24" class="h-6 w-6 transition-all" :class="detail.signal.is_supported ? 'fill-current text-primary scale-110' : 'opacity-40'">
                <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z" fill="currentColor" />
              </svg>
            </button>
            <span class="text-sm font-bold opacity-50">{{ detail.signal.support_count || 0 }} поддержали</span>
          </div>

          <div v-if="isOwnSignal" class="mt-6 rounded-[1.5rem] border border-base-200 bg-base-200/60 p-4">
            <div class="mb-3 text-xs font-black uppercase tracking-wider opacity-45">Управление сигналом</div>
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-sm rounded-xl font-bold" @click="router.push(`/signal-edit/${detail.signal.id}`)">Изменить</button>
              <button
                class="btn btn-sm rounded-xl font-bold"
                :class="detail.signal.status === 'resolved' ? 'btn-outline' : 'btn-success text-white'"
                @click="router.push(`/signal-resolve/${detail.signal.id}`)"
              >
                {{ detail.signal.status === 'resolved' ? 'Вернуть в активные' : 'Отметить как устраненный' }}
              </button>
              <button class="btn btn-outline btn-error btn-sm rounded-xl font-bold" @click="confirmDeleteOpen = true">Удалить</button>
            </div>
          </div>

          <div v-if="detail.signal.status === 'resolved'" class="mt-6 rounded-[1.5rem] border border-success/20 bg-success/10 p-5">
            <div class="text-xs font-black uppercase tracking-wider text-success/80">Итог завершения</div>
            <div class="mt-2 text-lg font-black text-success">{{ resolutionLabel(detail.signal.resolution_kind) }}</div>
            <p v-if="detail.signal.resolution_comment" class="mt-2 text-sm font-medium leading-relaxed opacity-75">
              {{ detail.signal.resolution_comment }}
            </p>
          </div>

          <div v-if="detail.photos?.length > 0" class="mt-8 grid grid-cols-2 gap-4 md:grid-cols-4">
            <div
              v-for="(photo, index) in detail.photos"
              :key="photo.id"
              class="group/img aspect-video cursor-pointer overflow-hidden rounded-2xl shadow-md sm:aspect-square"
              @click="openGallery(index)"
            >
              <img :src="filterPath(photo.preview_path || photo.file_path || photo.thumbnail_path)" class="h-full w-full object-cover transition-transform duration-500 group-hover/img:scale-110" />
            </div>
          </div>

          <div v-if="detail.signal.region || detail.signal.monument_name" class="mt-8 flex flex-wrap gap-3 border-t border-base-200 pt-6 text-sm font-bold opacity-60">
            <span v-if="detail.signal.region" class="rounded-lg bg-base-200 px-3 py-1">Регион: {{ detail.signal.region }}</span>
            <span v-if="detail.signal.monument_name" class="rounded-lg bg-base-200 px-3 py-1">Памятник: {{ detail.signal.monument_name }}</span>
            <router-link v-if="detail.signal.monument_id" :to="'/monument/' + detail.signal.monument_id" class="text-primary hover:underline">
              Перейти к памятнику &rarr;
            </router-link>
          </div>
        </section>

        <section class="rounded-[2.5rem] border border-base-200 bg-base-100 p-8 shadow-xl">
          <h2 class="mb-6 text-2xl font-black">Комментарии ({{ detail.comments?.length ?? 0 }})</h2>

          <form v-if="auth.isAuthenticated" class="mb-10" @submit.prevent="submitRootComment">
            <textarea v-model="newComment" class="textarea h-24 w-full resize-none rounded-2xl border-none bg-base-200 p-4 font-medium shadow-inner focus:ring-2 ring-primary" placeholder="Оставьте комментарий или уточнение..." />
            <div class="mt-3 flex justify-end">
              <button class="btn btn-primary rounded-xl px-8 font-bold" :disabled="committingRoot">
                <span v-if="committingRoot" class="loading loading-spinner"></span>
                <span v-else>Отправить</span>
              </button>
            </div>
          </form>

          <div v-else class="mb-10 rounded-2xl bg-base-200 p-6 text-center font-bold opacity-70">
            <router-link to="/login" class="text-primary hover:underline">Войдите</router-link>, чтобы оставлять комментарии
          </div>

          <div class="space-y-6">
            <CommentThread v-for="node in commentTree" :key="node.comment.id" :comment="node.comment" :replies="node.replies" @reply="handleReply" />
            <div v-if="commentTree.length === 0" class="py-10 text-center font-bold opacity-40">
              Пока нет комментариев.
            </div>
          </div>
        </section>
      </template>

      <div v-else class="py-20 text-center">
        <h2 class="text-2xl font-black opacity-50">Сигнал не найден</h2>
      </div>
    </div>

    <PhotoGallery
      v-model="galleryOpen"
      :photos="galleryPhotos"
      :initial-index="galleryPhotoIdx"
      :show-captions="false"
      :allow-report="auth.isAuthenticated"
      :report-subject="detail ? `Фотография сигнала ${signalTypeLabel(detail.signal.signal_type)}` : 'Фотография сигнала'"
      :report-duplicate-seed="detail?.signal.monument_name || ''"
    />

    <ReportDialog
      v-model="signalReportOpen"
      entity-type="signal"
      :entity-id="detail?.signal.id || ''"
      :subject="detail ? signalTypeLabel(detail.signal.signal_type) : ''"
      :duplicate-seed="detail?.signal.monument_name || ''"
    />

    <ConfirmActionDialog
      v-model="confirmDeleteOpen"
      title="Удаление сигнала"
      message="Сигнал будет удален вместе с прикрепленными фотографиями и связанными комментариями."
      :details="[
        'Действие применяется только к этому сигналу.',
        'После удаления сигнал исчезнет из раздела защиты и из профиля автора.'
      ]"
      confirm-label="Удалить сигнал"
      :loading="actionPending"
      destructive
      @confirm="deleteOwnSignal"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { MoreHorizontalIcon } from 'lucide-vue-next';
import api from '../api';
import type { SignalComment, SignalDetail } from '../types';
import { useAuthStore } from '../store/auth';
import { useToast } from '../composables/useToast';
import PhotoGallery from '../components/PhotoGallery.vue';
import CommentThread from '../components/CommentThread.vue';
import FormattedText from '../components/FormattedText.vue';
import ReportDialog from '../components/ReportDialog.vue';
import ConfirmActionDialog from '../components/ConfirmActionDialog.vue';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const loading = ref(true);
const detail = ref<SignalDetail | null>(null);
const supportPending = ref(false);
const actionPending = ref(false);
const newComment = ref('');
const committingRoot = ref(false);
const signalReportOpen = ref(false);
const confirmDeleteOpen = ref(false);
const galleryOpen = ref(false);
const galleryPhotoIdx = ref(0);

const fetchDetail = async () => {
  loading.value = true;
  try {
    const { data } = await api.get(`/signals/${route.params.id}`);
    detail.value = data;
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  void fetchDetail();
});

const urgencyColor = (urgency: string) => {
  if (urgency === 'high') return 'badge-error';
  if (urgency === 'medium') return 'badge-warning';
  return 'badge-info';
};

const urgencyLabel = (urgency: string) => {
  if (urgency === 'high') return 'Высокая';
  if (urgency === 'medium') return 'Средняя';
  return 'Низкая';
};

const signalTypeLabel = (value: string) => ({
  demolition: 'Риск сноса',
  vandalism: 'Вандализм или повреждение',
  poor_condition: 'Плохое состояние памятника',
  trash: 'Захламление территории',
  unsafe_work: 'Подозрительные работы рядом',
  other: 'Другая угроза',
  neglect: 'Плохое состояние памятника',
  damage: 'Вандализм или повреждение',
  reconstruction: 'Подозрительные работы рядом',
}[value] || value);

const resolutionLabel = (value?: string) => ({
  successful: 'Устранено результативно',
  partial: 'Частично решено',
  unsuccessful: 'Закрыто без результата',
}[value || ''] || 'Итог не указан');

const formatStatus = (status: string) => ({
  pending: 'На модерации',
  confirmed: 'Опубликовано',
  resolved: 'Решено',
  rejected: 'Отклонено',
}[status] || status);

const filterPath = (path?: string) => (path ? (path.startsWith('/') ? path : `/${path}`) : undefined);

const galleryPhotos = computed(() => {
  if (!detail.value?.photos) return [];
  return detail.value.photos.map((photo) => ({
    file_path: filterPath(photo.file_path || photo.preview_path),
    preview_path: filterPath(photo.preview_path),
  }));
});

const openGallery = (index: number) => {
  galleryPhotoIdx.value = index;
  galleryOpen.value = true;
};

const isOwnSignal = computed(() => !!detail.value?.signal?.author_id && detail.value.signal.author_id === auth.user?.id);

const deleteOwnSignal = async () => {
  if (!detail.value) return;
  actionPending.value = true;
  try {
    await api.delete(`/signals/${detail.value.signal.id}`);
    confirmDeleteOpen.value = false;
    toast.success('Сигнал удален');
    await router.push('/signals');
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось удалить сигнал');
  } finally {
    actionPending.value = false;
  }
};

interface CommentNode {
  comment: SignalComment;
  replies: CommentNode[];
}

const buildTree = (comments: SignalComment[]) => {
  const map: Record<string, CommentNode> = {};
  const roots: CommentNode[] = [];

  comments.forEach((comment) => {
    map[comment.id] = { comment, replies: [] };
  });

  comments.forEach((comment) => {
    if (comment.parent_id && map[comment.parent_id]) {
      map[comment.parent_id].replies.push(map[comment.id]);
    } else {
      roots.push(map[comment.id]);
    }
  });

  return roots;
};

const commentTree = computed(() => {
  if (!detail.value?.comments) return [];
  return buildTree(detail.value.comments);
});

const submitRootComment = async () => {
  if (!newComment.value.trim()) return;
  committingRoot.value = true;
  try {
    const { data } = await api.post(`/signals/${route.params.id}/comments`, { content: newComment.value });
    const addedComment: SignalComment = {
      id: data.comment_id,
      signal_id: String(route.params.id),
      author_id: auth.user?.id || '',
      author_name: auth.user?.username || 'Вы',
      content: newComment.value,
      is_hidden: data.is_hidden,
      created_at: new Date().toISOString(),
    };
    detail.value?.comments.push(addedComment);
    newComment.value = '';
    toast.success(data.is_hidden ? 'Комментарий отправлен и скрыт до проверки' : 'Комментарий отправлен');
  } catch {
    toast.error('Ошибка при отправке комментария');
  } finally {
    committingRoot.value = false;
  }
};

const handleReply = async (parentId: string, content: string, done: () => void) => {
  try {
    const { data } = await api.post(`/signals/${route.params.id}/comments`, {
      parent_id: parentId,
      content,
    });
    const addedComment: SignalComment = {
      id: data.comment_id,
      signal_id: String(route.params.id),
      author_id: auth.user?.id || '',
      author_name: auth.user?.username || 'Вы',
      parent_id: parentId,
      content,
      is_hidden: data.is_hidden,
      created_at: new Date().toISOString(),
    };
    detail.value?.comments.push(addedComment);
    toast.success(data.is_hidden ? 'Ответ отправлен и скрыт до проверки' : 'Ответ добавлен');
    done();
  } catch {
    toast.error('Не удалось отправить ответ');
  }
};

const toggleSupport = async () => {
  if (!auth.isAuthenticated) {
    router.push('/login');
    return;
  }
  if (!detail.value || supportPending.value) return;

  supportPending.value = true;
  try {
    if (detail.value.signal.is_supported) {
      await api.delete(`/signals/${route.params.id}/support`);
      detail.value.signal.is_supported = false;
      detail.value.signal.support_count = Math.max(0, (detail.value.signal.support_count || 0) - 1);
      toast.success('Поддержка снята');
    } else {
      await api.post(`/signals/${route.params.id}/support`);
      detail.value.signal.is_supported = true;
      detail.value.signal.support_count = (detail.value.signal.support_count || 0) + 1;
      toast.success('Сигнал поддержан');
    }
  } catch {
    toast.error('Не удалось обновить поддержку');
  } finally {
    supportPending.value = false;
  }
};
</script>
