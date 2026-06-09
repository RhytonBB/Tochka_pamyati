import { ref } from 'vue';

export type ToastType = 'info' | 'success' | 'warning' | 'error';

export interface Toast {
  id: string;
  type: ToastType;
  message: string;
}

const toasts = ref<Toast[]>([]);

export function useToast() {
  const addToast = (type: ToastType, message: string, duration = 3000) => {
    const id = Date.now().toString() + Math.random().toString(36).substr(2, 5);
    toasts.value.push({ id, type, message });
    setTimeout(() => {
      removeToast(id);
    }, duration);
  };

  const removeToast = (id: string) => {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  };

  const success = (message: string, duration?: number) => addToast('success', message, duration);
  const error = (message: string, duration?: number) => addToast('error', message, duration);
  const info = (message: string, duration?: number) => addToast('info', message, duration);
  const warning = (message: string, duration?: number) => addToast('warning', message, duration);

  return {
    toasts,
    success,
    error,
    info,
    warning,
    removeToast
  };
}
