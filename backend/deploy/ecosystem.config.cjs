const path = require('path')

const root = __dirname

module.exports = {
  apps: [
    {
      name: 'c7d5a6l',
      cwd: root,
      script: path.join(root, 'run.sh'),
      interpreter: '/bin/bash',
      instances: 1,
      exec_mode: 'fork',
      autorestart: true,
      max_restarts: 10,
      min_uptime: '5s',
      max_memory_restart: '512M',
      out_file: path.join(root, 'logs/out.log'),
      error_file: path.join(root, 'logs/error.log'),
      merge_logs: true,
    },
  ],
}
