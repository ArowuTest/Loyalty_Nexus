# Loyalty Nexus — Render Service (Remotion Video Templates)

Self-hosted [Remotion](https://remotion.dev) renderer that turns a composition + props
into an MP4. The Go API (`dispatchRender` → `callRemotionRender`) calls this over HTTP;
this service renders with headless Chromium and uploads the result to R2/S3.

**No AWS. Host-portable.** It's a plain Node + Chromium container: runs on **Render** now,
and the *same image* runs on **GCP Cloud Run** later — only env vars change, no code. It
never imports `@remotion/lambda` or `@remotion/cloudrun` (self-hosted `renderMedia()` only).

## API

- `GET /health` → `{ status: "ok" }`
- `POST /render` (Bearer `RENDER_SERVICE_TOKEN`) → body `{ composition, props }` → `{ url }`

## Env vars

| Var | Purpose |
|---|---|
| `PORT` | Listen port (Render injects; default 8090) |
| `RENDER_SERVICE_TOKEN` | Shared secret — must match the API service's value |
| `AWS_S3_BUCKET`, `AWS_S3_ENDPOINT`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION` | R2/S3 for MP4 upload (same bucket the Go backend uses) |
| `STORAGE_CDN_BASE_URL` / `STORAGE_PUBLIC_URL` | Public base URL for the returned MP4 |

## Go-live steps (owner)

1. **Deploy the service** — it's defined in `render.yaml` as `loyalty-nexus-render`
   (Docker, `plan: standard` — Chromium is RAM-heavy). Sync the blueprint or create the
   service manually; set the env vars above.
2. **Set a shared token** — pick a strong `RENDER_SERVICE_TOKEN` and set it on **both**
   `loyalty-nexus-render` and `loyalty-nexus-api`.
3. **Point the API at it** — set `RENDER_SERVICE_URL` on `loyalty-nexus-api` to this
   service's URL (Render internal or public).
4. **Activate the tool** — flip the `video-slideshow` row in `studio_tools` to
   `is_active=true` (admin), and optionally the `remotion-selfhosted` provider row.
   It's shipped inactive so it stays hidden until this service is live.

## Later: move to GCP

Build the same image and deploy it as a container on **GCP Cloud Run** (reuse the GCS
creds already in the project). Then just update `RENDER_SERVICE_URL` on the API service.
No code change.

## Local dev

```bash
npm install
RENDER_SERVICE_TOKEN=dev npm start
# POST http://localhost:8090/render  { "composition": "video-slideshow", "props": { "images": ["https://…1.jpg","https://…2.jpg","https://…3.jpg"], "aspectRatio": "9:16", "caption": "Hello" } }
```

## Compositions

- `remotion/index.ts` — registers the root.
- `remotion/Root.tsx` — `<Composition>` registry; `calculateMetadata` sets dimensions +
  duration from props.
- `remotion/VideoSlideshow.tsx` — the `video-slideshow` composition (Ken-Burns image
  montage + crossfades + caption + optional music).

Add new templates by dropping another composition into `remotion/`, registering it in
`Root.tsx`, and adding a matching `studio_tools` row + frontend form.
