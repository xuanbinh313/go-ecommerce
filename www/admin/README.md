# Admin

This app loads Medusa's official `@medusajs/dashboard` package and its Vite plugin. It keeps the Medusa dashboard UI and points its SDK at the Go API configured by `VITE_MEDUSA_BACKEND_URL` (default `http://localhost:8080/api/v1`).

Run from the repository root:

```powershell
pnpm --filter @micro-ecommerce/admin dev
```

The local URL is <http://localhost:7001>. Copy `.env.example` to `.env.local` if the API or storefront uses a different URL. The dashboard uses JWT authentication and stores its token under `micro_ecommerce_admin_jwt`.

The API authenticates admin routes against Medusa user records. If the database has Medusa RBAC enabled, the user also needs the matching Medusa policy; when RBAC is not enabled, authenticated admin users use Medusa's default admin access behavior. Storefront product reads are public.

For the Vietnamese setup and login flow through the Products list, see [Chạy Admin app và mở Products](../../docs/admin-products-run-vi.md).
