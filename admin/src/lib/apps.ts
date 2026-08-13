// User-facing apps that require account/password auth.
// Kept as a frontend constant rather than fetched from GET /admin/apps because:
// (a) apps are a product decision, not runtime data;
// (b) GET /admin/apps reverse-derives from users.app_scope, so a brand-new
//     app with no users yet would not appear.
// The "admin" pseudo-scope is intentionally excluded here — it's a cross-app
// pass-through managed by the backend based on role, never a user-visible app.

export interface AppMeta {
  code: string
  name: string
}

export const SUPPORTED_APPS: AppMeta[] = [
  { code: 'finflow', name: 'FinFlow' },
  { code: 'typresume', name: 'TypResume' },
]

export const SUPPORTED_APP_CODES = SUPPORTED_APPS.map(a => a.code)

export function appName(code: string): string {
  return SUPPORTED_APPS.find(a => a.code === code)?.name ?? code
}
