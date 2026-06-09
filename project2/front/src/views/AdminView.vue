<template>
  <div class="mx-auto max-w-7xl space-y-8 p-8">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
      <div>
        <div class="mb-3 badge badge-primary badge-outline font-bold">Админ-панель</div>
        <h1 class="text-4xl font-black tracking-tight">Пользователи и ограничения</h1>
        <p class="mt-2 opacity-60">Роли, временные ограничения, история санкций и статистика по материалам.</p>
      </div>
      <div class="flex flex-wrap gap-3">
        <input v-model="searchQuery" class="input input-bordered w-72 rounded-2xl" placeholder="Поиск по нику или email" @input="fetchUsers" />
        <button class="btn btn-outline rounded-2xl" @click="exportData('csv')">CSV</button>
        <button class="btn btn-outline rounded-2xl" @click="exportData('geojson')">GeoJSON</button>
      </div>
    </div>

    <div class="grid gap-4 md:grid-cols-4">
      <div v-for="stat in stats" :key="stat.label" class="rounded-[1.75rem] border border-base-200 bg-base-100 p-6 shadow-md">
        <div class="text-xs font-bold uppercase tracking-widest opacity-40">{{ stat.label }}</div>
        <div class="mt-2 text-4xl font-black">{{ stat.value }}</div>
      </div>
    </div>

    <div class="grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(340px,0.8fr)]">
      <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
        <div class="mb-4 flex items-center justify-between gap-3">
          <div>
            <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Памятники по регионам</div>
            <div class="mt-1 text-sm opacity-60">Показывается география сохраненных памятников.</div>
          </div>
          <button v-if="monumentsByRegion.length > visibleRegions.length" class="btn btn-ghost btn-sm rounded-xl font-bold" @click="showAllRegions = !showAllRegions">
            {{ showAllRegions ? 'Свернуть' : 'Показать все' }}
          </button>
        </div>
        <div v-if="visibleRegions.length" class="space-y-3">
          <div v-for="entry in visibleRegions" :key="entry.region" class="flex items-center justify-between rounded-2xl bg-base-200/60 px-4 py-3">
            <div class="min-w-0 truncate font-semibold">{{ entry.region }}</div>
            <div class="ml-4 rounded-full bg-base-100 px-3 py-1 text-sm font-black">{{ entry.count }}</div>
          </div>
        </div>
        <div v-else class="rounded-2xl bg-base-200/60 px-4 py-6 text-sm font-semibold opacity-60">
          Данных по регионам пока нет.
        </div>
      </section>

      <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
        <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Итоги завершения сигналов</div>
        <div class="mt-4 space-y-3">
          <div v-for="entry in signalResolutionStats" :key="entry.label" class="flex items-center justify-between rounded-2xl bg-base-200/60 px-4 py-3">
            <div class="font-semibold">{{ entry.label }}</div>
            <div class="rounded-full bg-base-100 px-3 py-1 text-sm font-black">{{ entry.value }}</div>
          </div>
        </div>
      </section>
    </div>

    <div class="overflow-hidden rounded-[2rem] border border-base-200 bg-base-100 shadow-xl">
      <div class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr>
              <th>Пользователь</th>
              <th>Роль</th>
              <th>Доверие</th>
              <th>Статус</th>
              <th>AI-комм. 24ч</th>
              <th>Ограничения</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td>
                <div class="font-bold">{{ user.username }}</div>
                <div class="text-xs opacity-50">{{ user.email }}</div>
              </td>
              <td>
                <select class="select select-sm rounded-xl" :value="user.role_name" @change="updateRole(user, ($event.target as HTMLSelectElement).value)">
                  <option value="user">Пользователь</option>
                  <option value="moderator">Модератор</option>
                  <option value="admin">Администратор</option>
                </select>
              </td>
              <td>
                <div class="font-black">{{ user.trust_score }}</div>
                <div class="mt-1">
                  <span class="badge badge-sm font-bold" :class="trustBadgeClass(user.trust_summary?.level)">
                    {{ trustLevelLabel(user.trust_summary?.level) }}
                  </span>
                </div>
              </td>
              <td>
                <span class="badge" :class="user.status === 'login_banned' ? 'badge-error' : user.status === 'restricted' ? 'badge-warning' : 'badge-success'">
                  {{ statusLabel(user.status) }}
                </span>
              </td>
              <td>{{ user.ai_hidden_comments_24h ?? 0 }}</td>
              <td>
                <div class="max-w-xs text-xs opacity-70">
                  {{ user.restriction_summary?.message || 'Нет' }}
                </div>
              </td>
              <td class="text-right">
                <div class="flex flex-wrap justify-end gap-2">
                  <button class="btn btn-ghost btn-sm rounded-xl" @click="selectUser(user)">Подробнее</button>
                  <router-link class="btn btn-ghost btn-sm rounded-xl" :to="`/admin/users/${user.id}`">Подробнее</router-link>
                  <button class="btn btn-ghost btn-sm rounded-xl text-error" @click="toggleBlock(user)">
                    {{ user.is_blocked ? 'Снять блок' : 'Блок входа' }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <section class="rounded-[2rem] border border-base-200 bg-base-100 p-6 shadow-xl">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="text-sm font-black uppercase tracking-[0.2em] opacity-45">Журнал событий</div>
          <div class="mt-1 text-sm opacity-60">Последние важные действия по пользователям, объектам и внутренним процессам системы.</div>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-ghost btn-sm rounded-xl font-bold" @click="fetchLogs">Обновить</button>
          <button class="btn btn-outline btn-sm rounded-xl font-bold" @click="logsExpanded = !logsExpanded">
            {{ logsExpanded ? 'Свернуть журнал' : 'Развернуть журнал' }}
          </button>
        </div>
      </div>

      <div v-if="logsExpanded" class="mt-5">
        <div v-if="systemLogs.length" class="space-y-3">
          <div v-for="entry in systemLogs" :key="entry.id" class="rounded-2xl border border-base-200 bg-base-50 px-4 py-3">
            <div class="flex flex-wrap items-center gap-2 text-xs font-bold uppercase tracking-wide opacity-45">
              <span>{{ entry.entity_type || 'system' }}</span>
              <span>{{ entry.action }}</span>
              <span>{{ entry.result }}</span>
            </div>
            <div class="mt-2 font-semibold">{{ entry.message }}</div>
            <div class="mt-1 text-xs opacity-55">{{ formatDate(entry.created_at) }}</div>
          </div>
        </div>
        <div v-else class="rounded-2xl bg-base-200/60 px-4 py-6 text-sm font-semibold opacity-60">
          Журнал пока пуст. Здесь появятся создание материалов, удаления, решения по жалобам и другие важные события.
        </div>
      </div>
    </section>

    <dialog v-if="false && selectedUser" class="modal modal-open">
      <div class="modal-box max-w-4xl rounded-[2rem]">
        <button class="btn btn-circle btn-sm absolute right-5 top-5" @click="selectedUser = null">
          <XIcon class="h-4 w-4" />
        </button>
        <h3 class="mb-2 text-2xl font-black">{{ selectedUser.username }}</h3>
        <p class="mb-6 opacity-60">{{ selectedUser.email }}</p>

        <div class="mb-6 rounded-2xl bg-base-200/60 p-4">
          <div class="mb-2 font-bold">Активные ограничения</div>
          <p class="text-sm opacity-70">{{ selectedUser.restriction_summary?.message || 'Ограничений нет' }}</p>
        </div>

        <div v-if="selectedUser.trust_summary" class="mb-6 rounded-[1.75rem] border border-base-200 bg-base-100/90 p-5">
          <div class="flex flex-wrap items-center gap-3">
            <div class="text-lg font-black">Доверие: {{ selectedUser.trust_summary.score }}</div>
            <span class="badge font-bold" :class="trustBadgeClass(selectedUser.trust_summary.level)">
              {{ trustLevelLabel(selectedUser.trust_summary.level) }}
            </span>
          </div>
          <p class="mt-3 text-sm opacity-70">{{ selectedUser.trust_summary.message }}</p>
          <div v-if="selectedUser.trust_summary.restrictions?.length" class="mt-4 flex flex-wrap gap-2">
            <span v-for="restriction in selectedUser.trust_summary.restrictions" :key="restriction" class="badge badge-outline h-auto px-4 py-3 text-left">
              {{ restriction }}
            </span>
          </div>
          <div v-if="selectedUser.trust_summary.recent_events?.length" class="mt-5 space-y-3">
            <div class="text-xs font-bold uppercase tracking-widest opacity-40">Последние изменения доверия</div>
            <div v-for="event in selectedUser.trust_summary.recent_events" :key="event.id" class="flex items-start gap-3 rounded-2xl bg-base-200/60 px-4 py-3">
              <div class="rounded-xl px-3 py-2 font-black" :class="event.delta >= 0 ? 'bg-success/15 text-success' : 'bg-error/15 text-error'">
                {{ event.delta > 0 ? `+${event.delta}` : event.delta }}
              </div>
              <div class="min-w-0">
                <div class="font-semibold">{{ event.comment || 'Изменение доверия' }}</div>
                <div class="mt-1 text-xs opacity-50">{{ formatDate(event.created_at) }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="mb-6 grid gap-6 lg:grid-cols-3">
          <section class="rounded-[1.5rem] border border-base-200 bg-base-100 p-4">
            <div class="mb-3 text-sm font-black uppercase tracking-[0.18em] opacity-45">Точки пользователя</div>
            <div v-if="selectedUser.monuments?.length" class="space-y-3">
              <div v-for="monument in selectedUser.monuments" :key="`mon-${monument.id}`" class="rounded-2xl bg-base-200/60 p-3">
                <div class="font-bold">{{ monument.name }}</div>
                <div class="mt-1 text-xs opacity-60">{{ monument.region || 'Регион не указан' }}</div>
                <button class="btn btn-ghost btn-xs mt-3 text-error" @click="deleteAdminEntity('monument', monument)">Удалить точку</button>
              </div>
            </div>
            <div v-else class="text-sm opacity-50">Точек пока нет.</div>
          </section>

          <section class="rounded-[1.5rem] border border-base-200 bg-base-100 p-4">
            <div class="mb-3 text-sm font-black uppercase tracking-[0.18em] opacity-45">Посты пользователя</div>
            <div v-if="selectedUser.posts?.length" class="space-y-3">
              <div v-for="post in selectedUser.posts" :key="`post-${post.id}`" class="rounded-2xl bg-base-200/60 p-3">
                <div class="font-bold">{{ post.monument_name || 'Пост' }}</div>
                <div class="mt-1 line-clamp-3 text-sm opacity-70">{{ post.description || 'Без описания' }}</div>
                <div class="mt-2 flex flex-wrap gap-2">
                  <span v-if="post.is_archived" class="badge badge-info badge-outline">В архиве</span>
                  <span v-if="post.monument_is_orphaned" class="badge badge-secondary badge-outline">Точка без автора</span>
                </div>
                <button class="btn btn-ghost btn-xs mt-3 text-error" @click="deleteAdminEntity('post', post)">Удалить пост</button>
              </div>
            </div>
            <div v-else class="text-sm opacity-50">Постов пока нет.</div>
          </section>

          <section class="rounded-[1.5rem] border border-base-200 bg-base-100 p-4">
            <div class="mb-3 text-sm font-black uppercase tracking-[0.18em] opacity-45">Сигналы пользователя</div>
            <div v-if="selectedUser.signals?.length" class="space-y-3">
              <div v-for="signal in selectedUser.signals" :key="`signal-${signal.id}`" class="rounded-2xl bg-base-200/60 p-3">
                <div class="font-bold">{{ signal.monument_name || signal.region || 'Сигнал' }}</div>
                <div class="mt-1 line-clamp-3 text-sm opacity-70">{{ signal.description }}</div>
                <button class="btn btn-ghost btn-xs mt-3 text-error" @click="deleteAdminEntity('signal', signal)">Удалить сигнал</button>
              </div>
            </div>
            <div v-else class="text-sm opacity-50">Сигналов пока нет.</div>
          </section>
        </div>

        <form class="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2" @submit.prevent="createSanction">
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
          <input v-model="sanctionForm.reason_text" class="input input-bordered rounded-xl md:col-span-2" placeholder="Причина ограничения" />
          <button class="btn btn-primary rounded-xl md:col-span-2">Выдать ограничение</button>
        </form>

        <div class="space-y-3">
          <h4 class="text-lg font-black">История санкций</h4>
          <div v-if="!selectedUser.sanctions_history?.length" class="opacity-50">История пуста</div>
          <div v-for="item in selectedUser.sanctions_history || []" :key="item.id" class="flex flex-col justify-between gap-3 rounded-2xl border border-base-200 p-4 md:flex-row md:items-center">
            <div>
              <div class="font-bold">{{ item.reason_text || sanctionReasonLabel(item.reason_code) }}</div>
              <div class="mt-1 text-xs opacity-50">{{ formatScopes(item.scopes || []) }} • {{ sanctionStatusLabel(item.status) }}</div>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-xs opacity-50">{{ formatDate(item.ends_at) }}</span>
              <button v-if="item.status === 'active'" class="btn btn-ghost btn-sm rounded-xl" @click="revokeSanction(item)">Снять</button>
            </div>
          </div>
        </div>
        <div class="mt-6 space-y-3">
          <h4 class="text-lg font-black">Журнал действий по пользователю</h4>
          <div v-if="!selectedUser.admin_logs?.length" class="opacity-50">Записей пока нет.</div>
          <div v-for="entry in selectedUser.admin_logs || []" :key="entry.id" class="rounded-2xl border border-base-200 p-4">
            <div class="font-bold">{{ entry.message }}</div>
            <div class="mt-1 text-xs opacity-55">{{ entry.action }} • {{ formatDate(entry.created_at) }}</div>
          </div>
        </div>
      </div>
    </dialog>

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
import { XIcon } from 'lucide-vue-next';
import api from '../api';
import ConfirmActionDialog from '../components/ConfirmActionDialog.vue';
import { useToast } from '../composables/useToast';

const toast = useToast();
const searchQuery = ref('');
const users = ref<any[]>([]);
const selectedUser = ref<any | null>(null);
const showAllRegions = ref(false);
const logsExpanded = ref(false);
const monumentsByRegion = ref<Array<{ region: string; count: number }>>([]);
const signalResolutionRaw = ref<Record<string, number>>({});
const systemLogs = ref<any[]>([]);
const confirmOpen = ref(false);
const confirmPending = ref(false);
const confirmAction = ref<null | (() => Promise<void>)>(null);
const confirmState = reactive({
  title: 'Подтверждение действия',
  message: '',
  confirmLabel: 'Подтвердить',
  details: [] as string[],
});

const stats = ref([
  { label: 'Памятники', value: '0' },
  { label: 'Посты', value: '0' },
  { label: 'Сигналы', value: '0' },
  { label: 'Пользователи', value: '0' },
]);

const sanctionForm = reactive({
  scopeMode: 'comment',
  duration: '6h',
  reason_text: '',
});

const visibleRegions = computed(() => {
  if (showAllRegions.value) return monumentsByRegion.value;
  return monumentsByRegion.value.slice(0, 8);
});

const signalResolutionStats = computed(() => ([
  { label: 'Устранено результативно', value: signalResolutionRaw.value.successful || 0 },
  { label: 'Частично решено', value: signalResolutionRaw.value.partial || 0 },
  { label: 'Закрыто без результата', value: signalResolutionRaw.value.unsuccessful || 0 },
  { label: 'Без указанного итога', value: signalResolutionRaw.value.unspecified || 0 },
]));

const fetchStats = async () => {
  const { data } = await api.get('/admin/stats');
  stats.value[0].value = String(data.monuments?.total || 0);
  stats.value[1].value = String(data.posts?.total || 0);
  stats.value[2].value = String(data.signals?.total || 0);
  stats.value[3].value = String(data.users?.total || 0);
  monumentsByRegion.value = Array.isArray(data.monuments?.by_region) ? data.monuments.by_region : [];
  signalResolutionRaw.value = data.signals?.by_resolution || {};
};

const fetchUsers = async () => {
  const { data } = await api.get('/admin/users', { params: { username: searchQuery.value || undefined } });
  users.value = data.users || data.items || [];
};

const fetchLogs = async () => {
  const { data } = await api.get('/admin/logs', { params: { limit: 12 } });
  systemLogs.value = data.items || [];
};

const updateRole = async (user: any, roleName: string) => {
  try {
    await api.post(`/admin/users/${user.id}/role`, { role_name: roleName });
    user.role_name = roleName;
    toast.success('Роль обновлена');
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось обновить роль');
  }
};

const toggleBlock = async (user: any) => {
  try {
    await api.post(`/admin/users/${user.id}/block`, { blocked: !user.is_blocked });
    user.is_blocked = !user.is_blocked;
    toast.success(user.is_blocked ? 'Вход заблокирован' : 'Блокировка снята');
    if (selectedUser.value?.id === user.id) {
      await selectUser(user);
    }
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось изменить блокировку');
  }
};

const selectUser = async (user: any) => {
  const { data } = await api.get(`/admin/users/${user.id}`);
  selectedUser.value = data;
};

const deleteAdminEntity = async (type: 'monument' | 'post' | 'signal', item: any) => {
  const labels: Record<string, string> = {
    monument: 'точку',
    post: 'пост',
    signal: 'сигнал',
  };
  confirmState.title = 'Удаление объекта';
  confirmState.message = `Будет удален ${labels[type]}, а связанный пользовательский контент будет обработан по правилам системы.`;
  confirmState.confirmLabel = `Удалить ${labels[type]}`;
  confirmState.details = ['Действие фиксируется в журнале системы.'];
  confirmAction.value = async () => {
    try {
      await api.post(`/admin/${type === 'monument' ? 'monuments' : type === 'post' ? 'posts' : 'signals'}/${item.id}/delete`, { confirm: true });
      toast.success(type === 'monument' ? 'Точка удалена.' : type === 'post' ? 'Пост удален.' : 'Сигнал удален.');
      if (selectedUser.value) {
        await selectUser(selectedUser.value);
      }
      await fetchUsers();
      await fetchLogs();
    } catch (error: any) {
      toast.error(error?.response?.data?.message || 'Не удалось удалить запись');
    }
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

const createSanction = async () => {
  if (!selectedUser.value) return;
  try {
    await api.post(`/admin/users/${selectedUser.value.id}/sanctions`, {
      reason_code: sanctionForm.scopeMode === 'login' ? 'manual_login_ban' : 'manual_restriction',
      reason_text: sanctionForm.reason_text,
      scopes: resolveScopes(sanctionForm.scopeMode),
      ends_at: resolveEndsAt(sanctionForm.duration),
    });
    toast.success('Ограничение выдано');
    sanctionForm.reason_text = '';
    await selectUser(selectedUser.value);
    await fetchUsers();
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось выдать ограничение');
  }
};

const revokeSanction = async (sanction: any) => {
  if (!selectedUser.value) return;
  try {
    await api.post(`/admin/users/${selectedUser.value.id}/sanctions/${sanction.id}/revoke`, { reason: 'Снято администратором' });
    toast.success('Ограничение снято');
    await selectUser(selectedUser.value);
    await fetchUsers();
  } catch (error: any) {
    toast.error(error?.response?.data?.message || 'Не удалось снять ограничение');
  }
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

const formatDate = (value?: string | null) => {
  if (!value) return 'Бессрочно';
  return new Date(value).toLocaleString();
};

const statusLabel = (status: string) => {
  if (status === 'login_banned') return 'Вход заблокирован';
  if (status === 'restricted') return 'Ограничен';
  return 'Активен';
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

const sanctionReasonLabel = (reasonCode?: string | null) => {
  if (reasonCode === 'manual_login_ban') return 'Ручная блокировка входа';
  if (reasonCode === 'manual_restriction') return 'Ручное ограничение функций';
  if (reasonCode === 'ai_comment_abuse') return 'Автоматическое ограничение за скрытые комментарии';
  if (reasonCode === 'legacy_blocked_user') return 'Перенесенная старая блокировка';
  return 'Служебная причина ограничения';
};

const scopeLabel = (scope: string) => ({
  comment_write: 'запрет комментариев',
  content_create: 'запрет публикации',
  content_edit: 'запрет редактирования',
  report_create: 'запрет жалоб',
  login: 'блокировка входа',
}[scope] || 'другое ограничение');

const formatScopes = (scopes: string[]) => {
  if (!scopes.length) return 'Без ограничения функций';
  return scopes.map(scopeLabel).join(', ');
};

const sanctionStatusLabel = (status: string) => {
  if (status === 'active') return 'активно';
  if (status === 'revoked') return 'снято';
  if (status === 'expired') return 'истекло';
  return 'неизвестно';
};

const exportData = async (format: string) => {
  try {
    const endpoint = format === 'geojson' ? '/admin/export/monuments/geojson' : '/admin/export/monuments/csv';
    const { data } = await api.get(endpoint, { responseType: 'blob' });
    const url = window.URL.createObjectURL(new Blob([data]));
    const link = document.createElement('a');
    link.href = url;
    link.download = format === 'geojson' ? 'monuments.geojson' : 'monuments.csv';
    document.body.appendChild(link);
    link.click();
    link.remove();
  } catch {
    toast.error('Не удалось выгрузить данные');
  }
};

onMounted(async () => {
  await Promise.all([fetchStats(), fetchUsers(), fetchLogs()]);
});
</script>

<style scoped>
.table td .flex.flex-wrap.justify-end.gap-2 > .btn:first-child {
  display: none;
}
</style>

