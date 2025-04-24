import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  // This middleware doesn't actually modify the request
  // It's here to document that you need to configure your hosting environment
  // to allow larger file uploads if using the App Router
  return NextResponse.next();
}

// Only apply this middleware to API routes that handle file uploads
export const config = {
  matcher: ["/api/parse-pdf", "/api/parse-office"],
};
