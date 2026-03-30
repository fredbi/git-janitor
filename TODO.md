* update "Facts": add progress bar (for github?)
* ~refresh repo: add progress bar~

go-fred-mcp addition:

 /btw just exporting all data structures is a seemingly simple refactoring action but it seems hard to execute. Could you
      describe what kind of extra tool you'd need to make this snappier and less error prone?

    Great question. Here's what would have made this dramatically easier:

    A semantic Go rename tool — something like gorename or gopls rename but designed for bulk operations. The core problem is
    that text-based tools (sed, string replace) can't distinguish between:
    - path as a struct field declaration
    - path as a local variable
    - path inside a string literal
    - .path as field access (always safe to rename)

    A bulk rename tool would accept a manifest like:
    type repoItem -> RepoItem
    field repoItem.path -> Path
    field repoItem.name -> Name

    and apply all renames atomically using AST-level understanding. No false positives on strings, no collisions with local
    variables, and it would catch the Title field vs Title() method clash upfront.

    What exists today but didn't quite fit:
    - gopls rename — correct and semantic, but one identifier at a time. For ~95 renames, that's ~95 sequential MCP calls, each
    needing file+position lookup
    - gorename — same limitation
    - gofmt -r — only handles expression rewrites, not identifier renames
    - sed — fast but purely textual, requires careful regex and manual fixup

    The ideal tool for Claude Code would be:
    1. A bulk-rename command that takes a list of (package, identifier, new_name) tuples
    2. Uses the Go AST/type checker to resolve all references semantically
    3. Applies all renames in one pass
    4. Reports conflicts (like the field/method clash) before writing

    Essentially: gopls rename with batch mode and dry-run conflict detection. The individual pieces exist in golang.org/x/tools
    — it's the batch orchestration and conflict pre-check that's missing.

