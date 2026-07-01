# Generates a DC-themed demo comic library (CBZ + ComicInfo.xml) for README screenshots.
# Covers are drawn with GDI+; each CBZ holds a cover + a few interior pages + ComicInfo.xml.
param(
  [string]$OutDir = (Join-Path $env:TEMP 'comichub-demo-library')
)

Add-Type -AssemblyName System.Drawing

if (Test-Path $OutDir) { Remove-Item -Recurse -Force $OutDir }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

function New-Cover {
  param(
    [string]$Path, [string]$Series, [string]$Number, [string]$Year,
    [System.Drawing.Color]$Top, [System.Drawing.Color]$Bottom,
    [System.Drawing.Color]$Accent, [string]$Emblem
  )
  $w = 700; $h = 1050
  $bmp = New-Object System.Drawing.Bitmap($w, $h)
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.SmoothingMode = 'AntiAlias'
  $g.TextRenderingHint = 'AntiAliasGridFit'

  # Background gradient
  $rect = New-Object System.Drawing.Rectangle(0, 0, $w, $h)
  $grad = New-Object System.Drawing.Drawing2D.LinearGradientBrush($rect, $Top, $Bottom, 90)
  $g.FillRectangle($grad, $rect)

  # Accent emblem circle
  $cx = $w / 2; $cy = 360; $r = 190
  $emblemBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(38, 255, 255, 255))
  $g.FillEllipse($emblemBrush, ($cx - $r), ($cy - $r), (2 * $r), (2 * $r))
  $accentBrush = New-Object System.Drawing.SolidBrush($Accent)
  $emFont = New-Object System.Drawing.Font('Segoe UI', 150, [System.Drawing.FontStyle]::Bold)
  $fmt = New-Object System.Drawing.StringFormat
  $fmt.Alignment = 'Center'; $fmt.LineAlignment = 'Center'
  $g.DrawString($Emblem, $emFont, $accentBrush, $cx, $cy, $fmt)

  # Bottom band
  $band = New-Object System.Drawing.Rectangle(0, ($h - 340), $w, 340)
  $bandBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(190, 8, 10, 16))
  $g.FillRectangle($bandBrush, $band)
  $g.FillRectangle($accentBrush, (New-Object System.Drawing.Rectangle(0, ($h - 340), $w, 8)))

  # Publisher kicker
  $kickFont = New-Object System.Drawing.Font('Segoe UI', 18, [System.Drawing.FontStyle]::Bold)
  $kickBrush = New-Object System.Drawing.SolidBrush($Accent)
  $g.DrawString('DC COMICS', $kickFont, $kickBrush, 44, ($h - 320))

  # Series title (wrapped)
  $titleFont = New-Object System.Drawing.Font('Segoe UI', 54, [System.Drawing.FontStyle]::Bold)
  $white = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)
  $titleRect = New-Object System.Drawing.RectangleF(40, ($h - 285), 620, 190)
  $g.DrawString($Series, $titleFont, $white, $titleRect)

  # Issue number + year
  $numFont = New-Object System.Drawing.Font('Segoe UI', 26, [System.Drawing.FontStyle]::Bold)
  $g.DrawString("#$Number   -   $Year", $numFont, $white, 44, ($h - 90))

  $g.Dispose()
  $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Jpeg)
  $bmp.Dispose()
}

function New-Page {
  param([string]$Path, [string]$Label, [System.Drawing.Color]$Tint)
  $w = 700; $h = 1050
  $bmp = New-Object System.Drawing.Bitmap($w, $h)
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.Clear([System.Drawing.Color]::FromArgb(24, 26, 33))
  $b = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(60, $Tint.R, $Tint.G, $Tint.B))
  $g.FillRectangle($b, 60, 80, ($w - 120), ($h - 160))
  $f = New-Object System.Drawing.Font('Segoe UI', 40, [System.Drawing.FontStyle]::Bold)
  $fmt = New-Object System.Drawing.StringFormat; $fmt.Alignment = 'Center'; $fmt.LineAlignment = 'Center'
  $wb = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(150, 255, 255, 255))
  $g.DrawString($Label, $f, $wb, ($w / 2), ($h / 2), $fmt)
  $g.Dispose()
  $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Jpeg)
  $bmp.Dispose()
}

function Add-Series {
  param(
    [string]$Series, [int]$Start, [int]$Count, [string]$Year, [int]$Volume,
    [string]$Writer, [string]$Summary, [string]$AgeRating,
    [System.Drawing.Color]$Top, [System.Drawing.Color]$Bottom, [System.Drawing.Color]$Accent, [string]$Emblem
  )
  $safe = ($Series -replace '[^\w \-]', '')
  $dir = Join-Path $OutDir $safe
  New-Item -ItemType Directory -Force -Path $dir | Out-Null
  for ($i = 0; $i -lt $Count; $i++) {
    $num = $Start + $i
    $stage = Join-Path $env:TEMP ("stage_{0}_{1}" -f $safe, $num)
    if (Test-Path $stage) { Remove-Item -Recurse -Force $stage }
    New-Item -ItemType Directory -Force -Path $stage | Out-Null

    New-Cover -Path (Join-Path $stage '000_cover.jpg') -Series $Series -Number $num -Year $Year -Top $Top -Bottom $Bottom -Accent $Accent -Emblem $Emblem
    New-Page -Path (Join-Path $stage '001.jpg') -Label 'PAGE 1' -Tint $Accent
    New-Page -Path (Join-Path $stage '002.jpg') -Label 'PAGE 2' -Tint $Accent
    New-Page -Path (Join-Path $stage '003.jpg') -Label 'PAGE 3' -Tint $Accent

    $xml = @"
<?xml version="1.0" encoding="utf-8"?>
<ComicInfo>
  <Series>$Series</Series>
  <Number>$num</Number>
  <Volume>$Volume</Volume>
  <Year>$Year</Year>
  <Month>3</Month>
  <Writer>$Writer</Writer>
  <Publisher>DC Comics</Publisher>
  <Genre>Superhero</Genre>
  <AgeRating>$AgeRating</AgeRating>
  <Summary>$Summary</Summary>
  <LanguageISO>en</LanguageISO>
</ComicInfo>
"@
    Set-Content -Path (Join-Path $stage 'ComicInfo.xml') -Value $xml -Encoding UTF8

    $cbz = Join-Path $dir ("{0} #{1:000}.cbz" -f $safe, $num)
    if (Test-Path $cbz) { Remove-Item -Force $cbz }
    Compress-Archive -Path (Join-Path $stage '*') -DestinationPath ($cbz -replace '\.cbz$', '.zip')
    Rename-Item -Path ($cbz -replace '\.cbz$', '.zip') -NewName ([System.IO.Path]::GetFileName($cbz))
    Remove-Item -Recurse -Force $stage
  }
  Write-Host "  $Series - $Count issues"
}

$C = [System.Drawing.Color]
Write-Host "Generating demo library at $OutDir"
Add-Series -Series 'Batman' -Start 1 -Count 8 -Year '2016' -Volume 3 -Writer 'Tom King' -AgeRating 'Teen' -Emblem 'B' `
  -Summary 'A new era for the Dark Knight begins as Batman faces two new heroes over the skies of Gotham City.' `
  -Top ($C::FromArgb(30,34,46)) -Bottom ($C::FromArgb(10,11,16)) -Accent ($C::FromArgb(255,214,10))
Add-Series -Series 'Detective Comics' -Start 1000 -Count 6 -Year '2019' -Volume 1 -Writer 'Peter J. Tomasi' -AgeRating 'Teen' -Emblem 'D' `
  -Summary 'The landmark 1000th issue: the greatest detective faces a mystery decades in the making.' `
  -Top ($C::FromArgb(20,30,54)) -Bottom ($C::FromArgb(8,12,22)) -Accent ($C::FromArgb(120,170,255))
Add-Series -Series 'Wonder Woman' -Start 1 -Count 6 -Year '2016' -Volume 5 -Writer 'Greg Rucka' -AgeRating 'Teen' -Emblem 'W' `
  -Summary 'Diana of Themyscira searches for the truth about her origins and the lies that shaped her.' `
  -Top ($C::FromArgb(120,20,30)) -Bottom ($C::FromArgb(40,8,14)) -Accent ($C::FromArgb(255,196,60))
Add-Series -Series 'The Flash' -Start 1 -Count 5 -Year '2016' -Volume 5 -Writer 'Joshua Williamson' -AgeRating 'Teen' -Emblem 'F' `
  -Summary 'A storm of the Speed Force grants Central City new speedsters - and new dangers.' `
  -Top ($C::FromArgb(150,30,20)) -Bottom ($C::FromArgb(50,10,8)) -Accent ($C::FromArgb(255,224,60))
Add-Series -Series 'Superman' -Start 1 -Count 5 -Year '2018' -Volume 5 -Writer 'Brian Michael Bendis' -AgeRating 'Everyone' -Emblem 'S' `
  -Summary 'The Man of Steel confronts a threat that spans the galaxy and strikes at the heart of his family.' `
  -Top ($C::FromArgb(20,40,90)) -Bottom ($C::FromArgb(8,14,32)) -Accent ($C::FromArgb(230,60,60))
Add-Series -Series 'Justice League' -Start 1 -Count 6 -Year '2018' -Volume 4 -Writer 'Scott Snyder' -AgeRating 'Teen' -Emblem 'JL' `
  -Summary 'The Legion of Doom rises as the League confronts the doorway to the Sixth Dimension.' `
  -Top ($C::FromArgb(24,32,64)) -Bottom ($C::FromArgb(10,12,24)) -Accent ($C::FromArgb(240,200,80))
Add-Series -Series 'Green Lantern' -Start 1 -Count 4 -Year '2018' -Volume 6 -Writer 'Grant Morrison' -AgeRating 'Teen' -Emblem 'GL' `
  -Summary 'Hal Jordan patrols the far reaches of space as an intergalactic cop for the Green Lantern Corps.' `
  -Top ($C::FromArgb(16,60,34)) -Bottom ($C::FromArgb(6,20,12)) -Accent ($C::FromArgb(90,240,140))
Add-Series -Series 'Aquaman' -Start 1 -Count 4 -Year '2016' -Volume 8 -Writer 'Dan Abnett' -AgeRating 'Teen' -Emblem 'A' `
  -Summary 'The King of Atlantis fights to keep peace between the surface world and the seven seas.' `
  -Top ($C::FromArgb(12,60,72)) -Bottom ($C::FromArgb(6,22,28)) -Accent ($C::FromArgb(255,150,70))

Write-Host "Done."
