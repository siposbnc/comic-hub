# Assembles an animated, looping GIF tour from PNG frames using WPF's GifBitmapEncoder,
# then patches the byte stream to add a NETSCAPE2.0 infinite-loop block and a per-frame
# delay (GifBitmapEncoder writes GCE blocks with delay 0). No ffmpeg/ImageMagick needed.
param(
  [string[]]$Frames,
  [string]$Out,
  [int]$DelayCs = 160,   # per-frame delay in centiseconds (160 = 1.6s)
  [int]$Width = 1000
)

Add-Type -AssemblyName PresentationCore
Add-Type -AssemblyName System.Drawing

function Load-Scaled([string]$path, [int]$w) {
  $src = New-Object System.Windows.Media.Imaging.BitmapImage
  $src.BeginInit()
  $src.CacheOption = 'OnLoad'
  $src.UriSource = New-Object System.Uri($path)
  $src.EndInit()
  $scale = $w / $src.PixelWidth
  $tb = New-Object System.Windows.Media.Imaging.TransformedBitmap($src, (New-Object System.Windows.Media.ScaleTransform($scale, $scale)))
  return $tb
}

$encoder = New-Object System.Windows.Media.Imaging.GifBitmapEncoder
foreach ($f in $Frames) {
  $bmp = Load-Scaled $f $Width
  $encoder.Frames.Add([System.Windows.Media.Imaging.BitmapFrame]::Create($bmp))
}
$ms = New-Object System.IO.MemoryStream
$encoder.Save($ms)
$bytes = $ms.ToArray()
$ms.Dispose()

# GifBitmapEncoder emits no Graphic Control Extensions, so we walk the GIF block structure
# and (a) insert a NETSCAPE2.0 infinite-loop block after the Global Color Table, and
# (b) emit a GCE carrying our frame delay before every Image Descriptor.
$buf = New-Object System.Collections.Generic.List[byte]
$p = 6                                   # skip "GIF89a"
$buf.AddRange([byte[]]$bytes[0..12])     # header(6) + Logical Screen Descriptor(7)
$packed = $bytes[10]
$p = 13
if ($packed -band 0x80) {                # Global Color Table present
  $gctSize = 3 * [math]::Pow(2, ($packed -band 0x07) + 1)
  $buf.AddRange([byte[]]$bytes[13..(13 + $gctSize - 1)])
  $p = 13 + $gctSize
}
# NETSCAPE2.0 application extension = loop forever
$buf.AddRange([byte[]](0x21,0xFF,0x0B,0x4E,0x45,0x54,0x53,0x43,0x41,0x50,0x45,0x32,0x2E,0x30,0x03,0x01,0x00,0x00,0x00))

$dLo = [byte]($DelayCs -band 0xFF); $dHi = [byte](($DelayCs -shr 8) -band 0xFF)
function Skip-SubBlocks([byte[]]$b, [ref]$pos) {
  while ($true) {
    $len = $b[$pos.Value]; $pos.Value++
    if ($len -eq 0) { break }
    $pos.Value += $len
  }
}
while ($p -lt $bytes.Length) {
  $marker = $bytes[$p]
  if ($marker -eq 0x3B) { $buf.Add(0x3B); break }           # trailer
  elseif ($marker -eq 0x2C) {                                # Image Descriptor
    $buf.AddRange([byte[]](0x21,0xF9,0x04,0x00,$dLo,$dHi,0x00,0x00))  # GCE with delay
    $buf.AddRange([byte[]]$bytes[$p..($p+9)])                # 10-byte descriptor
    $imgPacked = $bytes[$p+9]; $p += 10
    if ($imgPacked -band 0x80) {                             # Local Color Table
      $lct = 3 * [math]::Pow(2, ($imgPacked -band 0x07) + 1)
      $buf.AddRange([byte[]]$bytes[$p..($p + $lct - 1)]); $p += $lct
    }
    $buf.Add($bytes[$p]); $p++                               # LZW min code size
    $start = $p; $pp = [ref]$p; Skip-SubBlocks $bytes $pp    # image data sub-blocks
    $buf.AddRange([byte[]]$bytes[$start..($p-1)])
  }
  elseif ($marker -eq 0x21) {                                # existing extension: copy through
    $buf.Add(0x21); $buf.Add($bytes[$p+1]); $p += 2
    $start = $p; $pp = [ref]$p; Skip-SubBlocks $bytes $pp
    $buf.AddRange([byte[]]$bytes[$start..($p-1)])
  }
  else { throw ("unexpected GIF marker 0x{0:X2} at {1}" -f $marker, $p) }
}
[System.IO.File]::WriteAllBytes($Out, $buf.ToArray())

# Structure report for validation.
$b = [System.IO.File]::ReadAllBytes($Out)
$gce = 0; for ($i=0; $i -lt $b.Length-2; $i++){ if($b[$i]-eq0x21 -and $b[$i+1]-eq0xF9 -and $b[$i+2]-eq0x04){$gce++} }
$hasLoop = $false; for ($i=0;$i -lt $b.Length-10;$i++){ if($b[$i]-eq0x4E -and $b[$i+1]-eq0x45 -and $b[$i+2]-eq0x54 -and $b[$i+3]-eq0x53){$hasLoop=$true;break} }
Write-Host ("GIF written: {0}  ({1:N0} bytes, {2} frames, loop={3}, delay={4}cs)" -f $Out, $b.Length, $gce, $hasLoop, $DelayCs)
