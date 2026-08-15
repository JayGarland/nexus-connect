# nexus-connect

Nexus-owned extension and integration module for `cc-connect`.

## Architecture & Ownership

`nexus-connect` encapsulates all Nexus-specific transport wrappers, aggregators, debouncers, and Post Office boundary adaptations behind clean Go interfaces without polluting upstream `cc-connect` core or platform internals.

```text
chenhg5/cc-connect
        ↓ upstream

clean Nexus cc-connect fork
        │
        │ tiny compile-time glue only (cmd/cc-connect/plugin_extension_nexus.go)
        ▼
nexus-connect
        │
        └─ Nexus-owned integration implementation (DebouncedTelegramPlatform, TelegramAggregator)
```

## Features

- **Telegram Consecutive-Message Aggregation**: Transparently coalesces rapid sequential text messages within a configurable quiet window (`text_batch_window_ms`) into a single logical agent turn.
- **Deterministic Platform Wrapping**: Overrides `telegram` registration during Go package initialization, wrapping the upstream platform without modifying upstream source files.
- **Shutdown Safety**: Flushes buffered messages synchronously upon platform stop.
