package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"myxb/internal/api"
	"myxb/internal/config"
	"myxb/internal/models"
	"os"
	"strconv"
	"time"
)

const (
	taskDetailCacheVersion  = 1
	taskScoreMatchTolerance = 0.25
)

type cachedTaskDetail struct {
	Fingerprint string               `json:"fingerprint"`
	FetchedAt   time.Time            `json:"fetched_at"`
	Detail      models.SubjectDetail `json:"detail"`
}

type taskDetailCacheFile struct {
	Version int                         `json:"version"`
	Entries map[string]cachedTaskDetail `json:"entries"`
}

type taskDetailCache struct {
	path        string
	data        taskDetailCacheFile
	dirty       bool
	refresh     bool
	hits        int
	misses      int
	writes      int
	initialSize int
}

type taskDetailCacheStats struct {
	Hits        int
	Misses      int
	Writes      int
	InitialSize int
	FinalSize   int
	Refresh     bool
}

func loadTaskDetailCache(refresh bool) (*taskDetailCache, error) {
	path, err := config.GetTaskDetailCachePath()
	if err != nil {
		return nil, err
	}

	cache := &taskDetailCache{
		path: path,
		data: taskDetailCacheFile{
			Version: taskDetailCacheVersion,
			Entries: make(map[string]cachedTaskDetail),
		},
		refresh: refresh,
	}
	if refresh {
		cache.dirty = true
		return cache, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cache, nil
		}
		return nil, err
	}

	var decoded taskDetailCacheFile
	if err := json.Unmarshal(data, &decoded); err != nil {
		return cache, nil
	}
	if decoded.Version != taskDetailCacheVersion || decoded.Entries == nil {
		return cache, nil
	}

	cache.data = decoded
	cache.initialSize = len(decoded.Entries)
	return cache, nil
}

func (c *taskDetailCache) save() error {
	if c == nil || !c.dirty {
		return nil
	}

	encoded, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, encoded, 0600)
}

func (c *taskDetailCache) stats() taskDetailCacheStats {
	if c == nil {
		return taskDetailCacheStats{}
	}

	return taskDetailCacheStats{
		Hits:        c.hits,
		Misses:      c.misses,
		Writes:      c.writes,
		InitialSize: c.initialSize,
		FinalSize:   len(c.data.Entries),
		Refresh:     c.refresh,
	}
}

func (c *taskDetailCache) detailFor(apiClient *api.API, task models.TaskItem) (*models.SubjectDetail, bool, error) {
	if c == nil {
		detail, err := apiClient.GetTaskDetail(task.ID)
		return detail, false, err
	}

	key := strconv.FormatUint(task.ID, 10)
	fingerprint := taskDetailFingerprint(task)
	if !c.refresh {
		if cached, ok := c.data.Entries[key]; ok && cached.Fingerprint == fingerprint {
			c.hits++
			detail := cached.Detail
			return &detail, true, nil
		}
	}

	c.misses++
	detail, err := apiClient.GetTaskDetail(task.ID)
	if err != nil {
		return nil, false, err
	}

	c.data.Entries[key] = cachedTaskDetail{
		Fingerprint: fingerprint,
		FetchedAt:   time.Now(),
		Detail:      *detail,
	}
	c.dirty = true
	c.writes++

	return detail, false, nil
}

func taskDetailFingerprint(task models.TaskItem) string {
	payload := struct {
		ID                uint64   `json:"id"`
		Name              string   `json:"name"`
		SubjectName       string   `json:"subject_name"`
		TotalScore        float64  `json:"total_score"`
		FinishState       uint8    `json:"finish_state"`
		Score             *float64 `json:"score"`
		BeginTime         string   `json:"begin_time"`
		EndTime           string   `json:"end_time"`
		SyncTime          string   `json:"sync_time"`
		LearningTaskState uint8    `json:"learning_task_state"`
		TypeName          string   `json:"type_name"`
		TypeEName         string   `json:"type_ename"`
		IsExempt          bool     `json:"is_exempt"`
	}{
		ID:                task.ID,
		Name:              task.Name,
		SubjectName:       task.SubjectName,
		TotalScore:        task.TotalScore,
		FinishState:       task.FinishState,
		Score:             task.Score,
		BeginTime:         task.BeginTime,
		EndTime:           task.EndTime,
		SyncTime:          task.SyncTime,
		LearningTaskState: task.LearningTaskState,
		TypeName:          task.TypeName,
		TypeEName:         task.TypeEName,
		IsExempt:          task.IsExempt,
	}

	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func attachTaskDetailMetadata(tasks []models.TaskItem, details map[uint64]*models.SubjectDetail, projects []models.EvaluationProject) []models.TaskItem {
	if len(tasks) == 0 {
		return tasks
	}

	projectByID := mapEvaluationProjects(projects)
	groupCounts, groupAverages := taskCategoryGroups(tasks, details)

	enriched := make([]models.TaskItem, 0, len(tasks))
	for _, task := range tasks {
		detail := details[task.ID]
		if detail == nil {
			enriched = append(enriched, task)
			continue
		}

		inSubjectScore := detail.IsInSubjectScore
		task.IsInSubjectScore = &inSubjectScore

		category := taskLeafEvaluationProject(detail)
		if category == nil {
			enriched = append(enriched, task)
			continue
		}

		task.CategoryID = category.ID
		task.CategoryName = category.Name
		task.CategoryEName = category.EName
		task.CategoryProportion = category.Proportion

		if project, ok := projectByID[category.ID]; ok {
			if task.CategoryName == "" {
				task.CategoryName = project.EvaluationProjectName
				if task.CategoryName == "" {
					task.CategoryName = project.EvaluationProjectEName
				}
			}
			if task.CategoryEName == "" {
				task.CategoryEName = project.EvaluationProjectEName
			}
			task.CategoryProportion = project.Proportion

			count := groupCounts[category.ID]
			average := groupAverages[category.ID]
			if detail.IsInSubjectScore && taskHasScore(task) && count > 0 &&
				!project.ScoreIsNull && math.Abs(project.Score-average) <= taskScoreMatchTolerance {
				weight := project.Proportion / float64(count)
				task.EstimatedSubjectWeight = &weight
			}
		}

		enriched = append(enriched, task)
	}

	return enriched
}

func mapEvaluationProjects(projects []models.EvaluationProject) map[uint64]models.EvaluationProject {
	out := make(map[uint64]models.EvaluationProject)
	var walk func([]models.EvaluationProject)
	walk = func(items []models.EvaluationProject) {
		for _, project := range items {
			if project.EvaluationProjectID != 0 {
				out[project.EvaluationProjectID] = project
			}
			walk(project.EvaluationProjectList)
		}
	}
	walk(projects)
	return out
}

func taskCategoryGroups(tasks []models.TaskItem, details map[uint64]*models.SubjectDetail) (map[uint64]int, map[uint64]float64) {
	counts := make(map[uint64]int)
	averages := make(map[uint64]float64)

	for _, task := range tasks {
		detail := details[task.ID]
		if detail == nil || !detail.IsInSubjectScore || !taskHasScore(task) || task.TotalScore <= 0 {
			continue
		}

		category := taskLeafEvaluationProject(detail)
		if category == nil || category.ID == 0 {
			continue
		}

		counts[category.ID]++
		averages[category.ID] += *task.Score / task.TotalScore * 100.0
	}

	for categoryID, count := range counts {
		if count > 0 {
			averages[categoryID] /= float64(count)
		}
	}

	return counts, averages
}

func taskLeafEvaluationProject(detail *models.SubjectDetail) *models.TaskEvaluationProject {
	if detail == nil || len(detail.EvaProjects) == 0 {
		return nil
	}

	return &detail.EvaProjects[len(detail.EvaProjects)-1]
}

func taskHasScore(task models.TaskItem) bool {
	return task.FinishState != 0 && task.Score != nil
}

func taskCategoryDisplay(task models.TaskItem) string {
	name := task.CategoryEName
	if name == "" {
		name = task.CategoryName
	}
	if name == "" {
		return "-"
	}
	return asciiDisplayText(name)
}

func taskWeightDisplay(task models.TaskItem) string {
	if task.EstimatedSubjectWeight == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f%%", *task.EstimatedSubjectWeight)
}

func taskEstimatedWeightDisplay(task models.TaskItem) string {
	if task.EstimatedSubjectWeight == nil {
		return "- -"
	}
	return fmt.Sprintf("- %.2f%%", *task.EstimatedSubjectWeight)
}
