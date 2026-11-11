import { createApp } from 'vue'
import './style.css'
import './design.css'
import './styles/operations.css'
import './styles/consistency.css'
import App from './App.vue'
import router from './router/index.js'
import { installFormCare } from './plugins/formCare.js'
import { reportClientError } from './services/monitoring.js'

const app = createApp(App)
installFormCare(app, router)
app.config.errorHandler = () => reportClientError('unhandled')
router.onError(() => reportClientError('navigation'))
window.addEventListener('unhandledrejection', () => reportClientError('unhandled'))
app.use(router).mount('#app')

window.addEventListener('legui:logout', event => {
  if (router.currentRoute.value.meta.requiresAuth) router.replace({ path: '/login', query: event.detail ? { reason: event.detail } : {} });
});
