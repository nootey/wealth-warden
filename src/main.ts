import { createApp } from 'vue'
import './style/global.scss'
import App from './App.vue'
import router from "./services/router";
import { createPinia } from 'pinia';
import PrimeVue from 'primevue/config';
import Material from '@primevue/themes/material';
import {Button} from "primevue";


const app = createApp(App);
const pinia = createPinia();

app.component("Button", Button);

app.use(router);
app.use(PrimeVue, {
    // Default theme configuration
    theme: {
        preset: Material,
        options: {
            darkModeSelector: '.my-app-dark',
        }
    }
});

app.component('App', App);
app.mount('#app');
