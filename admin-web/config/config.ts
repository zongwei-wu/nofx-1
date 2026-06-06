import { defineConfig } from '@umijs/max';
import defaultSettings from './defaultSettings';
import routes from './routes';

export default defineConfig({
  base: '/admin/',
  publicPath: '/admin/',
  hash: true,
  routes,
  access: {},
  model: {},
  initialState: {},
  request: {},
  layout: {
    locale: false,
    ...defaultSettings,
  },
  proxy: {
    '/api': {
      target: 'http://localhost:8080',
      changeOrigin: true,
    },
  },
  npmClient: 'npm',
  mock: false,
});
