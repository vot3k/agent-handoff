# Use Chainguard Redis image
FROM cgr.dev/chainguard/redis:latest

# Default working directory is /data writable by redis user
# Start redis server in the foreground (not daemonized)
CMD ["redis-server", "--daemonize", "no"]

