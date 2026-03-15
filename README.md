# LFX Backend

Embedded PocketBase backend with custom sync APIs for the Flutter clinic app.

## Run

```bash
cd /home/peter/LFX-be
go run . serve --http=0.0.0.0:8090
```

Optional environment variables:

```bash
export PATH="$HOME/go-sdk/bin:$PATH"
export LFX_SYNC_API_KEY="change-me"
export LFX_PB_ENCRYPTION_KEY="change-me-too"
```

## Sync API

- `GET /api/lfx-sync/health`
- `GET /api/lfx-sync/pull`
- `POST /api/lfx-sync/customers/create`
- `POST /api/lfx-sync/customers/patch`
- `POST /api/lfx-sync/customers/delete`
- `POST /api/lfx-sync/recharges/create`
- `POST /api/lfx-sync/consumes/create`
- `POST /api/lfx-sync/logs/create`

The built-in PocketBase dashboard and APIs are still available, but collection CRUD rules require authenticated access. Flutter should use the custom sync routes instead of direct record writes.
