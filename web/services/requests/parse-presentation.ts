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

interface TeachingCard {
  subtopic: string;
  teaching: string;
}

interface ProcessingParams {
  text: string;
  serviceType: "testme" | "teachme";
  difficulty?: string;
  teachingStyle?: string;
}

export const initiateProcessing = async (params: ProcessingParams) => {
  const URL = `${process.env.NEXT_PUBLIC_API_URL}/initiate`;

  let requestBody = {};

  if (params.serviceType === "testme") {
    requestBody = {
      service_type: params.serviceType,
      presentation_text: params.text,
      difficulty: params.difficulty || "easy",
    };
  } else {
    requestBody = {
      service_type: params.serviceType,
      presentation_text: params.text,
      teaching_style: params.teachingStyle || "simple-language",
    };
  }

  const response = await fetch(URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(requestBody),
  });

  const data = await response.json();

  return data;
};

export default initiateProcessing;
