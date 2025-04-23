import { create } from "zustand";

export interface Flashcard {
  term: string;
  definition: string;
}

export interface QuizAnswer {
  text: string;
  correct: boolean;
  explanation: string;
}

export interface Quiz {
  question: string;
  answers: QuizAnswer[];
}

interface ResultState {
  flashcards: Flashcard[];
  quizzes: Quiz[];
  setResults: (data: { flashcards: Flashcard[]; quizzes: Quiz[] }) => void;
  clearResults: () => void;
  hasResults: () => boolean;
}

export const useResultStore = create<ResultState>((set, get) => ({
  flashcards: [],
  quizzes: [],
  setResults: (data) =>
    set({ flashcards: data.flashcards, quizzes: data.quizzes }),
  clearResults: () => set({ flashcards: [], quizzes: [] }),
  hasResults: () => get().flashcards.length > 0 || get().quizzes.length > 0,
}));
