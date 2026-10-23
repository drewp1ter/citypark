FROM node:24-alpine AS build
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL
ARG VITE_SITE_URL
ENV VITE_API_URL=$VITE_API_URL VITE_SITE_URL=$VITE_SITE_URL
RUN npm run build

FROM nginx:stable-alpine
# The official image runs envsubst over /etc/nginx/templates/*.template at start and writes the result to
# /etc/nginx/conf.d/ (so API_ORIGIN and SITE_ORIGIN are runtime values).
COPY nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY nginx/snippets/security-headers.conf.template /etc/nginx/templates/snippets/security-headers.conf.template
COPY --from=build /src/dist /usr/share/nginx/html
EXPOSE 80
