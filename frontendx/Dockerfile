FROM node:22-alpine AS build
WORKDIR /site
COPY . .
RUN npm ci && npm run typecheck && npm run build

FROM node:22-alpine
WORKDIR /site
ENV NODE_ENV=production
COPY --from=build /site/apps/visitor/dist ./apps/visitor/dist
COPY --from=build /site/apps/admin/dist ./apps/admin/dist
COPY tooling/backend-site.mjs ./tooling/backend-site.mjs
USER node
CMD ["node", "tooling/backend-site.mjs"]
