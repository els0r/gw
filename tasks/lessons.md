# Lessons

## Never nesting

**Date:** 2026-04-01
**Trigger:** Nested `if` inside a `for` loop with an inner `if` + `else` — three levels deep in `splitArgs`.

**Rule:** Use early `continue`/`return` to keep loop and function bodies at one level of indentation. When you catch yourself writing `if X { ... } else { ... }` inside a loop, invert the condition and `continue`.

**Pattern to avoid:**

```go
for i := 0; i < len(args); i++ {
    if condition {
        // work
        if nestedCondition {
            // more work
        }
    } else {
        // other work
    }
}
```

**Correct pattern:**

```go
for i := 0; i < len(args); i++ {
    if !condition {
        // other work
        continue
    }
    // work
    if !nestedCondition {
        continue
    }
    // more work
}
```

**Checkpoint:** Before writing any loop body, ask: "Can I flip this condition and `continue`/`return` early?"

## Comment casing

**Date:** 2026-04-01
**Trigger:** Inline comments capitalized inconsistently (`// Append to log`, `// Filter entries`). Coding-style skill mandates lowercase.

**Rule:** Comments start lowercase unless they document an exported (public) symbol. Go's `godoc` convention requires `// FuncName ...` for exported functions — everything else is lowercase.

**Pattern to avoid:**

```go
// Append to log
f.WriteString(line)

// Update state file
os.WriteFile(path, data, 0o644)
```

**Correct pattern:**

```go
// append to log
f.WriteString(line)

// update state file
os.WriteFile(path, data, 0o644)
```

**Checkpoint:** Before writing a comment, ask: "Is this documenting an exported symbol?" If no, lowercase.

## Context propagation for HTTP calls

**Date:** 2026-04-08
**Trigger:** `earlySignIn` created an `http.Request` without context and `EarlyToken` had no `context.Context` parameter, making downstream HTTP calls uncancellable.

**Rule:** Every function that performs I/O (HTTP calls, database queries, RPCs) must accept `context.Context` as its first parameter and propagate it to all outbound requests. Never use `http.NewRequest` — always use `http.NewRequestWithContext`. Never call `context.Background()` deep in a call chain; accept the context from the caller and pass it through.

**Pattern to avoid:**

```go
func fetchData(url string) ([]byte, error) {
    req, _ := http.NewRequest("GET", url, nil)  // no context
    return http.DefaultClient.Do(req)
}

func caller() {
    data, _ := fetchData(url)  // can't cancel
}
```

**Correct pattern:**

```go
func fetchData(ctx context.Context, url string) ([]byte, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    return client.Do(req)
}

func caller(ctx context.Context) {
    data, _ := fetchData(ctx, url)  // cancellation propagates
}
```

**Checkpoint:** Before writing any HTTP or I/O call, ask: "Does this function accept a context? Does the request use `WithContext`?"
