package gpa

import (
	"math"
	"myxb/internal/models"
)

// Subject represents a subject with calculated scores and GPA
type Subject struct {
	ID                uint64
	Name              string
	ClassID           uint64
	Score             float64  // Calculated score (0-100)
	OfficialScore     *float64 // Official score from API
	ExtraCredit       float64  // Extra credit points
	GPA               float64
	UnweightedGPA     float64
	MaxGPA            float64
	UnweightedMaxGPA  float64
	Weight            float64 // 1.0 for regular, 0.5 for elective
	IsWeighted        bool    // Whether this is a weighted course
	IsElective        bool
	IsInGrade         bool // Whether this subject counts toward GPA
	EvaluationDetails []models.EvaluationProject
}

// CalculatedGPA represents the final GPA result
type CalculatedGPA struct {
	WeightedGPA      float64
	MaxGPA           float64
	UnweightedGPA    float64
	UnweightedMaxGPA float64
	Subjects         []Subject
}

// AdjustProportions adjusts evaluation project proportions to sum to 100%
func AdjustProportions(projects []models.EvaluationProject) {
	// Calculate total proportion of valid (non-null score) projects
	totalProportion := 0.0
	for _, p := range projects {
		if !p.ScoreIsNull {
			totalProportion += p.Proportion
		}
	}

	if totalProportion == 0 {
		return
	}

	// Adjust proportions
	for i := range projects {
		if !projects[i].ScoreIsNull {
			projects[i].Proportion = projects[i].Proportion / totalProportion * 100.0
		}
	}

	// Recursively adjust nested projects
	for i := range projects {
		if len(projects[i].EvaluationProjectList) > 0 {
			AdjustNestedProportions(projects[i].EvaluationProjectList, projects[i].Proportion)
		}
	}
}

// AdjustNestedProportions adjusts nested evaluation projects
func AdjustNestedProportions(projects []models.EvaluationProject, parentProportion float64) {
	totalProportion := 0.0
	for _, p := range projects {
		if !p.ScoreIsNull {
			totalProportion += p.Proportion
		}
	}

	if totalProportion == 0 {
		return
	}

	for i := range projects {
		if !projects[i].ScoreIsNull {
			projects[i].Proportion = projects[i].Proportion / totalProportion * parentProportion
		}

		// Recursively adjust deeper nested projects
		if len(projects[i].EvaluationProjectList) > 0 {
			AdjustNestedProportions(projects[i].EvaluationProjectList, projects[i].Proportion)
		}
	}
}

// CalculateSubjectScore calculates the total score for a subject
func CalculateSubjectScore(projects []models.EvaluationProject) float64 {
	totalScore := 0.0

	for _, project := range projects {
		if !project.ScoreIsNull {
			// If has nested projects, calculate from them
			if len(project.EvaluationProjectList) > 0 {
				totalScore += calculateNestedScore(project.EvaluationProjectList)
			} else {
				totalScore += project.Score * project.Proportion / 100.0
			}
		}
	}

	return totalScore
}

func calculateNestedScore(projects []models.EvaluationProject) float64 {
	totalScore := 0.0

	for _, project := range projects {
		if !project.ScoreIsNull {
			if len(project.EvaluationProjectList) > 0 {
				totalScore += calculateNestedScore(project.EvaluationProjectList)
			} else {
				totalScore += project.Score * project.Proportion / 100.0
			}
		}
	}

	return totalScore
}

// ScoreToGPA converts a score to GPA using the mapping table
func ScoreToGPA(score float64, isWeighted bool) float64 {
	if scoreMappings == nil {
		return math.NaN()
	}

	// Round to 1 decimal place
	roundedScore := math.Round(score*10) / 10

	mappingList := scoreMappings.NonWeighted
	if isWeighted {
		mappingList = scoreMappings.Weighted
	}

	for _, mapping := range mappingList {
		if roundedScore >= mapping.MinValue && roundedScore <= mapping.MaxValue {
			return mapping.GPA
		}
	}

	return math.NaN()
}

// GetMaxGPA returns the maximum possible GPA for a course type
func GetMaxGPA(isWeighted bool) float64 {
	if scoreMappings == nil {
		return math.NaN()
	}

	if isWeighted {
		return scoreMappings.Weighted[0].GPA // First entry is highest
	}
	return scoreMappings.NonWeighted[0].GPA
}

// CalculateGPA calculates the weighted and unweighted GPA
func CalculateGPA(subjects []Subject) CalculatedGPA {
	totalWeight := 0.0
	totalWeightedGPA := 0.0
	totalUnweightedGPA := 0.0
	totalMaxGPA := 0.0

	validSubjects := []Subject{}

	for _, subject := range subjects {
		// Skip subjects that don't count toward GPA or have no GPA
		if !subject.IsInGrade || math.IsNaN(subject.GPA) {
			continue
		}

		validSubjects = append(validSubjects, subject)
		totalWeight += subject.Weight
		totalWeightedGPA += subject.GPA * subject.Weight
		totalUnweightedGPA += subject.UnweightedGPA * subject.Weight
		totalMaxGPA += subject.MaxGPA * subject.Weight
	}

	result := CalculatedGPA{
		Subjects: validSubjects,
	}

	if totalWeight > 0 {
		result.WeightedGPA = totalWeightedGPA / totalWeight
		result.UnweightedGPA = totalUnweightedGPA / totalWeight
		result.MaxGPA = totalMaxGPA / totalWeight
		result.UnweightedMaxGPA = GetMaxGPA(false)
	} else {
		result.WeightedGPA = math.NaN()
		result.UnweightedGPA = math.NaN()
		result.MaxGPA = math.NaN()
		result.UnweightedMaxGPA = math.NaN()
	}

	return result
}

// ProcessSubject processes a single subject and calculates its GPA
func ProcessSubject(detail *models.SubjectDetail, dynamicScore *models.DynamicScoreData,
	dynamicInfo *models.SubjectDynamicScore, isElective bool) Subject {

	subject := Subject{
		ID:        detail.SubjectID,
		Name:      detail.SubjectName,
		ClassID:   detail.ClassID,
		Weight:    1.0,
		IsElective: isElective,
		IsInGrade: true,
		EvaluationDetails: dynamicScore.EvaluationProjectList,
	}

	// Set elective weight
	if isElective || detail.SubjectName == "C-Humanities" {
		subject.Weight = 0.5
		subject.IsElective = true
	}

	// Determine if weighted subject
	subject.IsWeighted = IsWeightedSubject(detail.SubjectName)

	// Adjust proportions
	AdjustProportions(dynamicScore.EvaluationProjectList)

	// Calculate score
	subject.Score = CalculateSubjectScore(dynamicScore.EvaluationProjectList)

	// Use official score if available
	if dynamicInfo != nil {
		subject.IsInGrade = dynamicInfo.IsInGrade

		if dynamicInfo.SubjectScore != nil && dynamicInfo.SubjectTotalScore > 0 {
			officialScore := *dynamicInfo.SubjectScore / dynamicInfo.SubjectTotalScore * 100.0
			subject.OfficialScore = &officialScore

			// Calculate extra credit
			roundedCalculated := math.Round(subject.Score*10) / 10
			subject.ExtraCredit = officialScore - roundedCalculated

			// Use official score
			subject.Score = officialScore
		}
	}

	// Round score to 1 decimal
	subject.Score = math.Round(subject.Score*10) / 10

	// Calculate GPA
	subject.GPA = ScoreToGPA(subject.Score, subject.IsWeighted)
	subject.UnweightedGPA = ScoreToGPA(subject.Score, false)
	subject.MaxGPA = GetMaxGPA(subject.IsWeighted)
	subject.UnweightedMaxGPA = GetMaxGPA(false)

	return subject
}
