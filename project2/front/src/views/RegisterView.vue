<template>
  <div class="min-h-[calc(100vh-64px)] grid place-items-center p-4 sm:p-6 lg:p-10 bg-base-200/30">
    <div class="w-full max-w-[min(94vw,860px)] bg-base-100 p-6 sm:p-10 lg:p-12 rounded-[2.5rem] shadow-2xl border border-base-200 animate-in fade-in slide-in-from-bottom-8 duration-700">
      <div class="text-center mb-10">
        <div class="inline-flex bg-primary/10 p-4 rounded-3xl text-primary mb-6 animate-bounce duration-1000">
          <UserPlusIcon class="w-10 h-10" />
        </div>
        <h1 class="text-4xl font-black tracking-tighter mb-2">Создать аккаунт</h1>
        <p class="text-base-content/60 font-medium">Присоединяйтесь к сообществу</p>
      </div>

      <form @submit.prevent="handleRegister" class="space-y-7 max-w-[65ch] mx-auto">
        <div class="space-y-2 group">
          <label class="text-base font-bold ml-1 opacity-75 group-focus-within:text-primary transition-colors">Никнейм</label>
          <div class="relative">
            <UserIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input 
              v-model="username"
              type="text" 
              placeholder="ivan_ivanov" 
              class="input input-bordered w-full h-14 pl-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
              required
            />
          </div>
        </div>

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
          <label class="text-base font-bold ml-1 opacity-75 group-focus-within:text-primary transition-colors">Пароль</label>
          <div class="relative">
            <KeyIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input 
              v-model="password"
              :type="showPassword ? 'text' : 'password'" 
              placeholder="••••••••" 
              class="input input-bordered w-full h-14 pl-12 pr-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
              minlength="8"
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
          <div class="rounded-2xl border px-4 py-3 text-sm font-medium space-y-2" :class="isPasswordStrongEnough ? 'border-success/30 bg-success/10' : 'border-base-200 bg-base-200/60'">
            <div class="font-bold">Требования к паролю:</div>
            <div :class="passwordRules.minLength ? 'text-success' : 'opacity-70'">Не менее 8 символов</div>
            <div :class="passwordRules.uppercase ? 'text-success' : 'opacity-70'">Хотя бы одна заглавная буква</div>
            <div :class="passwordRules.special ? 'text-success' : 'opacity-70'">Хотя бы один специальный символ</div>
          </div>
        </div>

        <div class="space-y-2 group">
          <label class="text-base font-bold ml-1 opacity-75 group-focus-within:text-primary transition-colors">Подтверждение пароля</label>
          <div class="relative">
            <KeyIcon class="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 opacity-30" />
            <input
              v-model="confirmPassword"
              :type="showPassword ? 'text' : 'password'"
              placeholder="••••••••"
              class="input input-bordered w-full h-14 pl-12 pr-12 rounded-2xl bg-base-200/50 border-none transition-all font-semibold text-lg"
              required
            />
          </div>
        </div>

        <div class="flex items-start gap-3 px-1 py-2">
          <input type="checkbox" v-model="agree" class="checkbox checkbox-primary rounded-lg" required />
          <span class="text-sm opacity-60 font-medium leading-tight">
            Я согласен на <a href="#" class="text-primary font-bold hover:underline">обработку персональных данных</a> (152-ФЗ)
          </span>
        </div>

        <div v-if="error" class="alert alert-error bg-error/10 text-error border-none rounded-2xl p-4 flex gap-3 animate-shake">
          <AlertCircleIcon class="w-5 h-5" />
          <span class="text-sm font-bold">{{ error }}</span>
        </div>

        <button 
          type="submit" 
          class="btn btn-primary w-full h-16 rounded-2xl text-lg font-black shadow-xl shadow-primary/30 mt-4 transition-all hover:scale-[1.02] active:scale-95 disabled:bg-primary/50"
          :disabled="loading"
        >
          <span v-if="loading" class="loading loading-spinner"></span>
          <span v-else>Зарегистрироваться</span>
        </button>
      </form>

      <div class="divider my-8 opacity-20">или</div>

      <p class="text-center text-base-content/60 font-medium">
        Уже есть аккаунт? 
        <router-link to="/login" class="text-primary font-bold hover:underline underline-offset-4 decoration-2">Войти</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import api from '../api';
import { UserPlusIcon, UserIcon, MailIcon, KeyIcon, EyeIcon, EyeOffIcon, AlertCircleIcon } from 'lucide-vue-next';
import { getPasswordRuleState, isPasswordStrongEnough as isStrongPassword } from '../utils/passwordRules';

const username = ref('');
const email = ref('');
const password = ref('');
const confirmPassword = ref('');
const agree = ref(false);
const showPassword = ref(false);
const error = ref('');
const loading = ref(false);
const passwordRules = computed(() => getPasswordRuleState(password.value));
const isPasswordStrongEnough = computed(() => isStrongPassword(password.value));

const router = useRouter();

const handleRegister = async () => {
  if (!agree.value) return;
  if (!isPasswordStrongEnough.value) {
    error.value = 'Пароль должен содержать не менее 8 символов, заглавную букву и специальный символ';
    return;
  }
  if (password.value !== confirmPassword.value) {
    error.value = 'Пароли не совпадают';
    return;
  }
  loading.value = true;
  error.value = '';
  try {
    await api.post('/auth/register', { 
      username: username.value, 
      email: email.value, 
      password: password.value,
      consent: agree.value
    });
    router.push({ name: 'verify-email', query: { email: email.value } });
  } catch (err: any) {
    error.value = err.response?.data?.message || 'Ошибка регистрации. Возможно, email или никнейм заняты.';
  } finally {
    loading.value = false;
  }
};
</script>
