<template>
  <Teleport to="body">
    <div class="fixed bottom-6 right-6 z-[9999] flex flex-col gap-3 pointer-events-none">
      <TransitionGroup
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="opacity-0 translate-y-4 scale-95"
        enter-to-class="opacity-100 translate-y-0 scale-100"
        leave-active-class="transition-all duration-200 ease-in"
        leave-from-class="opacity-100 translate-y-0 scale-100"
        leave-to-class="opacity-0 translate-x-8 scale-95"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="pointer-events-auto flex items-center gap-3 px-6 py-4 rounded-2xl shadow-2xl backdrop-blur-xl border min-w-[320px] max-w-[420px] font-bold text-sm cursor-pointer"
          :class="toastClasses(toast.type)"
          @click="removeToast(toast.id)"
        >
          <component :is="toastIcon(toast.type)" class="w-5 h-5 shrink-0" />
          <span class="flex-grow">{{ toast.message }}</span>
          <button class="btn btn-ghost btn-xs btn-circle opacity-50 hover:opacity-100 shrink-0">
            <XIcon class="w-3.5 h-3.5" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useToast } from '../composables/useToast';
import { CheckCircle2Icon, AlertTriangleIcon, InfoIcon, XCircleIcon, XIcon } from 'lucide-vue-next';
import type { ToastType } from '../composables/useToast';
import { markRaw } from 'vue';

const { toasts, removeToast } = useToast();

const toastClasses = (type: ToastType) => {
  switch (type) {
    case 'success': return 'bg-success/90 text-success-content border-success/20';
    case 'error': return 'bg-error/90 text-error-content border-error/20';
    case 'warning': return 'bg-warning/90 text-warning-content border-warning/20';
    case 'info': return 'bg-info/90 text-info-content border-info/20';
  }
};

const toastIcon = (type: ToastType) => {
  switch (type) {
    case 'success': return markRaw(CheckCircle2Icon);
    case 'error': return markRaw(XCircleIcon);
    case 'warning': return markRaw(AlertTriangleIcon);
    case 'info': return markRaw(InfoIcon);
  }
};
</script>
