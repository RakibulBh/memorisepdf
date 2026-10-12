package llm

import "google.golang.org/genai"

// The settings a quiz can be asked for.
var Difficulties = []string{"easy", "medium", "hard"}

// The settings a teaching card can be asked for.
var TeachingStyles = []string{"simple-language", "analogy-driven", "scaffolded-learning"}

// Says whether a requested difficulty is one this app offers.
func ValidDifficulty(s string) bool {
	return contains(Difficulties, s)
}

// Says whether a requested teaching style is one this app offers.
func ValidTeachingStyle(s string) bool {
	return contains(TeachingStyles, s)
}

// Says whether a value appears in a list.
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// QuizResponse is the exact shape the model is required to answer with.
type QuizResponse struct {
	Questions []struct {
		Question string `json:"question"`
		Answers  []struct {
			Text        string `json:"text"`
			Correct     bool   `json:"correct"`
			Explanation string `json:"explanation"`
		} `json:"answers"`
	} `json:"questions"`
}

type FlashcardResponse struct {
	Flashcards []struct {
		Term       string `json:"term"`
		Definition string `json:"definition"`
	} `json:"flashcards"`
}

type TeachingResponse struct {
	TeachingCards []struct {
		Subtopic string `json:"subtopic"`
		Teaching string `json:"teaching"`
	} `json:"teaching_cards"`
}

// Describes the question format to the model, so the reply can be read without
// guessing and a stray sentence can never break parsing.
func quizSchema() *genai.Schema {
	return &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"questions"},
		Properties: map[string]*genai.Schema{
			"questions": {
				Type:     genai.TypeArray,
				MinItems: ptr(int64(1)),
				Items: &genai.Schema{
					Type:             genai.TypeObject,
					Required:         []string{"question", "answers"},
					PropertyOrdering: []string{"question", "answers"},
					Properties: map[string]*genai.Schema{
						"question": {
							Type:        genai.TypeString,
							Description: "A single self-contained question about the lecture content.",
						},
						"answers": {
							Type:        genai.TypeArray,
							MinItems:    ptr(int64(4)),
							MaxItems:    ptr(int64(4)),
							Description: "Exactly four options, of which exactly one is correct.",
							Items: &genai.Schema{
								Type:             genai.TypeObject,
								Required:         []string{"text", "correct", "explanation"},
								PropertyOrdering: []string{"text", "correct", "explanation"},
								Properties: map[string]*genai.Schema{
									"text":    {Type: genai.TypeString},
									"correct": {Type: genai.TypeBoolean},
									"explanation": {
										Type:        genai.TypeString,
										Description: "Why this option is right or wrong, in one or two sentences.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Describes the flashcard format to the model.
func flashcardSchema() *genai.Schema {
	return &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"flashcards"},
		Properties: map[string]*genai.Schema{
			"flashcards": {
				Type:     genai.TypeArray,
				MinItems: ptr(int64(1)),
				Items: &genai.Schema{
					Type:             genai.TypeObject,
					Required:         []string{"term", "definition"},
					PropertyOrdering: []string{"term", "definition"},
					Properties: map[string]*genai.Schema{
						"term": {
							Type:        genai.TypeString,
							Description: "The concept, term or name being tested.",
						},
						"definition": {
							Type:        genai.TypeString,
							Description: "A concise, self-contained explanation of the term.",
						},
					},
				},
			},
		},
	}
}

// Describes the teaching card format to the model.
func teachingSchema() *genai.Schema {
	return &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"teaching_cards"},
		Properties: map[string]*genai.Schema{
			"teaching_cards": {
				Type:     genai.TypeArray,
				MinItems: ptr(int64(1)),
				Items: &genai.Schema{
					Type:             genai.TypeObject,
					Required:         []string{"subtopic", "teaching"},
					PropertyOrdering: []string{"subtopic", "teaching"},
					Properties: map[string]*genai.Schema{
						"subtopic": {
							Type:        genai.TypeString,
							Description: "The specific idea this card teaches.",
						},
						"teaching": {
							Type:        genai.TypeString,
							Description: "An explanation of the subtopic in the requested style.",
						},
					},
				},
			},
		},
	}
}

// Gives back a pointer to a value, which the schema fields require.
func ptr[T any](v T) *T {
	return &v
}
