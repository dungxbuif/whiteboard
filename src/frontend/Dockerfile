FROM node:24.19.0-bookworm-slim AS build
WORKDIR /app
COPY src/package.json src/package-lock.json ./
COPY src/frontend/package.json frontend/package.json
COPY src/collab/package.json collab/package.json
COPY src/mcp/package.json mcp/package.json
COPY src/packages/collab-contracts/package.json packages/collab-contracts/package.json
COPY src/packages/diagram-kit/package.json packages/diagram-kit/package.json
RUN npm ci --ignore-scripts --no-audit
COPY src/packages packages
COPY src/frontend frontend
RUN npm run build --workspace ./packages/collab-contracts && npm run build --workspace ./frontend

FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY src/frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/frontend/dist /usr/share/nginx/html
USER 101
EXPOSE 80
