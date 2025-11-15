package gpa

import (
	"encoding/json"
	"os"
)

// ScoreMapping represents a score-to-GPA mapping entry
type ScoreMapping struct {
	MinValue float64 `json:"min_value"`
	MaxValue float64 `json:"max_value"`
	Level    string  `json:"level"`
	GPA      float64 `json:"gpa"`
}

// ScoreMappingData contains all score mappings
type ScoreMappingData struct {
	Weighted    []ScoreMapping `json:"weighted"`
	NonWeighted []ScoreMapping `json:"non-weighted"`
}

var scoreMappings *ScoreMappingData

// LoadScoreMappings loads the score mapping from a JSON file
func LoadScoreMappings(filepath string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	var mappings ScoreMappingData
	if err := json.Unmarshal(data, &mappings); err != nil {
		return err
	}

	scoreMappings = &mappings
	return nil
}

// GetScoreMappings returns the loaded score mappings
func GetScoreMappings() *ScoreMappingData {
	return scoreMappings
}

// IsWeightedSubject determines if a subject uses weighted GPA
func IsWeightedSubject(subjectName string) bool {
	// AP courses
	if contains(subjectName, "AP") {
		return true
	}

	// A Level courses
	if contains(subjectName, "A Level") {
		return true
	}

	// AS courses
	if contains(subjectName, "AS") {
		return true
	}

	// Specific advanced courses
	extraWeightedSubjects := []string{
		"Linear Algebra",
		"Modern Physics and Optics",
		"Multivariable Calculus",
	}

	for _, subject := range extraWeightedSubjects {
		if subjectName == subject {
			return true
		}
	}

	return false
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr ||
		(len(str) > len(substr) &&
			(hasPrefix(str, substr) || hasSuffix(str, substr) || hasInfix(str, substr))))
}

func hasPrefix(str, prefix string) bool {
	return len(str) >= len(prefix) && str[:len(prefix)] == prefix
}

func hasSuffix(str, suffix string) bool {
	return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
}

func hasInfix(str, infix string) bool {
	for i := 0; i <= len(str)-len(infix); i++ {
		if str[i:i+len(infix)] == infix {
			return true
		}
	}
	return false
}
