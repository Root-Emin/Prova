package usecase

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/domain/prova/model"
	infraAI "github.com/masterfabric-go/masterfabric/internal/infrastructure/ai"
)

// TestSubmitTurnPythonMockIntegration exercises the production path used by
// the session flow: use-case -> PersonaInferenceClient -> Go HTTP adapter ->
// the local Python mock provider. Repositories remain fakes so this test does
// not require MongoDB, while the network contract is real.
func TestSubmitTurnPythonMockIntegration(t *testing.T) {
	baseURL, stop := startPersonaMockService(t)
	defer stop()

	uc, repo, _, scope, employeeID, sessionID := newSessionFixture()
	uc.PersonaClient = infraAI.NewPersonaClient(baseURL, 2*time.Second)

	requestID := uuid.New()
	turn, err := uc.SubmitTurn(context.Background(), scope, employeeID, sessionID, requestID, "Siparişiniz için çözüm sürecini başlatıyorum.")
	if err != nil {
		t.Fatalf("integrated SubmitTurn() error = %v", err)
	}
	if turn.Text == "" || len(repo.turns) != 2 || repo.turns[0].Role != model.TurnRoleEmployee || repo.turns[1].Role != model.TurnRoleCharacter {
		t.Fatalf("integrated flow did not persist employee/persona pair: turn=%+v turns=%+v", turn, repo.turns)
	}
	if repo.turns[0].RequestID == nil || repo.turns[1].RequestID == nil || *repo.turns[0].RequestID != requestID || *repo.turns[1].RequestID != requestID {
		t.Fatal("integrated flow did not persist request id on both turns")
	}
}

func startPersonaMockService(t *testing.T) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve persona service port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test location")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(testFile), "../../../../.."))
	command := exec.Command("python3", "persona_service/server.py", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	command.Dir = filepath.Join(repoRoot, "backend", "ml")
	command.Env = append(os.Environ(), "AI_PERSONA_PROVIDER=mock", "PYTHONUNBUFFERED=1")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Start(); err != nil {
		t.Fatalf("start persona mock service: %v", err)
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{Timeout: 200 * time.Millisecond}
	ready := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response, requestErr := client.Get(baseURL + "/health")
		if requestErr == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("persona mock service did not become healthy")
	}

	return baseURL, func() {
		_ = command.Process.Kill()
		_ = command.Wait()
	}
}
