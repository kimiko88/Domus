import os
from PIL import Image

MASTER_PATH = r"C:\Users\chimi\.gemini\antigravity-ide\brain\5c42993f-0273-4822-ae6b-b7843c82ceea\domus_app_icon_1791129741504.jpg"
PUBLIC_DIR = r"c:\Users\chimi\Desktop\Programmazione\Domus\web\public"
ICONS_DIR = os.path.join(PUBLIC_DIR, "icons")

os.makedirs(ICONS_DIR, exist_ok=True)

img = Image.open(MASTER_PATH).convert("RGBA")
width, height = img.size
print(f"Loaded master image: {width}x{height}")

# Standard sizes for PWA manifest
pwa_sizes = [72, 96, 128, 144, 152, 192, 384, 512]
for size in pwa_sizes:
    resized = img.resize((size, size), Image.Resampling.LANCZOS)
    out_path = os.path.join(ICONS_DIR, f"icon-{size}x{size}.png")
    resized.save(out_path, format="PNG", optimize=True)
    print(f"Generated {out_path} ({size}x{size})")

# Also copy 192 and 512 directly into public/ for root fallbacks
img.resize((192, 192), Image.Resampling.LANCZOS).save(os.path.join(PUBLIC_DIR, "pwa-192x192.png"), format="PNG", optimize=True)
img.resize((512, 512), Image.Resampling.LANCZOS).save(os.path.join(PUBLIC_DIR, "pwa-512x512.png"), format="PNG", optimize=True)

# Maskable icons with 15% safe area margin on #0b1120 background
maskable_sizes = [192, 512]
bg_color = (11, 17, 32, 255) # #0b1120
for size in maskable_sizes:
    canvas = Image.new("RGBA", (size, size), bg_color)
    content_size = int(size * 0.80) # 80% content size, 10% padding on each side
    scaled = img.resize((content_size, content_size), Image.Resampling.LANCZOS)
    offset = (size - content_size) // 2
    canvas.paste(scaled, (offset, offset), scaled)
    out_path = os.path.join(ICONS_DIR, f"icon-maskable-{size}x{size}.png")
    canvas.save(out_path, format="PNG", optimize=True)
    print(f"Generated maskable {out_path} ({size}x{size})")

# Apple touch icons
apple_sizes = [152, 167, 180]
for size in apple_sizes:
    resized = img.resize((size, size), Image.Resampling.LANCZOS)
    out_path = os.path.join(PUBLIC_DIR, f"apple-touch-icon-{size}x{size}.png")
    resized.save(out_path, format="PNG", optimize=True)
    if size == 180:
        # Default apple-touch-icon.png
        default_apple = os.path.join(PUBLIC_DIR, "apple-touch-icon.png")
        resized.save(default_apple, format="PNG", optimize=True)
        print(f"Generated default apple-touch-icon.png (180x180)")

# Favicons
img.resize((16, 16), Image.Resampling.LANCZOS).save(os.path.join(PUBLIC_DIR, "favicon-16x16.png"), format="PNG")
img.resize((32, 32), Image.Resampling.LANCZOS).save(os.path.join(PUBLIC_DIR, "favicon-32x32.png"), format="PNG")

# Multi-resolution favicon.ico (16, 32, 48)
ico_img = img.resize((48, 48), Image.Resampling.LANCZOS)
ico_path = os.path.join(PUBLIC_DIR, "favicon.ico")
ico_img.save(ico_path, format="ICO", sizes=[(16, 16), (32, 32), (48, 48)])
print(f"Generated multi-resolution {ico_path}")

print("All PWA icons successfully generated!")
