import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../store/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    children: [
      {
        path: '',
        name: 'home',
        component: () => import('../views/HomeView.vue'),
      },
      {
        path: 'monument/:id',
        name: 'monument-detail',
        component: () => import('../views/MonumentDetailView.vue'),
      },
      {
        path: 'signals',
        name: 'signals',
        component: () => import('../views/SignalsView.vue'),
      },
      {
        path: 'signal/:id',
        name: 'signal-detail',
        component: () => import('../views/SignalDetailView.vue'),
      },
      {
        path: 'signal-edit/:id',
        name: 'signal-edit',
        component: () => import('../views/SignalEditorView.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'signal-resolve/:id',
        name: 'signal-resolve',
        component: () => import('../views/SignalResolutionView.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'profile',
        name: 'profile',
        component: () => import('../views/ProfileView.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'notifications',
        name: 'notifications',
        component: () => import('../views/NotificationsView.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'submission-edit/:type/:id',
        name: 'submission-edit',
        component: () => import('../views/SubmissionEditorView.vue'),
        meta: { requiresAuth: true },
      },
      {
        path: 'moderation',
        name: 'moderation',
        component: () => import('../views/ModerationView.vue'),
        meta: { requiresModerator: true },
      },
      {
        path: 'moderation/item/:type/:id',
        name: 'moderation-item',
        component: () => import('../views/ModerationItemView.vue'),
        meta: { requiresModerator: true },
      },
      {
        path: 'admin',
        name: 'admin',
        component: () => import('../views/AdminView.vue'),
        meta: { requiresAdmin: true },
      },
      {
        path: 'admin/users/:id',
        name: 'admin-user',
        component: () => import('../views/AdminUserView.vue'),
        meta: { requiresAdmin: true },
      },
    ],
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/LoginView.vue'),
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('../views/RegisterView.vue'),
  },
  {
    path: '/verify-email',
    name: 'verify-email',
    component: () => import('../views/VerifyEmailView.vue'),
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  
  if (!auth.initialized) {
    await auth.fetchMe();
  }

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' };
  }

  if (to.meta.requiresModerator && !auth.isModerator) {
    return { name: 'home' };
  }

  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { name: 'home' };
  }
});

export default router;
