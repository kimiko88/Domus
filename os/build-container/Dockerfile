# ─── Domus OS - Hermetic Buildroot Build Container ───────────────────────────
# Provides a completely reproducible, isolated environment to cross-compile
# Domus OS for Raspberry Pi 4 ARM64 from any host OS (Linux, macOS, Windows).
FROM ubuntu:22.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Etc/UTC

# Install all mandatory Buildroot host dependencies and image assembly tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash \
    bc \
    binutils \
    bison \
    build-essential \
    bzip2 \
    ca-certificates \
    cpio \
    curl \
    diffutils \
    dosfstools \
    file \
    flex \
    g++ \
    gawk \
    gcc \
    genimage \
    git \
    gzip \
    libncurses-dev \
    libssl-dev \
    locales \
    make \
    mtools \
    patch \
    perl \
    python3 \
    python3-dev \
    python3-pip \
    rsync \
    sed \
    squashfs-tools \
    tar \
    u-boot-tools \
    unzip \
    wget \
    xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Set up UTF-8 locale
RUN locale-gen en_US.UTF-8
ENV LANG=en_US.UTF-8
ENV LANGUAGE=en_US:en
ENV LC_ALL=en_US.UTF-8

# Create non-root build user (Buildroot refuses to build as root for safety)
ARG USER_ID=1000
ARG GROUP_ID=1000
RUN groupadd -g ${GROUP_ID} builder && \
    useradd -l -u ${USER_ID} -g builder -m builder

# Set workspace directory
WORKDIR /workspace
USER builder

# Default entrypoint
CMD ["/bin/bash"]
