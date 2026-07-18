// Loyalty Nexus — Remotion render service.
//
// A thin HTTP service that wraps Remotion's renderMedia(). The Go backend
// (dispatchRender → callRemotionRender) POSTs {composition, props} here; we
// render an MP4 with headless Chromium, upload it to R2/S3, and return {url}.
//
// HOST-PORTABLE BY DESIGN: this is a plain Node + Chromium container. It runs
// on Render today; the SAME container runs on GCP Cloud Run later with no code
// change — only env vars differ. It never imports @remotion/lambda or
// @remotion/cloudrun (self-hosted renderMedia only).
//
// Auth: protected by a shared secret (RENDER_SERVICE_TOKEN) that the Go backend
// sends as `Authorization: Bearer <token>`. This service is internal-only.

import express from 'express';
import { bundle } from '@remotion/bundler';
import { renderMedia, selectComposition, ensureBrowser } from '@remotion/renderer';
import { S3Client, PutObjectCommand } from '@aws-sdk/client-s3';
import { readFile, unlink } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { randomUUID } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { dirname } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));

const PORT = process.env.PORT || 8090;
const SHARED_TOKEN = process.env.RENDER_SERVICE_TOKEN || '';

// ── R2 / S3 storage (same bucket the Go backend already uses) ─────────────────
const s3 = new S3Client({
  region: process.env.AWS_REGION || 'auto',
  endpoint: process.env.AWS_S3_ENDPOINT || undefined, // R2 endpoint
  credentials: {
    accessKeyId: process.env.AWS_ACCESS_KEY_ID || '',
    secretAccessKey: process.env.AWS_SECRET_ACCESS_KEY || '',
  },
});
const BUCKET = process.env.AWS_S3_BUCKET || '';
const PUBLIC_BASE = (process.env.STORAGE_CDN_BASE_URL || process.env.STORAGE_PUBLIC_URL || '').replace(/\/$/, '');

// ── Bundle the Remotion project ONCE at startup (expensive; reused per render) ──
let bundlePromise = null;
async function getBundle() {
  if (!bundlePromise) {
    await ensureBrowser(); // download/verify headless Chromium
    bundlePromise = bundle({
      entryPoint: join(__dirname, 'remotion', 'index.ts'),
      // webpackOverride can be added here if templates need extra loaders
    });
  }
  return bundlePromise;
}

async function uploadMp4(localPath, key) {
  const body = await readFile(localPath);
  await s3.send(new PutObjectCommand({
    Bucket: BUCKET,
    Key: key,
    Body: body,
    ContentType: 'video/mp4',
  }));
  await unlink(localPath).catch(() => {});
  return PUBLIC_BASE ? `${PUBLIC_BASE}/${key}` : key;
}

const app = express();
app.use(express.json({ limit: '2mb' }));

app.get('/health', (_req, res) => res.json({ status: 'ok', service: 'render-service' }));

// POST /render { composition, props } → { url }
app.post('/render', async (req, res) => {
  // Shared-secret auth (internal service).
  if (SHARED_TOKEN) {
    const auth = req.headers['authorization'] || '';
    if (auth !== `Bearer ${SHARED_TOKEN}`) {
      return res.status(401).json({ error: 'unauthorized' });
    }
  }
  const { composition, props } = req.body || {};
  if (!composition) return res.status(400).json({ error: 'composition required' });

  const outPath = join(tmpdir(), `${composition}-${randomUUID()}.mp4`);
  try {
    const serveUrl = await getBundle();
    const comp = await selectComposition({
      serveUrl,
      id: composition,
      inputProps: props || {},
    });
    await renderMedia({
      serveUrl,
      composition: comp,
      codec: 'h264',
      inputProps: props || {},
      outputLocation: outPath,
      // Keep concurrency modest — this box renders one job at a time.
      concurrency: 1,
    });
    const key = `studio/video-templates/${composition}_${Date.now()}.mp4`;
    const url = await uploadMp4(outPath, key);
    return res.json({ url });
  } catch (err) {
    console.error('[render] failed:', err);
    await unlink(outPath).catch(() => {});
    return res.status(500).json({ error: String(err && err.message ? err.message : err) });
  }
});

app.listen(PORT, () => {
  console.log(`[render-service] listening on :${PORT}`);
  // Warm the bundle + browser in the background so the first render isn't slow.
  getBundle().then(() => console.log('[render-service] Remotion bundle ready')).catch((e) => console.error('[render-service] bundle warm failed:', e));
});
