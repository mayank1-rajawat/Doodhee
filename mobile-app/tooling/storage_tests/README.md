# Host-only regression tests

Run from this directory:

```bash
flutter pub get
flutter test
```

This separate package imports the real Doodhee repositories and adds desktop SQLite
only for host tests. Keeping `sqflite_common_ffi` out of the application's dependency
graph prevents its SQLite native build hook from bundling a second SQLite library in
debug Android APKs. The measured release baseline already tree-shook that library.
Mobile production code continues to use the `sqflite` platform plugin.
