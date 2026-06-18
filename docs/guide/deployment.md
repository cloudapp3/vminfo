---
title: Deployment
description: Deploy the docs site to Cloudflare Pages with the correct build output and custom domain settings.
---

# Deployment

This docs site is designed to deploy cleanly to Cloudflare Pages.

## Cloudflare Pages settings

| Setting | Value |
| --- | --- |
| Framework preset | VitePress or None |
| Build command | `pnpm docs:build` |
| Build output directory | `docs/.vitepress/dist` |
| Root directory | `/` |
| Node.js version | 20 or newer |

If your Pages project needs to install dependencies during build, use:

```bash
corepack enable && pnpm install --frozen-lockfile && pnpm docs:build
```

## Custom domain

- Use a docs subdomain such as `vminfo.bestcheapvps.org`
- Add the custom domain inside the Cloudflare Pages project first
- Then configure DNS as Cloudflare instructs
- Do not create only a manual CNAME without attaching the Pages project

## Base path

- For a custom domain at the site root, do not set `base`
- For a subpath deployment such as `https://example.com/vminfo/`, set `base: "/vminfo/"`

## Output

Cloudflare Pages should publish the static site from `docs/.vitepress/dist`.
