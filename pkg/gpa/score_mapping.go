package gpa

import (
	_ "embed"
	"encoding/json"
)

// ScoreMapping represents a score-to-GPA mapping entry
type ScoreMapping struct {
	MinValue float64 `json:"minValue"`
	MaxValue float64 `json:"maxValue"`
	Level    string  `json:"displayName"`
	GPA      float64 `json:"gpa"`
}

// ScoreMappingData contains all score mappings
type ScoreMappingData struct {
	Weighted    []ScoreMapping `json:"weighted"`
	NonWeighted []ScoreMapping `json:"non-weighted"`
}

// CourseClassification contains lists of weighted and unweighted courses
type CourseClassification struct {
	Weighted   []string `json:"weighted"`
	Unweighted []string `json:"unweighted"`
}

//go:embed score_mapping.json
var scoreMappingJSON []byte

//go:embed course_classification.json
var courseClassificationJSON []byte

var scoreMappings *ScoreMappingData
var courseClassification *CourseClassification

// init automatically loads embedded data when package is imported
func init() {
	// Load score mappings
	var mappings ScoreMappingData
	if err := json.Unmarshal(scoreMappingJSON, &mappings); err != nil {
		panic("failed to load embedded score_mapping.json: " + err.Error())
	}
	scoreMappings = &mappings

	// Load course classification
	var classification CourseClassification
	if err := json.Unmarshal(courseClassificationJSON, &classification); err != nil {
		panic("failed to load embedded course_classification.json: " + err.Error())
	}
	courseClassification = &classification
}

// GetScoreMappings returns the loaded score mappings
func GetScoreMappings() *ScoreMappingData {
	return scoreMappings
}

// IsWeightedSubject determines if a subject uses weighted GPA
func IsWeightedSubject(subjectName string) bool {
	if courseClassification == nil {
		return false
	}

	// First, check explicit unweighted list (highest priority)
	for _, course := range courseClassification.Unweighted {
		if subjectName == course {
			return false
		}
	}

	// Second, check explicit weighted list
	for _, course := range courseClassification.Weighted {
		if subjectName == course {
			return true
		}
	}

	// Fallback to keyword matching
	// A Level courses
	if contains(subjectName, "A Level") {
		return true
	}

	// AS courses (but not "AS" alone, must be "AS " with space)
	if contains(subjectName, "AS ") {
		return true
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
