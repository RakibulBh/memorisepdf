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

export const initiateProcessing = async (
  text: string,
  difficulty: string = "easy"
) => {
  const URL = `${process.env.NEXT_PUBLIC_API_URL}/initiate`;
  const response = await fetch(URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      presentation_text: text,
      difficulty: difficulty,
    }),
  });

  const data = await response.json();

  return data;
};

export default initiateProcessing;
