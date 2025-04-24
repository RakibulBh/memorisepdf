import { NextRequest, NextResponse } from "next/server";
import { promises as fs } from "fs";
import { v4 as uuidv4 } from "uuid";
import PDFParser from "pdf2json";

const MAX_FILE_SIZE = parseInt(
  process.env.NEXT_PUBLIC_MAX_FILE_SIZE || "104857600",
  10
); // 100MB

// Configure body parser to accept larger files
export const config = {
  api: {
    bodyParser: false,
  },
};

export async function POST(req: NextRequest) {
  const formData = await req.formData();
  const file = formData.get("file");

  if (!file || typeof file === "string") {
    return NextResponse.json({ error: "No file provided" }, { status: 400 });
  }

  // Check if file is a PDF by type or extension
  const fileName = file instanceof File ? file.name.toLowerCase() : "";
  if (file.type !== "application/pdf" && !fileName.endsWith(".pdf")) {
    return NextResponse.json(
      {
        error:
          "Only PDF files are allowed in this endpoint. Use parse-office for Office documents.",
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
  const tempFilePath = `/tmp/${uniqueFileName}.pdf`;

  const fileBuffer = Buffer.from(await file.arrayBuffer());
  await fs.writeFile(tempFilePath, fileBuffer);

  try {
    const pdfParser = new PDFParser(null, true);

    const pdfData = await new Promise((resolve, reject) => {
      pdfParser.on("pdfParser_dataError", (error) => {
        console.error("PDF parsing error:", error);
        reject(new Error("Failed to parse PDF content"));
      });

      pdfParser.on("pdfParser_dataReady", () => {
        resolve(pdfParser.getRawTextContent());
      });

      pdfParser.loadPDF(tempFilePath);
    });

    // Clean up the temporary file
    await fs
      .unlink(tempFilePath)
      .catch((err) => console.error("Failed to delete temp file:", err));

    return NextResponse.json({ parsedText: pdfData });
  } catch (error) {
    console.error("Error parsing PDF:", error);

    // Clean up the temporary file even if parsing fails
    await fs
      .unlink(tempFilePath)
      .catch((err) => console.error("Failed to delete temp file:", err));

    return NextResponse.json(
      { error: "Failed to parse PDF content" },
      { status: 500 }
    );
  }
}
