# WAFFLE Documentation To-Do

Documentation topics and guides to create in future passes, prioritized by impact.

---

## High Priority

These topics would benefit most users and fill significant gaps:

1. **JSON API patterns** — Full request/response handling, encoding, error responses, validation
2. **API versioning** — Patterns for `/v1`, `/v2` versioned endpoints
3. **Error handling patterns** — Consistent error responses across features
4. **Testing patterns** — Unit testing handlers with mocked dependencies
5. **Configuration scenarios** — Let's Encrypt (http-01, dns-01), local dev, reverse proxy setups

---

## Medium Priority

Important for specific use cases or deeper understanding:

6. **Role-based access control** — Full example with auth middleware, roles, protected routes
7. **Multiple databases in DBDeps** — Connecting to multiple data stores
8. **Database migrations** — Setup guide using `golang-migrate/migrate`
9. **Graceful shutdown patterns** — Cleanup in Shutdown hook, resource release
10. **Environment-specific configuration** — Development vs. staging vs. production patterns
11. **Logging best practices** — Using zap logger effectively, structured logging patterns

---

## Lower Priority

Specialized topics for specific scenarios:

12. **WebSocket endpoints** — Real-time chat, notifications, live updates
13. **High-security admin panels** — Admin auth, roles, audit logging
14. **CORS for external SPAs** — Frontend served from different domain
15. **Shared route prefixes** — Organizing large features with common prefixes
16. **Advanced DBDeps organization** — Complex dependency structures

---

## Windows-Specific

For Windows deployment scenarios (basic Windows service docs exist):

17. **Windows service lifecycle events** — Logging service start/stop/pause
18. **Windows service with DBDeps cleanup** — Proper resource cleanup on service stop
19. **Windows distribution packaging** — Creating installers, MSI packages

---

## Documentation Infrastructure

Improvements to documentation organization and accessibility:

20. **Troubleshooting/FAQ** — Common errors, debugging tips, "why isn't X working?"
21. **Migration guide** — For developers coming from other frameworks (Gin, Echo, Fiber)
22. **Examples directory** — Complete, runnable example applications (not just code snippets)
23. **Changelog/versioning docs** — What changed between versions, upgrade paths
24. **Contributing guide** — How to contribute to WAFFLE itself (separate from doc guidelines)
25. ~~**Reorganize development.md**~~ — ✅ Done: now `docs/guides/development/` with focused documents

---

## Notes

Items are prioritized by:
- How many users would benefit
- Gap in current documentation
- Complexity of figuring it out without a guide
