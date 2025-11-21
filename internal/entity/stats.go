package entity

type ReviewerAssignmentStat struct {
	UserID      string `json:"user_id"`
	Assignments int    `json:"assignments"`
}

type PRStatusStat struct {
	Total            int     `json:"total"`
	Open             int     `json:"open"`
	Merged           int     `json:"merged"`
	AverageReviewers float64 `json:"average_reviewers"`
}

type TeamMemberStat struct {
	TeamName string `json:"team_name"`
	Active   int    `json:"active_members"`
	Inactive int    `json:"inactive_members"`
}

type DurationBreakdown struct {
	Days    int `json:"days"`
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
}

type PRLifetimeStat struct {
	AverageMerge       DurationBreakdown `json:"average_merge"`
	OpenOlderThan7Days int               `json:"open_older_than_7_days"`
}

type StatsSummary struct {
	ReviewerAssignments []ReviewerAssignmentStat `json:"reviewer_assignments"`
	PRStatus            PRStatusStat             `json:"pr_status"`
	TeamMembers         []TeamMemberStat         `json:"team_members"`
	PRLifetime          PRLifetimeStat           `json:"pr_lifetime"`
}
