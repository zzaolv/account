# Dockerfile
# This file builds a single, final image containing both the Go backend and React frontend.

# --- STAGE 1: Build the Frontend Application ---
FROM node:20-alpine AS builder-frontend
WORKDIR /app
COPY simple-ledger-frontend/package*.json ./
RUN npm install
COPY simple-ledger-frontend/ ./
RUN npm run build

# --- STAGE 2: Build the Backend Application ---
FROM golang:1.22-alpine AS builder-backend
RUN apk add --no-cache build-base
WORKDIR /app
COPY bookkeeper-app/go.mod bookkeeper-app/go.sum ./
RUN go mod download
COPY bookkeeper-app/ ./
RUN CGO_ENABLED=1 go build -o /app/server -ldflags "-w -s" ./...

# --- STAGE 3: Build the Final Production Image ---
FROM alpine:latest

# Install Nginx, Supervisor, and SQLite dependencies
RUN apk add --no-cache nginx supervisor libc6-compat

# --- Nginx Setup ---
RUN rm -f /etc/nginx/http.d/default.conf
COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=builder-frontend /app/dist /usr/share/nginx/html

# --- Backend Setup ---
WORKDIR /app
COPY --from=builder-backend /app/server .

# --- Supervisor Setup ---
COPY supervisord.conf /etc/supervisord.conf

# --- Data & Permissions ---
RUN mkdir -p /app/data && \
    chown -R nginx:nginx /app/data && \
    chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/lib/nginx && \
    chown -R nginx:nginx /var/log/nginx
USER nginx

# Expose the port Nginx will listen on
EXPOSE 8080

# Use Supervisor to start and manage both Nginx and the Go backend
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]