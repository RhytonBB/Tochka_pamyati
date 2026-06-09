<template>
  <div class="relative w-full h-full overflow-hidden">
    <!-- Map Container -->
    <div id="map" class="map-container absolute inset-0 transition-transform duration-700 ease-in-out"></div>

    <!-- Search Overlay -->
    <div class="absolute top-6 left-4 right-4 xl:left-8 xl:right-auto z-20 xl:w-[24rem] 2xl:w-[26rem]">
      <div class="relative rounded-3xl border border-base-200/80 bg-base-100/95 backdrop-blur-xl shadow-[0_24px_56px_rgba(15,23,42,0.18)] overflow-visible transition-all focus-within:border-primary focus-within:ring-4 focus-within:ring-primary/10">
        <div class="absolute inset-y-0 left-0 pl-5 flex items-center pointer-events-none text-base-content/50">
          <SearchIcon class="w-5 h-5" />
        </div>
        <input 
          type="text" 
          placeholder="Поиск памятников, городов..." 
          class="search-input w-full h-16 pl-14 pr-14 bg-transparent text-[1.02rem] font-semibold placeholder:text-base-content/40 border-0"
          v-model="searchQuery"
          @input="handleSearch"
          @keydown.enter.prevent="selectFirstSearchResult"
        />
        <button
          v-if="searchQuery"
          type="button"
          class="btn btn-ghost btn-circle btn-sm absolute right-3 top-1/2 -translate-y-1/2 hover:bg-primary/10"
          @click="clearSearch"
        >
          <XIcon class="w-4 h-4 opacity-60" />
        </button>
        <div v-if="showSearchDropdown" class="absolute top-full left-0 right-0 mt-3 bg-base-100/97 backdrop-blur-xl shadow-2xl rounded-3xl border border-base-200 overflow-hidden animate-in fade-in slide-in-from-top-4">
          <ul v-if="searchResults.length > 0" class="p-2.5 space-y-1.5">
            <li v-for="res in searchResults" :key="res.id" class="w-full list-none">
              <button @click="selectMonument(res)" class="w-full py-3.5 px-4 flex items-center gap-3 hover:bg-base-200 rounded-2xl transition-colors text-left">
                <MapPinIcon class="w-5 h-5 text-primary opacity-75 shrink-0" />
                <span class="font-semibold truncate">{{ res.name }}</span>
              </button>
            </li>
          </ul>
          <div v-else-if="searchLoading" class="px-5 py-4 text-sm font-medium opacity-60">
            Ищем подходящие объекты...
          </div>
          <div v-else class="px-5 py-4 text-sm font-medium opacity-60">
            Ничего не найдено. Уточните запрос.
          </div>
        </div>
      </div>
    </div>

    <!-- Filter Buttons (Bottom Right) -->
    <div class="absolute bottom-10 right-10 flex flex-col gap-3 z-10">
      <button class="btn btn-circle btn-lg bg-base-100 shadow-2xl border-none hover:bg-base-200 transition-all" @click="recenterMap">
        <LocateIcon class="w-6 h-6 opacity-70" />
      </button>
      <button class="btn btn-primary btn-circle btn-lg shadow-2xl shadow-primary/30 border-none hover:scale-110 active:scale-95 transition-all" @click="showAddMonument">
        <PlusIcon class="w-8 h-8" />
      </button>
    </div>

    <!-- Side Panel / Bottom Sheet (Desktop/Mobile) -->
    <transition 
      enter-active-class="transition duration-500 ease-out"
      enter-from-class="translate-x-full opacity-0"
      enter-to-class="translate-x-0 opacity-100"
      leave-active-class="transition duration-400 ease-in"
      leave-from-class="translate-x-0 opacity-100"
      leave-to-class="translate-x-full opacity-0"
    >
      <div v-if="selectedMonument" class="fixed top-16 right-0 h-[calc(100dvh-64px)] w-full sm:w-[42rem] max-w-full bg-base-100/95 backdrop-blur-xl shadow-[-20px_0_50px_rgba(0,0,0,0.2)] z-40 overflow-y-auto border-l border-base-200">
        <div class="sticky top-0 p-6 flex items-center justify-between bg-base-100/50 backdrop-blur-md z-30 border-b border-base-200">
          <h2 class="text-2xl font-black tracking-tight truncate pr-4">{{ selectedMonument.name }}</h2>
          <button @click="selectedMonument = null" class="btn btn-ghost btn-circle btn-sm">
            <XIcon class="w-6 h-6 opacity-50" />
          </button>
        </div>
        
        <div class="p-8">
          <div v-if="selectedMonument.thumbnail" class="aspect-video bg-base-300 rounded-3xl overflow-hidden shadow-inner mb-8 group relative cursor-pointer" @click="openSidebarGallery(0)">
            <img :src="'/' + selectedMonument.thumbnail" class="w-full h-full object-cover transition-transform duration-700 group-hover:scale-110" />
            <div v-if="false" class="flex h-full w-full items-center justify-center bg-gradient-to-br from-base-100 to-base-200/80 p-7">
              <div class="max-w-md text-center">
                <div class="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                  <PlusIcon class="w-7 h-7" />
                </div>
                <div class="mt-4 text-xl font-black tracking-tight">Карточка ждет первый вклад</div>
                <p class="mt-2 text-sm font-medium leading-6 opacity-70">
                  У точки пока нет фотографий и подробных публикаций. Первый пост поможет наполнить карточку описанием, фактами и снимками объекта.
                </p>
                <div class="mt-4 flex flex-wrap justify-center gap-2">
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Нет фото</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Можно добавить первый пост</span>
                </div>
              </div>
            </div>
            <div class="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent opacity-0 group-hover:opacity-100 transition-opacity flex items-end p-6">
              <span class="text-white font-medium flex items-center gap-2">Смотреть все фото <ArrowRightIcon class="w-4 h-4" /></span>
            </div>
          </div>

          <div v-if="!selectedMonument.thumbnail && sidebarPosts.length === 0" class="hidden mb-8 rounded-3xl border border-base-200 bg-gradient-to-br from-base-100 to-base-200/80 p-6 shadow-sm">
            <div class="flex items-start gap-4">
              <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <PlusIcon class="w-7 h-7" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-lg font-black tracking-tight">РљР°СЂС‚РѕС‡РєР° Р¶РґРµС‚ РїРµСЂРІС‹Р№ РІРєР»Р°Рґ</div>
                <p class="mt-2 max-w-2xl text-sm font-medium leading-6 opacity-70">
                  Р­С‚Р° С‚РѕС‡РєР° СѓР¶Рµ РµСЃС‚СЊ РЅР° РєР°СЂС‚Рµ, РЅРѕ Рє РЅРµР№ РїРѕРєР° РЅРёРєС‚Рѕ РЅРµ РґРѕР±Р°РІРёР» РїРѕСЃС‚С‹ Рё С„РѕС‚РѕРіСЂР°С„РёРё. РџРµСЂРІС‹Р№ РјР°С‚РµСЂРёР°Р» РїРѕРјРѕР¶РµС‚ РЅР°РїРѕР»РЅРёС‚СЊ РєР°СЂС‚РѕС‡РєСѓ РѕРїРёСЃР°РЅРёРµРј, С„Р°РєС‚Р°РјРё Рё РёР»Р»СЋСЃС‚СЂР°С†РёСЏРјРё.
                </p>
                <div class="mt-4 flex flex-wrap gap-2">
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">РќРµС‚ С„РѕС‚Рѕ</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">РќРµС‚ РїРѕСЃС‚РѕРІ</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">РњРѕР¶РЅРѕ РґРѕР±Р°РІРёС‚СЊ РїРµСЂРІС‹Р№ РјР°С‚РµСЂРёР°Р»</span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="!selectedMonument.thumbnail && sidebarPosts.length === 0" class="hidden mb-8 rounded-3xl border border-base-200 bg-gradient-to-br from-base-100 to-base-200/80 p-6 shadow-sm">
            <div class="flex items-start gap-4">
              <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <PlusIcon class="w-7 h-7" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-lg font-black tracking-tight">Карточка ждет первый вклад</div>
                <p class="mt-2 max-w-2xl text-sm font-medium leading-6 opacity-70">
                  Эта точка уже есть на карте, но к ней пока никто не добавил посты и фотографии. Первый материал поможет наполнить карточку описанием, фактами и иллюстрациями.
                </p>
                <div class="mt-4 flex flex-wrap gap-2">
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Нет фото</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Нет постов</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Можно добавить первый материал</span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="!selectedMonument.thumbnail && sidebarPosts.length === 0" class="mb-8 rounded-3xl border border-base-200 bg-gradient-to-br from-base-100 to-base-200/80 p-6 shadow-sm">
            <div class="flex items-start gap-4">
              <div class="flex h-14 w-14 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                <PlusIcon class="w-7 h-7" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="text-lg font-black tracking-tight">Карточка ждет первый вклад</div>
                <p class="mt-2 max-w-2xl text-sm font-medium leading-6 opacity-70">
                  Эта точка уже есть на карте, но к ней пока никто не добавил посты и фотографии. Первый материал поможет наполнить карточку описанием, фактами и иллюстрациями.
                </p>
                <div class="mt-4 flex flex-wrap gap-2">
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Нет фото</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Нет постов</span>
                  <span class="badge badge-ghost rounded-xl px-3 py-3 font-bold">Можно добавить первый материал</span>
                </div>
                <button
                  @click="openAddPostForSelectedMonument"
                  class="btn btn-primary mt-5 h-12 rounded-2xl px-5 font-black shadow-lg shadow-primary/20"
                >
                  <PlusIcon class="w-5 h-5 mr-2" /> Добавить первый пост
                </button>
              </div>
            </div>
          </div>

          <div class="flex gap-4 mb-10 overflow-x-auto pb-4 no-scrollbar">
            <div class="badge badge-lg py-5 px-6 rounded-2xl bg-secondary/10 text-secondary border-none font-bold flex gap-2">
              <NavigationIcon class="w-4 h-4" /> {{ selectedMonument.lat.toFixed(4) }}, {{ selectedMonument.lon.toFixed(4) }}
            </div>
          </div>

          <!-- Sidebar Posts Feed -->
          <div v-if="sidebarPosts.length > 0" class="space-y-6 mb-10">
            <h3 class="text-[11px] font-extrabold opacity-45 uppercase tracking-[0.2em]">Посты</h3>
            <div v-for="post in sidebarPosts" :key="post.id" class="bg-base-200/50 p-5 rounded-2xl space-y-3">
              <div class="flex items-center gap-3">
                <div class="avatar placeholder">
                  <div class="bg-primary/10 text-primary rounded-lg w-8 h-8 text-xs font-black">
                    {{ post.author_name?.[0]?.toUpperCase() || 'U' }}
                  </div>
                </div>
                <div>
                  <div class="text-sm font-bold">{{ post.author_name || 'Волонтер' }}</div>
                  <div class="text-[10px] opacity-40 font-bold">{{ new Date(post.created_at).toLocaleDateString() }}</div>
                </div>
              </div>
              <FormattedText v-if="post.description" class="text-sm opacity-70 font-medium line-clamp-3" :text="post.description" />
              <div v-if="post.photos?.length" class="flex gap-2 overflow-x-auto no-scrollbar">
                <img 
                  v-for="(photo, pIdx) in post.photos" 
                  :key="photo.id" 
                  :src="photo.thumbnail_path" 
                  class="w-16 h-16 object-cover rounded-xl shadow-sm cursor-pointer hover:ring-2 hover:ring-primary transition-all shrink-0" 
                  @click="openSidebarGalleryForPost(post, Number(pIdx))" 
                />
              </div>
            </div>
          </div>

          <div class="flex flex-col gap-4">
            <router-link :to="'/monument/' + selectedMonument.id" class="btn btn-primary btn-lg rounded-2xl shadow-lg shadow-primary/20 h-16 font-bold text-lg group">
              Подробнее в карточке
              <ArrowRightIcon class="w-5 h-5 ml-2 transition-transform group-hover:translate-x-1" />
            </router-link>
            <button @click="showSignalForm(selectedMonument)" class="btn btn-outline btn-lg rounded-2xl h-16 border-base-300 hover:bg-base-200 hover:text-base-content hover:border-base-300 transition-all font-bold">
              <ShieldAlertIcon class="w-5 h-5 mr-2" /> Сообщить об угрозе
            </button>
          </div>
        </div>
      </div>
    </transition>

    <!-- Add Monument Modal -->
    <dialog id="add_monument_modal" class="modal modal-bottom sm:modal-middle bg-base-300/30 backdrop-blur-xl">
      <div class="modal-box w-[min(94vw,980px)] max-w-none rounded-[3rem] p-0 border border-base-200 shadow-2xl animate-in zoom-in duration-300 flex flex-col max-h-[90vh] relative">
        <button
          type="button"
          @click="closeAddMonument"
          class="btn btn-circle btn-sm absolute top-5 right-5 z-20 bg-base-100/85 border-base-300 hover:bg-base-200"
          aria-label="Закрыть форму"
        >
          <XIcon class="w-4 h-4" />
        </button>
        <div class="p-10 overflow-y-auto custom-scrollbar flex-grow">
        <div class="bg-primary/10 p-4 rounded-2xl inline-flex text-primary mb-6">
          <PlusIcon class="w-10 h-10" />
        </div>
        <h3 class="text-3xl font-black tracking-tight mb-4">Добавить памятник</h3>
        <p class="opacity-60 font-medium mb-8">Заполните данные о новом памятнике. Выберите точку на карте или укажите координаты.</p>
        
        <form @submit.prevent="submitMonument" class="space-y-6">
          <div class="form-control w-full gap-3">
            <label class="label font-bold opacity-70 text-base pb-0">Название</label>
            <input v-model="newMonument.name" type="text" class="input w-full rounded-2xl h-14 bg-base-200 border-2 font-semibold text-base px-5 transition-colors focus:outline-none focus:ring-4 focus:ring-primary/10" :class="badFields.name ? 'border-error' : 'border-transparent focus:border-primary'" />
            <span v-if="badFields.name" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('name', badFields.name) }}</span>
          </div>

          <button type="button" @click="selectLocationOnMap" class="btn btn-primary btn-lg rounded-2xl h-16 font-black w-full text-lg shadow-lg shadow-primary/20 hover:scale-[1.01] transition-transform">
            <MapPinIcon class="w-6 h-6 mr-3" /> Выбрать точку на карте
          </button>

          <div class="grid grid-cols-2 gap-4">
            <div class="form-control w-full gap-3">
              <label class="label font-bold opacity-50 text-xs uppercase tracking-wider">Долгота</label>
              <input v-model.number="newMonument.lon" type="number" step="any" class="input w-full rounded-2xl h-14 bg-base-200 border-2 font-semibold text-base px-5 transition-colors focus:outline-none focus:ring-4 focus:ring-primary/10" :class="badFields.lon ? 'border-error' : 'border-transparent focus:border-primary'" />
              <span v-if="badFields.lon" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('lon', badFields.lon) }}</span>
            </div>
            <div class="form-control w-full gap-3">
              <label class="label font-bold opacity-50 text-xs uppercase tracking-wider">Широта</label>
              <input v-model.number="newMonument.lat" type="number" step="any" class="input w-full rounded-2xl h-14 bg-base-200 border-2 font-semibold text-base px-5 transition-colors focus:outline-none focus:ring-4 focus:ring-primary/10" :class="badFields.lat ? 'border-error' : 'border-transparent focus:border-primary'" />
              <span v-if="badFields.lat" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('lat', badFields.lat) }}</span>
            </div>
          </div>

          <button
            type="button"
            class="flex w-full items-center justify-between rounded-[2rem] border border-base-200 bg-base-200/30 px-6 py-5 text-left transition hover:border-primary/30 hover:bg-primary/5"
            @click="newMonument.createPost = !newMonument.createPost"
          >
            <div>
              <div class="font-black text-lg">Добавить первый пост к точке</div>
              <div class="mt-1 text-sm font-medium opacity-60">Если блок не раскрывать, будет создана только точка с названием и меткой на карте.</div>
            </div>
            <div class="ml-4 flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-base-300 bg-base-100 text-xl font-black">
              {{ newMonument.createPost ? '−' : '+' }}
            </div>
          </button>
          <div v-if="false" class="rounded-[2rem] border border-base-200 bg-base-200/40 p-5">
            <label class="flex items-center justify-between gap-4">
              <div>
                <div class="font-black">Сразу добавить первый пост</div>
                <div class="mt-1 text-sm font-medium opacity-60">Если режим выключен, будет создана только точка. Пост можно добавить позже отдельно.</div>
              </div>
              <input v-model="newMonument.createPost" type="checkbox" class="toggle toggle-primary toggle-lg" />
            </label>
          </div>

          <div v-if="newMonument.createPost" class="space-y-6 rounded-[2rem] border border-primary/10 bg-primary/5 p-6">
            <div class="form-control w-full gap-3">
              <label class="label font-bold opacity-70 text-base pb-0">Текст первого поста</label>
              <RichTextEditor v-model="newMonument.description" :invalid="!!badFields.description" placeholder="Напишите первую заметку об объекте: что это за место, в каком оно состоянии и что важно зафиксировать..." />
              <span v-if="badFields.description" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('description', badFields.description) }}</span>
            </div>

            <div class="form-control w-full gap-3">
              <label class="label font-bold opacity-70 mb-1 text-base">Фотографии поста (до 10)</label>
              <input type="file" multiple @change="handleFiles" class="file-input file-input-bordered file-input-primary w-full rounded-2xl h-14 bg-base-200 border-none text-base" accept="image/*" />
              <span v-if="badFields.photos" class="block text-error text-sm font-bold leading-snug">{{ fieldMessage('photos', badFields.photos) }}</span>
              <div class="flex gap-3 mt-4 overflow-x-auto py-2">
                <div v-for="(file, idx) in previewUrls" :key="idx" class="relative shrink-0">
                  <img :src="file" class="w-20 h-20 object-cover rounded-xl shadow-md" :class="badPhotos.includes(idx) ? 'ring-4 ring-error' : ''" />
                  <button @click.prevent="removeFile(idx)" class="btn btn-circle btn-xs btn-error absolute top-1 right-1 z-10 shadow-md">
                    <XIcon class="w-3 h-3" />
                  </button>
                  <div v-if="badPhotos.includes(idx)" class="absolute bottom-0 inset-x-0 bg-error text-white text-[8px] font-bold text-center py-0.5 rounded-b-xl">AI</div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="aiWarnings.length > 0" class="alert alert-warning bg-warning/10 text-warning border-none rounded-2xl p-6 flex flex-col gap-4">
            <div class="flex gap-3">
              <AlertTriangleIcon class="w-6 h-6" />
              <div class="font-bold">Обнаружены потенциальные проблемы:</div>
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
            <button type="button" @click="closeAddMonument" class="btn btn-ghost flex-grow h-14 rounded-2xl font-bold">Отмена</button>
            <button type="submit" class="btn btn-primary flex-[2] h-14 rounded-2xl font-black text-white shadow-xl shadow-primary/20" :disabled="loading">
              <span v-if="loading" class="loading loading-spinner"></span>
              <span v-else>Отправить</span>
            </button>
          </div>
        </form>
        </div>
      </div>
    </dialog>

    <!-- Sidebar Photo Gallery -->
    <PhotoGallery
      :photos="sidebarAllPhotos"
      :initial-index="sidebarGalleryIdx"
      v-model="sidebarGalleryOpen"
      :show-captions="true"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, shallowRef, reactive, computed, watch } from 'vue';
import maplibregl from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import api from '../api';
import { 
  SearchIcon, MapPinIcon, LocateIcon, PlusIcon, XIcon, 
  ArrowRightIcon, NavigationIcon, ShieldAlertIcon,
  AlertTriangleIcon, AlertCircleIcon
} from 'lucide-vue-next';
import { useAuthStore } from '../store/auth';
import { useRouter } from 'vue-router';
import { useToast } from '../composables/useToast';
import PhotoGallery from '../components/PhotoGallery.vue';
import RichTextEditor from '../components/RichTextEditor.vue';
import FormattedText from '../components/FormattedText.vue';
import { applyValidationResult, buildFilesFingerprint, buildTextFingerprint, createValidationState } from '../composables/useContentValidation';
import { normalizeRichTextInput } from '../utils/richText';

const auth = useAuthStore();
const router = useRouter();
const toast = useToast();

const searchQuery = ref('');
const searchResults = ref<any[]>([]);
const searchLoading = ref(false);
const showSearchDropdown = computed(() => searchQuery.value.trim().length >= 2);
let searchTimer: number | undefined;
const selectedMonument = ref<any>(null);
const map = shallowRef<maplibregl.Map | null>(null);

const loading = ref(false);
const error = ref('');
const aiWarnings = ref<string[]>([]);
const contentAck = ref(false);
const previewUrls = ref<string[]>([]);
const selectedFiles = ref<File[]>([]);
const badFields = reactive<Record<string, string>>({});
const badPhotos = ref<number[]>([]);
const validation = createValidationState();
let validateTimer: number | undefined;

const newMonument = reactive({
  name: '',
  lon: 37.6173,
  lat: 55.7558,
  description: '',
  createPost: false,
});

onMounted(() => {
  map.value = new maplibregl.Map({
    container: 'map',
    style: {
      version: 8,
      sources: {
        'yandex-tiles': {
          type: 'raster',
          tiles: ['https://core-renderer-tiles.maps.yandex.net/tiles?l=map&x={x}&y={y}&z={z}&scale=1&lang=ru_RU'],
          tileSize: 256,
        }
      },
      layers: [
        {
          id: 'yandex-layer',
          type: 'raster',
          source: 'yandex-tiles',
          minzoom: 0,
          maxzoom: 19,
        }
      ]
    },
    center: [37.6173, 55.7558], // Moscow
    zoom: 10,
  });

  map.value.on('load', () => {
    // Add MVT source for monuments
    map.value?.addSource('monuments', {
      type: 'vector',
      tiles: [`${window.location.origin}/api/v1/tiles/monuments/{z}/{x}/{y}.mvt`],
      minzoom: 0,
      maxzoom: 14,
    });

    map.value?.addLayer({
      id: 'monuments-layer-glow',
      type: 'circle',
      source: 'monuments',
      'source-layer': 'monuments',
      paint: {
        'circle-radius': [
          'interpolate', ['linear'], ['zoom'],
          5, 10,
          10, 20,
          15, 30
        ],
        'circle-color': '#e11d48',
        'circle-opacity': 0.3,
        'circle-blur': 0.5,
      },
    });

    map.value?.addLayer({
      id: 'monuments-layer',
      type: 'circle',
      source: 'monuments',
      'source-layer': 'monuments',
      paint: {
        'circle-radius': [
          'interpolate', ['linear'], ['zoom'],
          5, 5,
          10, 8,
          15, 12
        ],
        'circle-color': '#e11d48',
        'circle-stroke-width': 3,
        'circle-stroke-color': '#ffffff',
        'circle-stroke-opacity': 0.9,
      },
    });

    // Click handler for points
    map.value?.on('click', 'monuments-layer', (e) => {
      if (e.features && e.features.length > 0) {
        const feature = e.features[0];
        const props = feature.properties;
        fetchMonumentDetails(props.id);
      }
    });

    // Change cursor on hover
    map.value?.on('mouseenter', 'monuments-layer', () => {
      if (map.value) map.value.getCanvas().style.cursor = 'pointer';
    });
    map.value?.on('mouseleave', 'monuments-layer', () => {
      if (map.value) map.value.getCanvas().style.cursor = '';
    });

    // Click on map to set coordinates when adding
    map.value?.on('click', (e) => {
      if (isPickingPoint.value) {
        newMonument.lon = e.lngLat.lng;
        newMonument.lat = e.lngLat.lat;
        isPickingPoint.value = false;
        (document.getElementById('add_monument_modal') as HTMLDialogElement).showModal();
      }
    });
  });
});

const isPickingPoint = ref(false);

const fetchMonumentDetails = async (id: string) => {
  try {
    const { data } = await api.get(`/monuments/${id}`);
    selectedMonument.value = {
      ...data.monument,
      thumbnail: data.photos?.length > 0 ? data.photos[0].thumbnail_path : null,
      photos: data.photos || []
    };
    sidebarPosts.value = data.posts || [];
  } catch (err) {
    console.error('Failed to fetch monument details', err);
  }
};

const handleSearch = async () => {
  if (searchTimer) window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(async () => {
    const q = searchQuery.value.trim();
    if (q.length < 2) {
      searchResults.value = [];
      searchLoading.value = false;
      return;
    }
    searchLoading.value = true;
    try {
      const { data } = await api.get('/search/suggest', { params: { q, limit: 8 } });
      searchResults.value = Array.isArray(data) ? data : (data.items || []);
    } catch (err) {
      searchResults.value = [];
    } finally {
      searchLoading.value = false;
    }
  }, 220);
};

const selectMonument = (mon: any) => {
  fetchMonumentDetails(mon.id);
  clearSearch();
  map.value?.flyTo({
    center: [Number(mon.lon), Number(mon.lat)],
    zoom: 15,
    speed: 1.5,
    curve: 1.2
  });
};

const selectFirstSearchResult = () => {
  if (!searchResults.value.length) return;
  selectMonument(searchResults.value[0]);
};

const clearSearch = () => {
  searchQuery.value = '';
  searchResults.value = [];
  searchLoading.value = false;
};

const recenterMap = () => {
  map.value?.flyTo({ center: [37.6173, 55.7558], zoom: 10 });
};

const showAddMonument = () => {
  if (!auth.isAuthenticated) {
    router.push('/login');
    return;
  }
  (document.getElementById('add_monument_modal') as HTMLDialogElement).showModal();
};

const closeAddMonument = () => {
  (document.getElementById('add_monument_modal') as HTMLDialogElement).close();
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
  invalid_input: 'Проверьте обязательные поля формы',
  text_flagged: 'Описание содержит недопустимый контент',
  image_flagged: 'Некоторые фотографии не прошли проверку',
  possible_duplicate: 'Похоже, такой памятник уже есть рядом',
  image_filter_unavailable: 'Сервис проверки изображений временно недоступен',
  text_filter_unavailable: 'Сервис проверки текста временно недоступен',
};

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
  if (value === 'required_or_photos') return 'Нужно добавить текст первого поста или фотографии';
  return fieldMaps[field]?.[value] || 'Проверьте заполнение поля';
};

const syncValidationState = () => {
  aiWarnings.value = [...validation.warnings];
  Object.keys(badFields).forEach((key) => delete badFields[key]);
  Object.assign(badFields, validation.badFields);
  badPhotos.value = [...validation.badPhotos];
};

const currentFingerprint = () => {
  return [
    newMonument.name.trim(),
    newMonument.lon,
    newMonument.lat,
    newMonument.createPost,
    buildTextFingerprint(normalizeRichTextInput(newMonument.description)),
    buildFilesFingerprint(selectedFiles.value),
  ].join('::');
};

const markValidationDirty = () => {
  validation.isDirtyAfterValidation = true;
  if (contentAck.value) {
    contentAck.value = false;
  }
  scheduleValidation();
};

const validateMonument = async () => {
  const fingerprint = currentFingerprint();
  if (!fingerprint || fingerprint === validation.lastValidatedFingerprint) {
    return;
  }
  validation.isValidating = true;
  try {
    const formData = new FormData();
    formData.append('name', newMonument.name);
    formData.append('lon', newMonument.lon.toString());
    formData.append('lat', newMonument.lat.toString());
    formData.append('create_post_with_monument', String(newMonument.createPost));
    if (newMonument.createPost) {
      formData.append('description', normalizeRichTextInput(newMonument.description));
      selectedFiles.value.forEach((file) => formData.append('photos', file));
    }
    const { data } = await api.post('/monuments/validate', formData);
    applyValidationResult(validation, data, warningMap);
    validation.lastValidatedFingerprint = fingerprint;
    validation.isDirtyAfterValidation = false;
    syncValidationState();
  } catch (err) {
    console.error('Monument validation failed', err);
  } finally {
    validation.isValidating = false;
  }
};

const scheduleValidation = () => {
  if (validateTimer) window.clearTimeout(validateTimer);
  const delay = selectedFiles.value.length > 0 ? 400 : 700;
  validateTimer = window.setTimeout(() => {
    validateMonument();
  }, delay);
};

watch(() => [newMonument.name, newMonument.lon, newMonument.lat, newMonument.description, newMonument.createPost], () => {
  markValidationDirty();
}, { deep: true });

const submitMonument = async () => {
  if (newMonument.createPost && selectedFiles.value.length === 0 && !normalizeRichTextInput(newMonument.description)) {
    error.value = 'Добавьте текст или фотографии для первого поста.';
    return;
  }
  loading.value = true;
  error.value = '';
  
  const formData = new FormData();
  formData.append('name', newMonument.name);
  formData.append('lon', newMonument.lon.toString());
  formData.append('lat', newMonument.lat.toString());
  formData.append('create_post_with_monument', String(newMonument.createPost));
  formData.append('content_ack', contentAck.value.toString());
  if (newMonument.createPost) {
    formData.append('description', normalizeRichTextInput(newMonument.description));
    selectedFiles.value.forEach(f => formData.append('photos', f));
  }

  // Clear field highlights
  Object.keys(badFields).forEach(k => delete badFields[k]);
  badPhotos.value = [];
  try {
    if (validation.isDirtyAfterValidation) {
      await validateMonument();
    }
    await api.post('/monuments', formData);
    closeAddMonument();
    toast.success(newMonument.createPost ? 'Точка и первый пост отправлены на модерацию.' : 'Точка отправлена на модерацию.');
    Object.assign(newMonument, { name: '', description: '', createPost: false });
    selectedFiles.value = [];
    previewUrls.value = [];
    aiWarnings.value = [];
    contentAck.value = false;
    validation.lastValidatedFingerprint = '';
    validation.isDirtyAfterValidation = false;
  } catch (err: any) {
    if (err.response?.status === 422) {
      applyValidationResult(validation, {
        requires_ack: true,
        reasons: err.response.data.data?.reasons || [],
        fields: err.response.data.fields || {},
      }, warningMap);
      syncValidationState();
    } else {
      error.value = err.response?.data?.message || 'Ошибка при сохранении';
    }
  } finally {
    loading.value = false;
  }
};

const showSignalForm = (mon: any) => {
  router.push({ name: 'signals', query: { monument_id: mon.id } });
};

const openAddPostForSelectedMonument = () => {
  if (!selectedMonument.value?.id) return;
  router.push({ path: `/monument/${selectedMonument.value.id}`, query: { compose: 'post' } });
};

const selectLocationOnMap = () => {
  closeAddMonument();
  isPickingPoint.value = true;
  toast.info('Кликните по карте, чтобы выбрать точку');
};

// Sidebar posts feed
const sidebarPosts = ref<any[]>([]);

// Sidebar gallery
const sidebarGalleryOpen = ref(false);
const sidebarGalleryIdx = ref(0);

const sidebarAllPhotos = computed(() => {
  const seen = new Set<string>();
  const photos: any[] = [];
  
  if (selectedMonument.value?.photos) {
    for (const p of selectedMonument.value.photos) {
      const key = p.preview_path || p.thumbnail_path;
      if (key && !seen.has(key)) {
        seen.add(key);
        photos.push({
          ...p,
          caption: p.description || '',
          file_path: ('/' + (p.file_path || p.preview_path)).replace('//', '/'),
          preview_path: p.preview_path ? ('/' + p.preview_path).replace('//', '/') : undefined,
          thumbnail_path: p.thumbnail_path ? ('/' + p.thumbnail_path).replace('//', '/') : undefined,
        });
      }
    }
  }
  
  sidebarPosts.value.forEach((post: any) => {
    if (post.photos) {
      for (const p of post.photos) {
        const key = p.preview_path || p.thumbnail_path;
        if (key && !seen.has(key)) {
          seen.add(key);
          photos.push({
            ...p,
            caption: post.description || '',
            file_path: ('/' + (p.file_path || p.preview_path)).replace('//', '/'),
            preview_path: p.preview_path ? ('/' + p.preview_path).replace('//', '/') : undefined,
            thumbnail_path: p.thumbnail_path ? ('/' + p.thumbnail_path).replace('//', '/') : undefined,
          });
        }
      }
    }
  });
  
  return photos;
});

const openSidebarGallery = (idx: number) => {
  selectedMonument.value = null;
  sidebarGalleryIdx.value = idx;
  sidebarGalleryOpen.value = true;
};

const openSidebarGalleryForPost = (post: any, photoIdxInPost: number) => {
  const photo = post.photos?.[photoIdxInPost];
  if (!photo) return;
  const globalIdx = sidebarAllPhotos.value.findIndex((p: any) => 
    p.preview_path === photo.preview_path || p.thumbnail_path === photo.thumbnail_path
  );
  openSidebarGallery(globalIdx !== -1 ? globalIdx : 0);
};
</script>

<style scoped>
.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
  margin: 20px 0;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background-color: rgba(0,0,0,0.1);
  border-radius: 20px;
}

.search-input {
  outline: none;
  box-shadow: none;
  -webkit-appearance: none;
  appearance: none;
}

.search-input:focus,
.search-input:focus-visible,
.search-input:focus-within {
  outline: none !important;
  box-shadow: none !important;
}
</style>
