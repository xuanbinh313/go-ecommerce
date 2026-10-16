import { NextRequest, NextResponse } from "next/server"

const DEFAULT_REGION = (process.env.NEXT_PUBLIC_DEFAULT_REGION || "us").toLowerCase()
const BACKEND_URL = process.env.NEXT_PUBLIC_MEDUSA_BACKEND_URL || "http://localhost:9000"
const PUBLISHABLE_KEY = process.env.NEXT_PUBLIC_MEDUSA_PUBLISHABLE_KEY

type RegionList = {
  regions?: Array<{ countries?: Array<{ iso_2?: string }> }>
}

let countryCodes: Set<string> | undefined
let defaultCountryCode = DEFAULT_REGION

const getCountryCodes = async () => {
  if (countryCodes) return countryCodes

  try {
    const response = await fetch(`${BACKEND_URL}/store/regions?limit=100`, {
      headers: PUBLISHABLE_KEY
        ? { "x-publishable-api-key": PUBLISHABLE_KEY }
        : undefined,
      cache: "force-cache",
    })
    if (!response.ok) return new Set<string>()

    const data = (await response.json()) as RegionList
    const codes = (data.regions || [])
      .flatMap((region) => region.countries || [])
      .map((country) => country.iso_2?.toLowerCase())
      .filter((code): code is string => Boolean(code))

    if (codes.length) {
      countryCodes = new Set(codes)
      if (!countryCodes.has(defaultCountryCode)) defaultCountryCode = codes[0]
    }
  } catch {
    // Keep using the configured default while the API is unavailable.
  }

  return countryCodes || new Set<string>()
}

export async function middleware(request: NextRequest) {
  if (request.nextUrl.pathname.includes(".")) return NextResponse.next()

  const firstSegment = request.nextUrl.pathname.split("/")[1]?.toLowerCase()
  const supportedCountries = await getCountryCodes()
  if (firstSegment && supportedCountries.has(firstSegment)) return NextResponse.next()

  const pathname = request.nextUrl.pathname === "/" ? "" : request.nextUrl.pathname
  return NextResponse.redirect(
    `${request.nextUrl.origin}/${defaultCountryCode}${pathname}${request.nextUrl.search}`,
    307
  )
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|images|assets|png|svg|jpg|jpeg|gif|webp).*)"],
}
