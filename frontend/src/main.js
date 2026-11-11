import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import Aura from '@primeuix/themes/aura'
import ToastService from 'primevue/toastservice'

import 'primeicons/primeicons.css'
import '@fortawesome/fontawesome-free/css/all.min.css'
import '@xterm/xterm/css/xterm.css'
import './styles/app.css'

import App from './App.vue'
import router from './router'

document.documentElement.classList.add('dark')

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(PrimeVue, {
  theme: {
    preset: Aura,
    options: {
      darkModeSelector: '.dark',
      cssLayer: { name: 'primevue', order: 'theme, base, primevue' },
    },
  },
})
app.use(ToastService)

app.mount('#app')
