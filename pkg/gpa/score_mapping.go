package gpa

import (
	_ "embed"
	"encoding/json"
	"strings"
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
	if strings.Contains(subjectName, "A Level") {
		return true
	}

	// AS courses (but not "AS" alone, must be "AS " with space)
	if strings.Contains(subjectName, "AS ") {
		return true
	}

	// AP courses
	if strings.Contains(subjectName, "AP") {
		return true
	}

	return false
}

// GetScoreLevelFromScore returns the score level (A+, A, B, etc.) for a given score
func GetScoreLevelFromScore(score float64, isWeighted bool) string {
	if scoreMappings == nil {
		return ""
	}

	mappingList := scoreMappings.NonWeighted
	if isWeighted {
		mappingList = scoreMappings.Weighted
	}

	for _, mapping := range mappingList {
		if score >= mapping.MinValue && score <= mapping.MaxValue {
			return mapping.Level
		}
	}

	return ""
}
