# Policy: Architecture & Patterns

## Hexagonal Strictness
1. **Domain (`/internal/domain`)**: Only pure Go structs and logic. No tags (except JSON), no imports from other internal folders.
2. **Ports (`/internal/ports`)**: Interfaces only. These define the "contracts" the service needs.
3. **Adapters (`/internal/adapters`)**: Technology-specific code. 
   - `adapters/kafka` can only import `ports` and `domain`.
   - Never import one adapter into another.

## Dependency Injection (DI)
- Use **Functional Options** for constructors.
- `main.go` is the only place where concrete adapters are instantiated and injected into the Service.
