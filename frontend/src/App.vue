<template>
  <div class="app-container">
    <!-- ================= VISTA 1: LOGIN ================= -->
    <div v-if="!isLoggedIn" class="login-wrapper">
      <div class="login-card">
        <div class="brand-header">
          <div class="logo-icon">🍔</div>
          <h2>Leguiburger SaaS</h2>
          <p>Iniciar Sesión</p>
        </div>

        <form @submit.prevent="handleLogin">
          <div class="input-group">
            <label for="email">Correo electrónico</label>
            <input type="email" id="email" v-model="email" required placeholder="admin@leguiburger.com" />
          </div>
          <div class="input-group">
            <label for="password">Contraseña</label>
            <input type="password" id="password" v-model="password" required placeholder="••••••••" />
          </div>
          <button type="submit" class="btn-submit" :disabled="loading">
            <span v-if="loading">Entrando...</span>
            <span v-else>Iniciar Sesión</span>
          </button>
          <p v-if="errorMessage" class="error-msg">{{ errorMessage }}</p>
        </form>
      </div>
    </div>

    <!-- ================= VISTA 2: PANEL DE OWNER ================= -->
    <div v-else-if="employee && employee.role === 'owner'" class="dashboard-layout">
      <!-- SIDEBAR -->
      <aside class="sidebar">
        <div class="sidebar-brand">
          <span>🍔</span> Leguiburger Admin
        </div>
        <nav class="sidebar-nav">
          <button :class="{ active: currentTab === 'tenants' }" @click="currentTab = 'tenants'">
            🏢 Gestión de Tenants
          </button>
          <button :class="{ active: currentTab === 'admins' }" @click="currentTab = 'admins'">
            👥 Gestión de Admins
          </button>
        </nav>
        <div class="sidebar-footer">
          <div class="owner-info">
            <span class="owner-name">{{ employee?.first_name || 'Super Owner' }}</span>
            <span class="owner-role">SUPER ADMIN</span>
          </div>
          <button @click="handleLogout" class="btn-logout-sidebar">Cerrar Sesión</button>
        </div>
      </aside>

      <!-- CONTENIDO PRINCIPAL -->
      <main class="main-content">
        <!-- TABS: TENANTS -->
        <div v-if="currentTab === 'tenants'" class="section-view">
          <div class="section-header">
            <div>
              <h2>Negocios Contratados (Tenants)</h2>
              <p>Administrá el estado de suscripción y acceso de cada comercio.</p>
            </div>
            <button class="btn-primary" @click="openTenantModal = true">+ Nuevo Negocio</button>
          </div>

          <div class="table-container">
            <table>
              <thead>
                <tr>
                  <th>Comercio</th>
                  <th>Subdominio</th>
                  <th>Tax ID</th>
                  <th>Estado</th>
                  <th>Acciones</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="tenant in tenantsList" :key="tenant.id">
                  <td><strong>{{ tenant.name }}</strong></td>
                  <td><code>{{ tenant.subdomain }}</code></td>
                  <td>{{ tenant.tax_id }}</td>
                  <td>
                    <span :class="['badge', tenant.active ? 'badge-active' : 'badge-inactive']">
                      {{ tenant.active ? 'Activo' : 'Suspendido' }}
                    </span>
                  </td>
                  <td>
                    <div class="action-buttons">
                      <button @click="toggleTenantStatus(tenant)" :class="['btn-sm', tenant.active ? 'btn-warning' : 'btn-success']">
                        {{ tenant.active ? 'Suspender' : 'Reactivar' }}
                      </button>
                      <button @click="openEditTenant(tenant)" class="btn-sm btn-info">Editar</button>
                      <button @click="deleteTenant(tenant.id)" class="btn-sm btn-danger">Eliminar</button>
                    </div>
                  </td>
                </tr>
                <tr v-if="tenantsList.length === 0">
                  <td colspan="5" class="empty-text">No hay negocios registrados.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- TABS: ADMINS -->
        <div v-if="currentTab === 'admins'" class="section-view">
          <div class="section-header">
            <div>
              <h2>Administradores de Negocios</h2>
              <p>Controlá los usuarios con acceso de gestión.</p>
            </div>
            <button class="btn-primary" @click="openAdminModal = true">+ Nuevo Admin</button>
          </div>

          <div class="table-container">
            <table>
              <thead>
                <tr>
                  <th>Nombre</th>
                  <th>Email</th>
                  <th>Tenant Asignado</th>
                  <th>Estado</th>
                  <th>Acciones</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="admin in adminsList" :key="admin.id">
                  <td><strong>{{ admin.first_name }} {{ admin.last_name }}</strong></td>
                  <td>{{ admin.email }}</td>
                  <td><code>{{ admin.tenant_id }}</code></td>
                  <td>
                    <span :class="['badge', admin.active ? 'badge-active' : 'badge-inactive']">
                      {{ admin.active ? 'Activo' : 'Inactivo' }}
                    </span>
                  </td>
                  <td>
                    <div class="action-buttons">
                      <button @click="toggleAdminStatus(admin)" :class="['btn-sm', admin.active ? 'btn-warning' : 'btn-success']">
                        {{ admin.active ? 'Desactivar' : 'Activar' }}
                      </button>
                      <button @click="openEditAdmin(admin)" class="btn-sm btn-info">Editar</button>
                      <button @click="deleteAdmin(admin.id)" class="btn-sm btn-danger">Eliminar</button>
                    </div>
                  </td>
                </tr>
                <tr v-if="adminsList.length === 0">
                  <td colspan="5" class="empty-text">No hay administradores registrados.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>

    <!-- ================= VISTA 3: ADMIN (HOLA MUNDO) ================= -->
    <div v-else class="admin-welcome-wrapper">
      <div class="welcome-card">
        <div class="logo-icon">👋</div>
        <h1>¡Hola Mundo!</h1>
        <p>Bienvenido al sistema, <strong>{{ employee?.email }}</strong>.</p>
        <span class="role-badge">Rol: Admin de Sucursal</span>
        <button @click="handleLogout" class="btn-primary" style="margin-top: 24px;">Cerrar Sesión</button>
      </div>
    </div>

    <!-- MODALES TENANTS -->
    <div v-if="openTenantModal || isEditingTenant" class="modal-overlay">
      <div class="modal-card">
        <h3>{{ isEditingTenant ? 'Editar Negocio' : 'Registrar Nuevo Negocio' }}</h3>
        <p>Completá los datos del comercio.</p>
        <form @submit.prevent="isEditingTenant ? updateTenant() : createTenant()">
          <div class="input-group">
            <label>Nombre del Comercio</label>
            <input type="text" v-model="tenantForm.name" required />
          </div>
          <div class="input-group">
            <label>Subdominio / ID único</label>
            <input type="text" v-model="tenantForm.subdomain" required :disabled="isEditingTenant" />
          </div>
          <div class="input-group">
            <label>Tax ID (CUIT / RUC)</label>
            <input type="text" v-model="tenantForm.tax_id" required />
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeTenantModal">Cancelar</button>
            <button type="submit" class="btn-primary">{{ isEditingTenant ? 'Guardar' : 'Crear' }}</button>
          </div>
        </form>
      </div>
    </div>

    <!-- MODALES ADMINS -->
    <div v-if="openAdminModal || isEditingAdmin" class="modal-overlay">
      <div class="modal-card">
        <h3>{{ isEditingAdmin ? 'Editar Administrador' : 'Nuevo Administrador' }}</h3>
        <p>Completá los datos del usuario.</p>
        <form @submit.prevent="isEditingAdmin ? updateAdmin() : createAdmin()">
          <div class="input-group">
            <label>Nombre</label>
            <input type="text" v-model="adminForm.first_name" required />
          </div>
          <div class="input-group">
            <label>Apellido</label>
            <input type="text" v-model="adminForm.last_name" required />
          </div>
          <div class="input-group">
            <label>Email</label>
            <input type="email" v-model="adminForm.email" required />
          </div>
          <div class="input-group" v-if="!isEditingAdmin">
            <label>Contraseña</label>
            <input type="password" v-model="adminForm.password" required />
          </div>
          <div class="input-group">
            <label>Tenant ID (Asignación)</label>
            <input type="text" v-model="adminForm.tenant_id" required />
          </div>
          <div class="modal-actions">
            <button type="button" class="btn-secondary" @click="closeAdminModal">Cancelar</button>
            <button type="submit" class="btn-primary">{{ isEditingAdmin ? 'Guardar' : 'Crear' }}</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      isLoggedIn: false,
      email: '',
      password: '',
      loading: false,
      errorMessage: '',
      employee: null,
      currentTab: 'tenants',
      
      tenantsList: [],
      openTenantModal: false,
      isEditingTenant: false,
      tenantForm: { id: null, name: '', subdomain: '', tax_id: '' },

      adminsList: [],
      openAdminModal: false,
      isEditingAdmin: false,
      adminForm: { id: null, first_name: '', last_name: '', email: '', password: '', tenant_id: '', role: 'admin' }
    };
  },
  mounted() {
    const token = localStorage.getItem('token');
    const storedEmployee = localStorage.getItem('employee');
    if (token && storedEmployee) {
      try {
        this.employee = JSON.parse(storedEmployee);
        this.isLoggedIn = true;
        if (this.employee.role === 'owner') {
          this.loadInitialData();
        }
      } catch (e) {
        this.handleLogout();
      }
    }
  },
  methods: {
    async handleLogin() {
      this.errorMessage = '';
      this.loading = true;
      try {
        const response = await fetch('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email: this.email, password: this.password })
        });
        const data = await response.json();
        if (!response.ok) throw new Error(data.message || 'Credenciales inválidas');

        localStorage.setItem('token', data.token);
        localStorage.setItem('employee', JSON.stringify(data.employee));
        this.employee = data.employee;
        this.isLoggedIn = true;
        
        if (this.employee.role === 'owner') {
          this.loadInitialData();
        }
      } catch (err) {
        this.errorMessage = err.message;
      } finally {
        this.loading = false;
      }
    },
    loadInitialData() {
      this.fetchTenants();
      this.fetchAdmins();
    },
    getHeaders() {
      return { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}` 
      };
    },

    async fetchTenants() {
      try {
        const res = await fetch('/api/tenants', {
          headers: this.getHeaders()
        });

        const data = await res.json();

        console.log(data);

        this.tenantsList = data;
      } catch (err) {
        console.error(err);
      }
    },
    async createTenant() {
      try {
        const res = await fetch('/api/tenants', {
          method: 'POST', headers: this.getHeaders(), body: JSON.stringify(this.tenantForm)
        });
        if (res.ok) {
          this.closeTenantModal();
          this.fetchTenants();
        } else throw new Error("Error al crear tenant");
      } catch (err) { alert(err.message); }
    },
    async updateTenant() {
      try {
        const res = await fetch(`/api/tenants/${this.tenantForm.id}`, {
          method: 'PUT', headers: this.getHeaders(), body: JSON.stringify(this.tenantForm)
        });
        if (res.ok) {
          this.closeTenantModal();
          this.fetchTenants();
        } else throw new Error("Error al actualizar tenant");
      } catch (err) { alert(err.message); }
    },
    async toggleTenantStatus(tenant) {
      try {
        const res = await fetch(`/api/tenants/${tenant.id}`, {
          method: 'PUT', headers: this.getHeaders(), body: JSON.stringify({ active: !tenant.active })
        });
        if (res.ok) tenant.active = !tenant.active;
      } catch (err) { console.error("Error al cambiar estado:", err); }
    },
    async deleteTenant(id) {
      if (!confirm("¿Seguro que querés eliminar este negocio?")) return;
      try {
        const res = await fetch(`/api/tenants/${id}`, { method: 'DELETE', headers: this.getHeaders() });
        if (res.ok) this.fetchTenants();
      } catch (err) { console.error("Error al eliminar:", err); }
    },
    openEditTenant(tenant) {
      this.tenantForm = { ...tenant };
      this.isEditingTenant = true;
    },
    closeTenantModal() {
      this.openTenantModal = false;
      this.isEditingTenant = false;
      this.tenantForm = { id: null, name: '', subdomain: '', tax_id: '' };
    },

    async fetchAdmins() {
      try {
        const res = await fetch('/api/employees', { headers: this.getHeaders() });
        if (res.ok) this.adminsList = await res.json();
      } catch (err) { console.error("Error al cargar admins:", err); }
    },
    async createAdmin() {
      try {
        const res = await fetch('/api/employees', {
          method: 'POST', headers: this.getHeaders(), body: JSON.stringify(this.adminForm)
        });
        if (res.ok) {
          this.closeAdminModal();
          this.fetchAdmins();
        } else throw new Error("Error al crear administrador");
      } catch (err) { alert(err.message); }
    },
    async updateAdmin() {
      try {
        const res = await fetch(`/api/employees/${this.adminForm.id}`, {
          method: 'PUT', headers: this.getHeaders(), body: JSON.stringify(this.adminForm)
        });
        if (res.ok) {
          this.closeAdminModal();
          this.fetchAdmins();
        } else throw new Error("Error al actualizar administrador");
      } catch (err) { alert(err.message); }
    },
    async toggleAdminStatus(admin) {
      try {
        const res = await fetch(`/api/employees/${admin.id}`, {
          method: 'PUT', headers: this.getHeaders(), body: JSON.stringify({ active: !admin.active })
        });
        if (res.ok) admin.active = !admin.active;
      } catch (err) { console.error("Error al cambiar estado:", err); }
    },
    async deleteAdmin(id) {
      if (!confirm("¿Seguro que querés eliminar este administrador?")) return;
      try {
        const res = await fetch(`/api/employees/${id}`, { method: 'DELETE', headers: this.getHeaders() });
        if (res.ok) this.fetchAdmins();
      } catch (err) { console.error("Error al eliminar:", err); }
    },
    openEditAdmin(admin) {
      this.adminForm = { ...admin };
      this.isEditingAdmin = true;
    },
    closeAdminModal() {
      this.openAdminModal = false;
      this.isEditingAdmin = false;
      this.adminForm = { id: null, first_name: '', last_name: '', email: '', password: '', tenant_id: '', role: 'admin' };
    },

    handleLogout() {
      localStorage.clear();
      this.isLoggedIn = false;
      this.employee = null;
      this.email = '';
      this.password = '';
    }
  }
};
</script>

<style>
/* Estilos Globales y Reset */
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background-color: #090d16; color: #f8fafc; font-size: 15px; }

/* Login */
.login-wrapper { display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 20px; }
.login-card { background: #111827; padding: 40px; border-radius: 16px; border: 1px solid #1f2937; width: 100%; max-width: 420px; box-shadow: 0 20px 25px rgba(0,0,0,0.5); }
.brand-header { margin-bottom: 28px; text-align: center; }
.logo-icon { font-size: 40px; margin-bottom: 10px; }
.brand-header h2 { font-size: 26px; color: #fff; font-weight: 700; }
.brand-header p { font-size: 14px; color: #9ca3af; }

.input-group { margin-bottom: 18px; text-align: left; }
.input-group label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 6px; color: #d1d5db; }
.input-group input { width: 100%; padding: 12px 14px; border: 1px solid #374151; border-radius: 8px; font-size: 14px; color: #fff; background-color: #1f2937; outline: none; }
.input-group input:focus { border-color: #3b82f6; box-shadow: 0 0 0 3px rgba(59,130,246,0.2); }

.btn-submit, .btn-primary { width: 100%; padding: 12px; background-color: #3b82f6; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; transition: background 0.2s; }
.btn-submit:hover, .btn-primary:hover { background-color: #2563eb; }

.error-msg { color: #f87171; background: rgba(248,113,113,0.1); border: 1px solid rgba(248,113,113,0.2); padding: 10px; border-radius: 8px; font-size: 13px; margin-top: 16px; text-align: center; }

/* Vista Admin (Hola Mundo) */
.admin-welcome-wrapper { display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 20px; }
.welcome-card { background: #111827; padding: 40px; border-radius: 16px; border: 1px solid #1f2937; width: 100%; max-width: 420px; text-align: center; box-shadow: 0 20px 25px rgba(0,0,0,0.5); }
.welcome-card h1 { font-size: 28px; color: #fff; margin-bottom: 10px; }
.welcome-card p { font-size: 14px; color: #9ca3af; margin-bottom: 16px; }
.role-badge { display: inline-block; background: rgba(59,130,246,0.15); color: #60a5fa; border: 1px solid rgba(59,130,246,0.3); padding: 6px 14px; border-radius: 20px; font-size: 12px; font-weight: 600; }

/* Dashboard Layout */
.dashboard-layout { display: flex; min-height: 100vh; }
.sidebar { width: 280px; background-color: #111827; border-right: 1px solid #1f2937; display: flex; flex-direction: column; padding: 28px 20px; }
.sidebar-brand { font-size: 18px; font-weight: 700; color: #fff; margin-bottom: 32px; display: flex; align-items: center; gap: 10px; }
.sidebar-nav { display: flex; flex-direction: column; gap: 8px; flex: 1; }
.sidebar-nav button { background: transparent; border: none; color: #9ca3af; padding: 12px 16px; text-align: left; border-radius: 8px; font-size: 14px; font-weight: 500; cursor: pointer; transition: all 0.2s; }
.sidebar-nav button:hover, .sidebar-nav button.active { background-color: #1f2937; color: #fff; }

.sidebar-footer { border-top: 1px solid #1f2937; padding-top: 20px; display: flex; flex-direction: column; gap: 12px; }
.owner-info { display: flex; flex-direction: column; }
.owner-name { font-size: 14px; font-weight: 600; color: #fff; }
.owner-role { font-size: 11px; color: #3b82f6; font-weight: 700; }
.btn-logout-sidebar { background: rgba(239,68,68,0.1); color: #f87171; border: 1px solid rgba(239,68,68,0.2); padding: 10px; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; }
.btn-logout-sidebar:hover { background: rgba(239,68,68,0.2); }

/* Main Content */
.main-content { flex: 1; padding: 40px; background-color: #090d16; overflow-y: auto; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 32px; }
.section-header h2 { font-size: 24px; color: #fff; font-weight: 700; margin-bottom: 4px; }
.section-header p { font-size: 14px; color: #9ca3af; }
.section-header .btn-primary { width: auto; padding: 10px 20px; }

/* Tablas y Badges */
.table-container { background: #111827; border: 1px solid #1f2937; border-radius: 12px; overflow: hidden; }
table { width: 100%; border-collapse: collapse; text-align: left; font-size: 14px; }
th { background-color: #1f2937; color: #9ca3af; font-weight: 600; padding: 16px 20px; }
td { padding: 16px 20px; border-top: 1px solid #1f2937; color: #e5e7eb; }
code { background: #1f2937; padding: 4px 8px; border-radius: 4px; font-size: 12px; color: #60a5fa; }
.empty-text { text-align: center; color: #9ca3af; padding: 30px !important; }

.badge { padding: 6px 12px; border-radius: 20px; font-size: 12px; font-weight: 600; display: inline-block; }
.badge-active { background: rgba(16,185,129,0.15); color: #34d399; border: 1px solid rgba(16,185,129,0.3); }
.badge-inactive { background: rgba(239,68,68,0.15); color: #f87171; border: 1px solid rgba(239,68,68,0.3); }

/* Botones de acción */
.action-buttons { display: flex; gap: 8px; flex-wrap: wrap; }
.btn-sm { padding: 6px 12px; border-radius: 6px; font-size: 12px; font-weight: 600; cursor: pointer; border: none; }
.btn-danger { background: rgba(239,68,68,0.2); color: #f87171; border: 1px solid rgba(239,68,68,0.3); }
.btn-danger:hover { background: rgba(239,68,68,0.4); }
.btn-success { background: rgba(16,185,129,0.2); color: #34d399; border: 1px solid rgba(16,185,129,0.3); }
.btn-success:hover { background: rgba(16,185,129,0.4); }
.btn-warning { background: rgba(245,158,11,0.2); color: #fbbf24; border: 1px solid rgba(245,158,11,0.3); }
.btn-warning:hover { background: rgba(245,158,11,0.4); }
.btn-info { background: rgba(59,130,246,0.2); color: #60a5fa; border: 1px solid rgba(59,130,246,0.3); }
.btn-info:hover { background: rgba(59,130,246,0.4); }

/* Modal */
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.7); display: flex; justify-content: center; align-items: center; z-index: 100; }
.modal-card { background: #111827; border: 1px solid #1f2937; padding: 32px; border-radius: 16px; width: 100%; max-width: 440px; box-shadow: 0 20px 25px rgba(0,0,0,0.5); }
.modal-card h3 { font-size: 20px; color: #fff; margin-bottom: 6px; font-weight: 700; }
.modal-card p { font-size: 13px; color: #9ca3af; margin-bottom: 20px; }
.modal-actions { display: flex; gap: 12px; margin-top: 24px; }
.btn-secondary { background: #374151; color: #fff; border: none; padding: 12px; border-radius: 8px; cursor: pointer; flex: 1; font-weight: 600; font-size: 14px; }
.btn-secondary:hover { background: #4b5563; }
.modal-actions .btn-primary { flex: 1; }
</style>