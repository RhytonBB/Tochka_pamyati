<template>
  <div class="min-h-[calc(100vh-64px)] grid place-items-center p-4 sm:p-6 lg:p-10 bg-base-200/30">
    <div class="w-full max-w-[min(94vw,920px)] bg-base-100 p-6 sm:p-10 lg:p-12 rounded-[2.5rem] shadow-2xl border border-base-200 animate-in fade-in slide-in-from-bottom-8 duration-700">
      <div class="text-center mb-10">
        <div class="inline-flex bg-primary/10 p-4 rounded-3xl text-primary mb-6 animate-bounce duration-1000">
          <LockIcon class="w-10 h-10" />
        </div>
        <h1 class="text-4xl font-black tracking-tighter mb-2">{{ forgotMode ? 'Восстановление пароля' : 'С возвращением!' }}</h1>
        <p class="text-base-content/60 font-medium">{{ forgotMode ? 'Получите код на почту и задайте новый пароль' : 'Войдите в свой аккаунт' }}</p>
      </div>

      <form v-if="!forgotMode" @submit.prevent="handleLogin" class="space-y-7 max-w-[65ch] mx-auto">
        <div class="space-y-2 group">
          <label class="text-base font-bold ml-1 opacity-75 group-focus-within:text-primary transition-colors">Электронная почта</label>
          <div class="relative">
            <MailIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input 
              v-model="email"
              type="email" 
              placeholder="example@mail.com" 
              class="input input-bordered w-full h-14 pl-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
              required
            />
          </div>
        </div>

        <div class="space-y-2 group">
          <div class="flex items-center justify-between gap-3">
            <label class="text-base font-bold ml-1 opacity-75 group-focus-within:text-primary transition-colors">Пароль</label>
            <button type="button" class="text-sm font-bold text-primary hover:underline" @click="openForgotMode">
              Забыли пароль?
            </button>
          </div>
          <div class="relative">
            <KeyIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input 
              v-model="password"
              :type="showPassword ? 'text' : 'password'" 
              placeholder="••••••••" 
              class="input input-bordered w-full h-14 pl-12 pr-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
              required
            />
            <button 
              type="button"
              @click="showPassword = !showPassword"
              class="absolute right-4 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle opacity-40 hover:opacity-100"
            >
              <EyeIcon v-if="!showPassword" class="w-4 h-4" />
              <EyeOffIcon v-else class="w-4 h-4" />
            </button>
          </div>
        </div>

        <div v-if="error" class="alert alert-error bg-error/10 text-error border-none rounded-2xl p-4 flex gap-3 animate-shake">
          <AlertCircleIcon class="w-5 h-5" />
          <span class="text-sm font-bold">{{ error }}</span>
        </div>

        <div v-if="route.query.verified === 'true'" class="alert alert-success bg-success/10 text-success border-none rounded-2xl p-4 flex gap-3">
          <CheckCircleIcon class="w-5 h-5" />
          <span class="text-sm font-bold">Почта подтверждена. Теперь можно войти.</span>
        </div>

        <div v-if="loginNotice" class="alert alert-success bg-success/10 text-success border-none rounded-2xl p-4 flex gap-3">
          <CheckCircleIcon class="w-5 h-5" />
          <span class="text-sm font-bold">{{ loginNotice }}</span>
        </div>

        <button 
          type="submit" 
          class="btn btn-primary w-full h-16 rounded-2xl text-lg font-black shadow-xl shadow-primary/30 mt-4 transition-all hover:scale-[1.02] active:scale-95 disabled:bg-primary/50"
          :disabled="loading"
        >
          <span v-if="loading" class="loading loading-spinner"></span>
          <span v-else>Войти</span>
        </button>
      </form>

      <div v-else class="max-w-[65ch] mx-auto space-y-6">
        <div class="rounded-[2rem] border border-base-200 bg-base-200/40 p-5">
          <div class="font-black text-lg mb-2">{{ resetCodeRequested ? 'Шаг 2. Новый пароль' : 'Шаг 1. Код на почту' }}</div>
          <p class="opacity-65 text-sm">
            {{ resetCodeRequested ? 'Введите код из письма и придумайте новый пароль.' : 'На указанную почту будет отправлен отдельный код для сброса пароля.' }}
          </p>
        </div>

        <div class="space-y-2">
          <label class="text-base font-bold ml-1 opacity-75">Электронная почта</label>
          <div class="relative">
            <MailIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input
              v-model="resetEmail"
              type="email"
              placeholder="example@mail.com"
              class="input input-bordered w-full h-14 pl-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
            />
          </div>
        </div>

        <div v-if="resetCodeRequested" class="space-y-2">
          <label class="text-base font-bold ml-1 opacity-75">Код из письма</label>
          <input
            v-model="resetCode"
            type="text"
            maxlength="6"
            placeholder="123456"
            class="input input-bordered w-full h-14 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg tracking-[0.35em] text-center"
          />
        </div>

        <div v-if="resetCodeRequested" class="space-y-2">
          <label class="text-base font-bold ml-1 opacity-75">Новый пароль</label>
          <div class="relative">
            <KeyIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input
              v-model="resetPassword"
              :type="showResetPassword ? 'text' : 'password'"
              placeholder="••••••••"
              class="input input-bordered w-full h-14 pl-12 pr-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
            />
            <button
              type="button"
              @click="showResetPassword = !showResetPassword"
              class="absolute right-4 top-1/2 -translate-y-1/2 btn btn-ghost btn-xs btn-circle opacity-40 hover:opacity-100"
            >
              <EyeIcon v-if="!showResetPassword" class="w-4 h-4" />
              <EyeOffIcon v-else class="w-4 h-4" />
            </button>
          </div>
          <div class="rounded-2xl border px-4 py-3 text-sm font-medium space-y-2" :class="isResetPasswordStrongEnough ? 'border-success/30 bg-success/10' : 'border-base-200 bg-base-200/60'">
            <div class="font-bold">Требования к паролю:</div>
            <div :class="resetPasswordRules.minLength ? 'text-success' : 'opacity-70'">Не менее 8 символов</div>
            <div :class="resetPasswordRules.uppercase ? 'text-success' : 'opacity-70'">Хотя бы одна заглавная буква</div>
            <div :class="resetPasswordRules.special ? 'text-success' : 'opacity-70'">Хотя бы один специальный символ</div>
          </div>
        </div>

        <div v-if="resetCodeRequested" class="space-y-2">
          <label class="text-base font-bold ml-1 opacity-75">Подтверждение нового пароля</label>
          <input
            v-model="resetPasswordConfirm"
            :type="showResetPassword ? 'text' : 'password'"
            placeholder="••••••••"
            class="input input-bordered w-full h-14 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
          />
        </div>

        <div v-if="resetError" class="alert alert-error bg-error/10 text-error border-none rounded-2xl p-4 flex gap-3">
          <AlertCircleIcon class="w-5 h-5" />
          <span class="text-sm font-bold">{{ resetError }}</span>
        </div>

        <div v-if="resetSuccess" class="alert alert-success bg-success/10 text-success border-none rounded-2xl p-4 flex gap-3">
          <CheckCircleIcon class="w-5 h-5" />
          <span class="text-sm font-bold">{{ resetSuccess }}</span>
        </div>

        <div class="flex flex-wrap gap-3">
          <button
            v-if="!resetCodeRequested"
            type="button"
            class="btn btn-primary flex-1 h-14 rounded-2xl font-black"
            :disabled="resetLoading"
            @click="requestResetCode"
          >
            <span v-if="resetLoading" class="loading loading-spinner"></span>
            <span v-else>Получить код</span>
          </button>
          <button
            v-else
            type="button"
            class="btn btn-primary flex-1 h-14 rounded-2xl font-black"
            :disabled="resetLoading"
            @click="submitPasswordReset"
          >
            <span v-if="resetLoading" class="loading loading-spinner"></span>
            <span v-else>Сменить пароль</span>
          </button>
          <button type="button" class="btn btn-ghost h-14 rounded-2xl font-bold" @click="closeForgotMode">
            Вернуться ко входу
          </button>
        </div>

        <button
          v-if="resetCodeRequested"
          type="button"
          class="btn btn-link px-0 text-primary font-bold"
          :disabled="resetLoading"
          @click="requestResetCode"
        >
          Отправить код повторно
        </button>
      </div>

      <div class="divider my-8 opacity-20">или</div>

      <p class="text-center text-base-content/60 font-medium">
        Нет аккаунта?
        <router-link to="/register" class="text-primary font-bold hover:underline underline-offset-4 decoration-2">Зарегистрироваться</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../store/auth';
import api from '../api';
import { LockIcon, MailIcon, KeyIcon, EyeIcon, EyeOffIcon, AlertCircleIcon, CheckCircleIcon } from 'lucide-vue-next';
import { getPasswordRuleState, isPasswordStrongEnough as isStrongPassword } from '../utils/passwordRules';

const email = ref('');
const password = ref('');
const showPassword = ref(false);
const error = ref('');
const loading = ref(false);
const forgotMode = ref(false);
const loginNotice = ref('');

const resetEmail = ref('');
const resetCode = ref('');
const resetPassword = ref('');
const resetPasswordConfirm = ref('');
const showResetPassword = ref(false);
const resetCodeRequested = ref(false);
const resetError = ref('');
const resetSuccess = ref('');
const resetLoading = ref(false);

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();

const resetPasswordRules = computed(() => getPasswordRuleState(resetPassword.value));
const isResetPasswordStrongEnough = computed(() => isStrongPassword(resetPassword.value));

const openForgotMode = () => {
  forgotMode.value = true;
  resetError.value = '';
  resetSuccess.value = '';
  resetEmail.value = email.value || String(route.query.email || '');
};

const closeForgotMode = async () => {
  forgotMode.value = false;
  resetCodeRequested.value = false;
  resetCode.value = '';
  resetPassword.value = '';
  resetPasswordConfirm.value = '';
  resetError.value = '';
  resetSuccess.value = '';
  const nextQuery: Record<string, string> = {};
  if (route.query.verified) nextQuery.verified = String(route.query.verified);
  if (route.query.passwordChanged) nextQuery.passwordChanged = String(route.query.passwordChanged);
  await router.replace({ name: 'login', query: Object.keys(nextQuery).length ? nextQuery : undefined });
};

const syncForgotModeFromRoute = () => {
  const shouldOpen = route.query.forgot === 'true';
  if (route.query.passwordChanged === 'true') {
    loginNotice.value = 'Пароль изменен. Войдите снова с новым паролем.';
  }
  if (shouldOpen) {
    forgotMode.value = true;
    resetEmail.value = String(route.query.email || email.value || '');
  }
};

onMounted(syncForgotModeFromRoute);
watch(() => route.query, syncForgotModeFromRoute, { deep: true });

const handleLogin = async () => {
  loading.value = true;
  error.value = '';
  try {
    await api.post('/auth/login', { email: email.value, password: password.value });
    await auth.fetchMe();
    router.push('/');
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Не удалось войти. Проверьте данные.';
  } finally {
    loading.value = false;
  }
};

const requestResetCode = async () => {
  if (!resetEmail.value.trim()) {
    resetError.value = 'Нужно указать электронную почту';
    return;
  }
  resetLoading.value = true;
  resetError.value = '';
  resetSuccess.value = '';
  try {
    await api.post('/auth/password/forgot', { email: resetEmail.value });
    resetCodeRequested.value = true;
    resetSuccess.value = 'Если аккаунт с такой почтой существует, код для сброса уже отправлен.';
  } catch (err: any) {
    resetError.value = err.response?.data?.message || 'Не удалось отправить код.';
  } finally {
    resetLoading.value = false;
  }
};

const submitPasswordReset = async () => {
  if (!resetCode.value.trim()) {
    resetError.value = 'Нужно указать код из письма';
    return;
  }
  if (!isResetPasswordStrongEnough.value) {
    resetError.value = 'Пароль должен содержать не менее 8 символов, заглавную букву и специальный символ';
    return;
  }
  if (resetPassword.value !== resetPasswordConfirm.value) {
    resetError.value = 'Пароли не совпадают';
    return;
  }
  resetLoading.value = true;
  resetError.value = '';
  resetSuccess.value = '';
  try {
    await api.post('/auth/password/reset', {
      email: resetEmail.value,
      code: resetCode.value,
      new_password: resetPassword.value,
    });
    loginNotice.value = 'Пароль успешно изменен. Теперь можно войти с новым паролем.';
    email.value = resetEmail.value;
    password.value = '';
    await closeForgotMode();
  } catch (err: any) {
    resetError.value = err.response?.data?.message || 'Не удалось сменить пароль.';
  } finally {
    resetLoading.value = false;
  }
};
</script>

<style scoped>
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}
.animate-shake {
  animation: shake 0.3s ease-in-out;
}
</style>
