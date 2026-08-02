import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './style.css'
import App from './App.vue'
import router from './router'

// 深色模式跟隨系統設定，並在系統偏好變動時即時切換（不需重新整理）
const darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)')
function syncDarkClass(matches: boolean) {
  document.documentElement.classList.toggle('dark', matches)
}
syncDarkClass(darkModeQuery.matches)
darkModeQuery.addEventListener('change', (e) => syncDarkClass(e.matches))

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.use(ElementPlus)
app.mount('#app')
