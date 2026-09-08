import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { useAuthStore } from './stores/auth'
import './assets/styles/main.css'

const app = createApp(App)

app.use(createPinia())

// The auth store owns the tokens the axios interceptors need, so it is wired up
// before the router runs its first guard.
useAuthStore().install()

app.use(router)
app.mount('#app')
