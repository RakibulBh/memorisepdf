import { parseOfficeAsync } from "officeparser";

import { NextRequest, NextResponse } from "next/server";
import { promises as fs } from "fs";
import { v4 as uuidv4 } from "uuid";

const MAX_FILE_SIZE = parseInt(
  process.env.NEXT_PUBLIC_MAX_FILE_SIZE || "104857600",
  10
); // 100MB

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
  const formData = await req.formData();
  const file = formData.get("file");

  if (!file || typeof file === "string") {
    return NextResponse.json({ error: "No file provided" }, { status: 400 });
  }

  if (
    file.type === "application/pdf" ||
    (file instanceof File && file.name.toLowerCase().endsWith(".pdf"))
  ) {
    return NextResponse.json(
      {
        error:
          "PDF files are not supported by this endpoint. Use the parse-pdf endpoint instead.",
      },
      { status: 400 }
    );
  }

  // Check if file is a supported office format
  const fileName = file instanceof File ? file.name.toLowerCase() : "";
  const isSupported = SUPPORTED_FORMATS.some((ext) => fileName.endsWith(ext));

  if (!isSupported) {
    return NextResponse.json(
      {
        error:
          "Unsupported file format. Please upload a valid Office document.",
      },
      { status: 400 }
    );
  }

  if (file instanceof File && file.size > MAX_FILE_SIZE) {
    return NextResponse.json(
      { error: "File size exceeds limit" },
      { status: 400 }
    );
  }

  const uniqueFileName = uuidv4();
  const fileExtension = fileName.substring(fileName.lastIndexOf("."));
  const tempFilePath = `/tmp/${uniqueFileName}${fileExtension}`;

  const fileBuffer = Buffer.from(await file.arrayBuffer());
  await fs.writeFile(tempFilePath, fileBuffer);

  try {
    const parsedData = await parseOfficeAsync(tempFilePath);

    // Clean up the temporary file
    await fs
      .unlink(tempFilePath)
      .catch((err) => console.error("Failed to delete temp file:", err));

    return NextResponse.json({ parsedText: parsedData });
  } catch (error) {
    console.error("Error parsing office document:", error);

    // Clean up the temporary file even if parsing fails
    await fs
      .unlink(tempFilePath)
      .catch((err) => console.error("Failed to delete temp file:", err));

    return NextResponse.json(
      { error: "Failed to parse document content" },
      { status: 500 }
    );
  }
}
