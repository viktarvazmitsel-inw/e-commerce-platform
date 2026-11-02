package domain

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const maxPayloadBytes = 64 * 1024

var (
	emailRegex   = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	channelRegex = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
)

type Status string

const (
	StatusPending Status = "pending"
	StatusSent    Status = "sent"
	StatusFailed  Status = "failed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusPending, StatusSent, StatusFailed:
		return true
	default:
		return false
	}
}

type Notification struct {
	ID           string
	Channel      string
	Receiver     string
	Payload      string
	Status       Status
	SendAttempts int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewNotification(channel, receiver, payload string) (*Notification, error) {
	channel = strings.TrimSpace(channel)
	if !isChannelValid(channel) {
		return nil, ErrInvalidChannel
	}

	receiver = strings.TrimSpace(receiver)
	if !isReceiverValid(receiver) {
		return nil, ErrInvalidReceiver
	}

	payload = strings.TrimSpace(payload)
	if !isPayloadValid(payload) {
		return nil, ErrInvalidPayload
	}

	id, err := newID()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return &Notification{
		ID:           id,
		Channel:      channel,
		Receiver:     receiver,
		Payload:      payload,
		Status:       StatusPending,
		SendAttempts: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (n *Notification) SetStatus(status Status) error {
	if !status.Valid() {
		return ErrInvalidStatus
	}
	if n.Status != StatusPending || status == StatusPending {
		return ErrInvalidStatusTransition
	}

	n.Status = status
	n.UpdatedAt = time.Now().UTC()
	return nil
}

func (n *Notification) IncreaseSendAttempts() {
	n.SendAttempts++
	n.UpdatedAt = time.Now().UTC()
}

func isChannelValid(channel string) bool {
	return channelRegex.MatchString(channel)
}

func isReceiverValid(receiver string) bool {
	if len(receiver) < 3 || len(receiver) > 254 {
		return false
	}
	return emailRegex.MatchString(receiver)
}

func isPayloadValid(payload string) bool {
	if payload == "" || len(payload) > maxPayloadBytes {
		return false
	}
	if !json.Valid([]byte(payload)) {
		return false
	}
	switch payload[0] {
	case '{', '[':
		return true
	default:
		return false
	}
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate notification id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
