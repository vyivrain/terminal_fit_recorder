package prompt

import "strings"

// TrainerGuidanceSection renders the AI trainer's latest weekly review as a
// labeled block for workout generation prompts. It is shared by every
// generation path so they all read the guidance the same way. Empty when there
// is no guidance yet, so prompts stay unchanged until the first review.
func TrainerGuidanceSection(guidance string) string {
	guidance = strings.TrimSpace(guidance)
	if guidance == "" {
		return ""
	}
	return "TRAINER_GUIDANCE\n" + guidance + "\nEND_TRAINER_GUIDANCE\n" +
		"TRAINER_GUIDANCE is the person's trainer's latest weekly review of their training. Follow it when choosing exercises, loads, and focus, unless it conflicts with PROFILE or the safety rules.\n\n"
}
