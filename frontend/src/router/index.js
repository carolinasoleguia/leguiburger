import { createRouter, createWebHistory } from "vue-router";
import { useAuth } from "../composables/useAuth.js";
import { bootstrapSession } from "../services/session.js";
import { useMembership } from "../composables/useMembership.js";
import LandingView from "../views/public/LandingView.vue";
import LoginView from "../views/auth/LoginView.vue";
import OwnerDashboard from "../views/owner/OwnerDashboard.vue";
import OwnerHomeView from "../views/owner/OwnerHomeView.vue";
import TenantListView from "../views/owner/TenantListView.vue";
import AdminListView from "../views/owner/AdminListView.vue";
import BrandListView from "../views/owner/BrandListView.vue";
import AdminDashboard from "../views/admins/AdminDashboard.vue";
import AdminLandingView from "../views/admins/AdminLandingView.vue";
import AdminHomeView from "../views/admins/AdminHomeView.vue";
import AdminEmployeesView from "../views/admins/AdminEmployeesView.vue";
import AdminProductionView from "../views/admins/AdminProductionView.vue";
import AdminCatalogView from "../views/admins/AdminCatalogView.vue";
import AdminCatalogTenantView from "../views/admins/AdminCatalogTenantView.vue";
import EmployeeDashboard from "../views/employees/EmployeeDashboard.vue";
import CatalogLandingView from "../views/buyers/CatalogLandingView.vue";
import UnauthorizedView from "../views/shared/UnauthorizedView.vue";
import NotFoundView from "../views/shared/NotFoundView.vue";

const workspaceViews = {
  CashView: () => import("../views/shared/CashView.vue"),
  OrdersView: () => import("../views/shared/OrdersView.vue"),
  KitchenView: () => import("../views/shared/KitchenView.vue"),
  InventoryView: () => import("../views/shared/InventoryView.vue"),
  ExpensesView: () => import("../views/shared/ExpensesView.vue"),
  CustomersView: () => import("../views/shared/CustomersView.vue"),
  NotificationsView: () => import("../views/shared/NotificationsView.vue"),
  OnboardingView: () => import("../views/shared/OnboardingView.vue"),
  HealthView: () => import("../views/shared/HealthView.vue"),
};
const routes = [
  {
    path: "/platform/plans",
    component: () => import("../views/owner/MembershipPlansView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin"] },
  },
  {
    path: "/admin/membership",
    component: () => import("../views/admins/AdminMembershipView.vue"),
    meta: { requiresAuth: true, roles: ["admin"] },
  },
  {
    path: "/employee/membership",
    component: () => import("../views/employees/EmployeeMembershipView.vue"),
    meta: { requiresAuth: true, roles: ["employee", "cashier", "kitchen"] },
  },
  {
    path: "/platform/security",
    component: () => import("../views/owner/AbuseIncidentsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin"] },
  },
  {
    path: "/product-options",
    component: () => import("../views/shared/ProductOptionsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/orders/:id/print",
    component: () => import("../views/shared/OrderPrintView.vue"),
    meta: {
      requiresAuth: true,
      roles: ["owner", "super_admin", "cashier", "kitchen", "employee"],
    },
  },
  {
    path: "/menu-availability",
    component: () => import("../views/shared/MenuAvailabilityView.vue"),
    meta: {
      requiresAuth: true,
      roles: ["owner", "super_admin", "admin", "cashier"],
    },
  },
  {
    path: "/delivery-zones",
    component: () => import("../views/shared/DeliveryZonesView.vue"),
    meta: {
      title: "Zonas de delivery",
      requiresAuth: true,
      roles: ["owner", "super_admin", "admin"],
    },
  },
  {
    path: "/baja",
    component: () => import("../views/buyers/NewsletterUnsubscribeView.vue"),
    meta: { publicBuyer: true },
  },
  {
    path: "/store-operations",
    component: () => import("../views/shared/StoreOperationsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/campaigns",
    component: () => import("../views/shared/CampaignsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/mi-cuenta",
    component: () => import("../views/buyers/ShopperAccountView.vue"),
    meta: { publicBuyer: true },
  },
  {
    path: "/restaurantes",
    component: () => import("../views/buyers/RestaurantDirectoryView.vue"),
    meta: { publicBuyer: true },
  },
  {
    path: "/marketing",
    component: () => import("../views/shared/MarketingView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/care",
    component: () => import("../views/shared/CareView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/restaurants",
    component: () => import("../views/shared/RestaurantsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/locations",
    component: () => import("../views/shared/LocationsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/team",
    component: () => import("../views/shared/TeamView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/accept-business-invitation",
    component: () => import("../views/auth/RecoveryView.vue"),
    meta: { authAction: "business-invite" },
  },
  ...[
    ["/cash", "CashView", ["owner", "super_admin", "cashier"]],
    ["/orders", "OrdersView", ["owner", "super_admin", "cashier"]],
    [
      "/kitchen",
      "KitchenView",
      ["owner", "super_admin", "employee", "cashier", "kitchen"],
    ],
    [
      "/inventory",
      "InventoryView",
      ["owner", "super_admin", "admin", "employee", "cashier", "kitchen"],
    ],
    ["/expenses", "ExpensesView", ["owner", "super_admin", "admin"]],
    [
      "/customers",
      "CustomersView",
      ["owner", "super_admin", "admin", "cashier"],
    ],
    ["/notifications", "NotificationsView", null],
    ["/onboarding", "OnboardingView", ["admin"]],
    ["/platform/health", "HealthView", ["owner", "super_admin"]],
  ].map(([path, view, roles]) => ({
    path,
    component: workspaceViews[view],
    meta: { requiresAuth: true, ...(roles ? { roles } : {}) },
  })),
  {
    path: "/order",
    component: () => import("../views/public/OrderTrackingView.vue"),
  },
  {
    path: "/confirm-email",
    component: () => import("../views/auth/ConfirmEmailView.vue"),
  },
  {
    path: "/payments",
    component: () => import("../views/shared/PaymentsEntryView.vue"),
    meta: {
      requiresAuth: true,
      roles: ["owner", "super_admin", "admin", "cashier"],
    },
  },
  {
    path: "/payments/return",
    component: () => import("../views/public/PaymentReturnView.vue"),
  },
  {
    path: "/forgot-password",
    component: () => import("../views/auth/RecoveryView.vue"),
    meta: { authAction: "forgot" },
  },
  {
    path: "/reset-password",
    component: () => import("../views/auth/RecoveryView.vue"),
    meta: { authAction: "reset" },
  },
  {
    path: "/accept-invitation",
    component: () => import("../views/auth/RecoveryView.vue"),
    meta: { authAction: "invite" },
  },
  {
    path: "/account/security",
    component: () => import("../views/shared/SecurityView.vue"),
    meta: { requiresAuth: true },
  },
  {
    path: "/team/invitations",
    component: () => import("../views/shared/InvitationsView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/audit",
    component: () => import("../views/shared/AuditView.vue"),
    meta: { requiresAuth: true, roles: ["owner", "super_admin", "admin"] },
  },
  {
    path: "/",
    name: "Landing",
    component: LandingView,
  },
  {
    path: "/login",
    name: "Login",
    component: LoginView,
  },
  {
    path: "/owner",
    component: OwnerDashboard,
    meta: { requiresAuth: true, roles: ["owner", "super_admin"] },
    children: [
      {
        path: "",
        name: "OwnerHome",
        component: OwnerHomeView,
      },
      {
        path: "brands",
        name: "OwnerBrands",
        component: BrandListView,
      },
      {
        path: "tenants",
        name: "OwnerTenants",
        component: TenantListView,
      },
      {
        path: "admins",
        name: "OwnerAdmins",
        redirect: "/restaurants",
      },
    ],
  },
  {
    path: "/admin/landing",
    name: "AdminLanding",
    component: AdminLandingView,
    meta: { requiresAuth: true, roles: ["admin"] },
  },
  {
    path: "/admin",
    component: AdminDashboard,
    meta: { requiresAuth: true, roles: ["admin"] },
    children: [
      {
        path: "",
        name: "AdminHome",
        component: AdminHomeView,
      },
      {
        path: "employees",
        name: "AdminEmployees",
        redirect: "/team",
      },
      {
        path: "production",
        name: "AdminProduction",
        component: AdminProductionView,
      },
      {
        path: "catalog",
        name: "AdminCatalog",
        component: AdminCatalogView,
      },
      {
        path: "catalog/tenants",
        name: "AdminCatalogTenants",
        component: AdminCatalogTenantView,
      },
    ],
  },
  {
    path: "/tenant/:tenantId/employee",
    name: "TenantEmployeeDashboard",
    component: EmployeeDashboard,
    meta: {
      requiresAuth: true,
      roles: ["employee", "cashier", "kitchen"],
      tenantMatch: true,
    },
  },
  {
    path: "/:brand/:local/carta",
    component: () => import("../views/buyers/SlugCatalogView.vue"),
    meta: { publicBuyer: true },
  },
  {
    path: "/catalogo/:tenantId",
    component: CatalogLandingView,
    meta: { publicBuyer: true },
  },
  {
    path: "/catalogo",
    name: "CatalogLanding",
    component: CatalogLandingView,
    meta: { publicBuyer: true },
  },
  {
    path: "/unauthorized",
    name: "Unauthorized",
    component: UnauthorizedView,
  },
  {
    path: "/:pathMatch(.*)*",
    name: "NotFound",
    component: NotFoundView,
  },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition;
    if (to.hash) return { el: to.hash, top: 90 };
    return { top: 0 };
  },
});

router.beforeEach(async (to, from, next) => {
  // Buying from a public menu never depends on an application session.
  if (to.meta.publicBuyer) return next();
  await bootstrapSession();
  const auth = useAuth();

  if (to.meta.requiresAuth && !auth.isLoggedIn.value) {
    return next({ name: "Login" });
  }

  if (to.name === "Login" && auth.isLoggedIn.value) {
    const role = auth.userRole.value;
    if (role === "owner" || role === "super_admin") {
      return next({ path: "/owner" });
    }
    if (role === "admin") {
      return next({ path: "/admin" });
    }
    if (["employee", "cashier", "kitchen"].includes(role)) {
      return next({ path: `/tenant/${auth.tenantId.value}/employee` });
    }
  }

  if (to.meta.roles && !to.meta.roles.includes(auth.userRole.value)) {
    return next({ name: "Unauthorized" });
  }

  if (
    to.meta.tenantMatch &&
    auth.tenantId.value &&
    auth.tenantId.value !== to.params.tenantId
  ) {
    return next({ name: "Unauthorized" });
  }

  if (
    to.meta.requiresAuth &&
    ["admin", "employee", "cashier", "kitchen"].includes(auth.userRole.value)
  ) {
    const membershipPage =
      auth.userRole.value === "admin"
        ? "/admin/membership"
        : "/employee/membership";
    const renewal =
      auth.userRole.value === "admin" &&
      ["/payments", "/care", "/account/security"].some(
        (path) => to.path === path || to.path.startsWith(path + "/"),
      );
    if (to.path !== membershipPage) {
      try {
        const access = await useMembership().refresh();
        if (access.operational === false && !renewal)
          return next(membershipPage);
        if (
          access.premium === false &&
          ["/inventory", "/expenses", "/admin/production"].some(
            (p) => to.path === p || to.path.startsWith(p + "/"),
          )
        )
          return next(membershipPage);
      } catch {
        return next(membershipPage);
      }
    }
  }
  return next();
});

export default router;

window.addEventListener("legui:membership-expired", () => {
  if (router.currentRoute.value.meta.publicBuyer) return;
  const role = useAuth().userRole.value;
  if (role === "admin" && router.currentRoute.value.path !== "/payments")
    router.replace("/admin/membership");
  else if (["employee", "cashier", "kitchen"].includes(role))
    router.replace("/employee/membership");
});
