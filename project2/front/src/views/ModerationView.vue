<template>
  <div class="p-8 max-w-7xl mx-auto space-y-12 animate-in fade-in slide-in-from-bottom-8 duration-700">
    <header class="flex flex-col md:flex-row md:items-end justify-between gap-6 pb-8 border-b border-base-200">
      <div>
        <div class="inline-flex bg-primary/10 p-3 rounded-2xl text-primary mb-4 font-bold tracking-tight text-sm uppercase">
          Раздел модерации
        </div>
        <h1 class="text-5xl font-black tracking-tighter mb-4">Очередь обработки</h1>
        <p class="text-lg opacity-60 font-medium">Проверяйте и одобряйте контент пользователей площадки.</p>
      </div>

      <div class="tabs tabs-boxed p-1.5 bg-base-200/50 rounded-2xl border border-base-200">
        <button 
          v-for="tab in tabs" 
          :key="tab.id"
          class="tab tab-lg font-bold rounded-xl transition-all h-14 px-8"
          :class="{ 'tab-active bg-primary text-primary-content shadow-lg shadow-primary/20': activeTab === tab.id }"
          @click="activeTab = tab.id"
        >
          {{ tab.name }}
          <div v-if="tab.count" class="badge badge-sm ml-2" :class="activeTab === tab.id ? 'badge-ghost opacity-50' : 'badge-primary'">
            {{ tab.count }}
          </div>
        </button>
      </div>
    </header>

    <div v-if="activeTab === 'reports'" class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <select v-model="reportFilters.entity_type" class="select select-bordered rounded-2xl bg-base-100 border-base-200">
        <option value="">Все типы</option>
        <option value="monument">Памятники</option>
        <option value="post">Посты</option>
        <option value="photo">Фото</option>
        <option value="signal">Сигналы</option>
        <option value="comment">Комментарии</option>
      </select>
      <select v-model="reportFilters.category" class="select select-bordered rounded-2xl bg-base-100 border-base-200">
        <option value="">Все категории</option>
        <option value="integrity">Проверка данных</option>
        <option value="abuse">Абьюз / нарушение</option>
      </select>
      <select v-model="reportFilters.reason_code" class="select select-bordered rounded-2xl bg-base-100 border-base-200">
        <option value="">Все причины</option>
        <option value="wrong_name">Неверное название</option>
        <option value="wrong_coords">Неверные координаты</option>
        <option value="duplicate">Дубликат</option>
        <option value="offensive">Оскорбительный контент</option>
        <option value="spam">Спам</option>
        <option value="other">Другое</option>
      </select>
    </div>

    <!-- Queue List -->
    <div class="grid gap-6">
      <template v-if="loading">
        <div v-for="i in 3" :key="i" class="h-48 w-full bg-base-200 rounded-[2.5rem] animate-pulse"></div>
      </template>
      
      <template v-else-if="items.length === 0">
        <div class="bg-base-200/30 p-20 rounded-[3rem] text-center border-2 border-dashed border-base-300">
          <div class="bg-base-100 p-6 rounded-3xl inline-flex shadow-xl mb-6">
            <CheckCircleIcon class="w-12 h-12 text-success opacity-50" />
          </div>
          <h3 class="text-2xl font-black mb-2 tracking-tight">Все чисто!</h3>
          <p class="opacity-50 font-medium">В этой очереди пока нет новых элементов для проверки.</p>
        </div>
      </template>

      <template v-else>
        <div 
          v-for="item in items" 
          :key="item.id" 
          class="group bg-base-100 p-8 rounded-[2.5rem] shadow-xl hover:shadow-2xl transition-all duration-500 border border-base-200 flex flex-col md:flex-row gap-10 relative overflow-hidden"
        >
          <!-- High Risk Badge -->
          <div v-if="item.high_risk" class="absolute top-0 right-0 bg-error text-error-content px-6 py-2 rounded-bl-3xl font-black text-xs uppercase tracking-widest shadow-lg">
            Высокий риск
          </div>

          <div class="w-full md:w-64 aspect-square bg-base-200 rounded-3xl overflow-hidden shadow-inner shrink-0">
            <img v-if="resolveThumbnail(item)" :src="resolveThumbnail(item)" class="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" />
            <div v-else class="w-full h-full flex items-center justify-center opacity-20">
              <ImageIcon class="w-16 h-16" />
            </div>
          </div>

          <div class="flex-grow flex flex-col justify-between py-2">
            <div>
              <div class="flex flex-wrap items-center gap-3 mb-4">
                <div class="avatar placeholder">
                  <div class="bg-neutral text-neutral-content rounded-xl w-8 h-8 font-bold text-xs uppercase shadow-sm">
                    {{ item.author_name?.[0] || 'U' }}
                  </div>
                </div>
                <span class="font-bold opacity-60 text-sm">{{ item.author_name || 'Аноним' }}</span>
                <span class="opacity-20">•</span>
                <span class="text-xs opacity-40 font-bold uppercase tracking-wider">{{ new Date(item.created_at).toLocaleString() }}</span>
                <span
                  v-if="activeTab === 'edits' || item.is_edit_request"
                  class="badge border-0 bg-amber-100 text-amber-900 rounded-xl px-3 py-2 text-[10px] font-black uppercase tracking-wider"
                >
                  Заявка на редактирование
                </span>
                
                <!-- AI Flags -->
                <div v-if="item.toxic_score !== undefined" class="badge badge-sm font-bold" :class="item.toxic_score > 0.7 ? 'badge-error' : 'badge-ghost'">
                  Токсичность: {{ (item.toxic_score * 100).toFixed(0) }}%
                </div>
                <div v-if="item.relevance_score !== undefined" class="badge badge-sm font-bold" :class="item.relevance_score < 0.5 ? 'badge-warning' : 'badge-ghost'">
                  Релевантность: {{ (item.relevance_score * 100).toFixed(0) }}%
                </div>
              </div>
              
              <h3 class="text-3xl font-black tracking-tighter mb-4">{{ resolveTitle(item) }}</h3>
              <p class="text-lg font-medium opacity-70 line-clamp-3 leading-relaxed max-w-2xl">
                {{ resolvePreviewText(item) }}
              </p>
              
              <!-- Diff for edits -->
              <div v-if="activeTab === 'edits' && item.diff" class="mt-6 p-6 bg-base-200 rounded-3xl space-y-4">
                <div v-for="(change, field) in item.diff" :key="field" class="text-sm">
                  <div class="font-black uppercase opacity-40 text-[10px] mb-1">{{ field }}</div>
                  <div class="flex flex-col gap-1">
                    <del class="text-error opacity-50 line-through">{{ change.old }}</del>
                    <ins class="text-success no-underline font-bold">{{ change.new }}</ins>
                  </div>
                </div>
              </div>
              <div v-else-if="activeTab === 'edits' && (item.old_value !== undefined || item.new_value !== undefined)" class="mt-6 p-6 bg-base-200 rounded-3xl space-y-3 text-sm">
                <div class="font-black uppercase opacity-40 text-[10px] mb-1">{{ item.field_name || 'Поле' }}</div>
                <div class="rounded-2xl bg-error/10 px-4 py-3 text-error-content/80 line-through">
                  {{ item.old_value || '(пусто)' }}
                </div>
                <div class="rounded-2xl bg-success/10 px-4 py-3 font-bold text-success-content/80">
                  {{ item.new_value || '(пусто)' }}
                </div>
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-4 mt-8">
              <button @click="viewDetails(item)" class="btn btn-primary px-8 h-14 rounded-2xl font-black text-base shadow-lg shadow-primary/20 hover:scale-[1.02] transition-transform">
                Изучить заявку
              </button>
            </div>
          </div>
        </div>
      </template>
    </div>

    <!-- Rejection Modal -->
    <dialog id="reject_modal" class="modal modal-bottom sm:modal-middle bg-base-300/30 backdrop-blur-xl" style="z-index: 1000">
      <div class="modal-box w-[min(94vw,860px)] max-w-none rounded-[3rem] p-0 border border-base-200 shadow-2xl animate-in zoom-in duration-300 flex flex-col max-h-[90vh]">
        <div class="p-10 overflow-y-auto custom-scrollbar flex-grow">
        <div class="bg-error/10 p-4 rounded-2xl inline-flex text-error mb-6">
          <AlertCircleIcon class="w-10 h-10" />
        </div>
        <h3 class="text-3xl font-black tracking-tight mb-4">Причина отклонения</h3>
        <p class="opacity-60 font-medium mb-6">Пожалуйста, укажите причину отклонения. Автор получит уведомление с этим комментарием.</p>
        
        <div class="flex flex-wrap gap-2 mb-4">
          <button
            v-for="preset in rejectPresets"
            :key="preset"
            type="button"
            @click="rejectComment = preset"
            class="badge badge-error badge-outline hover:bg-primary/10 hover:border-primary hover:text-primary cursor-pointer py-3 h-auto font-bold opacity-70 hover:opacity-100 transition-all"
          >
            {{ preset }}
          </button>
        </div>

        <textarea 
          v-model="rejectComment"
          class="textarea textarea-bordered w-full h-40 rounded-3xl bg-base-200 border-none focus:ring-4 focus:ring-error/10 text-lg font-medium p-6 transition-all" 
          placeholder="Например: недостаточно четкое фото или неверное описание..."
        ></textarea>
        
        <div class="modal-action gap-3 mt-8">
          <form method="dialog" class="flex-grow">
            <button class="btn btn-ghost w-full h-14 rounded-2xl font-bold">Отмена</button>
          </form>
          <button @click="confirmReject" class="btn btn-error flex-[2] h-14 rounded-2xl font-black text-white shadow-xl shadow-error/20">
            Подтвердить
          </button>
        </div>
        </div>
      </div>
    </dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue';
import { useRouter } from 'vue-router';
import api from '../api';
import { 
  CheckCircleIcon, ImageIcon, AlertCircleIcon
} from 'lucide-vue-next';
import { useToast } from '../composables/useToast';

const router = useRouter();
const toast = useToast();
const activeTab = ref('monuments');
const tabs = ref([
  { id: 'monuments', name: 'Точки', count: 0 },
  { id: 'posts', name: 'Посты', count: 0 },
  { id: 'edits', name: 'Правки', count: 0 },
  { id: 'signals', name: 'Сигналы', count: 0 },
  { id: 'reports', name: 'Жалобы', count: 0 }
]);

const items = ref<any[]>([]);
const loading = ref(false);
const rejectComment = ref('');
const pendingItem = ref<any>(null);
const reportFilters = ref({
  entity_type: '',
  category: '',
  reason_code: '',
});

const rejectPresets = computed(() => {
  if (activeTab.value === 'reports') {
    return [
      'Жалоба не подтвердилась после проверки.',
      'Недостаточно доказательств для подтверждения жалобы.',
      'Объект соответствует правилам и остается без изменений.'
    ];
  }
  if (activeTab.value === 'signals') {
    return [
      'Угроза не подтверждена по приложенным материалам.',
      'Недостаточно данных для подтверждения сигнала.',
      'Сигнал не соответствует критериям публикации.'
    ];
  }
  if (activeTab.value === 'posts') {
    return [
      'Пост не соответствует правилам публикации.',
      'Содержание поста недостаточно информативно.',
      'Недостаточно подтверждений фактов в публикации.'
    ];
  }
  if (activeTab.value === 'monuments') {
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


const fetchCounts = async () => {
  try {
    const { data } = await api.get('/moderation/stats');
    // Map backend status counts to tab badges
    tabs.value[0].count = data.monuments?.pending || 0;
    tabs.value[1].count = data.posts?.pending || 0;
    tabs.value[3].count = data.signals?.pending || 0;
    // For 'edits' and 'reports', check if they exist in stats
    tabs.value[2].count = data.edits?.pending || 0;
    tabs.value[4].count = data.reports?.pending || 0;
  } catch (err) {
    console.error('Failed to fetch moderation counts', err);
  }
};

const fetchQueue = async () => {
  loading.value = true;
  try {
    const params = activeTab.value === 'reports' ? { ...reportFilters.value } : undefined;
    const { data } = await api.get(`/moderation/${activeTab.value}`, { params });
    items.value = data.items || [];
    // Update count for active tab after fetch
    const currentTab = tabs.value.find(t => t.id === activeTab.value);
    if (currentTab) currentTab.count = items.value.length;
  } catch (err) {
    items.value = [];
  } finally {
    loading.value = false;
  }
};

watch(activeTab, fetchQueue);
watch(reportFilters, () => {
  if (activeTab.value === 'reports') {
    fetchQueue();
  }
}, { deep: true });

onMounted(() => {
  fetchQueue();
  fetchCounts();
});


const confirmReject = async () => {
  if (!rejectComment.value.trim()) {
    toast.warning('Нужно указать комментарий для автора');
    return;
  }
  try {
    const payload: Record<string, any> = { action: 'reject' };
    if (activeTab.value === 'signals') {
      payload.official_response = rejectComment.value;
    } else {
      payload.comment = rejectComment.value;
    }
    await api.post(`/moderation/${activeTab.value}/${pendingItem.value.id}/action`, payload);
    items.value = items.value.filter(i => i.id !== pendingItem.value.id);
    (document.getElementById('reject_modal') as HTMLDialogElement).close();
    toast.success('Заявка отклонена');
  } catch (err) {
    toast.error('Ошибка при отклонении');
  }
};

const resolvePreviewText = (item: any) => {
  if (activeTab.value === 'reports') {
    const snapshot = item.entity_snapshot || {};
    if (typeof snapshot.description === 'string' && snapshot.description.trim()) return snapshot.description.trim();
    if (typeof snapshot.content === 'string' && snapshot.content.trim()) return snapshot.content.trim();
    return `Причина: ${reasonLabel(item.reason_code)}. Жалоб: ${item.distinct_reporters_count || item.reports_count || 0}`;
  }
  const direct = typeof item?.description === 'string' ? item.description.trim() : '';
  if (direct) return direct;
  const fromProps = typeof item?.properties?.description === 'string' ? item.properties.description.trim() : '';
  if (fromProps) return fromProps;
  return 'Нет описания...';
};

const resolveThumbnail = (item: any) => {
  if (activeTab.value === 'reports') {
    const snapshot = item.entity_snapshot || {};
    return normalizeImagePath(
      snapshot.thumbnail ||
      snapshot.preview ||
      snapshot.preview_path ||
      snapshot.thumbnail_path ||
      snapshot.file_path ||
      (Array.isArray(snapshot.photos) && snapshot.photos.length > 0
        ? snapshot.photos[0].preview_path || snapshot.photos[0].thumbnail_path || snapshot.photos[0].file_path || ''
        : '')
    );
  }
  return normalizeImagePath(
    item.thumbnail ||
    item.preview_path ||
    item.file_path ||
    (Array.isArray(item.photos) && item.photos.length > 0
      ? item.photos[0].preview_path || item.photos[0].thumbnail_path || item.photos[0].file_path || ''
      : '')
  );
};

const normalizeImagePath = (path: string) => {
  if (!path) return '';
  return path.startsWith('/') ? path : `/${path}`;
};

const resolveTitle = (item: any) => {
  if (activeTab.value === 'reports') {
    return `${entityTypeLabel(item.entity_type)}: ${reasonLabel(item.reason_code)}`;
  }
  if (activeTab.value === 'edits') {
    return item.title || item.monument_name || item.name || 'Заявка на редактирование';
  }
  return item.name || item.monument_name || item.title || (activeTab.value === 'signals' ? (item.signal_type ? 'Сигнал: ' + signalTypeLabel(item.signal_type) : 'Сигнал угрозы') : 'Без названия');
};

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

const viewDetails = async (item: any) => {
  router.push(`/moderation/item/${activeTab.value}/${item.id}`);
};
</script>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 8px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: rgba(0,0,0,0.1);
  border-radius: 20px;
}
</style>

