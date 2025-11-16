package main

import (
	"fmt"
	"math"
	"myxb/internal/api"
	"myxb/internal/models"
	"myxb/pkg/gpa"
	"os"
	"strings"
)

func calculateGPA(apiClient *api.API) {
	// Get semesters
	printInfo("Fetching semesters...")
	semesters, err := apiClient.GetSemesters()
	if err != nil {
		printError(fmt.Sprintf("Failed to get semesters: %v", err))
		os.Exit(1)
	}

	fmt.Println()

	// Display semester list (reversed, latest at bottom, max 10)
	var currentIndex int = -1
	displayCount := len(semesters)
	if displayCount > 10 {
		displayCount = 10
	}

	// Display in reverse order
	for i := displayCount - 1; i >= 0; i-- {
		prefix := fmt.Sprintf("[%d]", i)
		suffix := ""
		if semesters[i].IsNow {
			suffix = cyan(" (current)")
			currentIndex = i
		}
		fmt.Printf("%s %d-%d Semester %d%s\n",
			gray(prefix),
			semesters[i].Year,
			semesters[i].Year+1,
			semesters[i].Semester,
			suffix)
	}

	fmt.Println()

	// Prompt user to select semester
	var selectedIndex int
	if currentIndex >= 0 {
		fmt.Printf("Select a semester to calculate GPA for (default %s): ", gray(fmt.Sprintf("[%d]", currentIndex)))
	} else {
		fmt.Print("Select a semester to calculate GPA for: ")
	}

	var input string
	fmt.Scanln(&input)

	if input == "" && currentIndex >= 0 {
		selectedIndex = currentIndex
	} else {
		_, err := fmt.Sscanf(input, "%d", &selectedIndex)
		if err != nil || selectedIndex < 0 || selectedIndex >= len(semesters) {
			printError("Invalid semester selection")
			os.Exit(1)
		}
	}

	selectedSemester := &semesters[selectedIndex]
	fmt.Println()
	printSuccess(fmt.Sprintf("Calculating GPA for %d-%d Semester %d",
		selectedSemester.Year, selectedSemester.Year+1, selectedSemester.Semester))
	fmt.Println()

	// Get subjects
	printInfo("Fetching subjects...")
	subjects, err := apiClient.GetSubjectList(selectedSemester.ID)
	if err != nil {
		printError(fmt.Sprintf("Failed to get subjects: %v", err))
		os.Exit(1)
	}

	printSuccess(fmt.Sprintf("Found %d subjects", len(subjects)))
	fmt.Println()

	// Get semester dynamic scores to identify which subjects count toward GPA
	printInfo("Fetching semester-wide scores...")
	semesterScores, err := apiClient.GetSemesterDynamicScore(selectedSemester.ID)
	if err != nil {
		printWarning(fmt.Sprintf("Failed to get semester scores: %v", err))
		semesterScores = nil
	}

	// Create a map for quick lookup
	semesterScoreMap := make(map[uint64]*models.SubjectDynamicScore)
	if semesterScores != nil {
		for i := range semesterScores {
			semesterScoreMap[semesterScores[i].SubjectID] = &semesterScores[i]
		}
	}

	// Process each subject
	calculatedSubjects := []gpa.Subject{}

	printInfo("Calculating scores for each subject...")
	fmt.Println()

	for i, subject := range subjects {
		fmt.Printf("  %s Processing %s...\n", gray(fmt.Sprintf("[%d/%d]", i+1, len(subjects))), cyan(subject.Name))

		// Get task list
		tasks, err := apiClient.GetTaskList(selectedSemester.ID, subject.ID)
		if err != nil || len(tasks) == 0 {
			printWarning(fmt.Sprintf("    No tasks found for %s", subject.Name))
			continue
		}

		// Get task detail
		detail, err := apiClient.GetTaskDetail(tasks[0].ID)
		if err != nil {
			printWarning(fmt.Sprintf("    Failed to get task detail: %v", err))
			continue
		}

		// Get dynamic score detail
		dynamicScore, err := apiClient.GetDynamicScoreDetail(detail.ClassID, subject.ID, selectedSemester.ID)
		if err != nil {
			printWarning(fmt.Sprintf("    Failed to get dynamic score: %v", err))
			continue
		}

		// Get semester dynamic info for this subject
		dynamicInfo := semesterScoreMap[subject.ID]

		// Determine if elective (simplified - we'll mark manually if needed)
		isElective := strings.Contains(subject.Name, "Ele")

		// Process subject
		calcSubject := gpa.ProcessSubject(detail, dynamicScore, dynamicInfo, isElective)
		calculatedSubjects = append(calculatedSubjects, calcSubject)

		// Display subject info
		scoreStr := fmt.Sprintf("%.1f", calcSubject.Score)
		if calcSubject.OfficialScore != nil && calcSubject.ExtraCredit != 0 {
			scoreStr += gray(fmt.Sprintf(" (official: %.1f, extra: +%.1f)", *calcSubject.OfficialScore, calcSubject.ExtraCredit))
		}

		gpaStr := fmt.Sprintf("%.2f", calcSubject.GPA)
		if !calcSubject.IsInGrade {
			gpaStr += gray(" (not in GPA)")
		}

		typeStr := "Regular"
		if calcSubject.IsWeighted {
			typeStr = "Weighted"
		}
		if calcSubject.IsElective {
			typeStr += " Elective"
		}

		fmt.Printf("    Score: %s | GPA: %s | Type: %s\n",
			green(scoreStr), cyan(gpaStr), gray(typeStr))
	}

	fmt.Println()

	// Calculate final GPA
	printInfo("Calculating final GPA...")
	result := gpa.CalculateGPA(calculatedSubjects)

	// Display results
	fmt.Println()
	fmt.Println("----------------------------------------------")

	if !math.IsNaN(result.WeightedGPA) {
		fmt.Printf("%s %.2f %s\n",
			bold("Weighted GPA:"),
			result.WeightedGPA,
			gray(fmt.Sprintf("(max: %.1f)", result.MaxGPA)))

		fmt.Printf("%s %.2f %s\n",
			bold("Unweighted GPA:"),
			result.UnweightedGPA,
			gray(fmt.Sprintf("(max: %.1f)", result.UnweightedMaxGPA)))

		fmt.Println()

		// Get official GPA for comparison
		officialGPA, err := apiClient.GetGPA(selectedSemester.ID)
		if err == nil && officialGPA != nil {
			fmt.Printf("%s %.2f\n",
				bold("Official GPA:"),
				*officialGPA)
		} else if err == nil {
			fmt.Println(gray("Official GPA not yet published"))
		}

		fmt.Println()
		printSuccess(fmt.Sprintf("Calculated GPA from %d subjects", len(result.Subjects)))
	} else {
		printWarning("Unable to calculate GPA - no valid subjects found")
	}
}
