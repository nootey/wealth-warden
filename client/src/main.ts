// Styles
import "@fontsource-variable/geist";
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
  primitive: {
    borderRadius: {
      none: "0",
      xs: "3px",
      sm: "5px",
      md: "8px",
      lg: "10px",
      xl: "14px",
    },
  },
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
    focusRing: { width: "2px", style: "solid", offset: "2px" },
    colorScheme: {
      light: {
        surface: surface("stone"),
        formField: {
          background: "{surface.0}",
          borderColor: "{surface.200}",
          hoverBorderColor: "{surface.400}",
        },
        content: { borderColor: "{surface.200}" },
      },
      dark: {
        surface: surface("neutral"),
        formField: {
          background: "{surface.950}",
          borderColor: "{surface.700}",
          hoverBorderColor: "{surface.500}",
        },
        content: { background: "{surface.900}", borderColor: "{surface.800}" },
        overlay: {
          popover: {
            background: "{surface.900}",
            borderColor: "{surface.800}",
          },
          modal: { background: "{surface.900}", borderColor: "{surface.800}" },
          select: { background: "{surface.900}", borderColor: "{surface.800}" },
        },
      },
    },
  },
  components: {
    panel: {
      header: { padding: "1.25rem 1.5rem 0.75rem" },
      content: { padding: "0.5rem 1.5rem 1.5rem" },
      title: { fontWeight: "600" },
    },
    datatable: {
      headerCell: { padding: "0.75rem 1rem" },
      bodyCell: { padding: "0.8rem 1rem" },
      columnTitle: { fontWeight: "500" },
      colorScheme: {
        light: {
          headerCell: { background: "transparent", color: "{surface.500}" },
          header: { background: "transparent" },
          row: { background: "transparent" },
        },
        dark: {
          headerCell: { background: "transparent", color: "{surface.400}" },
          header: { background: "transparent" },
          row: { background: "transparent" },
        },
      },
    },
    dialog: {
      header: { padding: "1.5rem 1.5rem 1rem" },
      content: { padding: "0 1.5rem 1.5rem" },
      title: { fontSize: "1.25rem", fontWeight: "600" },
    },
    drawer: {
      colorScheme: {
        dark: { root: { background: "{surface.900}" } },
      },
    },
    tooltip: {
      colorScheme: {
        light: { root: { background: "{surface.900}", color: "{surface.0}" } },
        dark: { root: { background: "{surface.100}", color: "{surface.950}" } },
      },
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
      root: { class: "rounded-2xl" },
      title: { class: "text-lg tracking-tight" },
    },
    dialog: {
      title: { class: "tracking-tight" },
    },
  },
});

app.use(ToastService);
app.use(ConfirmationService);

// Directives
app.directive("tooltip", Tooltip);
app.directive("ripple", Ripple);

app.mount("#app");
