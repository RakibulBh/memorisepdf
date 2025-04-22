const parsePresentation = async (presentationText: string) => {
  console.log("Making request to parse presentation", presentationText);
  const API_URL = process.env.NEXT_PUBLIC_API_URL;

  console.log("API URL", API_URL);
  console.log("Presentation text", presentationText);

  const response = await fetch(`${API_URL}/parse-presentation`, {
    method: "POST",
    body: JSON.stringify({ presentation_text: presentationText }),
  });

  return response.json();
};

export default parsePresentation;
