<template>
  <div class="p-8 max-w-5xl mx-auto space-y-8">
    <div class="flex flex-col md:flex-row md:items-end justify-between gap-4 pb-6 border-b border-base-200">
      <div>
        <div class="inline-flex bg-primary/10 p-3 rounded-2xl text-primary mb-4 font-bold tracking-tight text-sm uppercase">
          Центр уведомлений
        </div>
        <h1 class="text-4xl font-black tracking-tight">Уведомления</h1>
        <p class="text-base opacity-60 font-medium">Статусы заявок, региональные сигналы и модерационные действия.</p>
      </div>
      <button class="btn btn-ghost rounded-2xl font-bold" @click="auth.markAllNotificationsRead()" :disabled="auth.unreadNotificationsCount === 0">
        Прочитать все
      </button>
    </div>

    <div class="tabs tabs-boxed p-1.5 bg-base-200/50 rounded-2xl border border-base-200 inline-flex">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="tab tab-lg font-bold rounded-xl transition-all h-12 px-6"
        :class="{ 'tab-active bg-primary text-primary-content shadow-lg shadow-primary/20': activeFilter === tab.id }"
        @click="activeFilter = tab.id"
      >
        {{ tab.label }}
      </button>
    </div>

    <div class="space-y-4">
      <div v-if="filteredNotifications.length === 0" class="bg-base-100 rounded-[2rem] border border-base-200 p-12 text-center opacity-50 font-bold">
        Подходящих уведомлений пока нет.
      </div>
      <button
        v-for="notification in filteredNotifications"
        :key="notification.id"
        class="w-full text-left bg-base-100 rounded-[2rem] border p-6 transition-all hover:border-primary/30 hover:shadow-lg"
        :class="notification.is_read ? 'border-base-200' : 'border-primary/20 bg-primary/5'"
        @click="openNotification(notification)"
      >
        <div class="flex items-start justify-between gap-4 mb-2">
          <div>
            <div class="font-black text-lg">{{ notification.title }}</div>
            <div class="text-xs font-bold uppercase tracking-widest opacity-40">{{ typeLabel(notification.type) }}</div>
          </div>
          <div class="text-xs font-bold opacity-40 whitespace-nowrap">{{ formatDate(notification.created_at) }}</div>
        </div>
        <p class="text-sm opacity-70 font-medium leading-relaxed">{{ notification.content }}</p>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../store/auth';
import type { NotificationItem } from '../types';

const auth = useAuthStore();
const router = useRouter();
const route = useRoute();

const activeFilter = ref(String(route.query.filter || 'all'));

const tabs = [
  { id: 'all', label: 'Все' },
  { id: 'unread', label: 'Непрочитанные' },
  { id: 'status', label: 'Статусы' },
  { id: 'regional', label: 'Сигналы региона' },
  { id: 'moderation', label: 'Модерация' },
];

const filteredNotifications = computed(() => {
  return auth.notifications.filter((notification) => {
    if (activeFilter.value === 'all') return true;
    if (activeFilter.value === 'unread') return !notification.is_read;
    if (activeFilter.value === 'status') return ['monument_status', 'post_status', 'signal_status', 'report_status'].includes(notification.type);
    if (activeFilter.value === 'regional') return notification.type === 'regional_threat';
    if (activeFilter.value === 'moderation') return ['photo_deleted', 'comment_deleted', 'content_hidden_report', 'needs_revision'].includes(notification.type);
    return true;
  });
});

const typeLabel = (type: string) => {
  const map: Record<string, string> = {
    monument_status: 'Статус памятника',
    post_status: 'Статус поста',
    signal_status: 'Статус сигнала',
    regional_threat: 'Сигнал региона',
    photo_deleted: 'Модерация фото',
    comment_deleted: 'Модерация комментария',
    report_status: 'Статус жалобы',
    content_hidden_report: 'Скрытие контента',
    needs_revision: 'Требуется правка',
  };
  return map[type] || type;
};

const formatDate = (value: string) => new Date(value).toLocaleString();

const openNotification = async (notification: NotificationItem) => {
  if (!notification.is_read) {
    await auth.markAsRead(notification.id);
  }
  if (notification.link) {
    await router.push(notification.link);
  }
};

onMounted(() => {
  auth.fetchNotifications();
});
</script>
