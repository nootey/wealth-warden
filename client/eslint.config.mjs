import eslint from '@eslint/js';
import { defineConfig } from 'eslint/config';
import tseslint from 'typescript-eslint';
import pluginVue from 'eslint-plugin-vue';

export default defineConfig(
    {
        ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**', '**/node_modules/**']
    },
    eslint.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginVue.configs['flat/recommended'],
    {
        rules: {
            '@typescript-eslint/no-explicit-any': 'off',
            '@typescript-eslint/no-unused-vars': 'warn',
            'vue/multi-word-component-names': 'off',
            'vue/no-v-html': 'off',
            '@typescript-eslint/ban-ts-comment': 'off',
            'no-undef': 'off'
        }
    }
);