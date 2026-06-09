<template>
  <div class="space-y-3">
    <div class="flex items-center gap-2">
      <button type="button" class="btn btn-sm rounded-xl font-black" :class="isBold ? 'btn-primary text-primary-content' : 'btn-ghost'" @mousedown.prevent @click="applyCommand('bold')" title="Жирный">
        B
      </button>
      <button type="button" class="btn btn-sm rounded-xl italic font-black" :class="isItalic ? 'btn-primary text-primary-content' : 'btn-ghost'" @mousedown.prevent @click="applyCommand('italic')" title="Курсив">
        I
      </button>
      <button type="button" class="btn btn-sm rounded-xl underline font-black" :class="isUnderline ? 'btn-primary text-primary-content' : 'btn-ghost'" @mousedown.prevent @click="applyCommand('underline')" title="Подчёркнутый">
        U
      </button>
    </div>

    <div
      ref="editorRef"
      contenteditable="true"
      class="editor textarea w-full rounded-3xl min-h-72 bg-base-200 border-2 font-medium text-base leading-7 p-6 resize-y shadow-inner transition-colors focus:outline-none focus:ring-4"
      :class="invalid ? 'border-error focus:ring-error/10' : 'border-transparent focus:border-primary focus:ring-primary/10'"
      :data-placeholder="placeholder || 'Введите текст...'"
      @input="onEditorInput"
      @keydown="onEditorKeydown"
      @mouseup="updateToolbarState"
      @keyup="updateToolbarState"
      @blur="updateToolbarState"
    ></div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue';
import { editorHtmlToRichText, renderRichTextForEditor } from '../utils/richText';

const props = defineProps<{
  modelValue: string;
  placeholder?: string;
  invalid?: boolean;
}>();

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void;
}>();

const editorRef = ref<HTMLDivElement | null>(null);
const isBold = ref(false);
const isItalic = ref(false);
const isUnderline = ref(false);
let isSyncingFromModel = false;
let lastEmittedValue = '';

const syncEditorFromModel = () => {
  const el = editorRef.value;
  if (!el) return;
  const nextHtml = renderRichTextForEditor(props.modelValue || '');
  if (el.innerHTML === nextHtml) return;
  isSyncingFromModel = true;
  el.innerHTML = nextHtml;
  isSyncingFromModel = false;
};

const onEditorInput = () => {
  if (isSyncingFromModel) return;
  const el = editorRef.value;
  if (!el) return;
  const serialized = editorHtmlToRichText(el.innerHTML);
  lastEmittedValue = serialized;
  emit('update:modelValue', serialized);
  updateToolbarState();
};

const applyCommand = (command: 'bold' | 'italic' | 'underline') => {
  const el = editorRef.value;
  if (!el) return;
  el.focus();
  document.execCommand(command);
  updateToolbarState();
  requestAnimationFrame(() => onEditorInput());
};

const onEditorKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Tab') {
    event.preventDefault();
    document.execCommand('insertText', false, '  ');
    onEditorInput();
  }
};

const updateToolbarState = () => {
  isBold.value = document.queryCommandState('bold');
  isItalic.value = document.queryCommandState('italic');
  isUnderline.value = document.queryCommandState('underline');
};

watch(() => props.modelValue, async () => {
  if (props.modelValue === lastEmittedValue) return;
  await nextTick();
  syncEditorFromModel();
});

onMounted(() => {
  lastEmittedValue = props.modelValue || '';
  syncEditorFromModel();
  updateToolbarState();
});
</script>

<style scoped>
.editor:empty::before {
  content: attr(data-placeholder);
  color: rgba(100, 116, 139, 0.7);
  pointer-events: none;
}
</style>
