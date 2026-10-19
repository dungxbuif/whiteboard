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
COPY src/mcp mcp
RUN npm run build --workspace ./packages/collab-contracts && npm run build --workspace ./packages/diagram-kit && npm run build --workspace ./mcp
RUN npm prune --omit=dev --ignore-scripts --no-audit

FROM mcr.microsoft.com/playwright:v1.60.0-noble
ENV NODE_ENV=production PLAYWRIGHT_BROWSERS_PATH=/ms-playwright MCP_BIND_HOST=0.0.0.0 PORT=3002
WORKDIR /app
RUN install -d -o pwuser -g pwuser -m 0700 /var/lib/whiteboard-mcp
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/packages/collab-contracts ./packages/collab-contracts
COPY --from=build /app/packages/diagram-kit ./packages/diagram-kit
COPY --from=build /app/mcp/package.json ./mcp/package.json
COPY --from=build /app/mcp/dist ./mcp/dist
COPY --from=build /app/mcp/render-dist ./mcp/render-dist
USER pwuser
EXPOSE 3002
CMD ["node", "mcp/dist/http.js"]
