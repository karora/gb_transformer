# Start from scratch to create an absolutely minimal-sized image
# FROM scratch AS aifeeder

# For testing...
FROM linuxcontainers/debian-slim AS xformer

# We make https requests so we need the ca-certificates.crt file
COPY ca-certificates.crt /etc/ssl/certs/

# The actual binary is copied into the root
COPY xformer /

# All configuration should be in environment variables
CMD ["/xformer"]