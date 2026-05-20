// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@vueuse/nuxt',
    '@pinia/nuxt'
  ],

  devtools: {
    enabled: true
  },

  css: ['~/assets/css/main.css'],

  routeRules: {
    '/api/**': {
      proxy: `${process.env.BACKEND_URL ?? 'http://localhost:8080'}/api/**`
    }
  },

  icon: {
    serverBundle: false,
    clientBundle: {
      scan: true,
      sizeLimitKb: 2048,
      icons: [
        // Nuxt UI v3 internal icons (not in project files, must be listed explicitly)
        'lucide:arrow-up-right',
        'lucide:chevron-down',
        'lucide:chevron-left',
        'lucide:chevron-right',
        'lucide:chevron-up',
        'lucide:check',
        'lucide:circle-alert',
        'lucide:circle-check',
        'lucide:circle-x',
        'lucide:ellipsis',
        'lucide:ellipsis-vertical',
        'lucide:eye',
        'lucide:eye-off',
        'lucide:grip-vertical',
        'lucide:info',
        'lucide:loader',
        'lucide:menu',
        'lucide:minus',
        'lucide:monitor',
        'lucide:moon',
        'lucide:panel-left',
        'lucide:panel-left-close',
        'lucide:panel-left-open',
        'lucide:search',
        'lucide:square',
        'lucide:sun',
        'lucide:triangle-alert',
        'lucide:x'
      ]
    }
  },

  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080'
    }
  },

  vite: {
    optimizeDeps: {
      include: [
        'zod',
        '@vue/devtools-core',
        '@vue/devtools-kit'
      ]
    }
  },

  colorMode: {
    preference: 'dark',
    fallback: 'dark'
  },

  compatibilityDate: '2024-07-11',

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
