module.exports = {
  apps: [
    {
      name: 'nofx-backend',
      cwd: '/home/admin/nofx-1',
      script: './nofx',
      env: {
        NODE_ENV: 'production',
      },
      autorestart: true,
      max_restarts: 10,
      restart_delay: 3000,
      log_date_format: 'YYYY-MM-DD HH:mm:ss',
      error_file: '/tmp/nofx-pm2-error.log',
      out_file: '/tmp/nofx-pm2-out.log',
      merge_logs: true,
    },
    {
      name: 'nofx-frontend',
      cwd: '/home/admin/nofx-1/web',
      script: './node_modules/.bin/vite',
      args: '--host 0.0.0.0 --port 3000',
      env: {
        NODE_ENV: 'development',
      },
      autorestart: true,
      max_restarts: 10,
      restart_delay: 2000,
      log_date_format: 'YYYY-MM-DD HH:mm:ss',
      error_file: '/tmp/vite-pm2-error.log',
      out_file: '/tmp/vite-pm2-out.log',
      merge_logs: true,
    }
  ]
};
