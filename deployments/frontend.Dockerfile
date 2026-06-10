FROM node:22-alpine AS builder

WORKDIR /src/web/frontend
COPY web/frontend/package*.json ./
RUN npm ci
COPY web/frontend ./
ARG VITE_API_BASE_URL=/api
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN npm run build

FROM nginx:1.27-alpine

COPY deployments/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/web/frontend/dist /usr/share/nginx/html

EXPOSE 80
