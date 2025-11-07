# goctxid

**A lightweight Go middleware for managing and propagating request/correlation IDs through `context.Context`.**

`goctxid` provides a simple way to ensure every request has a unique identifier, making your services observable and traceable. It's built on the standard `context.Context` package, making it compatible with any Go HTTP framework (with adapters included for popular frameworks like **Fiber**).

## 🚀 Features

* **Framework Agnostic:** Core logic is built on standard `context.Context`.
* **Middleware Adapters:** Includes a ready-to-use middleware for [Fiber](https://gofiber.io/).
* **Extract or Generate:** Automatically extracts an existing ID from request headers (e.g., `X-Correlation-ID`) or generates a new one if not found.
* **Propagation:**
      * Injects the ID into the `context.Context` (via `c.UserContext()` in Fiber) for use in your application logic (logging, downstream API calls).
      * Adds the ID to the response headers so clients (like web frontends or mobile apps) can also use it for debugging.
* **Customizable:**
      * Easily change the default header key (e.g., use `X-Request-ID`, `X-Trace-ID`).
      * Provide your own custom ID generator function (e.g., UUID, nanoid).
