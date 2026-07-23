<template>
  <div class="login-wrapper">
    <!-- VISTA 1: LOGIN -->
    <div v-if="!isLoggedIn" class="login-card">
      <div class="brand-header">
        <div class="logo-icon">🍔</div>
        <h2>Leguiburger</h2>
        <p>Iniciá sesión en tu panel de control</p>
      </div>

      <form @submit.prevent="handleLogin">
        <div class="input-group">
          <label for="email">Correo electrónico</label>
          <input 
            type="email" 
            id="email" 
            v-model="email" 
            required 
            placeholder="admin@leguiburger.com" 
          />
        </div>

        <div class="input-group">
          <label for="password">Contraseña</label>
          <input 
            type="password" 
            id="password" 
            v-model="password" 
            required 
            placeholder="••••••••" 
          />
        </div>

        <button type="submit" class="btn-submit" :disabled="loading">
          <span v-if="loading" class="spinner-text">Entrando...</span>
          <span v-else>Iniciar Sesión</span>
        </button>

        <p v-if="errorMessage" class="error-msg">{{ errorMessage }}</p>
      </form>
    </div>

    <!-- VISTA 2: DASHBOARD / BIENVENIDA -->
    <div v-else class="dashboard-card">
      <div class="welcome-header">
        <div class="badge-online">🟢 Sesión Activa</div>
        <h1>¡Bienvenido de nuevo!</h1>
        <p class="subtitle">Panel de administración global</p>
      </div>

      <div class="user-info-box" v-if="employee">
        <div class="info-row">
          <span class="label">Nombre</span>
          <span class="value">{{ employee.first_name || 'Administrador' }} {{ employee.last_name || '' }}</span>
        </div>
        <div class="info-row">
          <span class="label">Correo</span>
          <span class="value">{{ employee.email }}</span>
        </div>
        <div class="info-row">
          <span class="label">Rol</span>
          <span class="role-badge">{{ employee.role }}</span>
        </div>
      </div>

      <button @click="handleLogout" class="btn-logout">Cerrar Sesión</button>
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
      employee: null
    };
  },
  mounted() {
    const token = localStorage.getItem('token');
    const storedEmployee = localStorage.getItem('employee');
    if (token && storedEmployee) {
      try {
        this.employee = JSON.parse(storedEmployee);
        this.isLoggedIn = true;
      } catch (e) {
        localStorage.clear();
      }
    }
  },
  methods: {
    async handleLogin() {
      this.errorMessage = '';
      this.loading = true;

      try {
        const headers = { 
          'Content-Type': 'application/json',
          'X-Tenant-ID': '' // Vacío para que el backend reconozca al Owner global
        };

        const response = await fetch('/api/auth/login', {
          method: 'POST',
          headers: headers,
          body: JSON.stringify({
            email: this.email,
            password: this.password
          })
        });

        const data = await response.json();

        if (!response.ok) {
          throw new Error(data.message || 'Credenciales inválidas');
        }

        localStorage.setItem('token', data.token);
        localStorage.setItem('employee', JSON.stringify(data.employee));

        this.employee = data.employee;
        this.isLoggedIn = true;
      } catch (err) {
        this.errorMessage = err.message;
      } finally {
        this.loading = false;
      }
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
/* Reset y estilos generales oscuros */
* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  background-color: #090d16; /* Fondo general ultra oscuro */
  color: #f8fafc;
  min-height: 100vh;
}

.login-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 20px;
}

/* Tarjetas con estilo Dark Mode / Glassmorphism sutil */
.login-card, .dashboard-card {
  background: #111827;
  padding: 40px;
  border-radius: 16px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5), 0 10px 10px -5px rgba(0, 0, 0, 0.4);
  width: 100%;
  max-width: 420px;
  border: 1px solid #1f2937;
}

.brand-header {
  margin-bottom: 32px;
  text-align: center;
}

.logo-icon {
  font-size: 36px;
  margin-bottom: 12px;
}

.brand-header h2 {
  font-size: 26px;
  font-weight: 700;
  color: #ffffff;
  letter-spacing: -0.5px;
  margin-bottom: 6px;
}

.brand-header p {
  font-size: 14px;
  color: #9ca3af;
}

.input-group {
  margin-bottom: 20px;
  text-align: left;
}

.input-group label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 8px;
  color: #d1d5db;
}

.input-group input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid #374151;
  border-radius: 10px;
  font-size: 15px;
  color: #ffffff !important;
  background-color: #1f2937 !important;
  outline: none;
  transition: all 0.2s ease;
}

.input-group input:focus {
  border_color: #3b82f6;
  border-color: #3b82f6;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.2);
  background-color: #111827 !important;
}

.input-group input::placeholder {
  color: #6b7280;
}

.btn-submit {
  width: 100%;
  padding: 13px;
  background-color: #3b82f6;
  color: white;
  border: none;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s, transform 0.1s;
  margin-top: 6px;
}

.btn-submit:hover {
  background-color: #2563eb;
}

.btn-submit:active {
  transform: scale(0.98);
}

.btn-submit:disabled {
  background-color: #4b5563;
  cursor: not-allowed;
}

.error-msg {
  color: #f87171;
  background-color: rgba(248, 113, 113, 0.1);
  border: 1px solid rgba(248, 113, 113, 0.2);
  padding: 10px;
  border-radius: 8px;
  font-size: 13px;
  margin-top: 20px;
  text-align: center;
  font-weight: 500;
}

.welcome-header {
  text-align: center;
  margin-bottom: 28px;
}

.badge-online {
  display: inline-block;
  background-color: rgba(16, 185, 129, 0.15);
  color: #34d399;
  font-size: 11px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 20px;
  margin-bottom: 12px;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.welcome-header h1 {
  font-size: 22px;
  font-weight: 700;
  color: #ffffff;
  margin-bottom: 4px;
}

.subtitle {
  font-size: 14px;
  color: #9ca3af;
}

.user-info-box {
  background-color: #1f2937;
  border: 1px solid #374151;
  border-radius: 12px;
  padding: 18px;
  margin-bottom: 24px;
}

.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  font-size: 14px;
  border-bottom: 1px solid #374151;
}

.info-row:last-child {
  border-bottom: none;
}

.info-row .label {
  color: #9ca3af;
  font-weight: 500;
}

.info-row .value {
  color: #f3f4f6;
  font-weight: 600;
}

.role-badge {
  background-color: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  border: 1px solid rgba(59, 130, 246, 0.3);
}

.btn-logout {
  width: 100%;
  padding: 11px;
  background-color: rgba(239, 68, 68, 0.15);
  color: #f87171;
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-logout:hover {
  background-color: rgba(239, 68, 68, 0.25);
}
</style>