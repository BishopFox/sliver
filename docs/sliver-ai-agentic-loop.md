# Sliver AI Agentic Loop

## Goal

Port the example agentic loop from `examples/codex/codex-rs` into Sliver's AI chat so the server can:

- run a multi-step Responses API turn
- emit turn and item lifecycle events similar to Codex
- persist reasoning and tool-call items for sync and restoration
- keep those non-chat items out of the model context window
- render the new items in the client TUI

## How The Codex Example Loop Works

The Codex example splits the loop into three layers.

### 1. Provider/Protocol Layer

The protocol layer receives raw model events such as:

- turn started / completed / failed
- reasoning deltas and completed reasoning items
- command/tool begin and end notifications
- assistant messages

These provider events are fine-grained and often arrive as partial updates.

### 2. Event Processor Layer

`event_processor_with_jsonl_output.rs` converts the raw provider stream into a stable thread event model.

The important behaviors are:

- a turn gets a lifecycle: `turn.started`, then `turn.completed` or `turn.failed`
- each logical item gets a stable id
- begin/end pairs are folded into a single item lifecycle
- reasoning is emitted as its own item type
- tool calls and command executions are emitted as their own item types
- active items can be updated in place before completion

This is the key abstraction that makes the TUI and persisted history simple.

### 3. TUI Layer

The Codex TUI consumes those stable events rather than reloading the whole thread.

That lets it:

- append new items incrementally
- update an existing in-flight item in place
- keep the transcript stable while a turn runs
- restore the same item types from stored history later

## Sliver Port

Sliver ports the same shape, adapted to its existing RPC and database model.

### Conversation State

`AIConversation` now stores:

- `ActiveTurnID`
- `TurnState`
- `TargetSessionID`
- `TargetBeaconID`

The turn fields let clients restore "waiting", "completed", and "failed" state without inferring it from the last chat message.

The target fields bind a conversation to the active session or beacon so server-side tools can default to the conversation target.

### Message State

`AIConversationMessage` now stores:

- `Kind`: chat, reasoning, or tool call
- `Visibility`: context or UI-only
- `IncludeInContext`: whether the block should be replayed into the provider context window
- `State`: in progress, completed, or failed
- `TurnID`
- `ItemID`
- tool call metadata: name, arguments, result, error text, tool call id

This keeps one ordered transcript table while still distinguishing chat context from UI-only telemetry.

### UI-Only Persistence

Reasoning, tool-call, and intermediate assistant chat items can be stored as normal conversation messages with `Visibility = UI_ONLY`.

That gives Sliver:

- cross-client sync through the existing conversation storage path
- transcript restoration after reconnect or TUI restart
- ordered replay of an entire turn, including intermediate items

At the same time, the model context builder keys off `IncludeInContext`, so UI visibility and provider visibility are now independent.

### Event Model

Sliver now publishes `AIConversationEvent` payloads instead of raw `AIConversation` snapshots.

Supported event types:

- `CONVERSATION_UPDATED`
- `CONVERSATION_DELETED`
- `TURN_STARTED`
- `TURN_COMPLETED`
- `TURN_FAILED`
- `MESSAGE_STARTED`
- `MESSAGE_UPDATED`
- `MESSAGE_COMPLETED`

Each event can carry:

- a conversation summary
- a single message/item
- the turn id
- optional error text

This mirrors the Codex pattern of stable turn and item lifecycle events.

## Server Turn Flow

For each new user message:

1. The user chat message is saved as a normal context-visible chat item.
2. The server resolves runtime provider/model settings.
3. A new turn id is generated and persisted with `TURN_STARTED`.
4. If the runtime supports the Responses API loop, Sliver runs the agentic loop.
5. Reasoning items are persisted as completed UI-only reasoning blocks.
6. Tool calls are persisted as in-progress UI-only tool blocks, executed locally, then updated in place to completed or failed.
7. Intermediate assistant chat blocks on tool-use turns are persisted as UI-only chat items with `IncludeInContext = false`.
8. Tool outputs are fed back into the Responses API with `previous_response_id`.
9. When the model returns final assistant text, that assistant message is persisted as a normal context-visible chat item with `IncludeInContext = true`.
10. The conversation turn state is cleared with `TURN_COMPLETED`.

If anything fails:

- a UI-only failed system message is persisted
- the conversation is marked `TURN_FAILED`
- the failure remains visible in the transcript without polluting future model context

## Tool Surface

The first port exposes a safe read-only tool set:

- `list_sessions_and_beacons`
- `fs_ls`
- `fs_cat`
- `fs_pwd`

Why read-only first:

- it proves the loop, storage, eventing, and TUI behavior
- it avoids surprising destructive remote actions in the first server-side tool release

Filesystem tools accept explicit `session_id` or `beacon_id`, but when omitted they fall back to the conversation target.

Beacon-backed tools wait for task completion on the server before emitting the completed item update.

## Context Filtering

Only chat messages with:

- `Kind = CHAT`
- `IncludeInContext = true`

are included in future provider requests.

That means these items are excluded from the model context window:

- reasoning blocks
- tool call blocks
- UI-only intermediate assistant chat blocks
- failed system status blocks

This is the main rule that preserves transcript richness without degrading prompt quality.

## TUI Behavior

The AI TUI now consumes `AIConversationEvent` incrementally.

Instead of reloading the whole conversation for every server event, it:

- merges conversation summaries into the sidebar
- upserts messages in place by message id or item id
- updates waiting state from `TurnState`
- removes deleted conversations immediately

Reasoning and tool-call blocks render as dedicated dark-grey transcript boxes.

### Reasoning Blocks

Reasoning blocks show:

- the `Reasoning` label
- the persisted reasoning text
- item state metadata

### Tool Call Blocks

Tool blocks show:

- `Tool: <name>`
- arguments
- result
- error text when present
- item state metadata

These blocks are restored exactly like chat messages because they are stored in the same ordered transcript.

## Restore And Sync

Because UI-only items live in the database:

- another Sliver client can attach to the same conversation and see the full turn history
- reconnecting the TUI restores reasoning and tool-call blocks
- turn state survives restart, so "in progress" and "failed" status can be reconstructed from persisted conversation metadata

## Differences From Codex

This port intentionally differs from the example in a few places.

- Sliver currently uses non-streaming Responses API requests, so reasoning items arrive as completed items rather than token deltas.
- Tool execution is currently a bounded read-only server-side function set, not the full Codex local tool surface.
- Sliver persists intermediate items in its conversation DB rather than only in a transient thread runtime.

The important invariant is still the same:

- turn lifecycle is explicit
- item lifecycle is explicit
- UI-only items are persisted and replayable
- only context-visible chat reaches the model
