## 2025-05-06 - Add Rate Limiting and ReCaptcha
**Vulnerability:** Added Rate Limiting and ReCaptcha to the application for enhanced security.
**Learning:** Rate Limiting and ReCaptcha help prevent brute force attacks and abuse by limiting the number of requests from a single IP address and verifying that the user is not a bot.


## 2025-05-06 - Configure SameSite Attribute for OAuth State Cookie
**Vulnerability:** The OAuth state cookie, used to protect against CSRF during the Google login process, did not explicitly have the SameSite attribute set. This might lead to inconsistent browser behavior and potential exposure to CSRF attacks if the browser defaults to a more permissive setting.
**Learning:** Explicitly configuring the SameSite attribute (e.g., `http.SameSiteLaxMode` or `Strict`) for sensitive cookies like OAuth state tokens ensures consistent and secure cross-site request handling across different browsers.
**Prevention:** Always explicitly set the SameSite attribute on cookies, especially those involved in authentication, authorization, or state management, to enforce CSRF protection.
