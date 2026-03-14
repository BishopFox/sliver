# client/command/ai

Implements the `ai` command for the Sliver client console.

The TUI loads server-backed AI conversation threads over gRPC, submits prompts to the server, and refreshes when `AIConversationEvent` updates arrive with assistant replies or failure messages.
