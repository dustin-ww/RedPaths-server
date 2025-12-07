# Go Base Image (Debian-basiert, stabil)
FROM golang:1.25.1-bookworm

# Installiere wichtige Tools
RUN apt-get update && apt-get install -y \
    git \
    make \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Arbeitsverzeichnis setzen
WORKDIR /workspace

# Standard-User (für Devcontainer-Setup)
RUN useradd -m redpaths && \
    chown -R redpaths:redpaths /workspace
USER redpaths
