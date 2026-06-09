<template>
  <div class="mx-auto max-w-7xl space-y-8 p-8">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div>
        <router-link to="/admin" class="btn btn-ghost rounded-2xl px-4 mb-4">Назад в админ-панель</router-link>
        <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Карточка пользователя</div>
        <h1 class="mt-2 text-4xl font-black tracking-tight">{{ user?.username || 'Пользователь' }}</h1>
        <p class="mt-2 text-base opacity-60">{{ user?.email }}</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <select v-model="roleName" class="select select-bordered rounded-2xl" @change="saveRole">
          <option value="user">Пользователь</option>
          <option value="moderator">Модератор</option>
          <option value="admin">Администратор</option>
        </select>
        <button class="btn btn-outline rounded-2xl" @click="toggleBlock" :disabled="!user">
          {{ user?.is_blocked ? 'Снять блок входа' : 'Заблокировать вход' }}
        </button>
      </div>
    </div>

    <div v-if="user" class="grid gap-4 xl:grid-cols-4">
      <div class="rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-md">
        <div class="text-xs font-black uppercase tracking-widest opacity-40">Роль</div>
        <div class="mt-2 text-xl font-black">{{ roleLabel(user.role_name) }}</div>
      </div>
      <div class="rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-md">
        <div class="text-xs font-black uppercase tracking-widest opacity-40">Доверие</div>
        <div class="mt-2 text-xl font-black">{{ user.trust_score }}</div>
        <div class="mt-2"><span class="badge font-bold" :class="trustBadgeClass(user.trust_summary?.level)">{{ trustLevelLabel(user.trust_summary?.level) }}</span></div>
      </div>
      <div class="rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-md">
        <div class="text-xs font-black uppercase tracking-widest opacity-40">Статус</div>
        <div class="mt-2 text-xl font-black">{{ statusLabel(user.status) }}</div>
      </div>
      <div class="rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-md">
        <div class="text-xs font-black uppercase tracking-widest opacity-40">AI-скрытия за 24 часа</div>
        <div class="mt-2 text-xl font-black">{{ user.ai_hidden_comments_24h ?? 0 }}</div>
      </div>
    </div>

    <div v-if="user" class="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(360px,0.9fr)]">
      <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Материалы пользователя</div>
            <div class="mt-1 text-sm opacity-60">Единый список объектов с типами, быстрыми ссылками и действиями.</div>
          </div>
          <div class="flex flex-wrap gap-2">
            <select v-model="contentFilter" class="select select-sm rounded-xl">
              <option value="all">Все объекты</option>
              <option value="monument">Точки</option>
              <option value="post">Посты</option>
              <option value="signal">Сигналы</option>
            </select>
            <input v-model="contentQuery" class="input input-sm input-bordered w-64 rounded-xl" placeholder="Поиск по названию или описанию" />
          </div>
        </div>

        <div class="mt-5 space-y-3">
          <div v-for="item in filteredItems" :key="`${item.type}-${item.id}`" class="rounded-[1.5rem] border border-base-200 bg-base-50 px-5 py-4">
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="badge font-bold" :class="item.typeBadge">{{ item.typeLabel }}</span>
                  <span v-if="item.extraBadge" class="badge badge-outline font-bold">{{ item.extraBadge }}</span>
                  <span v-if="item.statusLabel" class="badge badge-ghost font-bold">{{ item.statusLabel }}</span>
                </div>
                <div class="mt-3 text-lg font-black leading-tight">{{ item.title }}</div>
                <div v-if="item.subtitle" class="mt-1 text-sm font-semibold opacity-55">{{ item.subtitle }}</div>
                <div v-if="item.preview" class="mt-3 line-clamp-3 text-sm leading-6 opacity-75">{{ item.preview }}</div>
                <div class="mt-3 flex flex-wrap gap-2">
                  <router-link class="btn btn-ghost btn-sm rounded-xl" :to="item.link">Открыть объект</router-link>
                  <button class="btn btn-ghost btn-sm rounded-xl" @click="toggleExpanded(item.key)">
                    {{ expanded[item.key] ? 'Свернуть' : 'Развернуть' }}
                  </button>
                  <button class="btn btn-ghost btn-sm rounded-xl text-error" @click="deleteEntity(item)">Удалить</button>
                </div>
              </div>
              <div class="text-xs font-semibold opacity-45">{{ formatDate(item.createdAt) }}</div>
            </div>

            <div v-if="expanded[item.key]" class="mt-4 rounded-2xl bg-base-200/60 p-4 text-sm">
              <div><span class="font-black">Ссылка:</span> <span class="break-all">{{ item.link }}</span></div>
              <div v-if="item.region" class="mt-2"><span class="font-black">Регион:</span> {{ item.region }}</div>
              <div v-if="item.meta" class="mt-2"><span class="font-black">Дополнительно:</span> {{ item.meta }}</div>
            </div>
          </div>

          <div v-if="!filteredItems.length" class="rounded-2xl bg-base-200/60 px-4 py-6 text-sm font-semibold opacity-60">
            По текущим фильтрам ничего не найдено.
          </div>
        </div>
      </section>

      <div class="space-y-6">
        <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
          <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Ограничения и доверие</div>
          <p class="mt-3 text-sm leading-6 opacity-70">{{ user.restriction_summary?.message || 'Активных ограничений нет.' }}</p>
          <div v-if="user.trust_summary?.restrictions?.length" class="mt-4 flex flex-wrap gap-2">
            <span v-for="restriction in user.trust_summary.restrictions" :key="restriction" class="badge badge-outline h-auto px-4 py-3 text-left">
              {{ restriction }}
            </span>
          </div>
        </section>

        <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
          <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Новое ограничение</div>
          <form class="mt-4 grid gap-3" @submit.prevent="createSanction">
            <select v-model="sanctionForm.scopeMode" class="select select-bordered rounded-xl">
              <option value="comment">Запрет комментариев</option>
              <option value="content">Запрет публикации и правок</option>
              <option value="login">Блокировка входа</option>
            </select>
            <select v-model="sanctionForm.duration" class="select select-bordered rounded-xl">
              <option value="6h">6 часов</option>
              <option value="24h">24 часа</option>
              <option value="7d">7 дней</option>
              <option value="permanent">Бессрочно</option>
            </select>
            <input v-model="sanctionForm.reason_text" class="input input-bordered rounded-xl" placeholder="Причина ограничения" />
            <button class="btn btn-primary rounded-xl">Выдать ограничение</button>
          </form>
        </section>

        <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
          <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Журнал пользователя</div>
          <div class="mt-4 space-y-3">
            <div v-for="entry in userLogs" :key="entry.id" class="rounded-2xl border border-base-200 p-4">
              <div class="flex flex-wrap items-center gap-2 text-[11px] font-black uppercase tracking-wide opacity-40">
                <span>{{ entry.entity_type }}</span>
                <span>{{ entry.action }}</span>
                <span>{{ entry.result }}</span>
              </div>
              <div class="mt-2 font-semibold">{{ entry.message }}</div>
              <div class="mt-1 text-xs opacity-50">{{ formatDate(entry.created_at) }}</div>
            </div>
            <div v-if="!userLogs.length" class="rounded-2xl bg-base-200/60 px-4 py-5 text-sm font-semibold opacity-60">
              Подробных записей по этому пользователю пока нет.
            </div>
          </div>
        </section>
      </div>
    </div>
    <ConfirmActionDialog
    v-model="confirmOpen"
    :title="confirmState.title"
    :message="confirmState.message"
    :details="confirmState.details"
    :confirm-label="confirmState.confirmLabel"
    :loading="confirmPending"
    destructive
    @confirm="runConfirmedAction"
  />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute } from 'vue-router';
import api from '../api';
import ConfirmActionDialog from '../components/ConfirmActionDialog.vue';
import { useToast } from '../composables/useToast';

const route = useRoute();
const toast = useToast();

const user = ref<any | null>(null);
const userLogs = ref<any[]>([]);
const roleName = ref('user');
const contentFilter = ref<'all' | 'monument' | 'post' | 'signal'>('all');
const contentQuery = ref('');
const expanded = reactive<Record<string, boolean>>({});
const confirmOpen = ref(false);
const confirmPending = ref(false);
const confirmAction = ref<null | (() => Promise<void>)>(null);
const confirmState = reactive({
  title: 'Подтверждение действия',
  message: '',
  confirmLabel: 'Подтвердить',
  details: [] as string[],
});

const sanctionForm = reactive({
  scopeMode: 'comment',
  duration: '6h',
  reason_text: '',
});

const roleLabel = (value?: string) => {
  if (value === 'admin') return 'Администратор';
  if (value === 'moderator') return 'Модератор';
  return 'Пользователь';
};

const trustLevelLabel = (level?: string) => {
  if (level === 'trusted') return 'Высокий';
  if (level === 'risky') return 'Пониженный';
  if (level === 'restricted') return 'Ограниченный';
  return 'Стандартный';
};

const trustBadgeClass = (level?: string) => {
  if (level === 'trusted') return 'badge-success';
  if (level === 'risky') return 'badge-warning';
  if (level === 'restricted') return 'badge-error';
  return 'badge-info';
};

const statusLabel = (status?: string) => {
  if (status === 'login_banned') return 'Вход заблокирован';
  if (status === 'restricted') return 'Ограничен';
  return 'Активен';
};

const formatDate = (value?: string | null) => {
  if (!value) return 'Без срока';
  return new Date(value).toLocaleString();
};

const normalize = (value?: string | null) => String(value || '').toLowerCase();

const items = computed(() => {
  if (!user.value) return [];
  const monuments = (user.value.monuments || []).map((item: any) => ({
    key: `monument-${item.id}`,
    id: item.id,
    type: 'monument',
    typeLabel: 'Точка',
    typeBadge: 'badge-primary',
    extraBadge: item.is_orphaned ? 'Без автора' : '',
    statusLabel: item.status,
    title: item.name,
    subtitle: item.region || 'Регион не указан',
    preview: item.properties?.description || '',
    link: `/monument/${item.id}`,
    createdAt: item.created_at,
    region: item.region,
    meta: `Координаты: ${item.lat}, ${item.lon}`,
  }));
  const posts = (user.value.posts || []).map((item: any) => ({
    key: `post-${item.id}`,
    id: item.id,
    type: 'post',
    typeLabel: 'Пост',
    typeBadge: 'badge-secondary',
    extraBadge: item.is_archived ? 'В архиве' : item.monument_is_orphaned ? 'Точка без автора' : '',
    statusLabel: item.status,
    title: item.monument_name || 'Пост без названия точки',
    subtitle: 'Публикация к карточке памятника',
    preview: item.description || '',
    link: `/monument/${item.monument_id}#post-${item.id}`,
    createdAt: item.created_at,
    region: item.region,
    meta: item.moderation_comment || '',
  }));
  const signals = (user.value.signals || []).map((item: any) => ({
    key: `signal-${item.id}`,
    id: item.id,
    type: 'signal',
    typeLabel: 'Сигнал',
    typeBadge: 'badge-accent',
    extraBadge: item.resolution_kind ? resolutionLabel(item.resolution_kind) : '',
    statusLabel: item.status,
    title: item.monument_name || item.region || 'Сигнал',
    subtitle: item.signal_type,
    preview: item.description || '',
    link: `/signal/${item.id}`,
    createdAt: item.created_at,
    region: item.region,
    meta: item.resolution_comment || '',
  }));
  return [...monuments, ...posts, ...signals].sort((a, b) => +new Date(b.createdAt) - +new Date(a.createdAt));
});

const filteredItems = computed(() => {
  return items.value.filter((item) => {
    if (contentFilter.value !== 'all' && item.type !== contentFilter.value) return false;
    const q = normalize(contentQuery.value);
    if (!q) return true;
    return [item.title, item.subtitle, item.preview, item.region, item.meta].some((part) => normalize(part).includes(q));
  });
});

const resolutionLabel = (value?: string) => {
  if (value === 'successful') return 'Устранен';
  if (value === 'partial') return 'Частично решен';
  if (value === 'unsuccessful') return 'Закрыт без результата';
  return '';
};

const toggleExpanded = (key: string) => {
  expanded[key] = !expanded[key];
};

const fetchUser = async () => {
  const { data } = await api.get(`/admin/users/${route.params.id}`);
  user.value = data;
  roleName.value = data.role_name || 'user';
};

const fetchLogs = async () => {
  const { data } = await api.get('/admin/logs', {
    params: {
      target_user_id: route.params.id,
      limit: 80,
    },
  });
  userLogs.value = data.items || [];
};

const saveRole = async () => {
  if (!user.value) return;
  await api.post(`/admin/users/${user.value.id}/role`, { role_name: roleName.value });
  toast.success('Роль обновлена');
  await fetchUser();
  await fetchLogs();
};

const toggleBlock = async () => {
  if (!user.value) return;
  await api.post(`/admin/users/${user.value.id}/block`, { blocked: !user.value.is_blocked });
  toast.success(user.value.is_blocked ? 'Блокировка снята' : 'Вход заблокирован');
  await fetchUser();
  await fetchLogs();
};

const resolveScopes = (scopeMode: string) => {
  if (scopeMode === 'login') return ['login'];
  if (scopeMode === 'content') return ['comment_write', 'content_create', 'content_edit'];
  return ['comment_write'];
};

const resolveEndsAt = (duration: string) => {
  if (duration === 'permanent') return null;
  const date = new Date();
  if (duration === '6h') date.setHours(date.getHours() + 6);
  if (duration === '24h') date.setHours(date.getHours() + 24);
  if (duration === '7d') date.setDate(date.getDate() + 7);
  return date.toISOString();
};

const createSanction = async () => {
  if (!user.value) return;
  await api.post(`/admin/users/${user.value.id}/sanctions`, {
    reason_code: sanctionForm.scopeMode === 'login' ? 'manual_login_ban' : 'manual_restriction',
    reason_text: sanctionForm.reason_text,
    scopes: resolveScopes(sanctionForm.scopeMode),
    ends_at: resolveEndsAt(sanctionForm.duration),
  });
  sanctionForm.reason_text = '';
  toast.success('Ограничение выдано');
  await fetchUser();
  await fetchLogs();
};

const deleteEntity = async (item: any) => {
  const labels: Record<string, string> = {
    monument: 'точку',
    post: 'пост',
    signal: 'сигнал',
  };
  const endpoint = item.type === 'monument' ? 'monuments' : item.type === 'post' ? 'posts' : 'signals';
  confirmState.title = 'Удаление объекта';
  confirmState.message = `Будет удален ${labels[item.type]}, а связанный пользовательский контент будет обработан по правилам системы.`;
  confirmState.confirmLabel = `Удалить ${labels[item.type]}`;
  confirmState.details = ['Действие фиксируется в журнале системы.'];
  confirmAction.value = async () => {
    await api.post(`/admin/${endpoint}/${item.id}/delete`, { confirm: true });
    toast.success('Объект удален');
    await Promise.all([fetchUser(), fetchLogs()]);
  };
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

onMounted(async () => {
  await Promise.all([fetchUser(), fetchLogs()]);
});
</script>

