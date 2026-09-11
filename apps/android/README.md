# apps/android — "RexiO Pay Engine" (pay.rexio.engine)

Android SMS receiver app for RexiO Pay. Kotlin 2.x + Jetpack Compose + Material 3,
Room outbox, WorkManager delivery, Hilt, min SDK 26. Full spec: REQUIREMENT.md §13.

## Status

Placeholder — implemented in milestone M7. Sideloaded via GitHub Releases (not Play Store).

## Build

```bash
./gradlew assembleDebug
```

`local.properties` (gitignored):
```
rexio.serverUrl=https://api.pay.rexio.pro
```
