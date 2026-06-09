<template>
  <div class="comment-thread relative" :class="{ 'ml-8 pl-4 border-l-2 border-base-200 mt-4': depth > 0 }">
    <div v-if="!comment.is_hidden || showHidden" class="bg-base-100 p-4 sm:p-5 rounded-2xl shadow-sm border border-base-200" :class="{ 'opacity-60 bg-error/5 border-error/20': comment.is_hidden }">
      <div class="flex items-center justify-between mb-3">
        <div class="flex items-center gap-3">
          <div class="avatar placeholder">
            <div class="bg-neutral text-neutral-content rounded-full w-8 h-8 text-xs font-bold shadow-inner">
              {{ comment.author_name?.[0]?.toUpperCase() || 'U' }}
            </div>
          </div>
          <div>
            <div class="font-bold text-sm leading-tight">{{ comment.author_name || 'Волонтер' }}</div>
            <div class="text-[10px] opacity-50 font-bold uppercase tracking-wider">{{ new Date(comment.created_at).toLocaleString() }}</div>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <div v-if="comment.is_hidden" class="badge badge-error badge-sm font-bold uppercase text-[9px] tracking-wider">Скрыто AI</div>
          <div v-if="auth.isAuthenticated" class="dropdown dropdown-end">
            <button tabindex="0" class="btn btn-ghost btn-xs btn-circle">
              <MoreHorizontalIcon class="w-4 h-4 opacity-60" />
            </button>
            <ul tabindex="0" class="dropdown-content menu menu-sm mt-2 z-[2] p-2 shadow-xl bg-base-100 rounded-2xl w-48 border border-base-200">
              <li><button @click="reportOpen = true">Пожаловаться</button></li>
            </ul>
          </div>
        </div>
      </div>

      <p class="text-sm font-medium leading-relaxed opacity-90 break-words whitespace-pre-wrap">{{ comment.content }}</p>

      <div class="mt-3 flex gap-4">
        <button v-if="!hideReply" @click="isReplying = !isReplying" class="text-xs font-bold text-primary opacity-80 hover:opacity-100 hover:underline flex items-center gap-1 transition-all">
          <MessageCircleIcon class="w-3 h-3" /> Ответить
        </button>
      </div>
    </div>
    <div v-else class="bg-base-200/50 p-3 rounded-2xl border border-base-300 border-dashed text-xs font-bold opacity-50 flex justify-between items-center">
      <span>Комментарий скрыт модерацией (AI)</span>
      <button @click="showHidden = true" class="btn btn-xs btn-ghost text-primary">Показать</button>
    </div>

    <ReportDialog
      v-model="reportOpen"
      entity-type="comment"
      :entity-id="comment.id"
      :subject="`Комментарий от ${comment.author_name || 'пользователя'}`"
    />

    <div v-if="isReplying" class="mt-3 bg-base-100 p-4 rounded-2xl border border-primary/20 shadow-lg animate-in fade-in slide-in-from-top-2">
      <form @submit.prevent="submitReply">
        <textarea v-model="replyText" class="textarea border-none bg-base-200 w-full h-20 rounded-xl focus:ring-2 ring-primary p-3 text-sm resize-none font-medium" placeholder="Напишите ответ..." required autofocus></textarea>
        <div class="flex justify-end gap-2 mt-2">
          <button type="button" @click="isReplying = false" class="btn btn-xs btn-ghost rounded-lg font-bold">Отмена</button>
          <button type="submit" class="btn btn-xs btn-primary rounded-lg text-white font-bold px-4" :disabled="isSubmitting">
            <span v-if="isSubmitting" class="loading loading-spinner w-3 h-3"></span>
            <span v-else>Отправить</span>
          </button>
        </div>
      </form>
    </div>

    <div v-if="replies && replies.length > 0" class="replies-container">
      <div v-if="isCollapsed" class="mt-3 flex items-center gap-2 text-xs font-bold opacity-60 hover:opacity-100 cursor-pointer transition-opacity" @click="isCollapsed = false">
        <div class="h-px bg-base-300 flex-grow"></div>
        <span class="bg-base-200 px-3 py-1 rounded-full"><ChevronDownIcon class="w-3 h-3 inline-block align-text-bottom mr-1" /> Показать {{ replies.length }} ответов</span>
        <div class="h-px bg-base-300 flex-grow"></div>
      </div>

      <div v-else>
        <CommentThread
          v-for="reply in replies"
          :key="reply.comment.id"
          :comment="reply.comment"
          :replies="reply.replies"
          :depth="depth + 1"
          @reply="emitReply"
        />
        <div v-if="depth >= 2 && !isCollapsed" class="mt-3 flex items-center gap-2 text-[10px] font-bold opacity-40 hover:opacity-100 cursor-pointer transition-opacity" @click="isCollapsed = true">
          <ChevronUpIcon class="w-3 h-3" /> Скрыть ответы
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { ChevronDownIcon, ChevronUpIcon, MessageCircleIcon, MoreHorizontalIcon } from 'lucide-vue-next';
import type { SignalComment } from '../types';
import { useAuthStore } from '../store/auth';
import ReportDialog from './ReportDialog.vue';

interface CommentNode {
  comment: SignalComment;
  replies: CommentNode[];
}

const props = defineProps<{
  comment: SignalComment;
  replies?: CommentNode[];
  depth?: number;
}>();

const emit = defineEmits<{
  (e: 'reply', parentId: string, content: string, cb: () => void): void;
}>();

const auth = useAuthStore();
const depth = computed(() => props.depth || 0);
const isCollapsed = ref(depth.value > 1 && (props.replies?.length || 0) > 0);
const isReplying = ref(false);
const replyText = ref('');
const showHidden = ref(false);
const isSubmitting = ref(false);
const hideReply = depth.value >= 4;
const reportOpen = ref(false);

const submitReply = () => {
  if (!replyText.value.trim()) return;
  isSubmitting.value = true;
  emit('reply', props.comment.id, replyText.value, () => {
    isSubmitting.value = false;
    isReplying.value = false;
    replyText.value = '';
    isCollapsed.value = false;
  });
};

const emitReply = (parentId: string, content: string, cb: () => void) => {
  emit('reply', parentId, content, cb);
};
</script>
