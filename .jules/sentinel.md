## 2025-05-06 - Add Rate Limiting and ReCaptcha
**Vulnerability:** Added Rate Limiting and ReCaptcha to the application for enhanced security.
**Learning:** Rate Limiting and ReCaptcha help prevent brute force attacks and abuse by limiting the number of requests from a single IP address and verifying that the user is not a bot.

## 2026-06-23 - [Sentinel] Set SameSite Cookie for CSRF Protection\n**Vulnerability:** The Gin SetCookie usage was missing explicit SetSameSite calls, making the oauth_state cookie vulnerable to CSRF/cross-site timing attacks depending on browser defaults.\n**Learning:** Gin's c.SetCookie() does not explicitly apply SameSite options by itself; it relies on c.SetSameSite() being called beforehand.\n**Prevention:** Use c.SetSameSite(http.SameSiteLaxMode) or http.SameSiteStrictMode immediately before c.SetCookie() to enforce proper CSRF protections.
