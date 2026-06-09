<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-all duration-300 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-all duration-200 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="isOpen"
        style="z-index: 2147483647;"
        class="fixed inset-0 flex flex-col items-center justify-center bg-black/90 backdrop-blur-xl"
        @click.self="close"
      >
        <!-- Close -->
        <button
          @click="close"
          style="z-index: 2147483648;"
          class="absolute top-6 right-6 btn btn-circle btn-lg bg-white/10 border-white/10 text-white hover:bg-white/20 backdrop-blur-md shadow-2xl"
        >
          <XIcon class="w-7 h-7" />
        </button>

        <button
          v-if="allowReport && currentPhoto.id"
          @click.stop="reportOpen = true"
          style="z-index: 2147483648;"
          class="absolute top-6 left-6 btn btn-circle btn-lg bg-white/10 border-white/10 text-white hover:bg-white/20 backdrop-blur-md shadow-2xl"
        >
          <FlagIcon class="w-6 h-6" />
        </button>

        <!-- Counter -->
        <div style="z-index: 2147483648;" class="absolute top-8 left-1/2 -translate-x-1/2 text-white/70 font-bold text-lg">
          {{ currentIdx + 1 }} / {{ photos.length }}
        </div>

        <!-- Previous -->
        <button
          v-if="photos.length > 1"
          @click.stop="prev"
          style="z-index: 2147483648;"
          class="absolute left-4 top-1/2 -translate-y-1/2 btn btn-circle btn-lg bg-white/10 border-white/10 text-white hover:bg-white/20 backdrop-blur-md shadow-2xl"
        >
          <ChevronLeftIcon class="w-8 h-8" />
        </button>

        <!-- Next -->
        <button
          v-if="photos.length > 1"
          @click.stop="next"
          style="z-index: 2147483648;"
          class="absolute right-4 top-1/2 -translate-y-1/2 btn btn-circle btn-lg bg-white/10 border-white/10 text-white hover:bg-white/20 backdrop-blur-md shadow-2xl"
        >
          <ChevronRightIcon class="w-8 h-8" />
        </button>

        <!-- Image -->
        <div class="relative max-w-[92vw] max-h-[85vh] flex items-center justify-center">
          <Transition
            enter-active-class="transition-all duration-200 ease-out"
            enter-from-class="opacity-0 scale-95"
            enter-to-class="opacity-100 scale-100"
            leave-active-class="transition-all duration-150 ease-in"
            leave-from-class="opacity-100 scale-100"
            leave-to-class="opacity-0 scale-95"
            mode="out-in"
          >
            <img
              :key="currentIdx"
              :src="(() => {
                const path = currentPhoto.file_path || currentPhoto.preview_path || currentPhoto.thumbnail_path;
                if (!path) return '';
                return (path.startsWith('/') ? path : '/' + path).replace('//', '/');
              })()"
              class="max-w-[92vw] max-h-[85vh] object-contain rounded-2xl shadow-2xl select-none"
              draggable="false"
            />
          </Transition>
        </div>

        <!-- Caption (optional, only shown when showCaptions is true) -->
        <div
          v-if="showCaptions && currentCaption && false"
          style="z-index: 2147483648;"
          class="absolute bottom-8 left-1/2 -translate-x-1/2 max-w-2xl w-full px-6 max-h-32 overflow-y-auto"
        >
          <div class="bg-black/60 backdrop-blur-xl text-white px-8 py-5 rounded-2xl text-center font-medium text-base leading-relaxed border border-white/10 shadow-2xl line-clamp-3">
            {{ currentCaption }}
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
  <ReportDialog
    v-model="reportOpen"
    entity-type="photo"
    :entity-id="String(currentPhoto.id || '')"
    :subject="reportSubject"
    :duplicate-seed="reportDuplicateSeed"
    @submitted="emit('reported')"
  />
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
import { XIcon, ChevronLeftIcon, ChevronRightIcon, FlagIcon } from 'lucide-vue-next';
import ReportDialog from './ReportDialog.vue';

interface GalleryPhoto {
  id?: string;
  file_path?: string;
  preview_path?: string;
  thumbnail_path?: string;
  caption?: string;
}

const props = defineProps<{
  photos: GalleryPhoto[];
  initialIndex?: number;
  modelValue: boolean;
  showCaptions?: boolean;
  allowReport?: boolean;
  reportSubject?: string;
  reportDuplicateSeed?: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void;
  (e: 'reported'): void;
}>();

const isOpen = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
});

const currentIdx = ref(props.initialIndex || 0);

watch(() => props.initialIndex, (v) => {
  if (v !== undefined) currentIdx.value = v;
});

watch(() => props.modelValue, (open) => {
  if (open && props.initialIndex !== undefined) {
    currentIdx.value = props.initialIndex;
  }
});

const currentPhoto = computed(() => props.photos[currentIdx.value] || {});
const currentCaption = computed(() => currentPhoto.value?.caption || '');
const reportOpen = ref(false);

const prev = () => {
  currentIdx.value = (currentIdx.value - 1 + props.photos.length) % props.photos.length;
};

const next = () => {
  currentIdx.value = (currentIdx.value + 1) % props.photos.length;
};

const close = () => {
  isOpen.value = false;
};

const handleKeydown = (e: KeyboardEvent) => {
  if (!isOpen.value) return;
  if (e.key === 'Escape') close();
  if (e.key === 'ArrowLeft') prev();
  if (e.key === 'ArrowRight') next();
};

onMounted(() => window.addEventListener('keydown', handleKeydown));
onUnmounted(() => window.removeEventListener('keydown', handleKeydown));
</script>
