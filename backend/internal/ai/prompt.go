package ai

import (
	"fmt"
	"strings"
)

// PromptPayload holds the three-layer prompt for AI planning.
type PromptPayload struct {
	System  string `json:"system"`
	Context string `json:"context"`
	User    string `json:"user"`
}

// BuildPrompt assembles a three-layer prompt from trip context and user constraints.
func BuildPrompt(tripName, destination, startDate, endDate string,
	travelersCount int, preferences map[string]string, constraints map[string]any) PromptPayload {

	system := `You are a travel planning assistant. Generate a structured JSON itinerary.
Rules:
- Output MUST be valid JSON with keys: title, summary, days[]
- Each day has: date, theme, items[]
- Each item has: title, itemType (place_visit|meal|transit|hotel|free_time|custom), confidence (high|medium|low)
- Respect the user's budget, pace, and preferences
- Do NOT include any text outside the JSON structure`

	contextParts := []string{
		fmt.Sprintf("Trip: %s", tripName),
		fmt.Sprintf("Destination: %s", destination),
		fmt.Sprintf("Dates: %s to %s", startDate, endDate),
		fmt.Sprintf("Travelers: %d", travelersCount),
	}
	for k, v := range preferences {
		contextParts = append(contextParts, fmt.Sprintf("%s: %s", k, v))
	}

	userParts := []string{"Please generate an itinerary with the following constraints:"}
	for k, v := range constraints {
		userParts = append(userParts, fmt.Sprintf("- %s: %v", k, v))
	}

	return PromptPayload{
		System:  system,
		Context: strings.Join(contextParts, "\n"),
		User:    strings.Join(userParts, "\n"),
	}
}

// FullPrompt returns the assembled prompt as a single string for the provider.
func (p PromptPayload) FullPrompt() string {
	return fmt.Sprintf("[SYSTEM]\n%s\n\n[CONTEXT]\n%s\n\n[USER]\n%s", p.System, p.Context, p.User)
}
