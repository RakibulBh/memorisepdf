package main

type PresentationResponse struct {
	Flashcards []Flashcard `json:"flashcards"`
	Quizzes    []Quiz      `json:"quizzes"`
}

type Flashcard struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

type Quiz struct {
	Question string   `json:"question"`
	Answers  []Answer `json:"answers"`
}

type Answer struct {
	Text        string `json:"text"`
	Correct     bool   `json:"correct"`
	Explanation string `json:"explanation"`
}

var difficultyLevels = []string{"easy", "medium", "hard"}
var teachingStyles = []string{"simple-language", "analogy-driven", "scaffolded-learning"}

type TeachingCard struct {
	Subtopic string `json:"subtopic"`
	Teaching string `json:"teaching"`
}

type TeachingCardsResponse struct {
	TeachingCards []TeachingCard `json:"teaching_cards"`
}
