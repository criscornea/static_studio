import { defineConfig, globalIgnores } from 'eslint/config'
import tseslint from 'typescript-eslint'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import pluginVitest from '@vitest/eslint-plugin'
import pluginOxlint from 'eslint-plugin-oxlint'
import skipFormatting from 'eslint-config-prettier/flat'

export default defineConfig(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx}'],
  },

  globalIgnores(['**/dist/**', '**/dist-ssr/**', '**/coverage/**']),

  tseslint.configs.recommended,
  reactHooks.configs.flat.recommended,
  { files: ['src/**/*.tsx'], ...reactRefresh.configs.vite },

  // shadcn/ui files export variant helpers next to components.
  { files: ['src/components/ui/**'], rules: { 'react-refresh/only-export-components': 'off' } },

  {
    ...pluginVitest.configs.recommended,
    files: ['src/**/*.test.{ts,tsx}'],
  },

  ...pluginOxlint.buildFromOxlintConfigFile('.oxlintrc.json'),

  skipFormatting,
)
