<template>
  <div class="space-y-3">
    <div class="relative overflow-hidden transition-[max-height] duration-300" :style="containerStyle">
      <FormattedText ref="contentRef" :text="text" :class="contentClass" />
      <div
        v-if="canExpand && !expanded"
        class="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-base-100 via-base-100/90 to-transparent"
      />
    </div>

    <button
      v-if="canExpand"
      type="button"
      class="btn btn-ghost btn-sm rounded-xl px-0 font-black text-primary hover:bg-transparent hover:text-primary/80"
      @click="expanded = !expanded"
    >
      {{ expanded ? 'Свернуть' : 'Развернуть' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import FormattedText from './FormattedText.vue';

const props = withDefaults(defineProps<{
  text: string;
  maxHeight?: number;
  minHiddenHeight?: number;
  contentClass?: string;
}>(), {
  maxHeight: 240,
  minHiddenHeight: 32,
  contentClass: '',
});

const expanded = ref(false);
const canExpand = ref(false);
const measuredHeight = ref(0);
const contentRef = ref<any>(null);
let resizeObserver: ResizeObserver | null = null;

const measure = async () => {
  await nextTick();
  const contentElement = contentRef.value?.$el as HTMLElement | undefined;
  if (!contentElement) return;
  const fullHeight = Math.ceil(contentElement.scrollHeight);
  measuredHeight.value = fullHeight;
  const hiddenHeight = fullHeight - props.maxHeight;
  canExpand.value = hiddenHeight > props.minHiddenHeight;
  if (!canExpand.value) {
    expanded.value = false;
  }
};

const containerStyle = computed(() => {
  if (expanded.value || !canExpand.value) {
    return { maxHeight: `${measuredHeight.value || props.maxHeight}px` };
  }
  return { maxHeight: `${props.maxHeight}px` };
});

watch(() => props.text, () => {
  expanded.value = false;
  void measure();
});

onMounted(async () => {
  await measure();
  const contentElement = contentRef.value?.$el as HTMLElement | undefined;
  if (!contentElement) return;
  resizeObserver = new ResizeObserver(() => {
    void measure();
  });
  resizeObserver.observe(contentElement);
});

onBeforeUnmount(() => {
  resizeObserver?.disconnect();
  resizeObserver = null;
});
</script>
