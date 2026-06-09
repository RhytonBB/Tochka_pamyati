<template>
  <div class="flex flex-col h-dvh overflow-hidden bg-base-200/30">
    <!-- Navbar -->
    <header class="navbar bg-base-100/95 backdrop-blur-lg shadow-lg sticky top-0 z-50 px-4 lg:px-8 border-b border-base-200">
      <div class="navbar-start">
        <router-link to="/" class="flex items-center gap-3 group transition-all duration-300 hover:scale-105">
          <div class="bg-gradient-to-br from-primary to-primary/80 p-3 rounded-2xl text-primary-content shadow-lg shadow-primary/20">
            <MapPinIcon class="w-6 h-6" />
          </div>
          <div class="flex flex-col">
            <span class="text-xl font-black tracking-tight text-gradient bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent">Точка памяти</span>
            <span class="text-xs opacity-50 font-medium">Геопортал культурного наследия</span>
          </div>
        </router-link>
      </div>

      <div class="navbar-center hidden lg:flex">
        <nav class="flex gap-2 p-2 bg-base-200/50 rounded-2xl border border-base-200">
          <router-link 
            to="/" 
            :class="{ 'bg-primary text-primary-content shadow-lg shadow-primary/20': $route.name === 'home' }"
            class="px-6 py-3 rounded-xl font-semibold transition-all duration-300 hover:bg-primary/10 hover:text-primary"
          >
            <HomeIcon class="w-5 h-5 mr-2 inline" />
            Карта
          </router-link>
          <router-link 
            to="/signals" 
            :class="{ 'bg-secondary text-secondary-content shadow-lg shadow-secondary/20': $route.name === 'signals' }"
            class="px-6 py-3 rounded-xl font-semibold transition-all duration-300 hover:bg-secondary/10 hover:text-secondary"
          >
            <ShieldAlertIcon class="w-5 h-5 mr-2 inline" />
            Защита
          </router-link>
          <router-link 
            v-if="auth.isModerator"
            to="/moderation" 
            :class="{ 'bg-accent text-accent-content shadow-lg shadow-accent/20': $route.name === 'moderation' }"
            class="px-6 py-3 rounded-xl font-semibold transition-all duration-300 hover:bg-accent/10 hover:text-accent"
          >
            <CheckSquareIcon class="w-5 h-5 mr-2 inline" />
            Модерация
          </router-link>
          <router-link 
            v-if="auth.isAdmin"
            to="/admin" 
            :class="{ 'bg-neutral text-neutral-content shadow-lg shadow-neutral/20': $route.name === 'admin' }"
            class="px-6 py-3 rounded-xl font-semibold transition-all duration-300 hover:bg-neutral/10 hover:text-neutral"
          >
            <BarChartIcon class="w-5 h-5 mr-2 inline" />
            Админ-панель
          </router-link>
        </nav>
      </div>

      <div class="navbar-end gap-3">
        <template v-if="auth.isAuthenticated">
          <!-- Notifications Bell -->
          <div class="dropdown dropdown-end">
            <label tabindex="0" class="btn btn-ghost btn-circle btn-lg hover:bg-base-200/50 transition-all duration-300">
              <div class="indicator">
                <BellIcon class="w-6 h-6 opacity-70 group-hover:opacity-100 transition-opacity" />
                <span v-if="auth.unreadNotificationsCount > 0" class="indicator-item badge badge-primary badge-xs animate-pulse"></span>
              </div>
            </label>
            <div tabindex="0" class="dropdown-content mt-3 z-[1] p-4 shadow-xl bg-base-100 rounded-[2rem] w-80 border border-base-200 animate-in fade-in slide-in-from-top-2">
              <div class="flex items-center justify-between mb-4 px-2">
                <h3 class="font-black text-lg">Уведомления</h3>
                <span v-if="auth.unreadNotificationsCount > 0" class="text-xs font-bold text-primary">{{ auth.unreadNotificationsCount }} новых</span>
              </div>
              <div class="space-y-2 max-h-[400px] overflow-y-auto pr-1">
                <div v-if="auth.notifications.length === 0" class="text-center py-8 opacity-40">
                  <BellOffIcon class="w-10 h-10 mx-auto mb-2 opacity-20" />
                  <p class="text-sm font-medium">Нет новых уведомлений</p>
                </div>
                <button
                  v-for="n in auth.notifications" 
                  :key="n.id" 
                  class="w-full text-left p-4 rounded-2xl hover:bg-base-200/50 transition-colors border border-transparent hover:border-base-200 group cursor-pointer"
                  :class="{ 'bg-primary/5 border-primary/10': !n.is_read }"
                  @click="openNotification(n)"
                >
                  <div class="font-bold text-sm mb-1 group-hover:text-primary transition-colors">{{ n.title }}</div>
                  <p class="text-xs opacity-60 leading-relaxed mb-2">{{ n.content }}</p>
                  <span class="text-[10px] font-black uppercase tracking-widest opacity-30">{{ new Date(n.created_at).toLocaleDateString() }}</span>
                </button>
              </div>
              <div class="mt-4 pt-4 border-t border-base-200">
                <router-link to="/notifications" class="btn btn-ghost btn-sm btn-block rounded-xl font-bold opacity-60">Смотреть все</router-link>
              </div>
            </div>
          </div>

          <div class="dropdown dropdown-end">
            <label tabindex="0" class="btn btn-ghost btn-circle avatar border border-base-200 shadow-sm">
              <div class="w-10 rounded-full bg-base-300 grid place-items-center">
                <span class="text-lg font-medium">{{ auth.user?.username[0].toUpperCase() }}</span>
              </div>
            </label>
            <ul tabindex="0" class="menu menu-sm dropdown-content mt-3 z-[1] p-2 shadow-xl bg-base-100 rounded-box w-52 border border-base-200 animate-in fade-in slide-in-from-top-2">
              <li class="menu-title px-4 py-2 border-b border-base-200 mb-1">
                <span class="text-xs opacity-50 block">Вы вошли как</span>
                <span class="text-sm font-bold truncate block">{{ auth.user?.username }}</span>
              </li>
              <li>
                <router-link to="/profile" class="py-3">
                  <UserIcon class="w-4 h-4 opacity-70" /> Профиль
                </router-link>
              </li>
              <li>
                <button @click="auth.logout()" class="py-3 text-error">
                  <LogOutIcon class="w-4 h-4 opacity-70" /> Выйти
                </button>
              </li>
            </ul>
          </div>
        </template>
        <template v-else>
          <router-link to="/login" class="btn btn-ghost rounded-lg font-medium">Войти</router-link>
          <router-link to="/register" class="btn btn-primary rounded-lg font-medium shadow-lg shadow-primary/20">Регистрация</router-link>
        </template>
      </div>
    </header>

    <div
      v-if="showRestrictionBanner"
      class="pointer-events-none fixed inset-x-0 top-[5.25rem] z-40 px-4"
    >
      <div class="pointer-events-auto mx-auto flex max-w-3xl items-start gap-4 rounded-[1.75rem] border border-warning/25 bg-base-100/96 px-5 py-4 shadow-2xl shadow-warning/10 backdrop-blur-xl">
        <div class="mt-0.5 flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-warning/12 text-warning ring-1 ring-warning/15">
          <BellOffIcon class="w-5 h-5" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-sm font-black tracking-tight">Ограничение аккаунта</div>
          <div class="mt-1 text-sm leading-6 opacity-70">{{ auth.restrictionSummary?.message }}</div>
        </div>
        <button class="btn btn-ghost btn-circle btn-sm mt-0.5 shrink-0 opacity-60 hover:bg-base-200 hover:opacity-100" @click="dismissRestrictionBanner">
          <XIcon class="w-4 h-4" />
        </button>
      </div>
    </div>

    <!-- Content with Transitions -->
    <main class="flex-1 min-h-0" :class="isHomeRoute ? 'overflow-hidden' : 'overflow-y-auto'">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <!-- Bottom Navigation for Mobile -->
    <nav class="lg:hidden btm-nav btm-nav-md bg-base-100/90 backdrop-blur-md border-t border-base-200 z-[100]">
      <router-link to="/" :class="{ 'active text-primary': $route.name === 'home' }">
        <MapPinIcon class="w-6 h-6" />
        <span class="btm-nav-label text-[10px]">Карта</span>
      </router-link>
      <router-link to="/signals" :class="{ 'active text-primary': $route.name === 'signals' }">
        <ShieldAlertIcon class="w-6 h-6" />
        <span class="btm-nav-label text-[10px]">Защита</span>
      </router-link>
      <router-link v-if="auth.isModerator" to="/moderation" :class="{ 'active text-primary': $route.name === 'moderation' }">
        <CheckSquareIcon class="w-6 h-6" />
        <span class="btm-nav-label text-[10px]">Модерация</span>
      </router-link>
      <router-link to="/profile" :class="{ 'active text-primary': $route.name === 'profile' }">
        <UserIcon class="w-6 h-6" />
        <span class="btm-nav-label text-[10px]">Профиль</span>
      </router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '../store/auth';
import { useRouter } from 'vue-router';
import { 
  MapPinIcon, 
  HomeIcon, 
  ShieldAlertIcon, 
  CheckSquareIcon, 
  BarChartIcon,
  UserIcon,
  LogOutIcon,
  BellIcon,
  BellOffIcon,
  XIcon
} from 'lucide-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();
const isHomeRoute = computed(() => route.name === 'home');
const restrictionBannerDismissed = ref(false);
const showRestrictionBanner = computed(() => !!auth.restrictionSummary && auth.restrictionSummary.status !== 'active' && !restrictionBannerDismissed.value);

onMounted(() => {
  if (auth.isAuthenticated) {
    auth.fetchNotifications();
  }
});

watch(() => auth.restrictionSummary?.message, () => {
  restrictionBannerDismissed.value = false;
});

const dismissRestrictionBanner = () => {
  restrictionBannerDismissed.value = true;
};

const openNotification = async (notification: { id: string; is_read: boolean; link?: string }) => {
  if (!notification.is_read) {
    await auth.markAsRead(notification.id);
  }
  if (notification.link) {
    await router.push(notification.link);
  } else {
    await router.push('/notifications');
  }
};
</script>

<style scoped>
</style>
