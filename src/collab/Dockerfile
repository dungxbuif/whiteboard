FROM node:24.19.0-bookworm-slim AS build
WORKDIR /app
COPY src/package.json src/package-lock.json ./
COPY src/frontend/package.json frontend/package.json
COPY src/collab/package.json collab/package.json
COPY src/mcp/package.json mcp/package.json
COPY src/packages/collab-contracts/package.json packages/collab-contracts/package.json
COPY src/packages/diagram-kit/package.json packages/diagram-kit/package.json
RUN npm ci --ignore-scripts --no-audit
COPY src/packages/collab-contracts packages/collab-contracts
COPY src/collab collab
RUN npm run build --workspace ./collab
RUN npm prune --omit=dev --ignore-scripts --no-audit

FROM node:24.19.0-bookworm-slim
ENV NODE_ENV=production
WORKDIR /app
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/packages/collab-contracts ./packages/collab-contracts
COPY --from=build /app/collab/package.json ./collab/package.json
COPY --from=build /app/collab/dist ./collab/dist
USER node
EXPOSE 8082 8083
CMD ["node", "collab/dist/server.js"]
