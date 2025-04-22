interface PresentationResponse {
  flashcards: {
    term: string;
    definition: string;
  }[];
  quizzes: Quiz[];
}

interface Quiz {
  question: string;
  answers: {
    text: string;
    correct: boolean;
    explanation: string;
  }[];
}

interface Flashcard {
  term: string;
  definition: string;
}

const parsePresentation = async (presentationText: string) => {
  console.log("Making request to parse presentation", presentationText);
  const API_URL = process.env.NEXT_PUBLIC_API_URL;

  console.log("API URL", API_URL);
  console.log("Presentation text", presentationText);

  const response = await fetch(`${API_URL}/parse-presentation`, {
    method: "POST",
    body: JSON.stringify({ presentation_text: presentationText }),
  });

  const data = await response.json();

  const parsedData: PresentationResponse = data.data;

  return parsedData;
};

export default parsePresentation;
