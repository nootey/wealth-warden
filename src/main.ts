import '../node_modules/primeflex/primeflex.css';
import './style/global.scss'
import './style/overrride.scss'

import { createApp } from 'vue'
import App from './App.vue'
import router from "./services/router";
import { createPinia } from 'pinia';
import PrimeVue from 'primevue/config';
import Material from '@primevue/themes/material';
import Tooltip from "primevue/tooltip";
import {Toast} from "primevue";
import {ToastService} from "primevue";
import ConfirmationService from 'primevue/confirmationservice';
import {Button} from "primevue";
import {Checkbox} from "primevue";
import {InputText} from "primevue";
import {FloatLabel} from "primevue";
import {DataTable} from "primevue";
import {Column} from "primevue";
import {AutoComplete} from "primevue";
import {InputGroup} from "primevue";
import {InputGroupAddon} from "primevue";
import {DatePicker} from "primevue";
import {InputNumber} from "primevue";
import {Paginator} from "primevue";
import {Dialog} from "primevue";
import {MultiSelect} from "primevue";
import {ProgressSpinner} from "primevue";
import {ConfirmPopup} from "primevue";
import {Popover} from "primevue";
import {SelectButton} from "primevue";

const app = createApp(App);
const pinia = createPinia();

app.directive('tooltip', Tooltip);
app.component("Toast", Toast);
app.component("Button", Button);
app.component("Checkbox", Checkbox);
app.component("InputText", InputText);
app.component("FloatLabel", FloatLabel);
app.component("DataTable", DataTable);
app.component("Column", Column);
app.component("AutoComplete", AutoComplete);
app.component("InputGroup", InputGroup);
app.component("InputGroupAddon", InputGroupAddon);
app.component("DatePicker", DatePicker);
app.component("InputNumber", InputNumber);
app.component("Paginator", Paginator);
app.component("Dialog", Dialog);
app.component("MultiSelect", MultiSelect);
app.component("ProgressSpinner", ProgressSpinner);
app.component("ConfirmPopup", ConfirmPopup);
app.component("Popover", Popover);
app.component("SelectButton", SelectButton);

app.use(pinia);
app.use(router);
app.use(PrimeVue, {
    theme: {
        preset: Material,
        options: {
            prefix: 'p',
            darkModeSelector: '.my-app-dark',
            cssLayer: false
        }
    }
});
app.use(ToastService);
app.use(ConfirmationService);



app.component('App', App);
app.mount('#app');
