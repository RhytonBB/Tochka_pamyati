import { defineStore } from 'pinia';
import api from '../api';
import type { NotificationItem, User } from '../types';

interface AuthState {
  user: User | null;
  notifications: NotificationItem[];
  unreadCount: number;
  lastFetchedAt: string | null;
  loading: boolean;
  initialized: boolean;
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    notifications: [],
    unreadCount: 0,
    lastFetchedAt: null,
    loading: false,
    initialized: false,
  }),
  actions: {
    async fetchMe() {
      this.loading = true;
      try {
        const { data } = await api.get('/me');
        this.user = data;
        if (this.user) {
          this.fetchNotifications();
        }
      } catch (e) {
        this.user = null;
      } finally {
        this.loading = false;
        this.initialized = true;
      }
    },
    async fetchNotifications() {
      try {
        const { data } = await api.get('/notifications', { params: { limit: 20 } });
        this.notifications = data?.items || [];
        this.unreadCount = data?.unread_count ?? this.notifications.filter(n => !n.is_read).length;
        this.lastFetchedAt = new Date().toISOString();
      } catch (e) {
        console.error('Failed to fetch notifications', e);
      }
    },
    async markAsRead(id: string) {
      try {
        await api.post(`/notifications/${id}/read`);
        const n = this.notifications.find(n => n.id === id);
        if (n && !n.is_read) {
          n.is_read = true;
          this.unreadCount = Math.max(0, this.unreadCount - 1);
        }
      } catch (e) {
        console.error('Failed to mark notification as read', e);
      }
    },
    async markAllNotificationsRead() {
      try {
        await api.post('/notifications/read-all');
        this.notifications = this.notifications.map(n => ({ ...n, is_read: true }));
        this.unreadCount = 0;
      } catch (e) {
        console.error('Failed to mark all notifications as read', e);
      }
    },
    async logout() {
      try {
        await api.post('/auth/logout');
      } finally {
        this.user = null;
        this.notifications = [];
        this.unreadCount = 0;
        this.lastFetchedAt = null;
      }
    },
  },
  getters: {
    isAuthenticated: (state) => !!state.user,
    isModerator: (state) => state.user?.role_name === 'moderator' || state.user?.role_name === 'admin',
    isAdmin: (state) => state.user?.role_name === 'admin',
    unreadNotificationsCount: (state) => state.unreadCount,
    restrictionSummary: (state) => state.user?.restriction_summary || null,
    hasRestriction: (state) => (scope: string) => !!state.user?.restriction_summary?.scopes?.includes(scope),
    canComment: (state) => !state.user?.restriction_summary?.scopes?.includes('comment_write'),
    canCreateContent: (state) => !state.user?.restriction_summary?.scopes?.includes('content_create'),
    canEditContent: (state) => !state.user?.restriction_summary?.scopes?.includes('content_edit'),
  },
});
