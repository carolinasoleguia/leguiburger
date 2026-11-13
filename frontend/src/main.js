import { createApp } from 'vue'
import './style.css'
import './design.css'
import './styles/operations.css'
import './styles/consistency.css'
import './styles/motion.css'
import './styles/stable-loading.css'
import './styles/workspace.css'
import App from './App.vue'
import router from './router/index.js'
import { installFormCare } from './plugins/formCare.js'
import { reportClientError } from './services/monitoring.js'
import { installActivity } from './services/activity.js'

document.documentElement.dataset.motion = 'full'
localStorage.removeItem('legui-motion')
const app = createApp(App)
installFormCare(app, router)
installActivity(router)
app.config.errorHandler = () => reportClientError('unhandled')
router.onError(() => reportClientError('navigation'))
window.addEventListener('unhandledrejection', () => reportClientError('unhandled'))
app.use(router).mount('#app')

window.addEventListener('legui:logout', event => {
  if (router.currentRoute.value.meta.requiresAuth) router.replace({ path: '/login', query: event.detail ? { reason: event.detail } : {} });
});
