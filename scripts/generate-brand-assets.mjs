import { mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import sharp from "../admin-web/node_modules/sharp/dist/index.cjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const source = path.join(root, "design/brand/doodhee-icon-master.png");

const outputs = [
  // 256px is ample for the 48dp in-app mark on high-density phones.
  ["mobile-app/assets/images/doodhee-icon.png", 256],
  ["mobile-app/android/app/src/main/res/mipmap-mdpi/ic_launcher.png", 48],
  ["mobile-app/android/app/src/main/res/mipmap-hdpi/ic_launcher.png", 72],
  ["mobile-app/android/app/src/main/res/mipmap-xhdpi/ic_launcher.png", 96],
  ["mobile-app/android/app/src/main/res/mipmap-xxhdpi/ic_launcher.png", 144],
  ["mobile-app/android/app/src/main/res/mipmap-xxxhdpi/ic_launcher.png", 192],
  ["admin-web/public/brand/doodhee-icon.png", 512],
  ["admin-web/app/icon.png", 512],
  ["admin-web/app/apple-icon.png", 180],
];

const iosOutputs = [
  ["Icon-App-20x20@1x.png", 20],
  ["Icon-App-20x20@2x.png", 40],
  ["Icon-App-20x20@3x.png", 60],
  ["Icon-App-29x29@1x.png", 29],
  ["Icon-App-29x29@2x.png", 58],
  ["Icon-App-29x29@3x.png", 87],
  ["Icon-App-40x40@1x.png", 40],
  ["Icon-App-40x40@2x.png", 80],
  ["Icon-App-40x40@3x.png", 120],
  ["Icon-App-60x60@2x.png", 120],
  ["Icon-App-60x60@3x.png", 180],
  ["Icon-App-76x76@1x.png", 76],
  ["Icon-App-76x76@2x.png", 152],
  ["Icon-App-83.5x83.5@2x.png", 167],
  ["Icon-App-1024x1024@1x.png", 1024],
];

for (const [relativePath, size] of outputs) {
  const output = path.join(root, relativePath);
  await mkdir(path.dirname(output), { recursive: true });
  await sharp(source)
    .resize(size, size, { fit: "contain" })
    .png({ compressionLevel: 9, adaptiveFiltering: true })
    .toFile(output);
  console.log(`${relativePath} (${size}x${size})`);
}

const iosDirectory = path.join(
  root,
  "mobile-app/ios/Runner/Assets.xcassets/AppIcon.appiconset",
);
for (const [filename, size] of iosOutputs) {
  const output = path.join(iosDirectory, filename);
  await sharp(source)
    .resize(size, size, { fit: "contain" })
    .flatten({ background: "#075a35" })
    .png({ compressionLevel: 9, adaptiveFiltering: true })
    .toFile(output);
  console.log(`mobile-app/ios AppIcon/${filename} (${size}x${size})`);
}
