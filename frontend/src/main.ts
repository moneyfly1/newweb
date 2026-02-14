import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/styles/global.scss'
import './styles/admin-mobile.css'
import './styles/mobile-cards.css'
import './styles/text-selection.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
