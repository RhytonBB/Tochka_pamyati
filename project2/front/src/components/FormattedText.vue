<template>
  <div class="whitespace-normal" v-html="html"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { renderRichText } from '../utils/richText';

const props = defineProps<{
  text?: string;
}>();

const html = computed(() => renderRichText(props.text || ''));
</script>
