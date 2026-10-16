# Storefront integration

This is the Medusa v2 starter storefront. Its `/[countryCode]/store` page now reads the published product list from the Nest API at `NEXT_PUBLIC_MEDUSA_BACKEND_URL` (default `http://localhost:3000/api/v1`). The first integration includes search, pagination, product images, and categories.

Copy `.env.example` to `.env.local`, then run `pnpm --filter @micro-ecommerce/storefront dev` from the repository root. Open <http://localhost:8000/us/store>.

Other starter routes still use Medusa region, cart, customer, and checkout APIs and are outside this product-list run.
