# Loyalty Nexus render-service — self-hosted Remotion renderer.
# Portable container: runs on Render now, GCP Cloud Run later, unchanged.
FROM node:22-bookworm-slim

# Headless-Chromium system libraries required by Remotion's renderer.
RUN apt-get update && apt-get install -y --no-install-recommends \
    libnss3 libdbus-1-3 libatk1.0-0 libgbm-dev libasound2 libxrandr2 \
    libxkbcommon-dev libxfixes3 libxcomposite1 libxdamage1 libatk-bridge2.0-0 \
    libpango-1.0-0 libcairo2 libcups2 fonts-liberation ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install production deps first (better layer caching).
COPY package.json ./
RUN npm install --omit=dev

# App source (server + Remotion compositions).
COPY . .

ENV NODE_ENV=production
# Render injects PORT; default to 8090 locally.
ENV PORT=8090
EXPOSE 8090

CMD ["node", "server.mjs"]
