# Doodhee brand icon

`doodhee-icon-master.png` is the canonical Doodhee application mark. It was selected from the generated source `exec-127d12af-138d-44e7-a6a5-cae6a8cde252.png` and shows the core doorstep milk-delivery flow with a confirmation check.

Regenerate all Android, iOS, Flutter and admin-web icon sizes from the repository root:

```bash
node scripts/generate-brand-assets.mjs
```

Do not edit the generated platform files independently. iOS outputs are flattened onto the brand green because App Store icons cannot contain transparency; Android and web outputs retain the source transparency.
