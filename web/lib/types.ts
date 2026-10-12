export type Kind = "quiz" | "flashcard" | "teaching";

export type Difficulty = "easy" | "medium" | "hard";

export type TeachingStyle =
  | "simple-language"
  | "analogy-driven"
  | "scaffolded-learning";

export interface User {
  id: string;
  email: string;
  name: string;
  pictureUrl: string;
  createdAt: string;
  lastLoginAt: string;
}

export interface Subject {
  id: string;
  name: string;
  colour: string;
  createdAt: string;
  updatedAt: string;
  moduleCount: number;
}

export interface Module {
  id: string;
  subjectId: string;
  subjectName?: string;
  name: string;
  createdAt: string;
  updatedAt: string;
  fileCount: number;
  totalBytes: number;
}

export interface LectureFile {
  id: string;
  moduleId: string;
  originalName: string;
  sha256: string;
  byteSize: number;
  contentType: string;
  uploadedAt: string;
  /** Which kinds this file has already produced. Always present, possibly empty. */
  analysed: Kind[];
}

export interface QuizAnswer {
  id: string;
  text: string;
  correct: boolean;
  explanation: string;
}

export interface QuizQuestion {
  id: string;
  fileId: string;
  fileName?: string;
  question: string;
  answers: QuizAnswer[];
}

export interface Flashcard {
  id: string;
  fileId: string;
  fileName?: string;
  term: string;
  definition: string;
}

export interface TeachingCard {
  id: string;
  fileId: string;
  fileName?: string;
  subtopic: string;
  teaching: string;
}

export interface QuizPool {
  difficulty: Difficulty;
  questions: QuizQuestion[];
  filesPending: number;
}

export interface FlashcardPool {
  flashcards: Flashcard[];
  filesPending: number;
}

export interface TeachingPool {
  style: TeachingStyle;
  teachingCards: TeachingCard[];
  filesPending: number;
}

export interface UploadOutcome {
  name: string;
  status: "stored" | "skipped" | "rejected";
  reason?: string;
  file?: LectureFile;
}

export interface UploadResult {
  stored: number;
  results: UploadOutcome[];
  storageUsed: number;
  storageQuota: number;
}

export interface Usage {
  storageUsed: number;
  storageQuota: number;
  maxFileSize: number;
}

export interface Meta {
  difficulties: Difficulty[];
  teachingStyles: TeachingStyle[];
  supportedExtensions: string[];
  maxFileSize: number;
  maxFilesPerModule: number;
  storageQuota: number;
}

/** One update from the generation stream. */
export interface Progress {
  stage: "started" | "analysing" | "file-done" | "finished";
  message: string;
  fileName?: string;
  done: number;
  total: number;
  reused: number;
  generated: number;
  failed: number;
}

/** What a finished generation run did. */
export interface GenerateResult {
  reused: number;
  generated: number;
  failed: number;
  errors?: string[];
}
