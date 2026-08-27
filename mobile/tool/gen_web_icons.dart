// Generates the PWA icons the web manifest declares but which were missing.
//
// frontend/public/manifest.json requires /icon-192.png and /icon-512.png; neither
// existed, so the installed web app had no icon at all. Uses the `image` package
// (already on the dev dependency tree via flutter_launcher_icons) so this needs no
// extra tooling — run from mobile/:
//   dart run tool/gen_web_icons.dart
import 'dart:io';
import 'package:image/image.dart' as img;

void main() {
  final srcFile = File('assets/images/logo-square.png');
  if (!srcFile.existsSync()) {
    stderr.writeln('missing ${srcFile.path}');
    exit(1);
  }
  final src = img.decodePng(srcFile.readAsBytesSync());
  if (src == null) {
    stderr.writeln('could not decode source');
    exit(1);
  }
  for (final size in [192, 512]) {
    final resized = img.copyResize(
      src,
      width: size,
      height: size,
      interpolation: img.Interpolation.average, // best quality for downscaling
    );
    final out = File('../frontend/public/icon-$size.png');
    out.writeAsBytesSync(img.encodePng(resized, level: 9));
    stdout.writeln('wrote ${out.path}  ${size}x$size  ${out.lengthSync()} bytes');
  }
}
