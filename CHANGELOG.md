# Changelog

## 1.0.0 (2025-10-18)


### Features

* add OpenProject package ([c9af95b](https://github.com/LoriKarikari/compak/commit/c9af95bf4dddf04733d5836bdc9b50e5ea7a330a))
* add package search and index functionality ([b137e99](https://github.com/LoriKarikari/compak/commit/b137e99522eb7ce631d1e194f4d6f8d8bdbecf6b))
* add sync-upstream automation tool with GitHub Releases API and compose checksums ([e5202e9](https://github.com/LoriKarikari/compak/commit/e5202e927b795a48e904c58f504b826002c77a4e))
* add upgrade command with version validation ([8f24acd](https://github.com/LoriKarikari/compak/commit/8f24acd7d952d4409e55aaa145ec235d0d0add4f))
* add URL-based package installation ([bc85dec](https://github.com/LoriKarikari/compak/commit/bc85dec98cdd1621ab5b541bb94e2b8bb35d4c96))
* **cli:** add registry support to install command ([7492fc4](https://github.com/LoriKarikari/compak/commit/7492fc42d10f3fa21291608ff176dcb924753435))
* **cli:** add status and version commands ([5fd65aa](https://github.com/LoriKarikari/compak/commit/5fd65aaac0ddd1e42807705174e382cef0729462))
* **examples:** add nginx example package ([ba820a1](https://github.com/LoriKarikari/compak/commit/ba820a1e2867f16265e02e394465467b3003a694))
* implement complete OCI registry client with authentication ([f158555](https://github.com/LoriKarikari/compak/commit/f15855556cf8f7ebab3925f11dbac16dde0d4b97))
* implement core package management functionality ([3e6e06f](https://github.com/LoriKarikari/compak/commit/3e6e06f73df7dce68098092bbc0b985be0bb4f7f))
* implement native compose-go client ([351c25c](https://github.com/LoriKarikari/compak/commit/351c25ce982bf6ffca60c652c2598fa93e33f13d))
* implement OCI registry support with oras-go ([9feb80b](https://github.com/LoriKarikari/compak/commit/9feb80bff41cd4b6623b25b6ebc189f3c40ce545))
* initial boilerplate CLI setup ([23ba864](https://github.com/LoriKarikari/compak/commit/23ba864a403ca3a5d187b555b8a0c226dcb52158))
* **package:** add comprehensive security validation and lo library integration ([c613938](https://github.com/LoriKarikari/compak/commit/c6139384b6f345415b5a773e3bc25d6c1096c5df))
* **registry:** add OCI registry client foundation ([40b05e6](https://github.com/LoriKarikari/compak/commit/40b05e60d85cbafc4fd24fe8957a47c1f2eb12e0))
* **template:** add template engine with secure environment handling ([8e30e4e](https://github.com/LoriKarikari/compak/commit/8e30e4e57eda418d2df095565594ce74c96b858c))


### Bug Fixes

* **security:** remove COMPOSE_COMMAND env override to prevent command injection ([d81b5a4](https://github.com/LoriKarikari/compak/commit/d81b5a46d15a305fdb706bb2bef96be1edb191ed))
* support versioned packages with filename-based indexing ([acd9d32](https://github.com/LoriKarikari/compak/commit/acd9d321c770cf7a876eb0369ee3bd17c3b9974c))
* update golangci-lint config for latest version ([d80cca5](https://github.com/LoriKarikari/compak/commit/d80cca585c27edb8d7e0988ddb7e29b120779ab9))
* update gosec action to correct repository ([2965743](https://github.com/LoriKarikari/compak/commit/2965743dae1aa3910fc72df5861e4659331171a2))
* use correct env var names for Immich (DB_PASSWORD instead of POSTGRES_PASSWORD) ([bd37a17](https://github.com/LoriKarikari/compak/commit/bd37a170fda9fe5098021e09883c8da64468c40b))
