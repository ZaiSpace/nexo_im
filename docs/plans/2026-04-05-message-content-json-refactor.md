# Message Content JSON Refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor message persistence from multiple content columns to a single JSON content column while preserving existing HTTP and WebSocket API shapes.

**Architecture:** Store internal message content as a typed JSON payload on `entity.Message.Content` using GORM `serializer:json`. Keep gateway and HTTP APIs backward compatible by adapting flat `content.text/image/video/audio/file/custom` payloads to and from the new internal structure.

**Tech Stack:** Go, GORM, MySQL JSON columns, Hertz, Gorilla WebSocket, Go testing

---

### Task 1: Lock the compatibility boundaries with tests

**Files:**
- Create: `internal/entity/message_test.go`
- Create: `internal/gateway/ws_server_test.go`
- Create: `internal/service/message_service_test.go`

**Step 1: Write the failing tests**

- Add a test asserting `Message.ToMessageInfo()` still returns flat API content fields from the new internal typed content payload.
- Add a test asserting gateway mapping keeps the existing WebSocket `content.text/image/...` shape.
- Add a test asserting service validation rejects mismatched `msg_type` and payload.

**Step 2: Run test to verify it fails**

Run: `GOCACHE=/tmp/go-build go test ./internal/entity ./internal/gateway ./internal/service -run 'Test(Message|WsServer|ValidateMessageContent)' -count=1`

Expected: FAIL with compile or assertion failures because the new content model and validation do not exist yet.

### Task 2: Refactor entity and service models

**Files:**
- Modify: `internal/entity/message.go`
- Modify: `internal/service/message_service.go`

**Step 1: Write minimal implementation**

- Replace the multi-column content fields on `entity.Message` with a single `Content MessageContent` JSON field using `serializer:json`.
- Introduce typed nested content payload structs.
- Keep `MessageInfo` as the flat API DTO and convert from internal typed content in `ToMessageInfo()`.
- Add service-side content validation based on `msg_type`.

**Step 2: Run targeted tests**

Run: `GOCACHE=/tmp/go-build go test ./internal/entity ./internal/service -run 'Test(Message|ValidateMessageContent)' -count=1`

Expected: PASS

### Task 3: Adapt gateway and HTTP handlers without changing API shape

**Files:**
- Modify: `internal/gateway/ws_server.go`
- Modify: `internal/handler/message_handler.go`

**Step 1: Write minimal implementation**

- Convert flat WebSocket request content into internal typed `entity.MessageContent`.
- Convert internal typed message content back into flat WebSocket response content.
- Introduce HTTP request compatibility mapping so `/msg/send` still accepts flat content.

**Step 2: Run targeted tests**

Run: `GOCACHE=/tmp/go-build go test ./internal/gateway ./internal/handler -count=1`

Expected: PASS

### Task 4: Update schema and migration artifacts

**Files:**
- Modify: `migrations/001_init_schema.sql`
- Create: `migrations/002_message_content_json.sql`

**Step 1: Write migration SQL**

- Update fresh schema to use `content JSON`.
- Add a forward migration that backfills `content` from old columns and drops legacy columns when safe.

**Step 2: Verify SQL review**

Run: `sed -n '80,160p' migrations/001_init_schema.sql && sed -n '1,220p' migrations/002_message_content_json.sql`

Expected: Schema shows `content JSON` and migration backfill logic matches old column semantics.

### Task 5: Final verification

**Files:**
- Modify: `internal/entity/message.go`
- Modify: `internal/service/message_service.go`
- Modify: `internal/gateway/ws_server.go`
- Modify: `internal/handler/message_handler.go`
- Modify: `migrations/001_init_schema.sql`
- Create: `migrations/002_message_content_json.sql`

**Step 1: Run focused verification**

Run: `GOCACHE=/tmp/go-build go test ./internal/entity ./internal/gateway ./internal/handler ./internal/service -count=1`

Expected: PASS

**Step 2: Run broader repository verification if feasible**

Run: `GOCACHE=/tmp/go-build go test ./... -run '^$' -count=1`

Expected: compile-only signal for the repo; note unrelated failures explicitly if present.
