import { NextRequest, NextResponse } from "next/server";
import PDFParser from "pdf2json";
import { promises as fs } from "fs";
import { v4 as uuidv4 } from "uuid";
// Temporarily comment out blob deletion
// import { del } from "@vercel/blob";

export async function POST(req: NextRequest) {
  const { url, originalFileName } = await req.json();

  if (!url) {
    return NextResponse.json(
      { error: "No file URL provided" },
      { status: 400 }
    );
  }

  // Validate URL is a Vercel Blob URL (for security)
  if (!url.includes("vercel-storage.com")) {
    return NextResponse.json({ error: "Invalid blob URL" }, { status: 400 });
  }

  // Log debugging information
  console.log("Attempting to fetch PDF from URL:", url);
  console.log(
    "Original filename (if provided):",
    originalFileName || "Not provided"
  );

  try {
    // Create a temp file to process the PDF
    const uniqueFileName = uuidv4();
    const tempFilePath = `/tmp/${uniqueFileName}.pdf`;

    // Try to fetch with a direct URL first
    try {
      console.log("Attempting direct fetch of blob URL...");
      const response = await fetch(url, {
        headers: {
          // Add cache-busting header to prevent cached 404s
          "Cache-Control": "no-cache",
          Pragma: "no-cache",
        },
      });

      if (!response.ok) {
        console.error(
          `Direct fetch failed with status: ${response.status}, statusText: ${response.statusText}`
        );
        console.error(
          `Response headers: ${JSON.stringify([...response.headers.entries()])}`
        );
        const responseText = await response
          .text()
          .catch((e) => `Error reading response text: ${e.message}`);
        console.error(`Response body: ${responseText.substring(0, 200)}...`);

        // If we're getting a 404, we could try to construct the URL differently
        // But for now, just return the error
        return NextResponse.json(
          {
            error: `File not found. Status: ${response.status}`,
            details: responseText.substring(0, 200),
          },
          { status: 404 }
        );
      }

      console.log(
        "Successfully fetched PDF, content length:",
        response.headers.get("content-length")
      );

      const fileBuffer = Buffer.from(await response.arrayBuffer());
      console.log("Created buffer with size:", fileBuffer.length);

      await fs.writeFile(tempFilePath, fileBuffer);
      console.log("Wrote file to temp path:", tempFilePath);
    } catch (fetchError) {
      console.error("Error fetching the blob:", fetchError);
      return NextResponse.json(
        {
          error: `Error fetching the blob: ${
            fetchError instanceof Error
              ? fetchError.message
              : String(fetchError)
          }`,
        },
        { status: 500 }
      );
    }

    // Process the PDF
    try {
      console.log("Starting to parse PDF at:", tempFilePath);
      const pdfParser = new PDFParser(null, true);

      const pdfData = await new Promise((resolve, reject) => {
        pdfParser.on("pdfParser_dataError", (error) => {
          console.error("PDF parsing error:", error);
          reject(new Error("Failed to parse PDF content"));
        });

        pdfParser.on("pdfParser_dataReady", () => {
          const data = pdfParser.getRawTextContent();
          console.log("Successfully parsed PDF, content length:", data?.length);
          resolve(data);
        });

        pdfParser.loadPDF(tempFilePath);
      });

      // Clean up the temporary file
      await fs
        .unlink(tempFilePath)
        .catch((err) => console.error("Failed to delete temp file:", err));

      // Temporarily disable blob deletion to debug the issue
      /*
      // Delete the blob file since we no longer need it
      try {
        await del(url);
        console.log(`Successfully deleted blob at ${url}`);
      } catch (deleteError) {
        console.error("Error deleting blob:", deleteError);
        // Continue even if deletion fails - this is not critical
      }
      */

      return NextResponse.json({ parsedText: pdfData });
    } catch (parseError) {
      console.error("Error parsing PDF:", parseError);
      // Clean up temp file if parsing fails
      await fs
        .unlink(tempFilePath)
        .catch((err) =>
          console.error("Failed to delete temp file after parse error:", err)
        );

      return NextResponse.json(
        {
          error: `Failed to parse PDF content: ${
            parseError instanceof Error
              ? parseError.message
              : String(parseError)
          }`,
        },
        { status: 500 }
      );
    }
  } catch (error) {
    console.error("General error processing PDF:", error);
    return NextResponse.json(
      {
        error: `Failed to process PDF content: ${
          error instanceof Error ? error.message : String(error)
        }`,
      },
      { status: 500 }
    );
  }
}
