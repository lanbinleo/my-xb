# GPA计算说明

## 概述

GPA计算分为**加权GPA (Weighted GPA)** 和**非加权GPA (Unweighted GPA)** 两种。计算流程如下：

1. 获取每个科目的评分项目（EvaluationProject）
2. 调整评分项目的比例（排除无成绩的项目）
3. 计算科目总分
4. 根据分数映射表将分数转换为GPA
5. 应用科目权重
6. 计算加权平均GPA

---

## 分数映射表

分数映射表定义了分数区间到GPA的转换规则，分为两种类型：

### Weighted（加权课程）

适用于AP、A Level、AS课程，以及特定的高级课程：
- Linear Algebra
- Modern Physics and Optics
- Multivariable Calculus

| 分数范围 | 等级 | GPA |
|---------|------|-----|
| 97.0 - 100.0 | A+ | 4.8 |
| 93.0 - 96.9  | A  | 4.5 |
| 90.0 - 92.9  | A- | 4.2 |
| 87.0 - 89.9  | B+ | 3.8 |
| 83.0 - 86.9  | B  | 3.5 |
| 80.0 - 82.9  | B- | 3.2 |
| 77.0 - 79.9  | C+ | 2.8 |
| 73.0 - 76.9  | C  | 2.5 |
| 70.0 - 72.9  | C- | 2.2 |
| 67.0 - 69.9  | D+ | 1.8 |
| 63.0 - 66.9  | D  | 1.5 |
| 60.0 - 62.9  | D- | 1.2 |
| 0.0 - 59.9   | F  | 0.0 |

### Non-Weighted（非加权课程）

适用于常规课程：

| 分数范围 | 等级 | GPA |
|---------|------|-----|
| 97.0 - 100.0 | A+ | 4.3 |
| 93.0 - 96.9  | A  | 4.0 |
| 90.0 - 92.9  | A- | 3.7 |
| 87.0 - 89.9  | B+ | 3.3 |
| 83.0 - 86.9  | B  | 3.0 |
| 80.0 - 82.9  | B- | 2.7 |
| 77.0 - 79.9  | C+ | 2.3 |
| 73.0 - 76.9  | C  | 2.0 |
| 70.0 - 72.9  | C- | 1.7 |
| 67.0 - 69.9  | D+ | 1.3 |
| 63.0 - 66.9  | D  | 1.0 |
| 60.0 - 62.9  | D- | 0.7 |
| 0.0 - 59.9   | F  | 0.0 |

**映射表文件**: `score_mapping.json`

---

## 判定加权课程的规则

```go
func isWeightedSubject(subjectName string) bool {
    // AP课程
    if strings.Contains(subjectName, "AP") {
        return true
    }

    // A Level课程
    if strings.Contains(subjectName, "A Level") {
        return true
    }

    // AS课程
    if strings.Contains(subjectName, "AS") {
        return true
    }

    // 特定高级课程
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
```

---

## 科目权重规则

每个科目有一个权重系数，用于最终GPA计算：

### 默认权重: 1.0
大部分必修课使用默认权重1.0

### 选修课权重: 0.5
以下情况权重设为0.5：
1. **选修课（Elective）**: 课程名称包含"Ele"的课程
2. **C-Humanities**: 特殊指定的课程

### 识别选修课的方法

通过查询学生课表（Calendar API）识别选修课：
```go
// 伪代码
calendar := GetCalendar(client, currentTime-8days, currentTime+8days)
electiveClassIDs := []uint64{}

for _, block := range calendar.Blocks {
    if strings.Contains(block.ClassName, "Ele") {
        electiveClassIDs = append(electiveClassIDs, block.ID)
    }
}

// 对每个科目检查
if contains(electiveClassIDs, subject.ClassID) || subject.Name == "C-Humanities" {
    subject.Weight = 0.5
    subject.Elective = true
}
```

---

## 评分项目比例调整

从API获取的评分项目比例可能不准确，因为某些项目可能没有成绩。需要重新计算比例：

### 算法

```go
// 1. 过滤出有成绩的项目
validProjects := filter(evaluationProjects, func(p EvaluationProject) bool {
    return !p.ScoreIsNull
})

// 2. 计算有效项目的总比例
totalProportion := sum(validProjects, func(p EvaluationProject) float64 {
    return p.Proportion
})

// 3. 调整每个项目的比例，使总和为100%
for i := range evaluationProjects {
    evaluationProjects[i].AdjustedProportion =
        evaluationProjects[i].Proportion / totalProportion * 100.0
}
```

### 嵌套评分项目

如果评分项目包含子项目（`EvaluationProjectList`），需要递归调整：

```go
for i := range evaluationProjects {
    if len(evaluationProjects[i].EvaluationProjectList) > 0 {
        subProjects := evaluationProjects[i].EvaluationProjectList

        // 计算子项目的有效总比例
        subTotalProportion := sum(filter(subProjects, notNull), getProportion)

        // 调整子项目比例
        for j := range subProjects {
            subProjects[j].AdjustedProportion =
                subProjects[j].Proportion / subTotalProportion *
                evaluationProjects[i].AdjustedProportion
        }
    }
}
```

---

## 科目总分计算

```go
func calculateSubjectScore(evaluationProjects []EvaluationProject) float64 {
    totalScore := 0.0

    for _, project := range evaluationProjects {
        if !project.ScoreIsNull {
            totalScore += project.Score * project.AdjustedProportion / 100.0
        }
    }

    return totalScore
}
```

**注意**:
- 只计算`ScoreIsNull == false`的项目
- 使用调整后的比例（`AdjustedProportion`）而非原始比例
- 最终分数会四舍五入到小数点后1位

---

## 分数转GPA

```go
func scoreToGPA(score float64, mappingList []ScoreMapping) float64 {
    // 四舍五入到小数点后1位
    roundedScore := math.Round(score * 10) / 10

    for _, mapping := range mappingList {
        if roundedScore >= mapping.MinValue && roundedScore <= mapping.MaxValue {
            return mapping.GPA
        }
    }

    return math.NaN() // 未找到匹配
}
```

---

## 额外加分处理

某些科目可能有额外加分（Extra Credit）：

```go
// 从API获取的官方分数
officialScore := subjectDynamicScore.SubjectScore / subjectDynamicScore.SubjectTotalScore * 100.0

// 计算得出的分数
calculatedScore := calculateSubjectScore(evaluationProjects)

// 额外加分
extraCredit := officialScore - round(calculatedScore, 1)

// 使用官方分数作为最终分数
finalScore := officialScore
```

**说明**:
- 优先使用`GetStuSemesterDynamicScore` API返回的官方分数
- 额外加分可能来自特殊项目或教师调整

---

## 最终GPA计算

### 加权GPA计算

```go
type CalculatedGPA struct {
    WeightedGPA        float64
    MaxGPA             float64
    UnweightedGPA      float64
    UnweightedMaxGPA   float64
}

func calculateGPA(subjects []Subject) CalculatedGPA {
    totalWeight := 0.0
    totalWeightedGPA := 0.0
    totalMaxGPA := 0.0
    totalUnweightedGPA := 0.0

    // 只计算有GPA的科目
    for _, subject := range subjects {
        if math.IsNaN(subject.GPA) {
            continue
        }

        totalWeight += subject.Weight
        totalWeightedGPA += subject.GPA * subject.Weight
        totalUnweightedGPA += subject.UnweightedGPA * subject.Weight
        totalMaxGPA += subject.MaxGPA * subject.Weight
    }

    return CalculatedGPA{
        WeightedGPA:       totalWeightedGPA / totalWeight,
        UnweightedGPA:     totalUnweightedGPA / totalWeight,
        MaxGPA:            totalMaxGPA / totalWeight,
        UnweightedMaxGPA:  subjects[0].UnweightedMaxGPA, // 通常都是4.3
    }
}
```

### 公式

```
加权GPA = Σ(科目GPA × 科目权重) / Σ(科目权重)

非加权GPA = Σ(科目非加权GPA × 科目权重) / Σ(科目权重)
```

**示例**:

假设有3门课程：
1. AP Calculus: GPA=4.5, Weight=1.0 (Weighted)
2. English: GPA=3.7, Weight=1.0 (Non-Weighted)
3. Elective Art: GPA=4.0, Weight=0.5 (Non-Weighted)

```
总权重 = 1.0 + 1.0 + 0.5 = 2.5

加权GPA = (4.5×1.0 + 3.7×1.0 + 4.0×0.5) / 2.5
        = (4.5 + 3.7 + 2.0) / 2.5
        = 10.2 / 2.5
        = 4.08
```

---

## 特殊情况处理

### 1. 没有成绩的科目
- `ScoreIsNull == true` 的项目不参与计算
- 如果整个科目都没有成绩（所有项目都是null），科目总分为`NaN`
- 总分为`NaN`的科目不参与GPA计算

### 2. 是否计入GPA
- 优先使用`GetStuSemesterDynamicScore` API返回的`IsInGrade`字段
- 如果`IsInGrade == false`，该科目不计入最终GPA

### 3. 分数精度
- 科目总分四舍五入到小数点后1位
- GPA结果通常保留2位小数

---

## 完整计算流程

```
1. 获取学期的所有科目
   ↓
2. 对每个科目:
   a. 获取评分项目列表
   b. 调整评分项目比例（排除无成绩项目）
   c. 计算科目总分
   d. 判断使用Weighted还是Non-Weighted映射表
   e. 将分数转换为GPA
   ↓
3. 识别选修课，调整科目权重
   ↓
4. 叠加官方动态分数（处理额外加分和IsInGrade标识）
   ↓
5. 过滤掉不计入GPA的科目（IsInGrade==false或GPA为NaN）
   ↓
6. 计算加权平均GPA和非加权平均GPA
   ↓
7. 输出结果，与官方GPA对比
```

---

## 验证

建议实现以下验证机制：

1. **与官方GPA对比**: 使用`GetGpa` API获取官方GPA，与计算结果对比
2. **分数范围检查**: 确保所有分数在0-100范围内
3. **GPA范围检查**: 加权GPA应≤4.8，非加权GPA应≤4.3
4. **权重总和检查**: 所有科目权重之和应为合理值
5. **比例总和检查**: 每个科目的调整后比例总和应为100%

---

## 注意事项

1. **浮点数精度**: 使用`math.Round()`进行四舍五入，避免浮点数精度问题
2. **并发请求**: 可以并发获取多个科目的成绩，提高效率
3. **错误处理**: 某个科目获取失败不应影响其他科目的计算
4. **缓存**: 分数映射表可以缓存，避免重复解析JSON
5. **时区**: 日期时间使用UTC+8时区（Asia/Shanghai）
