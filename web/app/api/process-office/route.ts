import { NextRequest, NextResponse } from "next/server";
import { parseOfficeAsync } from "officeparser";
import { promises as fs } from "fs";
import { v4 as uuidv4 } from "uuid";
// Temporarily comment out blob deletion
// import { del } from "@vercel/blob";

// Supported office file extensions
const SUPPORTED_FORMATS = [
  ".docx", // Word
  ".pptx", // PowerPoint
  ".xlsx", // Excel
  ".odt", // OpenDocument Text
  ".odp", // OpenDocument Presentation
  ".ods", // OpenDocument Spreadsheet
];

export async function POST(req: NextRequest) {
  const { url, fileName, safeFileName } = await req.json();

  if (!url || !fileName) {
    return NextResponse.json(
      { error: "No file URL or filename provided" },
      { status: 400 }
    );
  }

  // Validate URL is a Vercel Blob URL (for security)
  if (!url.includes("vercel-storage.com")) {
    return NextResponse.json({ error: "Invalid blob URL" }, { status: 400 });
  }

  // Log debugging information
  console.log("Attempting to fetch Office document from URL:", url);
  console.log("Original filename:", fileName);
  console.log("Safe filename (if provided):", safeFileName || "Not provided");

  // Check if file is a supported office format
  const fileNameLower = fileName.toLowerCase();
  const isSupported = SUPPORTED_FORMATS.some((ext) =>
    fileNameLower.endsWith(ext)
  );

  if (!isSupported) {
    return NextResponse.json(
      {
        error:
          "Unsupported file format. Please upload a valid Office document.",
      },
      { status: 400 }
    );
  }

  try {
    // Create a temp file to process the document
    const uniqueFileName = uuidv4();
    const fileExtension = fileNameLower.substring(
      fileNameLower.lastIndexOf(".")
    );
    const tempFilePath = `/tmp/${uniqueFileName}${fileExtension}`;

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

        // If we're getting a 404, the URL might be incorrectly formatted
        // Let's try constructing the URL differently
        if (response.status === 404 && safeFileName) {
          console.log("Trying alternative URL construction...");

          // Extract the base URL (everything before the filename)
          const urlParts = url.split("/");
          urlParts.pop(); // Remove the last part (filename)
          const baseUrl = urlParts.join("/");

          // Try with the safeFileName directly
          const alternativeUrl = `${baseUrl}/${safeFileName}`;
          console.log("Trying alternative URL:", alternativeUrl);

          const alternativeResponse = await fetch(alternativeUrl, {
            headers: {
              "Cache-Control": "no-cache",
              Pragma: "no-cache",
            },
          });

          if (alternativeResponse.ok) {
            console.log("Alternative URL worked!");
            const fileBuffer = Buffer.from(
              await alternativeResponse.arrayBuffer()
            );
            console.log("Created buffer with size:", fileBuffer.length);
            await fs.writeFile(tempFilePath, fileBuffer);
            console.log("Wrote file to temp path:", tempFilePath);
          } else {
            console.error(
              `Alternative URL also failed with status: ${alternativeResponse.status}`
            );
            return NextResponse.json(
              {
                error: `File not found. Status: ${response.status}`,
                details: responseText.substring(0, 200),
              },
              { status: 404 }
            );
          }
        } else {
          return NextResponse.json(
            {
              error: `File not found. Status: ${response.status}`,
              details: responseText.substring(0, 200),
            },
            { status: 404 }
          );
        }
      } else {
        console.log(
          "Successfully fetched Office document, content length:",
          response.headers.get("content-length")
        );
        const fileBuffer = Buffer.from(await response.arrayBuffer());
        console.log("Created buffer with size:", fileBuffer.length);
        await fs.writeFile(tempFilePath, fileBuffer);
        console.log("Wrote file to temp path:", tempFilePath);
      }
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

    // Process the office document
    try {
      console.log("Starting to parse Office document at:", tempFilePath);
      const parsedData = await parseOfficeAsync(tempFilePath);
      console.log(
        "Successfully parsed Office document, content length:",
        parsedData?.length
      );

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

      return NextResponse.json({ parsedText: parsedData });
    } catch (parseError) {
      console.error("Error parsing office document:", parseError);
      // Clean up temp file if parsing fails
      await fs
        .unlink(tempFilePath)
        .catch((err) =>
          console.error("Failed to delete temp file after parse error:", err)
        );

      return NextResponse.json(
        {
          error: `Failed to parse document content: ${
            parseError instanceof Error
              ? parseError.message
              : String(parseError)
          }`,
        },
        { status: 500 }
      );
    }
  } catch (error) {
    console.error("General error processing office document:", error);
    return NextResponse.json(
      {
        error: `Failed to process document content: ${
          error instanceof Error ? error.message : String(error)
        }`,
      },
      { status: 500 }
    );
  }
}
