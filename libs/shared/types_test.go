package shared

import (
	"testing"
	"time"
)

// TestJobStatus_IsValid testa validação de status de job
func TestJobStatus_IsValid(t *testing.T) {
	tests := []struct {
		status JobStatus
		valid  bool
	}{
		{JobStatusPending, true},
		{JobStatusRunning, true},
		{JobStatusSuccess, true},
		{JobStatusFailed, true},
		{JobStatusCancelled, true},
		{JobStatus("invalid"), false},
		{JobStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// TestJobStatus_IsTerminal testa verificação de status terminal
func TestJobStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		status   JobStatus
		terminal bool
	}{
		{JobStatusPending, false},
		{JobStatusRunning, false},
		{JobStatusSuccess, true},
		{JobStatusFailed, true},
		{JobStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsTerminal(); got != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.terminal)
			}
		})
	}
}

// TestBuildPhase_IsValid testa validação de fase de build
func TestBuildPhase_IsValid(t *testing.T) {
	tests := []struct {
		phase BuildPhase
		valid bool
	}{
		{BuildPhaseGitSync, true},
		{BuildPhaseNXBuild, true},
		{BuildPhaseImageBuild, true},
		{BuildPhase("invalid"), false},
		{BuildPhase(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			if got := tt.phase.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// TestLanguage_IsValid testa validação de linguagem
func TestLanguage_IsValid(t *testing.T) {
	tests := []struct {
		lang  Language
		valid bool
	}{
		{LanguageJava, true},
		{LanguageDotNet, true},
		{LanguageGo, true},
		{LanguageUnknown, true},
		{Language("invalid"), false},
		{Language(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			if got := tt.lang.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// TestLanguage_IsSupported testa verificação de linguagem suportada
func TestLanguage_IsSupported(t *testing.T) {
	tests := []struct {
		lang      Language
		supported bool
	}{
		{LanguageJava, true},
		{LanguageDotNet, true},
		{LanguageGo, true},
		{LanguageUnknown, false},
		{Language("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			if got := tt.lang.IsSupported(); got != tt.supported {
				t.Errorf("IsSupported() = %v, want %v", got, tt.supported)
			}
		})
	}
}

// TestRepositoryInfo_FullName testa geração de nome completo do repositório
func TestRepositoryInfo_FullName(t *testing.T) {
	tests := []struct {
		name     string
		repo     RepositoryInfo
		expected string
	}{
		{
			"valid repo",
			RepositoryInfo{Owner: "owner", Name: "repo"},
			"owner/repo",
		},
		{
			"missing owner",
			RepositoryInfo{Owner: "", Name: "repo"},
			"",
		},
		{
			"missing name",
			RepositoryInfo{Owner: "owner", Name: ""},
			"",
		},
		{
			"both missing",
			RepositoryInfo{Owner: "", Name: ""},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.repo.FullName(); got != tt.expected {
				t.Errorf("FullName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestRepositoryInfo_IsValid testa validação de informações do repositório
func TestRepositoryInfo_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		repo  RepositoryInfo
		valid bool
	}{
		{
			"valid repo",
			RepositoryInfo{
				URL:   "https://github.com/owner/repo",
				Name:  "repo",
				Owner: "owner",
			},
			true,
		},
		{
			"missing URL",
			RepositoryInfo{Name: "repo", Owner: "owner"},
			false,
		},
		{
			"missing name",
			RepositoryInfo{URL: "https://github.com/owner/repo", Owner: "owner"},
			false,
		},
		{
			"missing owner",
			RepositoryInfo{URL: "https://github.com/owner/repo", Name: "repo"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.repo.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// TestPhaseMetric_IsComplete testa verificação de métrica completa
func TestPhaseMetric_IsComplete(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Minute)

	tests := []struct {
		name     string
		metric   PhaseMetric
		complete bool
	}{
		{
			"complete metric",
			PhaseMetric{StartTime: now, EndTime: later},
			true,
		},
		{
			"missing end time",
			PhaseMetric{StartTime: now},
			false,
		},
		{
			"missing start time",
			PhaseMetric{EndTime: later},
			false,
		},
		{
			"both missing",
			PhaseMetric{},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metric.IsComplete(); got != tt.complete {
				t.Errorf("IsComplete() = %v, want %v", got, tt.complete)
			}
		})
	}
}

// TestPhaseMetric_CalculateDuration testa cálculo de duração
func TestPhaseMetric_CalculateDuration(t *testing.T) {
	now := time.Now()
	later := now.Add(5 * time.Minute)

	metric := PhaseMetric{
		StartTime: now,
		EndTime:   later,
	}

	metric.CalculateDuration()

	expected := 5 * time.Minute
	if metric.Duration != expected {
		t.Errorf("CalculateDuration() = %v, want %v", metric.Duration, expected)
	}
}

// TestPhaseMetric_CalculateDuration_Incomplete testa que duração não é calculada se incompleta
func TestPhaseMetric_CalculateDuration_Incomplete(t *testing.T) {
	metric := PhaseMetric{
		StartTime: time.Now(),
	}

	metric.CalculateDuration()

	if metric.Duration != 0 {
		t.Errorf("CalculateDuration() for incomplete metric = %v, want 0", metric.Duration)
	}
}

// TestBuildJob_IsValid testa validação de job de build
func TestBuildJob_IsValid(t *testing.T) {
	validRepo := RepositoryInfo{
		URL:   "https://github.com/owner/repo",
		Name:  "repo",
		Owner: "owner",
	}

	tests := []struct {
		name  string
		job   BuildJob
		valid bool
	}{
		{
			"valid job",
			BuildJob{
				ID:         "job-123",
				Repository: validRepo,
				CommitHash: "abc123",
				Status:     JobStatusPending,
			},
			true,
		},
		{
			"missing ID",
			BuildJob{
				Repository: validRepo,
				CommitHash: "abc123",
				Status:     JobStatusPending,
			},
			false,
		},
		{
			"invalid repository",
			BuildJob{
				ID:         "job-123",
				Repository: RepositoryInfo{},
				CommitHash: "abc123",
				Status:     JobStatusPending,
			},
			false,
		},
		{
			"missing commit hash",
			BuildJob{
				ID:         "job-123",
				Repository: validRepo,
				Status:     JobStatusPending,
			},
			false,
		},
		{
			"invalid status",
			BuildJob{
				ID:         "job-123",
				Repository: validRepo,
				CommitHash: "abc123",
				Status:     JobStatus("invalid"),
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.job.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

// TestBuildJob_MarkStarted testa marcação de job como iniciado
func TestBuildJob_MarkStarted(t *testing.T) {
	job := BuildJob{
		ID:     "job-123",
		Status: JobStatusPending,
	}

	before := time.Now()
	job.MarkStarted()
	after := time.Now()

	if job.Status != JobStatusRunning {
		t.Errorf("Status = %v, want %v", job.Status, JobStatusRunning)
	}

	if job.StartedAt == nil {
		t.Error("StartedAt is nil, expected timestamp")
	} else {
		if job.StartedAt.Before(before) || job.StartedAt.After(after) {
			t.Errorf("StartedAt = %v, expected between %v and %v", job.StartedAt, before, after)
		}
	}
}

// TestBuildJob_MarkCompleted testa marcação de job como completado
func TestBuildJob_MarkCompleted(t *testing.T) {
	startTime := time.Now()
	job := BuildJob{
		ID:        "job-123",
		Status:    JobStatusRunning,
		StartedAt: &startTime,
	}

	time.Sleep(10 * time.Millisecond) // Pequeno delay para garantir duração > 0
	job.MarkCompleted()

	if job.Status != JobStatusSuccess {
		t.Errorf("Status = %v, want %v", job.Status, JobStatusSuccess)
	}

	if job.CompletedAt == nil {
		t.Error("CompletedAt is nil, expected timestamp")
	}

	if job.Duration <= 0 {
		t.Errorf("Duration = %v, expected > 0", job.Duration)
	}
}

// TestBuildJob_MarkFailed testa marcação de job como falho
func TestBuildJob_MarkFailed(t *testing.T) {
	startTime := time.Now()
	job := BuildJob{
		ID:        "job-123",
		Status:    JobStatusRunning,
		StartedAt: &startTime,
	}

	errorMsg := "build failed"
	time.Sleep(10 * time.Millisecond)
	job.MarkFailed(errorMsg)

	if job.Status != JobStatusFailed {
		t.Errorf("Status = %v, want %v", job.Status, JobStatusFailed)
	}

	if job.Error != errorMsg {
		t.Errorf("Error = %v, want %v", job.Error, errorMsg)
	}

	if job.CompletedAt == nil {
		t.Error("CompletedAt is nil, expected timestamp")
	}

	if job.Duration <= 0 {
		t.Errorf("Duration = %v, expected > 0", job.Duration)
	}
}

// TestBuildJob_MarkCancelled testa marcação de job como cancelado
func TestBuildJob_MarkCancelled(t *testing.T) {
	startTime := time.Now()
	job := BuildJob{
		ID:        "job-123",
		Status:    JobStatusRunning,
		StartedAt: &startTime,
	}

	time.Sleep(10 * time.Millisecond)
	job.MarkCancelled()

	if job.Status != JobStatusCancelled {
		t.Errorf("Status = %v, want %v", job.Status, JobStatusCancelled)
	}

	if job.CompletedAt == nil {
		t.Error("CompletedAt is nil, expected timestamp")
	}

	if job.Duration <= 0 {
		t.Errorf("Duration = %v, expected > 0", job.Duration)
	}
}

// TestBuildJob_AddPhase testa adição de fase ao job
func TestBuildJob_AddPhase(t *testing.T) {
	job := BuildJob{ID: "job-123"}

	phase := PhaseMetric{
		Phase:   BuildPhaseGitSync,
		Success: true,
	}

	job.AddPhase(phase)

	if len(job.Phases) != 1 {
		t.Errorf("len(Phases) = %d, want 1", len(job.Phases))
	}

	if job.Phases[0].Phase != BuildPhaseGitSync {
		t.Errorf("Phases[0].Phase = %v, want %v", job.Phases[0].Phase, BuildPhaseGitSync)
	}
}

// TestBuildJob_GetPhase testa obtenção de fase específica
func TestBuildJob_GetPhase(t *testing.T) {
	job := BuildJob{
		ID: "job-123",
		Phases: []PhaseMetric{
			{Phase: BuildPhaseGitSync, Success: true},
			{Phase: BuildPhaseNXBuild, Success: true},
		},
	}

	// Fase existente
	phase := job.GetPhase(BuildPhaseGitSync)
	if phase == nil {
		t.Error("GetPhase(BuildPhaseGitSync) = nil, expected phase")
	} else if phase.Phase != BuildPhaseGitSync {
		t.Errorf("GetPhase(BuildPhaseGitSync).Phase = %v, want %v", phase.Phase, BuildPhaseGitSync)
	}

	// Fase não existente
	phase = job.GetPhase(BuildPhaseImageBuild)
	if phase != nil {
		t.Errorf("GetPhase(BuildPhaseImageBuild) = %v, want nil", phase)
	}
}

// TestBuildJob_HasPhase testa verificação de existência de fase
func TestBuildJob_HasPhase(t *testing.T) {
	job := BuildJob{
		ID: "job-123",
		Phases: []PhaseMetric{
			{Phase: BuildPhaseGitSync, Success: true},
		},
	}

	if !job.HasPhase(BuildPhaseGitSync) {
		t.Error("HasPhase(BuildPhaseGitSync) = false, want true")
	}

	if job.HasPhase(BuildPhaseImageBuild) {
		t.Error("HasPhase(BuildPhaseImageBuild) = true, want false")
	}
}

// TestBuildJob_CalculateDuration testa cálculo de duração do job
func TestBuildJob_CalculateDuration(t *testing.T) {
	startTime := time.Now()
	completedTime := startTime.Add(5 * time.Minute)

	job := BuildJob{
		ID:          "job-123",
		StartedAt:   &startTime,
		CompletedAt: &completedTime,
	}

	job.CalculateDuration()

	expected := 5 * time.Minute
	if job.Duration != expected {
		t.Errorf("Duration = %v, want %v", job.Duration, expected)
	}
}

// TestBuildJob_CalculateDuration_NoStartTime testa que duração não é calculada sem start time
func TestBuildJob_CalculateDuration_NoStartTime(t *testing.T) {
	completedTime := time.Now()
	job := BuildJob{
		ID:          "job-123",
		CompletedAt: &completedTime,
	}

	job.CalculateDuration()

	if job.Duration != 0 {
		t.Errorf("Duration = %v, want 0", job.Duration)
	}
}

// TestBuildJob_CompleteWorkflow testa fluxo completo de um job
func TestBuildJob_CompleteWorkflow(t *testing.T) {
	// Criar job
	job := BuildJob{
		ID: "job-123",
		Repository: RepositoryInfo{
			URL:   "https://github.com/owner/repo",
			Name:  "repo",
			Owner: "owner",
		},
		CommitHash: "abc123",
		Status:     JobStatusPending,
		CreatedAt:  time.Now(),
	}

	// Validar job inicial
	if !job.IsValid() {
		t.Error("Initial job is invalid")
	}

	// Marcar como iniciado
	job.MarkStarted()
	if job.Status != JobStatusRunning {
		t.Errorf("After MarkStarted, Status = %v, want %v", job.Status, JobStatusRunning)
	}

	// Adicionar fases
	job.AddPhase(PhaseMetric{Phase: BuildPhaseGitSync, Success: true})
	job.AddPhase(PhaseMetric{Phase: BuildPhaseNXBuild, Success: true})
	job.AddPhase(PhaseMetric{Phase: BuildPhaseImageBuild, Success: true})

	if len(job.Phases) != 3 {
		t.Errorf("len(Phases) = %d, want 3", len(job.Phases))
	}

	// Marcar como completado
	time.Sleep(10 * time.Millisecond)
	job.MarkCompleted()

	if job.Status != JobStatusSuccess {
		t.Errorf("After MarkCompleted, Status = %v, want %v", job.Status, JobStatusSuccess)
	}

	if job.Duration <= 0 {
		t.Errorf("Duration = %v, expected > 0", job.Duration)
	}

	// Verificar que todas as fases estão presentes
	if !job.HasPhase(BuildPhaseGitSync) {
		t.Error("Missing BuildPhaseGitSync")
	}
	if !job.HasPhase(BuildPhaseNXBuild) {
		t.Error("Missing BuildPhaseNXBuild")
	}
	if !job.HasPhase(BuildPhaseImageBuild) {
		t.Error("Missing BuildPhaseImageBuild")
	}
}
