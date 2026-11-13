import { createRouter, createWebHistory } from 'vue-router'
import { useSessionStore } from '@/stores/session'
import HomeView from '../views/HomeView.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView,
      meta: { guestOnly: true },
    },
    {
      path: '/registration',
      name: 'registration',
      component: () => import('../views/RegistrationView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/check-email',
      name: 'check-email',
      component: () => import('../views/CheckEmailView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/verify',
      name: 'verify',
      component: () => import('../views/VerifyView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/password-recovery',
      name: 'password-recovery',
      component: () => import('../views/PasswordRecoveryView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/password-recovery/verify',
      name: 'password-recovery-verify',
      component: () => import('../views/PasswordRecoveryVerifyView.vue'),
      meta: { guestOnly: true },
    },
    {
      path: '/account',
      name: 'account',
      component: () => import('../views/AccountView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/profile',
      name: 'profile',
      component: () => import('../views/ProfileView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/profile/update-email',
      name: 'update-email',
      component: () => import('../views/UpdateEmailView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/profile/update-password',
      name: 'update-password',
      component: () => import('../views/UpdatePasswordView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/profile/update-email/verify',
      name: 'update-email-verify',
      component: () => import('../views/UpdateEmailVerifyView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/users',
      name: 'users',
      component: () => import('../views/UsersView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/users/:userId',
      name: 'user',
      component: () => import('../views/UserView.vue'),
      meta: { requiresAuth: true },
    },
    {
      path: '/notifications',
      name: 'notifications',
      component: () => import('../views/NotificationsView.vue'),
      meta: { requiresAuth: true, roles: 6 },
    },
    {
      path: '/notifications/:notificationId',
      name: 'notification',
      component: () => import('../views/NotificationView.vue'),
      meta: { requiresAuth: true, roles: 6 },
    },
    {
      path: '/404',
      name: 'not-found',
      component: () => import('../views/NotFoundView.vue'),
    },
  ],
})

router.beforeEach((to) => {
  const session = useSessionStore()
  const isAuthenticated = session.userId !== ''

  if (to.meta.requiresAuth && !isAuthenticated) {
    if (to.name === 'update-email-verify') {
      return { name: 'login', query: { redirect: to.fullPath } }
    }

    return { name: 'home' }
  }

  if (typeof to.meta.roles === 'number' && (Number(session.role) & to.meta.roles) === 0) {
    return { name: 'not-found' }
  }

  if (to.meta.guestOnly && isAuthenticated) {
    return { name: 'account' }
  }
})

export default router
