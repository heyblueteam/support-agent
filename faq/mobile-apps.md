# Mobile apps — current app broken for some workspaces, new apps in October

Use this for any mobile question: "is there an app?", "the app does not work
with my workspace", or "when is the mobile app fixed?".

## Status (as of September 2026)

1. **The current native iOS and Android apps do not work with certain
   workspace configurations.** If a customer reports the app failing, loading
   empty, or erroring on one workspace while others work, this is the known
   issue — do not debug their device or ask them to reinstall as a first step.
2. **New iOS and Android apps release in October 2026.** They replace the
   current apps.
3. **Meanwhile, the PWA is the answer.** The web app at `https://blue.app`
   works in mobile browsers and installs as a PWA with full web-app
   functionality — including the workspace configurations the native apps
   cannot handle.

## What to tell customers

- Confirm the problem matches the known issue (one workspace misbehaves in the
  native app).
- Point them to the PWA as the working option today, with install steps below.
- Say new iOS and Android apps are coming in October. No exact date — say
  "October".

## PWA install (30 seconds)

Full guide: <https://blue.app/docs/start-guide/download-apps>

- **iOS (must use Safari):** open `https://blue.app` → Share icon →
  **Add to Home Screen** → Add.
- **Android (Chrome):** open `https://blue.app` → three-dot menu →
  **Install** / **Add to Home screen** → Install.

The PWA logs in with the same account and shows the full web app, so every
workspace configuration works.

## Customer reply template

```
Hi [Name],

Thanks for reporting this. The current mobile apps have a known problem with
certain workspace configurations, and your workspace is affected.

Two things:

1. Right now, please use Blue through your phone's browser at
   https://blue.app — it has the full web app and works on mobile. To get an
   app-like icon, install it as a home-screen app:

   - iPhone (Safari): open https://blue.app, tap the Share icon, then
     "Add to Home Screen".
   - Android (Chrome): open https://blue.app, tap the three-dot menu, then
     "Install" or "Add to Home screen".

2. We are releasing new iOS and Android apps in October. They fix this.

Sorry for the trouble, and thank you for your patience.

Best regards,
Manny
Founder of Blue
```
