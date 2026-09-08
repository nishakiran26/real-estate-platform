# Build stage: Vite needs the API URL at build time, so it arrives as a build arg.
FROM node:20-alpine AS build

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

ARG VITE_API_BASE_URL=http://localhost:8080/api/v1
ARG VITE_APP_NAME=RERA Property Platform
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_APP_NAME=$VITE_APP_NAME

RUN npm run build

# Runtime stage: nginx serves the built SPA.
FROM nginx:1.27-alpine

COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --spider -q http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
