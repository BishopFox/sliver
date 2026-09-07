# Firebase Admin Go SDK - Agent Guide

This document provides a comprehensive guide for AI agents to understand the conventions, design patterns, and architecture of the Firebase Admin Go SDK. Adhering to these guidelines is crucial for making idiomatic and consistent code contributions.

## 1. High-Level Overview

The Firebase Admin Go SDK enables server-side (backend) applications to interact with Firebase services. Its design emphasizes idiomatic Go, thread-safety, and a consistent, discoverable API surface.

## 2. Directory Structure

-   `firebase.go`: The primary entry point for initializing a Firebase `App` instance.
-   `internal/`: Contains private implementation details, such as HTTP clients and utility functions, that are not part of the public API.
-   `auth/`, `db/`, `messaging/`, etc.: Each directory contains a specific Firebase service client.
-   `*_test.go`: Unit tests are located alongside the code they test (e.g., `auth/auth_test.go`).
-   `integration/`: Contains integration tests that make live network calls to Firebase services.
-   `snippets/`: Contains code snippets used in documentation.
-   `errorutils/`: Contains common error type checkers and other error handling utils.
-   `testdata/`: Contains mock data used in some tests.

## 3. Core Design Patterns

-   **Initialization:** The SDK is initialized by creating an `App` instance via `firebase.NewApp()`. This `App` object is the central point for accessing all service clients.
-   **Service Clients:** Service clients (e.g., `auth.Client`, `db.Client`) are obtained from the `App` instance (e.g., `app.Auth(ctx)`). These clients are lightweight and are typically created as needed.
-   **Error Handling:** Errors are handled using standard Go idioms. Firbase errors are defined in `internal/errors.go` however these errors can be further modified within each service. This modification is applied using that service's set `internal.HTTPClient.CreateErrFn` value.
-   **HTTP Communication:** All outgoing HTTP requests are managed by a centralized client located in `internal/http_client.go`. This ensures consistent handling of authentication, retries, and error parsing.
-   **Asynchronous Operations:** The SDK uses `context.Context` to manage deadlines, cancellations, and request-scoped values for all asynchronous operations.

## 4. Coding Style and Naming Conventions

-   **Naming:**
    -   Public functions, types, and fields use `PascalCase`.
    -   Private functions and types use `camelCase`.
    -   Constants are written in `PascalCase`.

## 5. Testing Philosophy

-   **Unit Tests:** Unit tests follow the `*_test.go` naming pattern and are placed in the same directory as the code under test. They use standard Go testing packages and mocks to isolate dependencies.
-   **Integration Tests:** Integration tests are located in the `integration/` directory. They are designed to run against actual Firebase services and require a configured Firebase project.

## 6. Dependency Management

-   **Manager:** The SDK uses Go Modules for dependency management.
-   **Manifest:** Dependencies are declared in the `go.mod` file.
-   **Command:** To add or update dependencies, use `go get` or `go mod tidy`.

## 7. Critical Developer Journeys

### Journey 1: How to Add a New API Method

1.  **Define Public Method:** Add the new method or change to the appropriate service client files (e.g., `auth/user_mgt.go`).
2.  **Internal Logic:** Implement the core logic within the service package.
3.  **HTTP Client:** Use the client in `internal/http_client.go` to make the API calls.
4.  **Error Handling:** New or updated error codes implemented in the appropriate location.
5.  **Testing:**
    -   Add unit tests in the corresponding `*_test.go` file (e.g., `auth/user_mgt_test.go`).
    -   Add integration tests in the `integration/` directory if applicable.
6.  **Snippets:** (Optional) Add or update code snippets in the `snippets/` directory.

### Journey 2: How to Deprecate a Field/Method in an Existing API

1.  **Add Deprecation Note:** Locate where the deprecated object is defined and add a deprecation warning with a note (e.g. `// Deprecated: Use X instead.`).

## 8. Critical Do's and Don'ts

-   **DO:** Use the centralized HTTP client in `internal/http_client.go` for all network calls.
-   **DO:** Pass `context.Context` as the first argument to all functions that perform I/O or other blocking operations.
-   **DO:** Run `go fmt` after implementing a change and fix any linting errors.
-   **DON'T:** Expose types or functions from the `internal/` directory in the public API.
-   **DON'T:** Introduce new third-party dependencies without a strong, documented justification and team consensus.

## 9. Branch Creation
- When creating a new barnch use the format `agentName-short-description`.
    * Example: `jules-auth-token-parsing`
    * Example: `gemini-add-storage-file-signer`


## 10. Commit and Pull Request Generation

After implementing and testing a change, you may create a commit and pull request which must follow the following these rules:

### Commit and Pull Request Title Format:
Use the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification: `type(scope): subject`
- `type` should be one of `feat`, `fix` or `chore`.
- `scope` should be the service package changed (e.g., `auth`, `rtdb`, `deps`).
    - **Note**: Some services use specific abbreviations. Use the abbreviation if one exists. Common abbreviations include:
        - `messaging` -> `fcm`
        - `dataconnect` -> `fdc`
        - `database` -> `rtdb`
        - `appcheck` -> `fac`
- `subject` should be a brief summary of the change depending on the action:
    - For pull requests this should focus on the larger goal the included commits achieve.
        - Example: `fix(auth): Resolved issue with custom token verification`
    - For commits this should focus on the specific changes made in that commit.
        - Example: `fix(auth): Added a new token verification check`

### Commit Body:
    This should be a brief explanation of code changes.

Example:
```
feat(fcm): Added `SendEachForMulticast` support for multicast messages

Added a new `SendEachForMulticast` method to the messaging client. This method wraps the `SendEach` method and sends the same message to each token.
```

### Pull Request Body:
- A brief explanation of the problem and the solution.
- A summary of the testing strategy (e.g., "Added a new unit test to verify the fix.").
- A **Context Sources** section that lists the `id` and repository path of every `AGENTS.md` file you used.

Example:
```
feat(fcm): Added support for multicast messages

This change introduces a new `SendEachForMulticast` method to the messaging client, allowing developers to send a single message to multiple tokens efficiently.

Testing: Added unit tests in `messaging_test.go` with a mock server and an integration test in `integration/messaging_test.go`.

Context Sources Used:
- id: firebase-admin-go (`/AGENTS.md`)
```

## 11. Metadata
- id: firebase-admin-go