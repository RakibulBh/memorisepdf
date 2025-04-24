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

export interface TeachingCard {
  subtopic: string;
  teaching: string;
}

interface ResultState {
  flashcards: Flashcard[];
  quizzes: Quiz[];
  teachingCards: TeachingCard[];
  contentType: "test" | "teach" | null;
  setResults: (data: { flashcards: Flashcard[]; quizzes: Quiz[] }) => void;
  setTeachingCards: (cards: TeachingCard[]) => void;
  clearResults: () => void;
  hasResults: () => boolean;
  hasTeachingContent: () => boolean;
}

export const useResultStore = create<ResultState>((set, get) => ({
  flashcards: [],
  quizzes: [],
  teachingCards: [],
  contentType: null,
  setResults: (data) =>
    set({
      flashcards: data.flashcards,
      quizzes: data.quizzes,
      contentType: "test",
    }),
  setTeachingCards: (cards) =>
    set({
      teachingCards: cards,
      contentType: "teach",
    }),
  clearResults: () =>
    set({
      flashcards: [],
      quizzes: [],
      teachingCards: [],
      contentType: null,
    }),
  hasResults: () => get().flashcards.length > 0 || get().quizzes.length > 0,
  hasTeachingContent: () => get().teachingCards.length > 0,
}));
