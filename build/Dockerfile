# Stage 1: Build the Vite app
FROM node:20-alpine as build
WORKDIR /app

COPY . ./
RUN --mount=type=cache,target=/root/.npm npm install
RUN npm run build

# Stage 2: Serve with Nginx
FROM nginx:alpine

COPY --from=build /app/dist /usr/share/nginx/html
COPY --from=build /app/nginx.conf /etc/nginx/conf.d/default.conf