import { handleUpload, type HandleUploadBody } from "@vercel/blob/client";
import { NextResponse } from "next/server";

export async function POST(request: Request): Promise<NextResponse> {
  const body = (await request.json()) as HandleUploadBody;

  try {
    const jsonResponse = await handleUpload({
      body,
      request,
      onBeforeGenerateToken: async (pathname: string) => {
        // Generate a client token for the browser to upload the file
        return {
          allowOverwrite: true, // Always allow overwriting existing blobs
          allowedContentTypes: [
            "application/pdf",
            "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
            "application/vnd.openxmlformats-officedocument.presentationml.presentation",
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
            "application/vnd.oasis.opendocument.text",
            "application/vnd.oasis.opendocument.presentation",
            "application/vnd.oasis.opendocument.spreadsheet",
          ],
          tokenPayload: JSON.stringify({
            // Any custom data you want to receive on upload completion
            timestamp: Date.now(),
          }),
        };
      },
      onUploadCompleted: async ({ blob, tokenPayload }) => {
        // This won't work with localhost
        console.log("blob upload completed", blob, tokenPayload);

        try {
          // You could store the blob URL in your database here
          // const { timestamp } = JSON.parse(tokenPayload);
          // await db.insert({ fileUrl: blob.url, uploadedAt: timestamp });
        } catch (error) {
          console.error("Error processing completed upload:", error);
        }
      },
    });

    return NextResponse.json(jsonResponse);
  } catch (error) {
    console.error("Upload error:", error);
    return NextResponse.json(
      { error: (error as Error).message },
      { status: 400 }
    );
  }
}
