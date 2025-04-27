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

export const initiateProcessing = async (formData: FormData) => {
  const URL = `${process.env.NEXT_PUBLIC_API_URL}/initiate`;

  // validate data in the client side
  if (!formData.get("file")) {
    return { error: "File is required" };
  }
  if (!formData.get("service_type")) {
    return { error: "Service type is required" };
  }
  if (
    formData.get("service_type") !== "testme" &&
    formData.get("service_type") !== "teachme"
  ) {
    return { error: "Invalid service type" };
  }
  if (formData.get("service_type") === "teachme") {
    if (!formData.get("teaching_style")) {
      return { error: "Teaching style is required" };
    }
  }
  if (formData.get("service_type") === "testme") {
    if (!formData.get("difficulty")) {
      return { error: "Difficulty is required" };
    }
  }

  const response = await fetch(URL, {
    method: "POST",
    body: formData,
  });

  if (!response.ok) {
    return { error: (await response.json()).error };
  }

  const data = await response.json();
  console.log(data);

  return data;
};

export default initiateProcessing;
