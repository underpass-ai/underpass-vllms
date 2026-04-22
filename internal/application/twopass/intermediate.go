package twopassapp

import "strings"

func sanitizeIntermediate(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	if extracted := extractFromFinalIRMarker(candidate); extracted != "" {
		candidate = extracted
	}
	if extracted := extractLastCodeFence(candidate); extracted != "" {
		candidate = extracted
	}

	return strings.TrimSpace(candidate)
}

func extractFromFinalIRMarker(raw string) string {
	markers := []string{
		"**Final Intermediate Representation:**",
		"Final Intermediate Representation:",
		"Final IR:",
	}
	for _, marker := range markers {
		index := strings.LastIndex(raw, marker)
		if index == -1 {
			continue
		}
		return strings.TrimSpace(raw[index+len(marker):])
	}
	return ""
}

func extractLastCodeFence(raw string) string {
	const fence = "```"
	end := strings.LastIndex(raw, fence)
	if end == -1 {
		return ""
	}
	start := strings.LastIndex(raw[:end], fence)
	if start == -1 {
		return ""
	}

	block := strings.TrimSpace(raw[start+len(fence) : end])
	if block == "" {
		return ""
	}

	if newline := strings.IndexByte(block, '\n'); newline != -1 {
		header := strings.TrimSpace(block[:newline])
		if isFenceLanguage(header) {
			block = strings.TrimSpace(block[newline+1:])
		}
	}

	return strings.TrimSpace(block)
}

func isFenceLanguage(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
