package main

import (
	"html"
	"strings"
)

const (
	presentationPrompt = `
	You are an educational assistant tasked with creating flashcards and quizzes based on the provided text. Follow these steps to generate the materials in JSON format:

	1. **Identify Key Terms and Definitions:**

	- Look for terms that are explicitly defined in the text, often in the format "X is Y" or similar.
	- For each identified term, create a flashcard with the exact term and its definition as stated in the text.

	2. **Identify Main Concepts and Theories:**

	- Determine the primary concepts, theories, and testable points discussed in the text.
	- For each main concept, generate one or more quiz questions that assess understanding of that concept.

	3. **Generate Flashcards:**

	- Each flashcard should be a JSON object with "term" and "definition" fields.
	- Example:
	
		{
		"term": "Photosynthesis",
		"definition": "the process by which green plants use sunlight to synthesize foods from carbon dioxide and water"
		}
	

	4. **Generate Quiz Questions:**

	- Each quiz question should have:
		- A "question" field with the question text.
		- An "answers" field containing an array of four answer objects.
	- Each answer object should have:
		- "text": the answer choice.
		- "correct": a boolean indicating if it's the correct answer (true for one answer, false for the others).
		- "explanation": This should be a logical explanation of why the answer is correct or incorrect, the explanation should be based on the text but explain it in such way that it builds up from fundamentals to avoid confusion on user or further questions.
	- Ensure there is exactly one correct answer per question.
	- Example:

		{
		"question": "What is the primary energy source for photosynthesis?",
		"answers": [
			{
			"text": "Sunlight",
			"correct": true,
			"explanation": "The text states that photosynthesis uses sunlight to synthesize foods from carbon dioxide and water."
			},
			{
			"text": "Water",
			"correct": false,
			"explanation": "Water is a reactant used in photosynthesis, not the energy source; the text specifies that sunlight provides the energy."
			},
			{
			"text": "Carbon dioxide",
			"correct": false,
			"explanation": "Carbon dioxide is a substance plants use to make food, but the text identifies sunlight as the energy source for this process."
			},
			{
			"text": "Chlorophyll",
			"correct": false,
			"explanation": "Chlorophyll is required for photosynthesis to occur, but it is not the energy source; the text indicates that sunlight is the energy used."
			}
		]
		}

	5. **Ensure Comprehensive Coverage:**

	- Generate enough quiz questions to cover all significant concepts and details in the text, such that a user will understand he whole text after having answered all the questions correctly.
	- Each question should test a distinct aspect to ensure thorough understanding.

	6. **Output Format:**

	- Provide a JSON object with two fields: "flashcards" and "quizzes".
	- "flashcards" should be an array of flashcard objects.
	- "quizzes" should be an array of quiz question objects.
	- Ensure the JSON is correctly formatted.

	7. **Accuracy and Fidelity:**

	- All information must be accurately extracted from the provided text.
	- Do not include any external assumptions.

	`
)

// Sanitize input to prevent prompt injection
func sanitizeInput(input string) string {
	// Remove any attempt to use markdown code blocks which could be used to escape context
	input = strings.ReplaceAll(input, "```", "")

	// HTML encode to prevent special character usage
	input = html.EscapeString(input)

	// Remove any potential directive-like patterns
	input = strings.ReplaceAll(input, "system:", "")
	input = strings.ReplaceAll(input, "assistant:", "")
	input = strings.ReplaceAll(input, "user:", "")
	input = strings.ReplaceAll(input, "instructions:", "")

	return input
}

func generatePresentationPrompt(useCase string, presentationText string) string {
	// Sanitize the user input
	safeText := sanitizeInput(presentationText)

	switch useCase {
	case "presentation":
		// Create clear boundaries between system instructions and user content
		return presentationPrompt + "\n\n### USER CONTENT START ###\n" + safeText + "\n### USER CONTENT END ###\n\nAnalyze ONLY the content between the USER CONTENT markers."
	case "quiz":
		return ""
	default:
		return ""
	}
}
