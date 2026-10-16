# Script per generare le icone Android a partire dalle icone della PWA Domus
Add-Type -AssemblyName System.Drawing

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Resolve-Path "$scriptDir\..\.."
$pwaMaster = Join-Path $projectRoot "web\public\pwa-512x512.png"
$resDir = Join-Path $projectRoot "mobile\android\app\src\main\res"

if (-not (Test-Path $pwaMaster)) {
    throw "PWA Master icon not found at: $pwaMaster"
}

Write-Output "Caricamento master PWA: $pwaMaster"
$masterImg = [System.Drawing.Bitmap]::FromFile($pwaMaster)

# Densità Android e dimensioni relative
$densities = @(
    @{ Name = "mdpi";    Scale = 1.0; Legacy = 48;  Adaptive = 108; ForeSize = 60  },
    @{ Name = "hdpi";    Scale = 1.5; Legacy = 72;  Adaptive = 162; ForeSize = 90  },
    @{ Name = "xhdpi";   Scale = 2.0; Legacy = 96;  Adaptive = 216; ForeSize = 120 },
    @{ Name = "xxhdpi";  Scale = 3.0; Legacy = 144; Adaptive = 324; ForeSize = 180 },
    @{ Name = "xxxhdpi"; Scale = 4.0; Legacy = 192; Adaptive = 432; ForeSize = 240 }
)

foreach ($d in $densities) {
    $mipmapFolder = Join-Path $resDir "mipmap-$($d.Name)"
    if (-not (Test-Path $mipmapFolder)) {
        New-Item -ItemType Directory -Path $mipmapFolder -Force | Out-Null
    }

    # Rimuovi eventuali vecchi file .webp template per evitare conflitti AAPT
    $oldWebp = Get-ChildItem -Path $mipmapFolder -Filter "*.webp" -ErrorAction SilentlyContinue
    foreach ($w in $oldWebp) {
        Remove-Item $w.FullName -Force
        Write-Output "Rimosso vecchio template: $($w.Name) in mipmap-$($d.Name)"
    }

    # 1. ic_launcher.png (Legacy Square/Squircle)
    $legacySize = $d.Legacy
    $legacyBmp = New-Object System.Drawing.Bitmap $legacySize, $legacySize, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $gLegacy = [System.Drawing.Graphics]::FromImage($legacyBmp)
    $gLegacy.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $gLegacy.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $gLegacy.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $gLegacy.DrawImage($masterImg, 0, 0, $legacySize, $legacySize)
    $gLegacy.Dispose()

    $legacyPath = Join-Path $mipmapFolder "ic_launcher.png"
    $legacyBmp.Save($legacyPath, [System.Drawing.Imaging.ImageFormat]::Png)
    $legacyBmp.Dispose()
    Write-Output "Generato: $legacyPath ($legacySize x $legacySize)"

    # 2. ic_launcher_round.png (Legacy Round)
    $roundBmp = New-Object System.Drawing.Bitmap $legacySize, $legacySize, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $gRound = [System.Drawing.Graphics]::FromImage($roundBmp)
    $gRound.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $gRound.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $gRound.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

    $margin = [Math]::Max(1.0, $legacySize * 0.015)
    $clipSize = $legacySize - (2 * $margin)
    $roundPath = New-Object System.Drawing.Drawing2D.GraphicsPath
    $roundPath.AddEllipse([float]$margin, [float]$margin, [float]$clipSize, [float]$clipSize)
    $gRound.SetClip($roundPath)
    $gRound.DrawImage($masterImg, 0, 0, $legacySize, $legacySize)
    $gRound.ResetClip()

    # Bordo circolare ciano delicato
    $penWidth = [Math]::Max(1.0, [float]($legacySize * 0.012))
    $pen = New-Object System.Drawing.Pen ([System.Drawing.Color]::FromArgb(70, 56, 189, 248), $penWidth)
    $gRound.DrawEllipse($pen, [float]$margin, [float]$margin, [float]$clipSize, [float]$clipSize)
    $pen.Dispose()
    $roundPath.Dispose()
    $gRound.Dispose()

    $roundOutPath = Join-Path $mipmapFolder "ic_launcher_round.png"
    $roundBmp.Save($roundOutPath, [System.Drawing.Imaging.ImageFormat]::Png)
    $roundBmp.Dispose()
    Write-Output "Generato: $roundOutPath ($legacySize x $legacySize round)"

    # 3. ic_launcher_foreground.png (Adaptive Icon Foreground su sfondo trasparente con Safe Zone standard)
    $adaptSize = $d.Adaptive
    $foreSize = $d.ForeSize
    $offset = [int](($adaptSize - $foreSize) / 2)

    $foreBmp = New-Object System.Drawing.Bitmap $adaptSize, $adaptSize, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $gFore = [System.Drawing.Graphics]::FromImage($foreBmp)
    $gFore.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $gFore.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
    $gFore.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    # Trasparente di base, disegna il tile Domus perfettamente centrato nella safe area (72dp)
    $gFore.DrawImage($masterImg, $offset, $offset, $foreSize, $foreSize)
    $gFore.Dispose()

    $forePath = Join-Path $mipmapFolder "ic_launcher_foreground.png"
    $foreBmp.Save($forePath, [System.Drawing.Imaging.ImageFormat]::Png)
    $foreBmp.Dispose()
    Write-Output "Generato: $forePath ($adaptSize x $adaptSize adaptive foreground)"
}

# 4. Google Play Store icon (512x512)
$playStorePath = Join-Path $projectRoot "mobile\android\app\src\main\ic_launcher-web.png"
$masterImg.Save($playStorePath, [System.Drawing.Imaging.ImageFormat]::Png)
Write-Output "Generato Play Store asset: $playStorePath (512x512)"

$masterImg.Dispose()
Write-Output "Tutte le immagini raster generate con successo!"
