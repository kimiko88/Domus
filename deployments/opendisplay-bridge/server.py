"""Small authenticated local HTTP companion for the upstream py-opendisplay SDK.

Only this process talks to BlueZ / BLE. Domus calls the narrow HTTP surface below;
it never receives direct Bluetooth privileges or arbitrary command execution.
"""
from __future__ import annotations

import asyncio
import base64
import json
import hmac
import logging
import os
import re
import warnings
from contextlib import asynccontextmanager
from urllib.parse import quote
from dataclasses import asdict
from io import BytesIO
from typing import Any

from aiohttp import web
from PIL import Image, UnidentifiedImageError
from odl_renderer import generate_image
from opendisplay import (
    BuzzerActivateConfig,
    DitherMode,
    FitMode,
    LedFlashConfig,
    OpenDisplayDevice,
    RefreshMode,
    Rotation,
    discover_devices_with_adv,
)
from opendisplay.discovery_ip import discover_ip_devices

_LOG = logging.getLogger("opendisplay.bridge")
_MAC_RE = re.compile(r"^(?:[0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$")
_MAX_IMAGE_BYTES = 8 * 1024 * 1024
_MAX_BASE64_CHARS = ((_MAX_IMAGE_BYTES + 2) // 3) * 4
Image.MAX_IMAGE_PIXELS = 20_000_000
_ALLOWED_FITS = {"contain", "cover", "stretch", "crop"}
_ALLOWED_REFRESH = {"full", "fast"}


def _required_token() -> str:
    token = os.environ.get("OPENDISPLAY_BRIDGE_TOKEN", "").strip()
    if len(token) < 32:
        raise RuntimeError("OPENDISPLAY_BRIDGE_TOKEN must contain at least 32 characters")
    if "\r" in token or "\n" in token:
        raise RuntimeError("OPENDISPLAY_BRIDGE_TOKEN must not contain newlines")
    return token


def _authorised(request: web.Request) -> bool:
    expected = request.app["token"]
    supplied = request.headers.get("Authorization", "")
    if not supplied.startswith("Bearer "):
        return False
    token = supplied[7:]
    return hmac.compare_digest(token.encode("utf-8"), expected.encode("utf-8"))


@web.middleware
async def auth_middleware(request: web.Request, handler):
    if request.path == "/health":
        return await handler(request)
    if not _authorised(request):
        raise web.HTTPUnauthorized(text="missing or invalid bridge token")
    return await handler(request)


def _serialise_advertisement(adv: Any) -> dict[str, Any] | None:
    if adv is None:
        return None
    payload = asdict(adv)
    payload.pop("dynamic_data", None)
    payload.pop("raw_data", None)
    return payload


async def list_devices(request: web.Request) -> web.Response:
    timeout = float(request.app["ble_scan_seconds"])
    found: dict[str, dict[str, Any]] = {}
    try:
        ble_devices = await discover_devices_with_adv(timeout=timeout)
        for name, (address, advertisement) in ble_devices.items():
            mac = address.upper()
            found[mac] = {
                "name": name,
                "address": mac,
                "transport": "ble",
                "online": True,
                "advertisement": _serialise_advertisement(advertisement),
            }
    except Exception as exc:  # discovery errors should not prevent a LAN scan
        _LOG.warning("BLE discovery failed: %s", exc)

    if request.app["enable_wifi"]:
        try:
            lan_devices = await discover_ip_devices(scan_seconds=min(timeout, 5.0))
            for name, info in lan_devices.items():
                mac = (info.mac or "").upper()
                key = mac if _MAC_RE.fullmatch(mac) else f"lan:{info.host}:{info.port}"
                previous = found.get(key, {})
                found[key] = {
                    **previous,
                    "name": name,
                    "address": key,
                    "mac": mac or None,
                    "transport": "wifi",
                    "host": info.host,
                    "port": info.port,
                    "tls": info.tls,
                    "firmware": info.fw,
                    "online": True,
                }
        except Exception as exc:
            _LOG.warning("OpenDisplay mDNS discovery failed: %s", exc)

    devices = sorted(found.values(), key=lambda item: item["name"].lower())
    return web.json_response({"devices": devices})


def _decode_image(value: Any) -> Image.Image:
    if not isinstance(value, str) or len(value) > _MAX_BASE64_CHARS:
        raise web.HTTPRequestEntityTooLarge(max_size=_MAX_IMAGE_BYTES, actual_size=0)
    try:
        raw = base64.b64decode(value, validate=True)
    except (ValueError, TypeError) as exc:
        raise web.HTTPBadRequest(text="image_base64 must be valid Base64") from exc
    if not raw or len(raw) > _MAX_IMAGE_BYTES:
        raise web.HTTPRequestEntityTooLarge(max_size=_MAX_IMAGE_BYTES, actual_size=len(raw))
    try:
        with warnings.catch_warnings():
            warnings.simplefilter("error", Image.DecompressionBombWarning)
            with Image.open(BytesIO(raw)) as source:
                if source.format not in {"PNG", "JPEG", "WEBP", "BMP"}:
                    raise web.HTTPUnsupportedMediaType(text="only PNG, JPEG, WEBP and BMP images are supported")
                source.verify()
            with Image.open(BytesIO(raw)) as source:
                source.load()
                return source.convert("RGB")
    except (UnidentifiedImageError, OSError, Image.DecompressionBombError, Image.DecompressionBombWarning) as exc:
        raise web.HTTPBadRequest(text="invalid or unsafe image") from exc



_MAX_ACTION_BODY_BYTES = 64 * 1024
_MAX_ODL_ELEMENTS = 128
_ALLOWED_ODL_ELEMENTS = {
    "text", "multiline", "rectangle", "rectangle_pattern", "circle", "ellipse",
    "polygon", "arc", "line", "icon", "icon_sequence", "qrcode", "progress_bar",
    "diagram", "debug_grid",
}
_HA_TAG_PREFIX = "https://www.home-assistant.io/tag/"


def _load_device_keys() -> dict[str, bytes]:
    """Load per-device 16-byte BLE keys without ever exposing them in API output."""
    raw = os.environ.get("OPENDISPLAY_BRIDGE_DEVICE_KEYS_JSON", "{}").strip() or "{}"
    if len(raw) > 64 * 1024:
        raise RuntimeError("OPENDISPLAY_BRIDGE_DEVICE_KEYS_JSON exceeds 64 KiB")
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise RuntimeError("OPENDISPLAY_BRIDGE_DEVICE_KEYS_JSON must be a JSON object") from exc
    if not isinstance(parsed, dict) or len(parsed) > 256:
        raise RuntimeError("OpenDisplay device keys must be a JSON object with at most 256 MAC entries")
    result: dict[str, bytes] = {}
    for mac, key in parsed.items():
        if not isinstance(mac, str) or not _MAC_RE.fullmatch(mac):
            raise RuntimeError("OpenDisplay device-key map contains an invalid MAC address")
        if not isinstance(key, str) or not re.fullmatch(r"[0-9A-Fa-f]{32}", key):
            raise RuntimeError(f"OpenDisplay key for {mac.upper()} must be exactly 32 hexadecimal characters")
        result[mac.upper()] = bytes.fromhex(key)
    return result


def _json_depth(value: Any, depth: int = 0) -> int:
    if depth > 8:
        raise web.HTTPBadRequest(text="JSON drawing payload is nested too deeply")
    if isinstance(value, dict):
        for child in value.values():
            _json_depth(child, depth + 1)
    elif isinstance(value, list):
        if len(value) > 512:
            raise web.HTTPBadRequest(text="JSON drawing arrays may contain at most 512 values")
        for child in value:
            _json_depth(child, depth + 1)
    return depth


async def _read_action_payload(request: web.Request) -> dict[str, Any]:
    if request.content_length is not None and request.content_length > _MAX_ACTION_BODY_BYTES:
        raise web.HTTPRequestEntityTooLarge(max_size=_MAX_ACTION_BODY_BYTES, actual_size=request.content_length)
    try:
        payload = await request.json()
    except Exception as exc:
        raise web.HTTPBadRequest(text="invalid JSON body") from exc
    if not isinstance(payload, dict):
        raise web.HTTPBadRequest(text="JSON body must be an object")
    if len(json.dumps(payload, separators=(",", ":")).encode("utf-8")) > _MAX_ACTION_BODY_BYTES:
        raise web.HTTPRequestEntityTooLarge(max_size=_MAX_ACTION_BODY_BYTES, actual_size=_MAX_ACTION_BODY_BYTES + 1)
    _json_depth(payload)
    return payload


def _allowed_fields(payload: dict[str, Any], allowed: set[str]) -> None:
    unknown = set(payload) - allowed
    if unknown:
        raise web.HTTPBadRequest(text=f"unsupported field(s): {', '.join(sorted(unknown))}")


def _integer(payload: dict[str, Any], name: str, default: int, minimum: int, maximum: int) -> int:
    value = payload.get(name, default)
    if isinstance(value, bool) or not isinstance(value, int) or not minimum <= value <= maximum:
        raise web.HTTPBadRequest(text=f"{name} must be an integer between {minimum} and {maximum}")
    return value


def _text(payload: dict[str, Any], name: str, default: str = "", maximum: int = 1024) -> str:
    value = payload.get(name, default)
    if not isinstance(value, str) or len(value) > maximum or (not value and not default):
        raise web.HTTPBadRequest(text=f"{name} must be a non-empty string up to {maximum} characters")
    return value


def _rgb_to_led_color(value: Any) -> int:
    if (not isinstance(value, list) or len(value) != 3 or
            any(isinstance(v, bool) or not isinstance(v, int) or not 0 <= v <= 255 for v in value)):
        raise web.HTTPBadRequest(text="color must be an RGB array with three integers in the range 0..255")
    red, green, blue = value
    return ((round(red * 7 / 255) << 5) | (round(green * 7 / 255) << 2) | round(blue * 3 / 255))


@asynccontextmanager
async def _open_device(request: web.Request, address: str, payload: dict[str, Any]):
    transport = payload.get("transport", "ble")
    mac = str(payload.get("mac", address)).upper()
    key = request.app["device_keys"].get(mac)
    if transport == "ble":
        if not _MAC_RE.fullmatch(address):
            raise web.HTTPBadRequest(text="BLE address must be a MAC address")
        async with OpenDisplayDevice(mac_address=address.upper(), encryption_key=key) as device:
            yield device
        return
    if transport != "wifi":
        raise web.HTTPBadRequest(text="transport must be ble or wifi")
    host = payload.get("host")
    port = payload.get("port")
    tls = payload.get("tls", False)
    if not isinstance(host, str) or not host or len(host) > 253:
        raise web.HTTPBadRequest(text="Wi-Fi host is required")
    if isinstance(port, bool) or not isinstance(port, int) or not 1 <= port <= 65535:
        raise web.HTTPBadRequest(text="Wi-Fi port is invalid")
    if not isinstance(tls, bool):
        raise web.HTTPBadRequest(text="Wi-Fi TLS flag is invalid")
    if tls:
        raise web.HTTPNotImplemented(text="Wi-Fi TLS-PSK is not enabled; configure BLE or non-TLS LAN transport")
    async with OpenDisplayDevice(host=host, port=port, tls=False, encryption_key=key) as device:
        yield device


def _validate_draw_elements(elements: Any) -> list[dict[str, Any]]:
    if not isinstance(elements, list) or not 1 <= len(elements) <= _MAX_ODL_ELEMENTS:
        raise web.HTTPBadRequest(text=f"elements must be a list containing 1..{_MAX_ODL_ELEMENTS} entries")
    checked: list[dict[str, Any]] = []
    for index, item in enumerate(elements):
        if not isinstance(item, dict):
            raise web.HTTPBadRequest(text=f"element {index} must be an object")
        kind = item.get("type")
        if not isinstance(kind, str) or kind not in _ALLOWED_ODL_ELEMENTS:
            raise web.HTTPBadRequest(text=f"element {index} has an unsupported type")
        # dlimg and plot are intentionally excluded: they can fetch external resources
        # or require a host-provided data provider. Callers can upload a prepared image instead.
        checked.append(item)
    return checked


async def device_action(request: web.Request) -> web.Response:
    action = request.match_info["action"]
    if action not in {"led", "buzzer", "melody", "nfc", "drawcustom"}:
        raise web.HTTPNotFound(text="unknown OpenDisplay action")
    payload = await _read_action_payload(request)
    address = request.match_info["address"]
    transport_fields = {"transport", "host", "port", "tls", "mac"}

    try:
        async with _open_device(request, address, payload) as device:
            if action == "led":
                _allowed_fields(payload, transport_fields | {"instance", "brightness", "color", "flash_count", "loop_delay_ms", "inter_delay_ms", "repeats"})
                instance = _integer(payload, "instance", 0, 0, 255)
                brightness = _integer(payload, "brightness", 8, 1, 16)
                color = _rgb_to_led_color(payload.get("color", [255, 0, 0]))
                flash_count = _integer(payload, "flash_count", 1, 0, 15)
                loop_delay = round(_integer(payload, "loop_delay_ms", 0, 0, 1500) / 100)
                inter_delay = round(_integer(payload, "inter_delay_ms", 0, 0, 25500) / 100)
                repeats = _integer(payload, "repeats", 1, 0, 255)
                config = LedFlashConfig.single(
                    color=color, flash_count=flash_count, loop_delay_units=loop_delay,
                    inter_delay_units=inter_delay, brightness=brightness,
                    group_repeats=None if repeats == 0 else repeats,
                )
                await device.activate_led(led_instance=instance, flash_config=config)
            elif action == "buzzer":
                _allowed_fields(payload, transport_fields | {"instance", "frequency_hz", "duration_ms", "repeats"})
                config = BuzzerActivateConfig.single_tone(
                    frequency_hz=_integer(payload, "frequency_hz", 1000, 0, 12000),
                    duration_ms=_integer(payload, "duration_ms", 100, 5, 1275),
                    repeats=_integer(payload, "repeats", 1, 1, 255),
                )
                await device.activate_buzzer(
                    buzzer_instance=_integer(payload, "instance", 0, 0, 3), config=config
                )
            elif action == "melody":
                _allowed_fields(payload, transport_fields | {"instance", "notes", "tempo", "repeats", "default_note_ms", "default_length"})
                notes = _text(payload, "notes", maximum=2048)
                default_length = payload.get("default_length")
                if default_length is not None and (isinstance(default_length, bool) or default_length not in {1, 2, 4, 8, 16, 32}):
                    raise web.HTTPBadRequest(text="default_length must be one of 1, 2, 4, 8, 16 or 32")
                try:
                    config = BuzzerActivateConfig.melody(
                        notes, tempo=_integer(payload, "tempo", 120, 40, 400),
                        repeats=_integer(payload, "repeats", 1, 1, 255),
                        default_ms=_integer(payload, "default_note_ms", 200, 5, 1275),
                        default_length=default_length,
                    )
                except ValueError as exc:
                    raise web.HTTPBadRequest(text="invalid melody notation") from exc
                await device.activate_buzzer(
                    buzzer_instance=_integer(payload, "instance", 0, 0, 3), config=config
                )
            elif action == "nfc":
                _allowed_fields(payload, transport_fields | {"record_type", "content", "mime_type"})
                record_type = _text(payload, "record_type", "url", 16)
                content = _text(payload, "content", maximum=512)
                mime_type = payload.get("mime_type", "text/vcard")
                if record_type == "ha_tag":
                    content = _HA_TAG_PREFIX + quote(content, safe="")
                    record_type = "url"
                encoded = content.encode("utf-8")
                if record_type == "mime":
                    if not isinstance(mime_type, str) or not 1 <= len(mime_type.encode("utf-8")) <= 255:
                        raise web.HTTPBadRequest(text="mime_type must encode to 1..255 UTF-8 bytes")
                    if len(mime_type.encode("utf-8")) + 1 + len(encoded) > 512:
                        raise web.HTTPRequestEntityTooLarge(max_size=512, actual_size=len(mime_type.encode("utf-8")) + 1 + len(encoded))
                    await device.write_nfc_mime(mime_type, content)
                elif record_type == "url":
                    if len(encoded) > 512:
                        raise web.HTTPRequestEntityTooLarge(max_size=512, actual_size=len(encoded))
                    await device.write_nfc_url(content)
                elif record_type == "text":
                    if len(encoded) > 512:
                        raise web.HTTPRequestEntityTooLarge(max_size=512, actual_size=len(encoded))
                    await device.write_nfc_text(content)
                else:
                    raise web.HTTPBadRequest(text="record_type must be url, text, mime or ha_tag")
            else:  # drawcustom
                _allowed_fields(payload, transport_fields | {"elements", "background", "rotation", "fit", "refresh_mode", "dither_mode"})
                elements = _validate_draw_elements(payload.get("elements"))
                displays = getattr(getattr(device, "config", None), "displays", None)
                if not displays:
                    raise web.HTTPBadRequest(text="display does not expose a renderable configuration")
                display = displays[0]
                color_scheme = getattr(display, "color_scheme_enum", None)
                if color_scheme is None:
                    raise web.HTTPBadRequest(text="display color scheme is unavailable")
                image = await generate_image(
                    width=display.pixel_width,
                    height=display.pixel_height,
                    elements=elements,
                    background=_text(payload, "background", "white", 24),
                    accent_color=color_scheme.accent_color,
                )
                fit = str(payload.get("fit", "contain")).lower()
                refresh = str(payload.get("refresh_mode", "full")).lower()
                dither = str(payload.get("dither_mode", "burkes")).upper()
                if fit not in _ALLOWED_FITS or refresh not in _ALLOWED_REFRESH:
                    raise web.HTTPBadRequest(text="unsupported fit or refresh mode")
                if dither not in DitherMode.__members__:
                    raise web.HTTPBadRequest(text="unsupported dither mode")
                rotation = _integer(payload, "rotation", 0, 0, 270)
                if rotation not in {0, 90, 180, 270}:
                    raise web.HTTPBadRequest(text="rotation must be 0, 90, 180 or 270")
                await device.upload_image(
                    image,
                    fit=FitMode[fit.upper()],
                    refresh_mode=RefreshMode[refresh.upper()],
                    dither_mode=DitherMode[dither],
                    rotate=Rotation(rotation),
                )
    except web.HTTPException:
        raise
    except Exception as exc:
        _LOG.warning("OpenDisplay %s action failed for %s: %s", action, address, exc)
        raise web.HTTPBadGateway(text=f"OpenDisplay {action} action failed") from exc

    return web.json_response({"ok": True, "address": address, "action": action})


async def upload_image(request: web.Request) -> web.Response:
    if request.content_length is not None and request.content_length > 12 * 1024 * 1024:
        raise web.HTTPRequestEntityTooLarge(max_size=12 * 1024 * 1024, actual_size=request.content_length)
    try:
        payload = await request.json()
    except Exception as exc:
        raise web.HTTPBadRequest(text="invalid JSON body") from exc
    if not isinstance(payload, dict):
        raise web.HTTPBadRequest(text="JSON body must be an object")
    _allowed_fields(payload, {"image_base64", "fit", "refresh_mode", "transport", "host", "port", "tls", "mac"})
    address = request.match_info["address"]
    fit = str(payload.get("fit", "contain")).lower()
    refresh_mode = str(payload.get("refresh_mode", "full")).lower()
    if fit not in _ALLOWED_FITS:
        raise web.HTTPBadRequest(text="unsupported fit mode")
    if refresh_mode not in _ALLOWED_REFRESH:
        raise web.HTTPBadRequest(text="unsupported refresh mode")
    image = _decode_image(payload.get("image_base64"))
    try:
        async with _open_device(request, address, payload) as device:
            await device.upload_image(
                image, fit=FitMode[fit.upper()], refresh_mode=RefreshMode[refresh_mode.upper()]
            )
    except web.HTTPException:
        raise
    except Exception as exc:
        _LOG.warning("OpenDisplay image upload failed for %s: %s", address, exc)
        raise web.HTTPBadGateway(text="OpenDisplay image upload failed") from exc
    return web.json_response({"ok": True, "address": address, "transport": payload.get("transport", "ble")})


async def health(_: web.Request) -> web.Response:
    return web.json_response({"status": "ok", "component": "opendisplay-bridge"})


def create_app() -> web.Application:
    token = _required_token()
    app = web.Application(middlewares=[auth_middleware], client_max_size=12 * 1024 * 1024)
    app["token"] = token
    app["ble_scan_seconds"] = max(2.0, min(float(os.environ.get("OPENDISPLAY_BRIDGE_BLE_SCAN_SECONDS", "6")), 20.0))
    app["enable_wifi"] = os.environ.get("OPENDISPLAY_BRIDGE_ENABLE_WIFI", "true").lower() in {"1", "true", "yes"}
    app["device_keys"] = _load_device_keys()
    app.router.add_get("/health", health)
    app.router.add_get("/api/v1/devices", list_devices)
    app.router.add_post("/api/v1/devices/{address}/image", upload_image)
    app.router.add_post("/api/v1/devices/{address}/actions/{action}", device_action)
    return app


if __name__ == "__main__":
    logging.basicConfig(level=os.environ.get("LOG_LEVEL", "INFO").upper())
    host = os.environ.get("OPENDISPLAY_BRIDGE_BIND", "127.0.0.1")
    port = int(os.environ.get("OPENDISPLAY_BRIDGE_PORT", "10320"))
    web.run_app(create_app(), host=host, port=port, access_log=None)
