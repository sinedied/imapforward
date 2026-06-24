package main

import "strings"

func normalizeTargetLabels(primaryFolder string, targetLabels []string) []string {
	seen := make(map[string]bool)
	var labels []string
	for _, label := range targetLabels {
		label = strings.TrimSpace(label)
		if label == "" || label == primaryFolder || seen[label] {
			continue
		}
		seen[label] = true
		labels = append(labels, label)
	}
	return labels
}

func labelParents(label string) []string {
	parts := strings.Split(label, "/")
	if len(parts) < 2 {
		return nil
	}

	parents := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		if parent != "" {
			parents = append(parents, parent)
		}
	}
	return parents
}

func orderedLabelsForCreation(labels []string) []string {
	seen := make(map[string]bool)
	var ordered []string
	for _, label := range labels {
		for _, parent := range labelParents(label) {
			if !seen[parent] {
				seen[parent] = true
				ordered = append(ordered, parent)
			}
		}
		if !seen[label] {
			seen[label] = true
			ordered = append(ordered, label)
		}
	}
	return ordered
}
