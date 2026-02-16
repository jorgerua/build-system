package shared

import (
	"time"
)

// JobStatus representa o status de um job de build
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusSuccess   JobStatus = "success"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// IsValid verifica se o status é válido
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusPending, JobStatusRunning, JobStatusSuccess, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal verifica se o status é terminal (não pode mais mudar)
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusSuccess || s == JobStatusFailed || s == JobStatusCancelled
}

// BuildPhase representa uma fase do processo de build
type BuildPhase string

const (
	BuildPhaseGitSync   BuildPhase = "git_sync"
	BuildPhaseNXBuild   BuildPhase = "nx_build"
	BuildPhaseImageBuild BuildPhase = "image_build"
)

// IsValid verifica se a fase é válida
func (p BuildPhase) IsValid() bool {
	switch p {
	case BuildPhaseGitSync, BuildPhaseNXBuild, BuildPhaseImageBuild:
		return true
	default:
		return false
	}
}

// Language representa uma linguagem de programação suportada
type Language string

const (
	LanguageJava   Language = "java"
	LanguageDotNet Language = "dotnet"
	LanguageGo     Language = "go"
	LanguageUnknown Language = "unknown"
)

// IsValid verifica se a linguagem é válida
func (l Language) IsValid() bool {
	switch l {
	case LanguageJava, LanguageDotNet, LanguageGo, LanguageUnknown:
		return true
	default:
		return false
	}
}

// IsSupported verifica se a linguagem é suportada (não unknown)
func (l Language) IsSupported() bool {
	return l != LanguageUnknown && l.IsValid()
}

// RepositoryInfo contém informações sobre um repositório
type RepositoryInfo struct {
	URL   string `json:"url"`
	Name  string `json:"name"`
	Owner string `json:"owner"`
	Branch string `json:"branch"`
}

// FullName retorna o nome completo do repositório (owner/name)
func (r *RepositoryInfo) FullName() string {
	if r.Owner == "" || r.Name == "" {
		return ""
	}
	return r.Owner + "/" + r.Name
}

// IsValid verifica se as informações do repositório são válidas
func (r *RepositoryInfo) IsValid() bool {
	return r.URL != "" && r.Name != "" && r.Owner != ""
}

// PhaseMetric contém métricas de uma fase do build
type PhaseMetric struct {
	Phase     BuildPhase    `json:"phase"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

// IsComplete verifica se a métrica está completa (tem start e end time)
func (m *PhaseMetric) IsComplete() bool {
	return !m.StartTime.IsZero() && !m.EndTime.IsZero()
}

// CalculateDuration calcula a duração baseada nos timestamps
func (m *PhaseMetric) CalculateDuration() {
	if m.IsComplete() {
		m.Duration = m.EndTime.Sub(m.StartTime)
	}
}

// BuildJob representa um job de build na fila
type BuildJob struct {
	ID           string          `json:"id"`
	Repository   RepositoryInfo  `json:"repository"`
	CommitHash   string          `json:"commit_hash"`
	CommitAuthor string          `json:"commit_author"`
	CommitMsg    string          `json:"commit_message"`
	Branch       string          `json:"branch"`
	Status       JobStatus       `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	Duration     time.Duration   `json:"duration"`
	Error        string          `json:"error,omitempty"`
	Phases       []PhaseMetric   `json:"phases"`
}

// IsValid verifica se o job é válido
func (j *BuildJob) IsValid() bool {
	return j.ID != "" && j.Repository.IsValid() && j.CommitHash != "" && j.Status.IsValid()
}

// MarkStarted marca o job como iniciado
func (j *BuildJob) MarkStarted() {
	now := time.Now()
	j.StartedAt = &now
	j.Status = JobStatusRunning
}

// MarkCompleted marca o job como completado com sucesso
func (j *BuildJob) MarkCompleted() {
	now := time.Now()
	j.CompletedAt = &now
	j.Status = JobStatusSuccess
	j.CalculateDuration()
}

// MarkFailed marca o job como falho
func (j *BuildJob) MarkFailed(err string) {
	now := time.Now()
	j.CompletedAt = &now
	j.Status = JobStatusFailed
	j.Error = err
	j.CalculateDuration()
}

// MarkCancelled marca o job como cancelado
func (j *BuildJob) MarkCancelled() {
	now := time.Now()
	j.CompletedAt = &now
	j.Status = JobStatusCancelled
	j.CalculateDuration()
}

// CalculateDuration calcula a duração total do job
func (j *BuildJob) CalculateDuration() {
	if j.StartedAt != nil && j.CompletedAt != nil {
		j.Duration = j.CompletedAt.Sub(*j.StartedAt)
	}
}

// AddPhase adiciona uma métrica de fase ao job
func (j *BuildJob) AddPhase(phase PhaseMetric) {
	j.Phases = append(j.Phases, phase)
}

// GetPhase retorna a métrica de uma fase específica
func (j *BuildJob) GetPhase(phase BuildPhase) *PhaseMetric {
	for i := range j.Phases {
		if j.Phases[i].Phase == phase {
			return &j.Phases[i]
		}
	}
	return nil
}

// HasPhase verifica se o job tem uma fase específica
func (j *BuildJob) HasPhase(phase BuildPhase) bool {
	return j.GetPhase(phase) != nil
}
