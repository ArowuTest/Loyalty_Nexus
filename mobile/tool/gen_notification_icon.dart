// Generates the Android status-bar notification icon.
//
// Android 5+ renders notification icons as a MASK: only the alpha channel is used,
// and it is drawn in a single tint colour. Passing a full-colour launcher icon
// (which push_notification_service.dart did via '@mipmap/ic_launcher') therefore
// produces an undifferentiated grey/white square in the status bar.
//
// This builds a white-on-transparent silhouette at every density.
// Run from mobile/:  dart run tool/gen_notification_icon.dart
import 'dart:io';
import 'package:image/image.dart' as img;

/// Android notification icon sizes, in dp-scaled pixels.
const _densities = <String, int>{
  'drawable-mdpi': 24,
  'drawable-hdpi': 36,
  'drawable-xhdpi': 48,
  'drawable-xxhdpi': 72,
  'drawable-xxxhdpi': 96,
};

void main() {
  final srcFile = File('assets/images/logo-square.png');
  final src = img.decodePng(srcFile.readAsBytesSync());
  if (src == null) {
    stderr.writeln('could not decode ${srcFile.path}');
    exit(1);
  }

  // Decide how to derive the silhouette. If the source is largely transparent it
  // already carries shape information in its alpha; otherwise (an opaque logo on a
  // solid plate) shape has to come from luminance.
  var transparent = 0;
  final total = src.width * src.height;
  for (final p in src) {
    if (p.a < 40) transparent++;
  }
  final useAlpha = transparent > total * 0.15;
  stdout.writeln(useAlpha
      ? 'source has real transparency (${(100 * transparent / total).round()}%) — using alpha as the mask'
      : 'source is largely opaque — deriving the mask from luminance');

  for (final entry in _densities.entries) {
    final size = entry.value;
    final resized = img.copyResize(src,
        width: size, height: size, interpolation: img.Interpolation.average);

    final out = img.Image(width: size, height: size, numChannels: 4);
    for (var y = 0; y < size; y++) {
      for (var x = 0; x < size; x++) {
        final p = resized.getPixel(x, y);
        final lum = (0.299 * p.r + 0.587 * p.g + 0.114 * p.b) / 255.0;
        // White pixel; alpha carries the shape.
        final a = useAlpha ? p.a.toDouble() : (lum * 255.0);
        out.setPixelRgba(x, y, 255, 255, 255, a.clamp(0, 255).round());
      }
    }

    final dir = Directory('android/app/src/main/res/${entry.key}');
    dir.createSync(recursive: true);
    final f = File('${dir.path}/ic_stat_notification.png');
    f.writeAsBytesSync(img.encodePng(out, level: 9));
    stdout.writeln('  ${f.path}  ${size}x$size  ${f.lengthSync()}b');
  }
}
