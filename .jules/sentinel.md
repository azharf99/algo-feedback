## 2025-05-06 - Add Rate Limiting and ReCaptcha
**Vulnerability:** Added Rate Limiting and ReCaptcha to the application for enhanced security.
**Learning:** Rate Limiting and ReCaptcha help prevent brute force attacks and abuse by limiting the number of requests from a single IP address and verifying that the user is not a bot.


## 2025-07-07 - Add SameSite attribute to Cookie creation using Gin Framework
**Vulnerability:** The OAuth state parameter was stored in a cookie but lacked an explicit `SameSite` attribute configuration, rendering it susceptible to specific Cross-Site Request Forgery (CSRF) vectors depending on browser defaults.
**Learning:** In the Gin Framework, simply using `c.SetCookie` does not explicitly set the `SameSite` attribute. Modern browsers might default to `Lax`, but explicitly defining it is critical for robust security.
**Prevention:** Always call `c.SetSameSite(http.SameSiteLaxMode)` (or `Strict` if applicable) immediately before calling `c.SetCookie` when setting any security-sensitive cookies within a Gin handler.
