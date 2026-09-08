package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var automationCommandFile = os.Getenv("IKEMEN_AUTOMATION_FILE")
var automationAckFile = os.Getenv("IKEMEN_AUTOMATION_ACK_FILE")
var automationCommandOffset int

func automationAcknowledge(command string, err error) {
	if automationAckFile == "" {
		return
	}
	status := "OK"
	if err != nil {
		status = "ERROR: " + err.Error()
	}
	line := fmt.Sprintf("%s\t%s\t%s\n", time.Now().Format(time.RFC3339Nano), status, command)
	f, openErr := os.OpenFile(automationAckFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if openErr != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

func automationKey(name string) (Key, error) {
	trimmed := strings.TrimSpace(name)
	key := StringToKey(trimmed)
	if key == KeyUnknown {
		key = StringToKey(strings.ToUpper(trimmed))
	}
	if key == KeyUnknown {
		key = StringToKey(strings.ToLower(trimmed))
	}
	if key == KeyUnknown {
		return KeyUnknown, fmt.Errorf("unknown key %q", name)
	}
	return key, nil
}

func executeAutomationCommand(command string) error {
	if strings.EqualFold(strings.TrimSpace(command), "storyboard_cancel") {
		if !sys.storyboard.active {
			return fmt.Errorf("no active storyboard")
		}
		sys.storyboard.canceled = true
		return nil
	}
	fields := strings.Fields(command)
	if len(fields) != 2 {
		return fmt.Errorf("expected: tap|pulse|down|up KEY")
	}
	key, err := automationKey(fields[1])
	if err != nil {
		return err
	}
	switch strings.ToLower(fields[0]) {
	case "tap":
		if pending, ok := deferredKeyReleases[key]; ok {
			OnKeyReleased(key, pending.mod)
			delete(deferredKeyReleases, key)
		}
		OnKeyPressed(key, 0)
		OnKeyReleased(key, 0)
	case "pulse":
		if pending, ok := deferredKeyReleases[key]; ok {
			OnKeyReleased(key, pending.mod)
			delete(deferredKeyReleases, key)
		}
		OnKeyPressed(key, 0)
		deferredKeyReleases[key] = deferredKeyRelease{
			releaseAt: time.Now().Add(50 * time.Millisecond),
		}
	case "down":
		delete(deferredKeyReleases, key)
		OnKeyPressed(key, 0)
	case "up":
		delete(deferredKeyReleases, key)
		OnKeyReleased(key, 0)
	default:
		return fmt.Errorf("unknown action %q", fields[0])
	}
	return nil
}

func pollAutomationCommands() {
	pollAutomationHTTPInput()
	updateAutomationRuntimeState()
	if automationCommandFile == "" {
		return
	}
	data, err := os.ReadFile(automationCommandFile)
	if err != nil || len(data) <= automationCommandOffset {
		return
	}
	newData := string(data[automationCommandOffset:])
	automationCommandOffset = len(data)
	for _, raw := range strings.Split(newData, "\n") {
		command := strings.TrimSpace(raw)
		if command == "" {
			continue
		}
		err := executeAutomationCommand(command)
		automationAcknowledge(command, err)
	}
}
