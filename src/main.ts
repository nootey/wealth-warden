import './style/global.scss'
import './style/auth.scss'
import './style/auth_input_shake_animation.scss';

import { createApp } from 'vue'
import App from './App.vue'
import router from "./services/router";
import { createPinia } from 'pinia';
import PrimeVue from 'primevue/config';
import Material from '@primevue/themes/material';
import Tooltip from "primevue/tooltip";
import {Button} from "primevue";
import {Checkbox} from "primevue";
import {InputText} from "primevue";



const app = createApp(App);
const pinia = createPinia();

app.directive('tooltip', Tooltip);
app.component("Button", Button);
app.component("Checkbox", Checkbox);
app.component("InputText", InputText);


app.use(pinia);
app.use(router);
app.use(PrimeVue, {
    theme: {
        preset: Material,
        options: {
            darkModeSelector: '.my-app-dark',
        }
    }
});

app.component('App', App);
app.mount('#app');
