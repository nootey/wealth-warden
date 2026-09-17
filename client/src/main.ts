// Styles
import "./style/tailwind.css";
import "./style/global.scss";
import "./style/overrride.scss";

import { createApp } from "vue";
import App from "./App.vue";
import router from "./services/router/main.ts";
import { createPinia } from "pinia";

// PrimeVue core + theme
import PrimeVue from "primevue/config";
import Aura from "@primeuix/themes/aura";
import { definePreset } from "@primeuix/themes";

const surface = (name: string) => ({
  0: "#ffffff",
  50: `{${name}.50}`,
  100: `{${name}.100}`,
  200: `{${name}.200}`,
  300: `{${name}.300}`,
  400: `{${name}.400}`,
  500: `{${name}.500}`,
  600: `{${name}.600}`,
  700: `{${name}.700}`,
  800: `{${name}.800}`,
  900: `{${name}.900}`,
  950: `{${name}.950}`,
});

const AppPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: "{indigo.50}",
      100: "{indigo.100}",
      200: "{indigo.200}",
      300: "{indigo.300}",
      400: "{indigo.400}",
      500: "{indigo.500}",
      600: "{indigo.600}",
      700: "{indigo.700}",
      800: "{indigo.800}",
      900: "{indigo.900}",
      950: "{indigo.950}",
    },
    colorScheme: {
      light: { surface: surface("stone") },
      dark: { surface: surface("neutral") },
    },
  },
});

// PrimeVue services & directives
import ConfirmationService from "primevue/confirmationservice";
import Tooltip from "primevue/tooltip";
import Ripple from "primevue/ripple";
import ToastService from "primevue/toastservice";

// App
const app = createApp(App);

// Plugins
app.use(createPinia());
app.use(router);
app.use(PrimeVue, {
  theme: {
    preset: AppPreset,
    options: {
      prefix: "p",
      darkModeSelector: ".my-app-dark",
      cssLayer: false,
    },
  },
  ripple: true,
  pt: {
    panel: {
      root: { class: "rounded-xl" },
    },
  },
});

app.use(ToastService);
app.use(ConfirmationService);

// Directives
app.directive("tooltip", Tooltip);
app.directive("ripple", Ripple);

app.mount("#app");
