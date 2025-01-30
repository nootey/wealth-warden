import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router"; // if using Vue Router

const app = createApp(App);
app.use(createPinia()); // Register Pinia
app.use(router); // Register Router (if applicable)
app.mount("#app");