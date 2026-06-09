<template>
  <div class="min-h-screen bg-base-200/30 pb-20">
    <template v-if="loading">
      <div class="h-96 w-full bg-base-200 animate-pulse"></div>
      <div class="max-w-7xl mx-auto p-8 space-y-8">
        <div class="h-12 w-1/3 bg-base-200 animate-pulse rounded-2xl"></div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
          <div class="col-span-2 space-y-6">
            <div v-for="i in 3" :key="i" class="h-64 bg-base-200 animate-pulse rounded-3xl"></div>
          </div>
          <div class="h-96 bg-base-200 animate-pulse rounded-3xl"></div>
        </div>
      </div>
    </template>

    <template v-else-if="monument">
      <!-- Hero Gallery -->
      <div class="h-[50vh] relative overflow-hidden bg-black">
        <div class="absolute inset-0 flex transition-transform duration-700" :style="{ transform: `translateX(-${activePhotoIdx * 100}%)` }">
          <div v-for="photo in allPhotos" :key="photo.id" class="w-full h-full shrink-0">
            <img :src="resolvePhotoSrc(photo)" class="w-full h-full object-cover opacity-70" />
          </div>
        </div>
        
        <div class="absolute inset-0 bg-gradient-to-t from-base-100 via-transparent to-black/30"></div>

        <!-- Gallery Navigation -->
        <div v-if="allPhotos.length > 1" class="absolute bottom-10 left-1/2 -translate-x-1/2 flex gap-3 z-10">
          <button 
            v-for="(_, idx) in allPhotos" 
            :key="idx"
            @click="activePhotoIdx = idx"
            class="w-3 h-3 rounded-full transition-all"
            :class="activePhotoIdx === idx ? 'bg-primary w-10' : 'bg-white/50 hover:bg-white'"
          ></button>
        </div>

      </div>

      <div class="mx-auto max-w-[1500px] px-5 lg:px-8 -mt-32 relative z-10">
        <div class="grid grid-cols-1 gap-8 xl:grid-cols-[minmax(0,1.85fr)_minmax(340px,0.9fr)]">
          <!-- Main Content -->
          <div class="space-y-10">
            <!-- Monument Header Card -->
            <div class="bg-base-100 p-8 lg:p-9 rounded-[3rem] shadow-2xl shadow-base-300/50 border border-base-200">
              <div class="flex flex-wrap items-center gap-4 mb-6">
                <div class="badge badge-primary badge-lg py-4 px-6 rounded-2xl font-black text-xs uppercase tracking-widest">
                  Памятник
                </div>
                <div class="flex items-center gap-2 text-sm font-bold opacity-40">
                  <CalendarIcon class="w-4 h-4" />
                  {{ new Date(monument.created_at).toLocaleDateString() }}
                </div>
                <button @click="openGallery" class="btn btn-ghost rounded-2xl font-bold ml-auto">
                  <ImageIcon class="w-5 h-5 mr-2" /> {{ allPhotos.length }} фото
                </button>
              </div>

              <div class="flex items-start gap-4 mb-6">
                <div class="flex-1">
                  <h1 class="text-4xl xl:text-[3.25rem] font-black tracking-tighter leading-none">{{ monument.name }}</h1>
                  <div class="mt-4 flex flex-wrap items-center justify-between gap-3">
                    <div class="text-sm font-bold opacity-55">
                      {{ monument.region || monument.properties?.city || 'Не указан' }}
                    </div>
                    <div class="badge badge-ghost font-bold px-4 py-4 rounded-2xl">{{ posts.length }} постов</div>
                  </div>
                </div>
                <div v-if="auth.isAuthenticated" class="dropdown dropdown-end">
                  <button tabindex="0" class="btn btn-ghost btn-circle">
                    <MoreHorizontalIcon class="w-5 h-5 opacity-60" />
                  </button>
                  <ul tabindex="0" class="dropdown-content menu menu-sm mt-2 z-[2] p-2 shadow-xl bg-base-100 rounded-2xl w-48 border border-base-200">
                    <li><button @click="monumentReportOpen = true">Пожаловаться</button></li>
                  </ul>
                </div>
              </div>
              
              <div v-if="visibleProperties.length > 0" class="grid grid-cols-2 xl:grid-cols-3 gap-4 mb-6">
                <div v-for="item in visibleProperties" :key="item.key" class="bg-base-200/50 p-4 rounded-2xl">
                  <div class="text-[10px] font-black uppercase opacity-40 mb-1 tracking-widest">{{ item.key }}</div>
                  <div class="font-bold text-sm truncate">{{ item.value }}</div>
                </div>
              </div>

            </div>

            <div v-if="monumentDescription" class="bg-base-100 p-8 rounded-[2.5rem] shadow-xl border border-base-200">
              <ExpandableFormattedText
                :text="monumentDescription"
                :max-height="260"
                :min-hidden-height="44"
                content-class="text-lg font-medium opacity-80 leading-relaxed"
              />
            </div>

            <!-- Posts List -->
            <div class="space-y-8">
              <div v-if="posts.length === 0" class="rounded-[2.5rem] border border-dashed border-base-300 bg-base-100 p-8 shadow-xl lg:p-10">
                <div class="grid gap-8 xl:grid-cols-[minmax(0,1.35fr)_minmax(360px,0.65fr)] xl:items-center">
                  <div class="min-w-0">
                    <div class="text-3xl font-black tracking-tight lg:text-[2.35rem]">Карточка еще ждет первый пост</div>
                    <p class="mt-4 max-w-3xl text-base font-medium leading-8 opacity-65 lg:text-lg">
                      Точка уже добавлена на карту, но подробного материала о ней пока нет. Первый пост поможет наполнить карточку фактами, описанием, фотографиями и полезными уточнениями для других пользователей.
                    </p>
                    <div class="mt-5 flex flex-wrap gap-2">
                      <span class="badge badge-ghost rounded-xl px-4 py-3 font-bold">Пока нет публикаций</span>
                      <span class="badge badge-ghost rounded-xl px-4 py-3 font-bold">Можно добавить описание и фото</span>
                    </div>
                  </div>
                  <div class="w-full rounded-[2rem] border border-base-200 bg-base-200/45 p-6 xl:max-w-[420px] xl:justify-self-end">
                    <button @click="showAddPostModal" class="btn btn-primary h-14 w-full rounded-2xl px-6 font-black shadow-lg shadow-primary/20">
                      <PlusIcon class="mr-2 h-5 w-5" /> Добавить первый пост
                    </button>
                    <div class="mt-4 text-sm font-semibold leading-7 opacity-60">
                      Чем подробнее первый материал, тем полезнее карточка для остальных пользователей.
                    </div>
                  </div>
                </div>
              </div>
              <div v-for="post in posts" :id="`post-${post.id}`" :key="post.id" class="bg-base-100 rounded-[2.5rem] shadow-xl border border-base-200 overflow-hidden group">
                <div class="p-8">
                  <div class="flex items-center gap-4 mb-6">
                    <div class="avatar placeholder">
                      <div class="bg-neutral text-neutral-content rounded-xl w-10 h-10 font-bold">
                        {{ post.author_name?.[0]?.toUpperCase() || 'U' }}
                      </div>
                    </div>
                    <div>
                      <div class="font-bold">{{ post.author_name || 'Волонтёр' }}</div>
                      <div class="text-xs opacity-40 font-bold uppercase tracking-wider">{{ new Date(post.created_at).toLocaleDateString() }}</div>
                    </div>
                  </div>

                  <div v-if="auth.isAuthenticated" class="flex justify-end mb-4">
                    <div class="dropdown dropdown-end">
                      <button tabindex="0" class="btn btn-ghost btn-sm btn-circle">
                        <MoreHorizontalIcon class="w-4 h-4 opacity-60" />
                      </button>
                      <ul tabindex="0" class="dropdown-content menu menu-sm mt-2 z-[2] p-2 shadow-xl bg-base-100 rounded-2xl w-48 border border-base-200">
                        <li><button @click="openPostReport(post)">Пожаловаться</button></li>
                      </ul>
                    </div>
                  </div>

                  <ExpandableFormattedText
                    :text="post.description"
                    :max-height="220"
                    :min-hidden-height="38"
                    content-class="text-xl font-medium opacity-80 leading-relaxed mb-2"
                  />

                  <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
                    <div 
                      v-for="photo in post.photos" 
                      :key="photo.id"
                      class="aspect-square rounded-2xl overflow-hidden shadow-md cursor-pointer group/img"
                      @click="viewPhoto(photo)"
                    >
                      <img :src="resolvePhotoThumb(photo)" class="w-full h-full object-cover transition-transform duration-500 group-hover/img:scale-110" />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Sidebar Actions -->
          <div class="space-y-6">
            <div class="sticky top-24 space-y-6">
              <div class="bg-primary p-8 rounded-[2.5rem] text-primary-content shadow-2xl shadow-primary/30">
                <h3 class="text-2xl font-black mb-4">Внести вклад</h3>
                <p class="font-medium opacity-80 mb-8 leading-snug">Поделитесь своими фотографиями или информацией об этом месте.</p>
                <button @click="showAddPostModal" class="btn btn-white w-full h-16 rounded-2xl font-black text-lg shadow-xl hover:scale-105 transition-transform">
                  <PlusIcon class="w-6 h-6 mr-2" /> Добавить пост
                </button>
              </div>

              <button @click="showSignalForm" class="btn w-full h-16 rounded-2xl font-black border-none bg-secondary text-secondary-content shadow-xl shadow-secondary/20 hover:bg-secondary/90 group">
                <ShieldAlertIcon class="w-6 h-6 mr-2 transition-transform group-hover:scale-110" />
                Сообщить об угрозе
              </button>

              <button @click="showOnMap" class="btn btn-outline w-full h-14 rounded-2xl border-base-300 font-bold">
                Показать на карте
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Add Post Modal -->
    <dialog id="add_post_modal" class="modal modal-bottom sm:modal-middle bg-base-300/30 backdrop-blur-xl">
      <div class="modal-box w-[min(94vw,920px)] max-w-none rounded-[3rem] p-0 border border-base-200 shadow-2xl animate-in zoom-in duration-300 flex flex-col max-h-[90vh] relative">
        <button
          type="button"
          @click="closeAddPost"
          class="btn btn-circle btn-sm absolute top-5 right-5 z-20 bg-base-100/85 border-base-300 hover:bg-base-200"
          aria-label="Закрыть форму"
        >
          <XIcon class="w-4 h-4" />
        </button>
        <div class="p-10 overflow-y-auto custom-scrollbar flex-grow">
          <div class="bg-primary/10 p-4 rounded-2xl inline-flex text-primary mb-6">
            <PlusIcon class="w-10 h-10" />
          </div>
          <h3 class="text-3xl font-black tracking-tight mb-4">Новый пост</h3>
          <p class="opacity-60 font-medium mb-8">Добавьте обновление по памятнику, новые детали или свежие фотографии.</p>
          
          <form @submit.prevent="submitPost" class="space-y-7">
            <div class="form-control w-full gap-3">
              <label class="label font-bold opacity-70 text-base">Текст поста</label>
              <RichTextEditor v-model="newPost.description" :invalid="!!badFields.description" placeholder="Напишите, что изменилось, что важно зафиксировать или что нужно проверить..." />
              <span v-if="badFields.description" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('description', badFields.description) }}</span>
            </div>

            <div class="form-control w-full gap-3">
              <label class="label font-bold opacity-70 text-base">Фотографии (1-10)</label>
              <input type="file" multiple @change="handleFiles" class="file-input file-input-bordered file-input-primary w-full rounded-2xl h-14 bg-base-200 border-none text-base" accept="image/*" />
              <span v-if="badFields.photos" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('photos', badFields.photos) }}</span>
              <div class="flex gap-3 mt-4 overflow-x-auto py-2">
                <div v-for="(file, idx) in previewUrls" :key="idx" class="relative shrink-0">
                  <img :src="file" class="w-24 h-24 object-cover rounded-2xl shadow-md transition-all" :class="validation.badPhotos.includes(idx) ? 'ring-4 ring-error' : ''" />
                  <button @click.prevent="removeFile(idx)" class="btn btn-circle btn-xs btn-error absolute -top-1.5 -right-1.5 z-10 shadow-md">
                    <XIcon class="w-3 h-3" />
                  </button>
                  <div v-if="validation.badPhotos.includes(idx)" class="absolute bottom-0 inset-x-0 bg-error text-white text-[8px] font-bold text-center py-0.5 rounded-b-2xl">AI</div>
                </div>
              </div>
            </div>

            <div v-if="aiWarnings.length > 0" class="alert alert-warning bg-warning/10 text-warning border-none rounded-2xl p-6 flex flex-col gap-4">
              <div class="flex gap-3">
                <AlertTriangleIcon class="w-6 h-6" />
                <div class="font-bold">AI-предупреждение</div>
              </div>
              <ul class="list-disc list-inside text-sm opacity-80 font-medium">
                <li v-for="w in aiWarnings" :key="w">{{ w }}</li>
              </ul>
              <div class="flex items-center gap-3 pt-2">
                <input type="checkbox" v-model="contentAck" class="checkbox checkbox-warning rounded-lg" />
                <span class="text-sm font-bold">Я уверен, что контент не нарушает правила</span>
              </div>
            </div>

            <div v-if="error" class="alert alert-error bg-error/10 text-error border-none rounded-2xl p-4 flex gap-3">
              <AlertCircleIcon class="w-5 h-5" />
              <span class="text-sm font-bold">{{ error }}</span>
            </div>

            <div class="modal-action gap-3">
              <button type="button" @click="closeAddPost" class="btn btn-ghost flex-grow h-14 rounded-2xl font-bold">Отмена</button>
              <button type="submit" class="btn btn-primary flex-[2] h-14 rounded-2xl font-black text-white shadow-xl shadow-primary/20" :disabled="submitting">
                <span v-if="submitting" class="loading loading-spinner"></span>
                <span v-else>Отправить пост</span>
              </button>
            </div>
          </form>
        </div>
      </div>
    </dialog>

    <PhotoGallery
      :photos="galleryPhotos"
      :initial-index="galleryPhotoIdx"
      v-model="galleryOpen"
      :show-captions="false"
      :allow-report="auth.isAuthenticated"
      :report-subject="galleryReportSubject"
      :report-duplicate-seed="monument?.name"
    />
    <ReportDialog
      v-model="monumentReportOpen"
      entity-type="monument"
      :entity-id="monument?.id || ''"
      :subject="monument?.name"
      :duplicate-seed="monument?.name"
    />
    <ReportDialog
      v-model="postReportOpen"
      entity-type="post"
      :entity-id="selectedPostForReport?.id || ''"
      :subject="selectedPostForReport ? `Пост о памятнике ${monument?.name || ''}` : ''"
      :duplicate-seed="monument?.name"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, reactive, watch, nextTick } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import { 
  ImageIcon, CalendarIcon, PlusIcon,
  ShieldAlertIcon, XIcon, AlertTriangleIcon, AlertCircleIcon, MoreHorizontalIcon 
} from 'lucide-vue-next';
import PhotoGallery from '../components/PhotoGallery.vue';
import ReportDialog from '../components/ReportDialog.vue';
import RichTextEditor from '../components/RichTextEditor.vue';
import ExpandableFormattedText from '../components/ExpandableFormattedText.vue';
import type { Monument, Post, Photo } from '../types';
import { useAuthStore } from '../store/auth';
import { applyValidationResult, buildFilesFingerprint, buildTextFingerprint, createValidationState } from '../composables/useContentValidation';
import { normalizeRichTextInput } from '../utils/richText';
import { useToast } from '../composables/useToast';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const toast = useToast();

const loading = ref(true);
const monument = ref<Monument | null>(null);
const posts = ref<Post[]>([]);
const activePhotoIdx = ref(0);
const galleryOpen = ref(false);
const galleryPhotoIdx = ref(0);
const monumentReportOpen = ref(false);
const postReportOpen = ref(false);
const selectedPostForReport = ref<Post | null>(null);

const fetchMonument = async () => {
  loading.value = true;
  try {
    const { data } = await api.get(`/monuments/${route.params.id}`);
    monument.value = data.monument;
    posts.value = data.posts || [];
    await nextTick();
    scrollToRequestedPost();
    maybeOpenComposerFromQuery();
  } catch (err) {
    console.error(err);
  } finally {
    loading.value = false;
  }
};

onMounted(fetchMonument);

const scrollToRequestedPost = () => {
  const hash = route.hash?.trim();
  if (!hash) return;
  const target = document.querySelector(hash);
  if (!target) return;
  target.scrollIntoView({ behavior: 'smooth', block: 'start' });
};

watch(() => route.hash, async () => {
  await nextTick();
  scrollToRequestedPost();
});

watch(() => route.query.compose, async () => {
  await nextTick();
  maybeOpenComposerFromQuery();
});

const allPhotos = computed(() => {
  const photos: Photo[] = [];
  posts.value.forEach(p => {
    if (p.photos && Array.isArray(p.photos)) {
      photos.push(...p.photos);
    }
  });
  return photos;
});

const visibleProperties = computed(() => {
  if (!monument.value?.properties) return [];
  return Object.entries(monument.value.properties)
    .filter(([key, value]) => key !== 'description' && value !== null && value !== undefined && String(value).trim() !== '')
    .map(([key, value]) => ({
      key,
      value: String(value),
    }));
});

const normalizeComparableText = (value: unknown) => {
  if (typeof value !== 'string') return '';
  return value
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .toLowerCase();
};

const monumentDescription = computed(() => {
  const raw = monument.value?.properties?.description;
  const description = typeof raw === 'string' ? raw.trim() : '';
  if (!description) return '';

  const firstPostDescription = posts.value[0]?.description || '';
  if (normalizeComparableText(description) && normalizeComparableText(description) === normalizeComparableText(firstPostDescription)) {
    return '';
  }

  return description;
});

const galleryPhotos = computed(() => {
  return allPhotos.value.map((photo) => ({
    ...photo,
    file_path: normalizePath(photo.file_path),
    preview_path: normalizePath(photo.preview_path),
    thumbnail_path: normalizePath(photo.thumbnail_path),
  }));
});

const galleryReportSubject = computed(() => {
  if (!monument.value) return 'Фотография памятника';
  return `Фотография памятника ${monument.value.name}`;
});

const openPostReport = (post: Post) => {
  selectedPostForReport.value = post;
  postReportOpen.value = true;
};

// Post creation
const submitting = ref(false);
const error = ref('');
const aiWarnings = ref<string[]>([]);
const contentAck = ref(false);
const previewUrls = ref<string[]>([]);
const selectedFiles = ref<File[]>([]);
const badFields = reactive<Record<string, string>>({});
const validation = createValidationState();
let validateTimer: number | undefined;
const newPost = reactive({
  description: '',
});

const showAddPostModal = () => {
  if (!auth.isAuthenticated) {
    router.push('/login');
    return;
  }
  (document.getElementById('add_post_modal') as HTMLDialogElement).showModal();
};

const maybeOpenComposerFromQuery = () => {
  if (route.query.compose !== 'post' || !monument.value) return;
  showAddPostModal();
  const nextQuery = { ...route.query };
  delete nextQuery.compose;
  router.replace({ path: route.path, query: nextQuery, hash: route.hash || undefined });
};

const closeAddPost = () => {
  (document.getElementById('add_post_modal') as HTMLDialogElement).close();
};

const handleFiles = (e: Event) => {
  const files = (e.target as HTMLInputElement).files;
  if (!files) return;
  for (let i = 0; i < files.length; i++) {
    if (selectedFiles.value.length >= 10) break;
    const file = files[i];
    selectedFiles.value.push(file);
    previewUrls.value.push(URL.createObjectURL(file));
  }
  markValidationDirty();
};

const removeFile = (idx: number) => {
  selectedFiles.value.splice(idx, 1);
  URL.revokeObjectURL(previewUrls.value[idx]);
  previewUrls.value.splice(idx, 1);
  markValidationDirty();
};

const warningMap: Record<string, string> = {
  text_flagged: 'Текст поста содержит недопустимый контент',
  image_flagged: 'Некоторые фотографии не прошли проверку',
  image_filter_unavailable: 'Сервис проверки изображений временно недоступен',
  text_filter_unavailable: 'Сервис проверки текста временно недоступен',
};

warningMap.invalid_input = 'Проверьте обязательные поля формы';

const fieldMessage = (field: string, value: string) => {
  const fieldMaps: Record<string, Record<string, string>> = {
    description: {
      required: 'Это поле обязательно для заполнения',
      flagged: 'Текст не прошёл AI-модерацию',
    },
    photos: {
      min_1: 'Добавьте хотя бы одну фотографию или текст',
      max_10: 'Можно загрузить не более 10 фотографий',
    },
  };
  if (value === 'required_or_photos') return 'Нужно добавить текст поста или фотографии';
  return fieldMaps[field]?.[value] || 'Проверьте заполнение поля';
};

const syncValidationState = () => {
  aiWarnings.value = [...validation.warnings];
  Object.keys(badFields).forEach((key) => delete badFields[key]);
  Object.assign(badFields, validation.badFields);
};

const currentFingerprint = () => {
  return [buildTextFingerprint(normalizeRichTextInput(newPost.description)), buildFilesFingerprint(selectedFiles.value)].join('::');
};

const markValidationDirty = () => {
  validation.isDirtyAfterValidation = true;
  if (contentAck.value) contentAck.value = false;
  scheduleValidation();
};

const validatePost = async () => {
  if (!monument.value) return;
  const fingerprint = currentFingerprint();
  if (!fingerprint || fingerprint === validation.lastValidatedFingerprint) return;
  validation.isValidating = true;
  try {
    const formData = new FormData();
    formData.append('description', normalizeRichTextInput(newPost.description));
    selectedFiles.value.forEach((file) => formData.append('photos', file));
    const { data } = await api.post(`/monuments/${monument.value.id}/posts/validate`, formData);
    applyValidationResult(validation, data, warningMap);
    validation.lastValidatedFingerprint = fingerprint;
    validation.isDirtyAfterValidation = false;
    syncValidationState();
  } catch (err) {
    console.error('Post validation failed', err);
  } finally {
    validation.isValidating = false;
  }
};

const scheduleValidation = () => {
  if (validateTimer) window.clearTimeout(validateTimer);
  const delay = selectedFiles.value.length > 0 ? 400 : 700;
  validateTimer = window.setTimeout(() => validatePost(), delay);
};

watch(() => newPost.description, () => markValidationDirty());

const submitPost = async () => {
  if (selectedFiles.value.length === 0 && !newPost.description) {
    error.value = 'Добавьте хотя бы текст или фото.';
    return;
  }
  submitting.value = true;
  error.value = '';
  
  const formData = new FormData();
  formData.append('description', normalizeRichTextInput(newPost.description));
  formData.append('content_ack', contentAck.value.toString());
  selectedFiles.value.forEach(f => formData.append('photos', f));

  try {
    if (validation.isDirtyAfterValidation) {
      await validatePost();
    }
    await api.post(`/monuments/${monument.value?.id}/posts`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    });
    closeAddPost();
    toast.success('Пост отправлен на модерацию!');
    newPost.description = '';
    selectedFiles.value = [];
    previewUrls.value = [];
    aiWarnings.value = [];
    Object.keys(badFields).forEach((key) => delete badFields[key]);
    contentAck.value = false;
    validation.lastValidatedFingerprint = '';
    validation.isDirtyAfterValidation = false;
  } catch (err: any) {
    if (err.response?.status === 422) {
      applyValidationResult(validation, {
        requires_ack: !!err.response.data?.data?.requires_ack,
        reasons: err.response.data.data?.reasons || [],
        fields: err.response.data.fields || {},
      }, warningMap);
      syncValidationState();
    } else {
      error.value = err.response?.data?.message || 'Ошибка при сохранении';
    }
  } finally {
    submitting.value = false;
  }
};

const showOnMap = () => {
  router.push({ name: 'home', query: { lat: monument.value?.lat, lon: monument.value?.lon } });
};

const showSignalForm = () => {
  router.push({ name: 'signals', query: { monument_id: monument.value?.id } });
};

const openGallery = () => {
  if (allPhotos.value.length === 0) return;
  galleryPhotoIdx.value = Math.min(activePhotoIdx.value, allPhotos.value.length - 1);
  galleryOpen.value = true;
};

const normalizePath = (value?: string) => {
  if (!value) return '';
  return value.startsWith('/') ? value : `/${value}`;
};

const resolvePhotoSrc = (photo: Photo) => {
  return normalizePath(photo.preview_path || photo.file_path || photo.thumbnail_path);
};

const resolvePhotoThumb = (photo: Photo) => {
  return normalizePath(photo.thumbnail_path || photo.preview_path || photo.file_path);
};

const viewPhoto = (photo: Photo) => {
  const idx = allPhotos.value.findIndex(p => p.id === photo.id);
  if (idx !== -1) {
    activePhotoIdx.value = idx;
    galleryPhotoIdx.value = idx;
    galleryOpen.value = true;
  }
};
</script>

<style scoped>
.btn-white {
  background: rgba(255, 255, 255, 0.94);
  color: #2563eb;
  border: none;
}
.btn-white:hover {
  background: rgba(255, 255, 255, 0.88);
}
.btn-glass {
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.2);
}
.btn-glass:hover {
  background: rgba(255, 255, 255, 0.2);
}
</style>
