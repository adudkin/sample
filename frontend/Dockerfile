# syntax=docker/dockerfile:1
FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
# Optional certificate bundle for trusted HTTPS proxies used by cloud builders.
RUN --mount=type=secret,id=cloud_ca,target=/tmp/cloud-ca.pem \
    if [ -f /tmp/cloud-ca.pem ]; then export NODE_EXTRA_CA_CERTS=/tmp/cloud-ca.pem; fi; \
    npm ci
COPY . .
RUN npm run build

FROM nginx:1.28-alpine
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx/https.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80 443
