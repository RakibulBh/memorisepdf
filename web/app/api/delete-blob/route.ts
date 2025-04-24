import { NextRequest, NextResponse } from "next/server";
import { del } from "@vercel/blob";

export async function POST(req: NextRequest) {
  try {
    const { url } = await req.json();

    if (!url) {
      return NextResponse.json({ error: "URL is required" }, { status: 400 });
    }

    // Validate URL is a Vercel Blob URL (for security)
    if (!url.includes("vercel-storage.com")) {
      return NextResponse.json({ error: "Invalid blob URL" }, { status: 400 });
    }

    // Delete the blob
    await del(url);

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error("Error deleting blob:", error);
    return NextResponse.json(
      { error: "Failed to delete blob" },
      { status: 500 }
    );
  }
}
