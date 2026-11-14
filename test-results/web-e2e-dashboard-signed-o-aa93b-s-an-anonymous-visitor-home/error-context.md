# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: web/e2e/dashboard.spec.ts >> signed out >> the dashboard sends an anonymous visitor home
- Location: web/e2e/dashboard.spec.ts:23:2

# Error details

```
Error: browserType.launch: Executable doesn't exist at /var/folders/ys/4by_31t93397sdbnxfhcspz40000gn/T/cursor-sandbox-cache/654d084a224a3ea9bf61edb53d261df4/playwright/chromium_headless_shell-1248/chrome-headless-shell-mac-arm64/chrome-headless-shell
╔════════════════════════════════════════════════════════════╗
║ Looks like Playwright was just installed or updated.       ║
║ Please run the following command to download new browsers: ║
║                                                            ║
║     npx playwright install                                 ║
║                                                            ║
║ <3 Playwright Team                                         ║
╚════════════════════════════════════════════════════════════╝
```