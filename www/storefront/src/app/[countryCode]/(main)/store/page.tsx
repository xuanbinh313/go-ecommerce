import { Metadata } from "next"

type Product = {
  id: string
  title: string
  handle: string
  subtitle?: string | null
  description?: string | null
  thumbnail?: string | null
  images?: { url: string }[]
  categories?: { id: string; name: string }[]
}

type ProductList = { products: Product[]; count: number; limit: number; offset: number }

export const metadata: Metadata = {
  title: "Products",
  description: "Browse products from our catalog.",
}

const backendUrl = process.env.NEXT_PUBLIC_MEDUSA_BACKEND_URL || "http://localhost:3000/api/v1"

async function getProducts(query: string, page: number): Promise<ProductList> {
  const limit = 12
  const params = new URLSearchParams({ limit: String(limit), offset: String((page - 1) * limit) })
  if (query) params.set("q", query)
  const response = await fetch(`${backendUrl}/store/products?${params}`, { cache: "no-store" })
  if (!response.ok) throw new Error(`Product API returned ${response.status}`)
  return response.json()
}

export default async function StorePage(props: {
  searchParams: Promise<{ q?: string; page?: string }>
}) {
  const searchParams = await props.searchParams
  const query = searchParams.q?.trim() || ""
  const page = Math.max(1, Number(searchParams.page) || 1)
  let result: ProductList | null = null
  let loadError = false

  try {
    result = await getProducts(query, page)
  } catch {
    loadError = true
  }

  const hasPrevious = page > 1
  const hasNext = result ? result.count > result.offset + result.limit : false

  return (
    <div className="content-container py-10 small:py-16">
      <div className="mb-8 flex flex-col gap-5 small:flex-row small:items-end small:justify-between">
        <div>
          <p className="text-small-regular text-ui-fg-subtle">Store</p>
          <h1 className="text-3xl-semi mt-2">Products</h1>
          <p className="text-base-regular mt-2 text-ui-fg-subtle">Browse the published product catalog.</p>
        </div>
        <form action="/store" method="get" className="flex w-full gap-2 small:w-auto">
          <input
            type="search"
            name="q"
            defaultValue={query}
            placeholder="Search products"
            className="border-ui-border-base bg-ui-bg-field text-ui-fg-base placeholder:text-ui-fg-muted focus:border-ui-border-interactive h-10 min-w-0 flex-1 rounded-md border px-3 small:w-72"
          />
          <button className="bg-ui-button-neutral text-ui-fg-on-color h-10 rounded-md px-4" type="submit">Search</button>
        </form>
      </div>

      {loadError ? (
        <div className="rounded-md border border-ui-border-base p-6 text-ui-fg-subtle">
          Could not load products. Check that the Nest API is running and the storefront API URL is configured.
        </div>
      ) : result?.products.length ? (
        <>
          <p className="text-small-regular mb-5 text-ui-fg-subtle">{result.count} products</p>
          <ul className="grid grid-cols-2 gap-x-4 gap-y-8 small:grid-cols-3 small:gap-x-6 medium:grid-cols-4">
            {result.products.map((product) => {
              const image = product.thumbnail || product.images?.[0]?.url
              return (
                <li key={product.id}>
                  <article>
                    <div className="bg-ui-bg-subtle relative aspect-[3/4] overflow-hidden rounded-md">
                      {image ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={image} alt={product.title} className="h-full w-full object-cover" />
                      ) : (
                        <div className="text-small-regular text-ui-fg-muted flex h-full items-center justify-center">No image</div>
                      )}
                    </div>
                    {product.categories?.length ? (
                      <p className="text-small-regular mt-3 text-ui-fg-subtle">{product.categories.map((category) => category.name).join(", ")}</p>
                    ) : null}
                    <h2 className="text-base-regular mt-1 font-medium">{product.title}</h2>
                    {product.subtitle ? <p className="text-small-regular mt-1 text-ui-fg-subtle">{product.subtitle}</p> : null}
                  </article>
                </li>
              )
            })}
          </ul>
          {(hasPrevious || hasNext) && (
            <nav aria-label="Product pages" className="mt-10 flex justify-between">
              {hasPrevious ? <a className="underline" href={`/store?${new URLSearchParams({ ...(query ? { q: query } : {}), page: String(page - 1) })}`}>Previous</a> : <span />}
              {hasNext ? <a className="underline" href={`/store?${new URLSearchParams({ ...(query ? { q: query } : {}), page: String(page + 1) })}`}>Next</a> : null}
            </nav>
          )}
        </>
      ) : (
        <div className="rounded-md border border-ui-border-base p-6 text-ui-fg-subtle">No products found.</div>
      )}
    </div>
  )
}
