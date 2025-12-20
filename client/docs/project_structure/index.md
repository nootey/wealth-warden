## Project structure

The project uses a hybrid approach when organizing folders.

```
root/
├── docs/               # Documentation for the project
├── src/
│   ├── assets/         # Static assets (images, fonts, icons, etc.)
│   ├── models/         # TypeScript interfaces and types
│   │   └── Auth.ts
│   ├── services/       # Global services (API calls, authentication, state management)
│   │   ├── router.ts   # Vue Router configuration
│   │   ├── store.ts    # Pinia store
│   │   └── api/        # Axios instance setup (global API configuration)
│   ├── utils/          # Helper functions and utilities
│   ├── _vue/           # Vue-related source files
│   │   ├── views/      # Page-level views (used in routing)
│   │   │   └── Dashboard.vue
│   │   ├── components/ # Shared components across features
│   │   │   ├── auth/
│   │   │      └── AuthSkeleton.vue
│   │   ├── features/   # Feature modules
│   │   │   ├── auth/
│   │   └──    └── Login.vue
│   ├── style/          # Global styles (SCSS, Tailwind, variables)
│   └── App.vue         # Root Vue component
```
