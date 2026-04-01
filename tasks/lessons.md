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
