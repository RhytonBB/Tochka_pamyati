<template>
  <div class="min-h-[calc(100vh-64px)] bg-base-200/30 py-12 px-6">
    <div class="w-[min(94vw,980px)] mx-auto bg-base-100 p-10 md:p-12 rounded-[2.5rem] shadow-2xl border border-base-200 animate-in fade-in slide-in-from-bottom-8 duration-700">
      <div class="grid grid-cols-1 md:grid-cols-[1.15fr_1fr] gap-10 md:gap-12 items-start">
        <div>
          <div class="inline-flex bg-primary/10 p-4 rounded-3xl text-primary mb-6">
            <MailCheckIcon class="w-9 h-9" />
          </div>

          <h1 class="text-4xl md:text-5xl font-black tracking-tight leading-[1.05] mb-4">Подтверждение регистрации</h1>
          <p class="text-base-content/60 font-medium mb-2">Мы отправили 6-значный код на вашу почту:</p>
          <p class="text-base-content font-bold break-words">{{ email }}</p>
        </div>

        <div class="bg-base-200/45 rounded-3xl p-6 md:p-7 border border-base-200">
          <form @submit.prevent="handleVerify" class="space-y-5">
            <div class="relative">
              <input 
                v-model="code"
                type="text" 
                maxlength="6" 
                placeholder="000000"
                class="input input-bordered w-full h-20 text-center text-4xl tracking-[0.35em] font-black rounded-3xl bg-base-100 border-none focus:ring-4 focus:ring-primary/10 transition-all"
                required
                :disabled="loading"
              />
            </div>

            <div v-if="error" class="alert alert-error bg-error/10 text-error border-none rounded-2xl p-4 flex gap-3 animate-shake">
              <AlertCircleIcon class="w-5 h-5" />
              <span class="text-sm font-bold">{{ error }}</span>
            </div>

            <button 
              type="submit" 
              class="btn btn-primary w-full h-14 rounded-2xl text-lg font-black shadow-xl shadow-primary/30 transition-all hover:scale-[1.01] active:scale-95 disabled:bg-primary/50"
              :disabled="loading || code.length !== 6"
            >
              <span v-if="loading" class="loading loading-spinner"></span>
              <span v-else>Подтвердить</span>
            </button>
          </form>

          <div class="mt-7 pt-6 border-t border-base-200/70">
            <p class="text-sm font-medium opacity-50 mb-3">Не получили код?</p>
            <button 
              @click="handleResend" 
              class="btn btn-ghost btn-sm font-bold text-primary disabled:opacity-30"
              :disabled="resendTimer > 0 || resendLoading"
            >
              <span v-if="resendLoading" class="loading loading-spinner loading-xs"></span>
              <span v-else-if="resendTimer > 0">Отправить снова через {{ resendTimer }}с</span>
              <span v-else>Отправить код повторно</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import api from '../api';
import { MailCheckIcon, AlertCircleIcon } from 'lucide-vue-next';

const route = useRoute();
const router = useRouter();

const email = ref((route.query.email as string) || '');
const code = ref('');
const error = ref('');
const loading = ref(false);
const resendLoading = ref(false);
const resendTimer = ref(0);
let timerInterval: any = null;

const startTimer = (seconds: number) => {
  resendTimer.value = seconds;
  if (timerInterval) clearInterval(timerInterval);
  timerInterval = setInterval(() => {
    if (resendTimer.value > 0) {
      resendTimer.value--;
    } else {
      clearInterval(timerInterval);
    }
  }, 1000);
};

onMounted(() => {
  if (!email.value) {
    router.push({ name: 'register' });
  }
});

onUnmounted(() => {
  if (timerInterval) clearInterval(timerInterval);
});

const handleVerify = async () => {
  loading.value = true;
  error.value = '';
  try {
    await api.post('/auth/verify-email', { 
      email: email.value, 
      code: code.value 
    });
    router.push({ name: 'login', query: { verified: 'true' } });
  } catch (err: any) {
    const rawMessage = String(err.response?.data?.message || '').toLowerCase();
    if (rawMessage.includes('invalid code') || rawMessage.includes('invalid') || rawMessage.includes('expired')) {
      error.value = 'Неверный или истекший код.';
    } else {
      error.value = err.response?.data?.message || 'Не удалось подтвердить код. Попробуйте еще раз.';
    }
  } finally {
    loading.value = false;
  }
};

const handleResend = async () => {
  resendLoading.value = true;
  error.value = '';
  try {
    await api.post('/auth/resend-code', { email: email.value });
    startTimer(120); // 2 minutes as per tasks.md
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Ошибка при повторной отправке.';
  } finally {
    resendLoading.value = false;
  }
};
</script>
