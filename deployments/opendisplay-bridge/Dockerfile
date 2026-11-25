FROM python:3.13-slim

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

RUN apt-get update && apt-get install -y --no-install-recommends \
      dbus \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt/opendisplay-bridge
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY server.py .
RUN useradd --system --uid 10001 --home-dir /nonexistent opendisplay \
    && chown -R opendisplay:opendisplay /opt/opendisplay-bridge

USER opendisplay
EXPOSE 10320
CMD ["python", "/opt/opendisplay-bridge/server.py"]
