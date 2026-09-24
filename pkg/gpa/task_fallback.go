package gpa

import (
	"myxb/internal/models"
	"sort"
	"strings"
)

// BuildEvaluationFromTasks rebuilds a subject's evaluation-project tree from
// learning tasks and their details. It is used when GetDynamicScoreDetail
// rejects a course (for example "该课程成绩未同步到移动端"): task-level scores
// and category paths stay available through GetList/GetDetail, so the subject
// can still be scored locally instead of being skipped.
//
// Aggregation rules mirror the server-side behavior already observed by
// attachTaskDetailMetadata:
//   - a leaf category's score is the average of its member tasks' percentage
//     scores (score/totalScore*100, synced tasks only)
//   - a parent category's score is the proportion-weighted average of its
//     scored children; proportions in a task's category path are relative to
//     the parent category (top-level ones relative to the whole subject)
//   - categories without any scored task keep ScoreIsNull and are ignored by
//     AdjustProportions/CalculateSubjectScore, matching real API data
//
// The rebuilt tree is an estimate: server-side adjustments such as dropped
// lowest scores cannot be reproduced.
func BuildEvaluationFromTasks(tasks []models.TaskItem, details map[uint64]*models.SubjectDetail, isWeighted bool) []models.EvaluationProject {
	type builderNode struct {
		id         uint64
		name       string
		eName      string
		proportion float64
		scoreSum   float64
		scoreCount int
		children   []*builderNode
	}

	root := &builderNode{}
	nodes := make(map[uint64]*builderNode)

	for _, task := range tasks {
		detail := details[task.ID]
		if detail == nil || !detail.IsInSubjectScore || len(detail.EvaProjects) == 0 {
			continue
		}

		path := append([]models.TaskEvaluationProject(nil), detail.EvaProjects...)
		// The API returns the path leaf-first; walk it root-first so parents
		// exist before their children are attached.
		sort.SliceStable(path, func(i, j int) bool {
			return proPathDepth(path[i].ProPath) < proPathDepth(path[j].ProPath)
		})

		parent := root
		for _, step := range path {
			node, ok := nodes[step.ID]
			if !ok {
				node = &builderNode{
					id:         step.ID,
					name:       step.Name,
					eName:      step.EName,
					proportion: step.Proportion,
				}
				nodes[step.ID] = node
				parent.children = append(parent.children, node)
			}
			parent = node
		}

		if taskIsScored(task) && parent != root {
			parent.scoreSum += *task.Score / task.TotalScore * 100.0
			parent.scoreCount++
		}
	}

	var convert func(node *builderNode) models.EvaluationProject
	convert = func(node *builderNode) models.EvaluationProject {
		project := models.EvaluationProject{
			EvaluationProjectID:    node.id,
			EvaluationProjectName:  node.name,
			EvaluationProjectEName: node.eName,
			Proportion:             node.proportion,
			ScoreIsNull:            true,
		}

		childProportionSum := 0.0
		weightedChildSum := 0.0
		for _, child := range node.children {
			childProject := convert(child)
			project.EvaluationProjectList = append(project.EvaluationProjectList, childProject)
			if !childProject.ScoreIsNull {
				childProportionSum += childProject.Proportion
				weightedChildSum += childProject.Score * childProject.Proportion
			}
		}

		score, hasScore := 0.0, false
		if childProportionSum > 0 {
			score, hasScore = weightedChildSum/childProportionSum, true
		} else if node.scoreCount > 0 {
			score, hasScore = node.scoreSum/float64(node.scoreCount), true
		}

		if hasScore {
			project.Score = score
			project.ScoreIsNull = false
			project.ScoreLevel = GetScoreLevelFromScore(score, isWeighted)
			project.GPA = ScoreToGPA(score, isWeighted)
		}

		return project
	}

	var sortTree func(node *builderNode)
	sortTree = func(node *builderNode) {
		sort.SliceStable(node.children, func(i, j int) bool { return node.children[i].id < node.children[j].id })
		for _, child := range node.children {
			sortTree(child)
		}
	}
	sortTree(root)

	projects := make([]models.EvaluationProject, 0, len(root.children))
	for _, child := range root.children {
		projects = append(projects, convert(child))
	}
	return projects
}

// taskIsScored reports whether a task exposes a real grade. Tasks whose scores
// have not been synced to the mobile API report score=0 even before grading,
// so IsSynchroToMobiled must be set for the score to count.
func taskIsScored(task models.TaskItem) bool {
	return task.IsSynchroToMobiled == 1 && task.FinishState != 0 &&
		task.Score != nil && task.TotalScore > 0
}

func proPathDepth(proPath string) int {
	depth := 0
	for _, segment := range strings.Split(proPath, ",") {
		if strings.TrimSpace(segment) != "" {
			depth++
		}
	}
	return depth
}
