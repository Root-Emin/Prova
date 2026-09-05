package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// startSession, yayınlanmış bir senaryodan oturum açar.
func (h *harness) startSession(token, scenarioLineage string) (string, error) {
	resp, err := h.query(token, fmt.Sprintf(
		`mutation { startSession(input:{scenarioLineageId:%q}) { id status } }`, scenarioLineage))
	if err != nil {
		return "", err
	}
	if len(resp.Errors) > 0 {
		return "", fmt.Errorf("oturum açılamadı: %s", resp.firstError())
	}

	var payload struct {
		StartSession struct {
			ID string `json:"id"`
		} `json:"startSession"`
	}
	if err := resp.decode(&payload); err != nil {
		return "", err
	}
	return payload.StartSession.ID, nil
}

// submitTurn, çalışanın mesajını gönderir.
func (h *harness) submitTurn(token, sessionID, text string) (gqlResponse, error) {
	return h.query(token, fmt.Sprintf(
		`mutation { submitTurn(input:{sessionId:%q, text:%q}) { index role text maskedFieldCount } }`,
		sessionID, text))
}

// endSession, oturumu bitirir ve puanlar.
func (h *harness) endSession(token, sessionID string) error {
	resp, err := h.query(token, fmt.Sprintf(
		`mutation { endSession(sessionId:%q) { id status } }`, sessionID))
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("oturum bitirilemedi: %s", resp.firstError())
	}
	return nil
}

// scoreView, doğrulamanın ilgilendiği puan alanları.
type scoreView struct {
	ID                  string   `json:"id"`
	Total               float64  `json:"total"`
	MaxTotal            float64  `json:"maxTotal"`
	Passed              bool     `json:"passed"`
	FailedMandatoryKeys []string `json:"failedMandatoryKeys"`
	Model               string   `json:"model"`
	Criteria            []struct {
		CriterionKey  string `json:"criterionKey"`
		Quote         string `json:"quote"`
		QuoteVerified bool   `json:"quoteVerified"`
		TurnIndex     *int   `json:"turnIndex"`
		Mandatory     bool   `json:"mandatory"`
	} `json:"criteria"`
}

// sessionScore, oturumun puanını okur.
func (h *harness) sessionScore(token, sessionID string) (scoreView, error) {
	resp, err := h.query(token, fmt.Sprintf(
		`{ session(id:%q) { score {
			id total maxTotal passed failedMandatoryKeys model
			criteria { criterionKey quote quoteVerified turnIndex mandatory }
		} } }`, sessionID))
	if err != nil {
		return scoreView{}, err
	}
	if len(resp.Errors) > 0 {
		return scoreView{}, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		Session struct {
			Score *scoreView `json:"score"`
		} `json:"session"`
	}
	if err := resp.decode(&payload); err != nil {
		return scoreView{}, err
	}
	if payload.Session.Score == nil {
		return scoreView{}, fmt.Errorf("oturumun puanı yok")
	}
	return *payload.Session.Score, nil
}

// sessionTurns, transkripti okur.
func (h *harness) sessionTurns(token, sessionID string) ([]turnView, error) {
	resp, err := h.query(token, fmt.Sprintf(
		`{ session(id:%q) { turns { index role text signals maskedFieldCount } } }`, sessionID))
	if err != nil {
		return nil, err
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("%s", resp.firstError())
	}

	var payload struct {
		Session struct {
			Turns []turnView `json:"turns"`
		} `json:"session"`
	}
	if err := resp.decode(&payload); err != nil {
		return nil, err
	}
	return payload.Session.Turns, nil
}

// turnView, konuşma sırasının doğrulamaya giren alanları.
type turnView struct {
	Index            int      `json:"index"`
	Role             string   `json:"role"`
	Text             string   `json:"text"`
	Signals          []string `json:"signals"`
	MaskedFieldCount int      `json:"maskedFieldCount"`
}

// playAndScore, tek konuşma sıralı bir oturum oynar ve puan kimliğini döndürür.
func (h *harness) playAndScore(token, scenario, message string) (sessionID, scoreID string, err error) {
	sessionID, err = h.startSession(token, scenario)
	if err != nil {
		return "", "", err
	}
	resp, err := h.submitTurn(token, sessionID, message)
	if err != nil {
		return "", "", err
	}
	if len(resp.Errors) > 0 {
		return "", "", fmt.Errorf("konuşma sırası gönderilemedi: %s", resp.firstError())
	}
	if err := h.endSession(token, sessionID); err != nil {
		return "", "", err
	}
	score, err := h.sessionScore(token, sessionID)
	if err != nil {
		return "", "", err
	}
	return sessionID, score.ID, nil
}

// subscribeSessionEvents, abonelik kanalına bağlanır ve gelen olayları sayar.
//
// Gerçek bir WebSocket el sıkışması yapılıyor ve connectionParams içinde
// token gönderiliyor: transport bağlanmamış olsaydı şema yine geçerli
// görünürdü, ve şemaya bakan bir kontrol bunu yakalayamazdı.
func (h *harness) subscribeSessionEvents(token, sessionID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	wsURL := strings.Replace(h.graphQL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		Subprotocols: []string{"graphql-transport-ws"},
		HTTPHeader:   http.Header{"Origin": []string{"http://localhost:3000"}},
	})
	if err != nil {
		return 0, fmt.Errorf("websocket bağlanamadı: %w", err)
	}
	defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()

	send := func(payload map[string]any) error {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return conn.Write(ctx, websocket.MessageText, encoded)
	}
	read := func() (map[string]any, error) {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return nil, err
		}
		var message map[string]any
		if err := json.Unmarshal(data, &message); err != nil {
			return nil, err
		}
		return message, nil
	}

	// connectionParams içindeki token doğrulanıyor.
	if err := send(map[string]any{
		"type":    "connection_init",
		"payload": map[string]any{"Authorization": "Bearer " + token},
	}); err != nil {
		return 0, err
	}
	ack, err := read()
	if err != nil {
		return 0, fmt.Errorf("connection_ack alınamadı: %w", err)
	}
	if ack["type"] != "connection_ack" {
		return 0, fmt.Errorf("connection_ack bekleniyordu, %v geldi", ack["type"])
	}

	if err := send(map[string]any{
		"id":   "1",
		"type": "subscribe",
		"payload": map[string]any{
			"query": fmt.Sprintf(
				`subscription { sessionEvents(sessionId:%q) { type sessionId text progress } }`, sessionID),
		},
	}); err != nil {
		return 0, err
	}

	// Abonelik kurulduktan sonra oturumu ilerletiyoruz; olaylar akmaya
	// başlamalı. Yayın tarafı abonelik açılmadan önce olayları üretirse
	// hiçbir şey gelmez, ve bu da kontrolün yakalaması gereken bir hata.
	errCh := make(chan error, 1)
	go func() {
		resp, err := h.submitTurn(token, sessionID, "Merhaba, size nasıl yardımcı olabilirim?")
		if err != nil {
			errCh <- err
			return
		}
		if len(resp.Errors) > 0 {
			errCh <- fmt.Errorf("%s", resp.firstError())
			return
		}
		errCh <- nil
	}()

	received := 0
	deadline := time.After(15 * time.Second)
	for received < 2 {
		type result struct {
			message map[string]any
			err     error
		}
		messageCh := make(chan result, 1)
		go func() {
			m, err := read()
			messageCh <- result{m, err}
		}()

		select {
		case r := <-messageCh:
			if r.err != nil {
				return received, fmt.Errorf("olay okunamadı: %w", r.err)
			}
			if r.message["type"] == "next" {
				received++
			}
			if r.message["type"] == "error" {
				return received, fmt.Errorf("abonelik hatası: %v", r.message["payload"])
			}
		case <-deadline:
			if turnErr := <-errCh; turnErr != nil {
				return received, fmt.Errorf("konuşma sırası gönderilemedi: %w", turnErr)
			}
			return received, fmt.Errorf("olay beklenirken zaman aşımı (%d olay alındı)", received)
		}
	}

	return received, nil
}
