## Usage

```bash
$ npm install
$ npm run dev          # http://localhost:3000
```

### Local HTTPS (`c7d5a6l.lo`) for Telegram Login Widget

Telegram’s iframe only allows **HTTPS on port 443**. No Apache vhosts needed — Vite serves TLS directly.

1. Install [mkcert](https://github.com/FiloSottile/mkcert) and run `mkcert -install`
2. Map the host: `echo '127.0.0.1  c7d5a6l.lo' | sudo tee -a /etc/hosts`
3. From `frontend/`: `npm run setup:https`, then once `npm run allow:https`, then `npm run dev:https`
4. BotFather → `/setdomain` → `c7d5a6l.lo`
5. Open **https://c7d5a6l.lo/**

## Available Scripts

### `npm run dev`

HTTP on [http://localhost:3000](http://localhost:3000).

### `npm run dev:https`

HTTPS on [https://c7d5a6l.lo/](https://c7d5a6l.lo/) (port 443; needs certs from `setup:https`).

### `npm run build`

Production build to `dist/`.

## Deployment

See [Vite static deploy](https://vite.dev/guide/static-deploy.html).
