<template>
  <div ref="container" class="relative min-w-0 w-full">
    <div
      class="min-h-14 rounded-2xl border-2 border-transparent bg-base-200 px-5 transition-all focus-within:border-primary focus-within:ring-4 focus-within:ring-primary/10"
      @click="openDropdown"
    >
      <div class="flex min-w-0 items-center">
        <SearchIcon class="mr-3 h-5 w-5 shrink-0 opacity-40" />
        <input
          v-model="query"
          type="text"
          class="region-input min-w-0 w-full bg-transparent py-4 text-base font-medium"
          :placeholder="selected || 'Выберите регион...'"
          @focus="openDropdown"
          @input="onInput"
        />
        <button
          v-if="selected || query"
          type="button"
          class="btn btn-ghost btn-circle btn-xs ml-2 shrink-0"
          @click.stop="clear"
        >
          <XIcon class="h-4 w-4" />
        </button>
      </div>
    </div>

    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="scale-95 opacity-0 -translate-y-2"
      enter-to-class="scale-100 opacity-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="scale-100 opacity-100 translate-y-0"
      leave-to-class="scale-95 opacity-0 -translate-y-2"
    >
      <div
        v-if="isOpen"
        class="absolute z-[100] mt-3 max-h-64 w-full overflow-y-auto rounded-2xl border border-base-200 bg-base-100 p-2 shadow-2xl"
      >
        <div v-if="regionsLoading" class="px-4 py-8 text-center text-sm font-bold opacity-50">
          Загрузка регионов...
        </div>

        <template v-else>
          <button
            v-for="region in filteredRegions"
            :key="region"
            type="button"
            class="mb-1 block w-full rounded-xl px-4 py-3 text-left text-sm font-bold leading-snug transition-colors hover:bg-primary/10 hover:text-primary last:mb-0"
            :class="selected === region ? 'bg-primary/20 text-primary' : ''"
            @click="select(region)"
          >
            {{ region }}
          </button>

          <div v-if="filteredRegions.length === 0" class="px-4 py-8 text-center text-sm font-bold opacity-40">
            Регион не найден
          </div>
        </template>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { SearchIcon, XIcon } from 'lucide-vue-next';
import { useRegions } from '../composables/useRegions';

const props = defineProps<{
  modelValue: string;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const query = ref('');
const isOpen = ref(false);
const container = ref<HTMLElement | null>(null);
const selected = computed(() => props.modelValue);
const { regions, loading: regionsLoading, fetchRegions } = useRegions();

const filteredRegions = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase();
  if (!normalizedQuery) return regions.value;
  return regions.value.filter((region) => region.toLowerCase().includes(normalizedQuery));
});

const openDropdown = async () => {
  isOpen.value = true;
  await fetchRegions(true);
};

const select = (region: string) => {
  emit('update:modelValue', region);
  query.value = '';
  isOpen.value = false;
};

const clear = () => {
  emit('update:modelValue', '');
  query.value = '';
};

const onInput = async () => {
  isOpen.value = true;
  await fetchRegions(true);
};

const handleClickOutside = (event: MouseEvent) => {
  if (container.value && !container.value.contains(event.target as Node)) {
    isOpen.value = false;
  }
};

onMounted(async () => {
  document.addEventListener('mousedown', handleClickOutside);
  await fetchRegions(true);
});

onUnmounted(() => {
  document.removeEventListener('mousedown', handleClickOutside);
});
</script>

<style scoped>
.region-input {
  border: none !important;
  outline: none !important;
  box-shadow: none !important;
}

.region-input:focus {
  border: none !important;
  outline: none !important;
  box-shadow: none !important;
}
</style>
