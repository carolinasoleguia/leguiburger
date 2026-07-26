import { createRouter, createWebHistory } from 'vue-router';
import { useAuth } from '../composables/useAuth.js';
import LoginView from '../views/auth/LoginView.vue';
import OwnerDashboard from '../views/owner/OwnerDashboard.vue';
import OwnerHomeView from '../views/owner/OwnerHomeView.vue';
import TenantListView from '../views/owner/TenantListView.vue';
import AdminListView from '../views/owner/AdminListView.vue';
import BrandListView from '../views/owner/BrandListView.vue';
import AdminDashboard from '../views/admins/AdminDashboard.vue';
import AdminLandingView from '../views/admins/AdminLandingView.vue';
import AdminHomeView from '../views/admins/AdminHomeView.vue';
import AdminEmployeesView from '../views/admins/AdminEmployeesView.vue';
import AdminProductionView from '../views/admins/AdminProductionView.vue';
import EmployeeDashboard from '../views/employees/EmployeeDashboard.vue';
import CatalogLandingView from '../views/buyers/CatalogLandingView.vue';
import UnauthorizedView from '../views/shared/UnauthorizedView.vue';
import NotFoundView from '../views/shared/NotFoundView.vue';

const routes = [
  {
    path: '/',
    redirect: { name: 'Login' }
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginView
  },
  {
    path: '/owner',
    component: OwnerDashboard,
    meta: { requiresAuth: true, roles: ['owner'] },
    children: [
      {
        path: '',
        name: 'OwnerHome',
        component: OwnerHomeView
      },
      {
        path: 'brands',
        name: 'OwnerBrands',
        component: BrandListView
      },
      {
        path: 'tenants',
        name: 'OwnerTenants',
        component: TenantListView
      },
      {
        path: 'admins',
        name: 'OwnerAdmins',
        component: AdminListView
      }
    ]
  },
  {
    path: '/admin/landing',
    name: 'AdminLanding',
    component: AdminLandingView,
    meta: { requiresAuth: true, roles: ['admin'] }
  },
  {
    path: '/admin',
    component: AdminDashboard,
    meta: { requiresAuth: true, roles: ['admin'] },
    children: [
      {
        path: '',
        name: 'AdminHome',
        component: AdminHomeView
      },
      {
        path: 'employees',
        name: 'AdminEmployees',
        component: AdminEmployeesView
      },
      {
        path: 'production',
        name: 'AdminProduction',
        component: AdminProductionView
      }
    ]
  },
  {
    path: '/tenant/:tenantId/employee',
    name: 'TenantEmployeeDashboard',
    component: EmployeeDashboard,
    meta: { requiresAuth: true, roles: ['employee'], tenantMatch: true }
  },
  {
    path: '/catalogo',
    name: 'CatalogLanding',
    component: CatalogLandingView
  },
  {
    path: '/unauthorized',
    name: 'Unauthorized',
    component: UnauthorizedView
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: NotFoundView
  }
];

const router = createRouter({
  history: createWebHistory(),
  routes
});

router.beforeEach((to, from, next) => {
  const auth = useAuth();

  if (to.meta.requiresAuth && !auth.isLoggedIn.value) {
    return next({ name: 'Login' });
  }

  if (to.name === 'Login' && auth.isLoggedIn.value) {
    const role = auth.userRole.value;
    if (role === 'owner') {
      return next({ path: '/owner' });
    }
    if (role === 'admin') {
      return next({ path: '/admin/landing' });
    }
    if (role === 'employee') {
      return next({ path: `/tenant/${auth.tenantId.value}/employee` });
    }
  }

  if (to.meta.roles && !to.meta.roles.includes(auth.userRole.value)) {
    return next({ name: 'Unauthorized' });
  }

  if (to.meta.tenantMatch && auth.tenantId.value && auth.tenantId.value !== to.params.tenantId) {
    return next({ name: 'Unauthorized' });
  }

  return next();
});

export default router;
