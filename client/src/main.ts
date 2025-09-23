// Styles
import '../node_modules/primeflex/primeflex.css';
import './style/global.scss'
import './style/overrride.scss'

import { createApp } from 'vue'
import App from './App.vue'
import router from "./services/router/main.ts";
import { createPinia } from 'pinia';

// PrimeVue core + theme
import PrimeVue from 'primevue/config';
import Material from '@primevue/themes/material';

// PrimeVue services & directives
import ConfirmationService from 'primevue/confirmationservice';
import Tooltip from "primevue/tooltip";
import Ripple from 'primevue/ripple';
import ToastService from 'primevue/toastservice';

// App
const app = createApp(App);

// Plugins
app.use(createPinia());
app.use(router);
app.use(PrimeVue, {
    theme: {
        preset: Material,
        options: {
            prefix: 'p',
            darkModeSelector: '.my-app-dark',
            cssLayer: false
        }
    },
    ripple: true
});

app.use(ToastService);
app.use(ConfirmationService);

// Directives
app.directive('tooltip', Tooltip);
app.directive('ripple', Ripple);

app.mount('#app');
