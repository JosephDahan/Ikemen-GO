package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const automationProtocolVersion = "3.14.0"

type automationQueuedCommand struct {
	command string
	done    chan error
}

type automationScreenshotResult struct {
	data []byte
	err  error
}

type automationScreenshotRequest struct {
	done chan automationScreenshotResult
}

type automationParticipant struct {
	Slot        int    `json:"slot"`
	Team        int    `json:"team"`
	Definition  string `json:"definition"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Palette     int32  `json:"palette"`
}

type automationFighter struct {
	Slot         int        `json:"slot"`
	Team         int        `json:"team"`
	Name         string     `json:"name"`
	DisplayName  string     `json:"display_name"`
	Alive        bool       `json:"alive"`
	Control      bool       `json:"control"`
	Life         int32      `json:"life"`
	LifeMax      int32      `json:"life_max"`
	Power        int32      `json:"power"`
	PowerMax     int32      `json:"power_max"`
	Position     [3]float32 `json:"position"`
	Velocity     [3]float32 `json:"velocity"`
	Facing       float32    `json:"facing"`
	StateNumber  int32      `json:"state_number"`
	StateTime    int32      `json:"state_time"`
	StateType    string     `json:"state_type"`
	MoveType     string     `json:"move_type"`
	Animation    int32      `json:"animation"`
	ReceivedHits int32      `json:"received_hits"`
	ReceivedDmg  int32      `json:"received_damage"`
	Guarding     bool       `json:"guarding"`
	HitPause     bool       `json:"hit_pause"`
	HelperCount  int        `json:"helper_count"`
}

type automationHelper struct {
	RootSlot    int        `json:"root_slot"`
	Index       int        `json:"index"`
	PlayerID    int32      `json:"player_id"`
	HelperID    int32      `json:"helper_id"`
	ParentID    int32      `json:"parent_id"`
	Team        int        `json:"team"`
	Name        string     `json:"name"`
	Alive       bool       `json:"alive"`
	Control     bool       `json:"control"`
	Position    [3]float32 `json:"position"`
	StateNumber int32      `json:"state_number"`
	StateTime   int32      `json:"state_time"`
	StateType   string     `json:"state_type"`
	MoveType    string     `json:"move_type"`
	Animation   int32      `json:"animation"`
}

type automationGlobalAssertSource struct {
	Flag        string `json:"flag"`
	RootSlot    int    `json:"root_slot"`
	Index       int    `json:"index"`
	PlayerID    int32  `json:"player_id"`
	HelperID    int32  `json:"helper_id"`
	ParentID    int32  `json:"parent_id"`
	Team        int    `json:"team"`
	Name        string `json:"name"`
	StateNumber int32  `json:"state_number"`
	StateTime   int32  `json:"state_time"`
}

type automationRenderState struct {
	Brightness            float32 `json:"brightness"`
	AllPalFXActive        bool    `json:"all_palfx_active"`
	BackgroundPalFXActive bool    `json:"background_palfx_active"`
	FadeInActive          bool    `json:"fade_in_active"`
	FadeInRemaining       int32   `json:"fade_in_remaining"`
	FadeOutActive         bool    `json:"fade_out_active"`
	FadeOutRemaining      int32   `json:"fade_out_remaining"`
	ShutterTimer          int32   `json:"shutter_timer"`
	EnvironmentColorTime  int32   `json:"environment_color_time"`
}

type automationCamera struct {
	Position       [2]float32 `json:"position"`
	ScreenPosition [2]float32 `json:"screen_position"`
	Offset         [2]float32 `json:"offset"`
	Scale          float32    `json:"scale"`
	MinScale       float32    `json:"min_scale"`
	XMin           float32    `json:"x_min"`
	XMax           float32    `json:"x_max"`
	ZoomEnabled    bool       `json:"zoom_enabled"`
	View           int        `json:"view"`
}

type automationSelectionCursor struct {
	Side          int    `json:"side"`
	Active        bool   `json:"active"`
	Player        int    `json:"player"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Cell          int    `json:"cell"`
	Reference     int    `json:"reference"`
	Definition    string `json:"definition"`
	Name          string `json:"name"`
	SelectedCount int    `json:"selected_count"`
	RequiredCount int    `json:"required_count"`
	PaletteMenu   bool   `json:"palette_menu"`
	PreloadStatus string `json:"preload_status"`
}

type automationStageCursor struct {
	Active            bool   `json:"active"`
	Index             int    `json:"index"`
	Reference         int    `json:"reference"`
	Definition        string `json:"definition"`
	Name              string `json:"name"`
	Random            bool   `json:"random"`
	PreloadStatus     string `json:"preload_status"`
	PortraitAvailable bool   `json:"portrait_available"`
}

type automationPauseMenu struct {
	Active            bool   `json:"active"`
	Root              string `json:"root"`
	Menu              string `json:"menu"`
	Item              string `json:"item"`
	Value             string `json:"value"`
	Special           string `json:"special"`
	MovelistCharacter string `json:"movelist_character"`
	MovelistIndex     int    `json:"movelist_index"`
	MovelistLine      int    `json:"movelist_line"`
	MovelistAvailable bool   `json:"movelist_available"`
	MovelistText      string `json:"movelist_text"`
	MovelistEntries   int    `json:"movelist_entries"`
}

type automationReplayState struct {
	MenuActive       bool   `json:"menu_active"`
	SelectedIndex    int    `json:"selected_index"`
	InventoryCount   int    `json:"inventory_count"`
	SelectedPath     string `json:"selected_path"`
	SelectedName     string `json:"selected_name"`
	PlaybackActive   bool   `json:"playback_active"`
	RecordingActive  bool   `json:"recording_active"`
	LocalRecording   bool   `json:"local_recording_supported"`
	CompatibilityTip string `json:"compatibility_tip"`
}

type automationInputConfig struct {
	ButtonAssist               bool    `json:"button_assist"`
	ButtonAssistScope          string  `json:"button_assist_scope"`
	SOCDResolution             int     `json:"socd_resolution"`
	ControllerStickSensitivity float32 `json:"controller_stick_sensitivity"`
	XInputTriggerSensitivity   float32 `json:"xinput_trigger_sensitivity"`
	UIRepeatDelay              int32   `json:"ui_repeat_delay"`
	UIRepeatRate               int32   `json:"ui_repeat_rate"`
	PauseExitDelay             int32   `json:"pause_exit_delay"`
}

type automationReplayInventoryItem struct {
	Name               string `json:"name"`
	Path               string `json:"path"`
	Size               int64  `json:"size"`
	ModifiedAt         string `json:"modified_at"`
	HeaderValid        bool   `json:"header_valid"`
	FormatVersion      uint16 `json:"format_version,omitempty"`
	SyncVersion        uint16 `json:"sync_version,omitempty"`
	SyncCompatible     bool   `json:"sync_compatible"`
	StrictSettingCount int    `json:"strict_setting_count,omitempty"`
	HostSettingCount   int    `json:"host_setting_count,omitempty"`
	ContentFingerprint string `json:"content_fingerprint,omitempty"`
	Error              string `json:"error,omitempty"`
}

type automationRuntimeState struct {
	Screen           string                         `json:"screen"`
	SystemScript     string                         `json:"system_script"`
	Common1Override  string                         `json:"common1_override"`
	GameMode         string                         `json:"game_mode"`
	GameRunning      bool                           `json:"game_running"`
	Paused           bool                           `json:"paused"`
	RenderWidth      int32                          `json:"render_width"`
	RenderHeight     int32                          `json:"render_height"`
	FPS              float32                        `json:"fps"`
	Turbo            float32                        `json:"turbo"`
	MatchNumber      int32                          `json:"match_number"`
	RoundNumber      int32                          `json:"round_number"`
	RoundState       int32                          `json:"round_state"`
	GameTime         int32                          `json:"game_time"`
	MatchTime        int32                          `json:"match_time"`
	CurrentRoundTime int32                          `json:"current_round_time"`
	TeamModes        [2]string                      `json:"team_modes"`
	TeamSizes        [2]int32                       `json:"team_sizes"`
	StageDefinition  string                         `json:"stage_definition"`
	StageName        string                         `json:"stage_name"`
	Participants     []automationParticipant        `json:"participants"`
	SelectionCursors [2]automationSelectionCursor   `json:"selection_cursors"`
	StageCursor      automationStageCursor          `json:"stage_cursor"`
	PauseMenu        automationPauseMenu            `json:"pause_menu"`
	Replay           automationReplayState          `json:"replay"`
	InputConfig      automationInputConfig          `json:"input_config"`
	Fighters         []automationFighter            `json:"fighters"`
	Helpers          []automationHelper             `json:"helpers"`
	GlobalAssertMask uint32                         `json:"global_assert_mask"`
	GlobalAsserts    []string                       `json:"global_asserts"`
	GlobalSources    []automationGlobalAssertSource `json:"global_assert_sources"`
	Render           automationRenderState          `json:"render"`
	ComboCounts      [2]int32                       `json:"combo_counts"`
	WinnerTeam       int                            `json:"winner_team"`
	FinishReason     string                         `json:"finish_reason"`
	WinTypes         [2]string                      `json:"win_types"`
	Camera           automationCamera               `json:"camera"`
	UpdatedAt        string                         `json:"updated_at"`
}

type automationEvent struct {
	ID        uint64 `json:"id"`
	Timestamp string `json:"timestamp"`
	Kind      string `json:"kind"`
	Detail    string `json:"detail"`
}

var automationHTTPStartedAt = time.Now()
var automationHTTPAddress string
var automationHTTPInput = make(chan automationQueuedCommand, 64)
var automationHTTPScreenshot = make(chan automationScreenshotRequest, 8)
var automationRuntimeMu sync.RWMutex
var automationRuntime = automationRuntimeState{Screen: "startup"}
var automationEventMu sync.RWMutex
var automationEvents = make([]automationEvent, 0, 256)
var automationEventID uint64

func automationEnabled() bool {
	if os.Getenv("IKEMEN_AUTOMATION_DISCOVERY_FILE") != "" {
		return true
	}
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(filepath.Base(executable)), "native-control")
}

func automationDiscoveryFilePath() (string, error) {
	if configured := os.Getenv("IKEMEN_AUTOMATION_DISCOVERY_FILE"); configured != "" {
		return configured, nil
	}
	if !automationEnabled() {
		return "", nil
	}
	automationRoot := filepath.Join(sys.baseDir, "save", "automation")
	if err := os.MkdirAll(automationRoot, 0o755); err != nil {
		return "", fmt.Errorf("create automation directory: %w", err)
	}
	name := fmt.Sprintf("native-%s-%d.endpoint.json", time.Now().Format("20060102-150405-000"), os.Getpid())
	return filepath.Join(automationRoot, name), nil
}

func recordAutomationEvent(kind, detail string) {
	automationEventMu.Lock()
	automationEventID++
	automationEvents = append(automationEvents, automationEvent{
		ID:        automationEventID,
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Kind:      kind,
		Detail:    detail,
	})
	if len(automationEvents) > 500 {
		automationEvents = append([]automationEvent(nil), automationEvents[len(automationEvents)-500:]...)
	}
	automationEventMu.Unlock()
}

func setAutomationScreen(screen string) {
	automationRuntimeMu.Lock()
	automationRuntime.Screen = screen
	automationRuntime.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	automationRuntimeMu.Unlock()
	recordAutomationEvent("screen", screen)
}
func setAutomationSelectionCursor(side, player, x, y, cell, reference int, definition, name string, selectedCount, requiredCount int, paletteMenu, active bool, preloadStatus string) {
	if side < 1 || side > 2 {
		return
	}
	cursor := automationSelectionCursor{
		Side:          side,
		Active:        active,
		Player:        player,
		X:             x,
		Y:             y,
		Cell:          cell,
		Reference:     reference,
		Definition:    definition,
		Name:          name,
		SelectedCount: selectedCount,
		RequiredCount: requiredCount,
		PaletteMenu:   paletteMenu,
		PreloadStatus: preloadStatus,
	}
	automationRuntimeMu.Lock()
	automationRuntime.SelectionCursors[side-1] = cursor
	automationRuntime.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	automationRuntimeMu.Unlock()
}

func setAutomationStageCursor(index, reference int, definition, name string, random, active bool, preloadStatus string, portraitAvailable bool) {
	automationRuntimeMu.Lock()
	automationRuntime.StageCursor = automationStageCursor{
		Active:            active,
		Index:             index,
		Reference:         reference,
		Definition:        definition,
		Name:              name,
		Random:            random,
		PreloadStatus:     preloadStatus,
		PortraitAvailable: portraitAvailable,
	}
	automationRuntime.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	automationRuntimeMu.Unlock()
}

func setAutomationPauseMenu(active bool, root, menu, item, value, special, movelistCharacter string, movelistIndex, movelistLine int, movelistAvailable bool, movelistText string, movelistEntries int) {
	automationRuntimeMu.Lock()
	automationRuntime.PauseMenu = automationPauseMenu{
		Active:            active,
		Root:              root,
		Menu:              menu,
		Item:              item,
		Value:             value,
		Special:           special,
		MovelistCharacter: movelistCharacter,
		MovelistIndex:     movelistIndex,
		MovelistLine:      movelistLine,
		MovelistAvailable: movelistAvailable,
		MovelistText:      movelistText,
		MovelistEntries:   movelistEntries,
	}
	automationRuntime.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	automationRuntimeMu.Unlock()
}

func setAutomationReplayMenu(active bool, selectedIndex, inventoryCount int, selectedPath, selectedName string) {
	automationRuntimeMu.Lock()
	enteringReplayMenu := active && (!automationRuntime.Replay.MenuActive || automationRuntime.Screen != "replay_menu")
	automationRuntime.Replay.MenuActive = active
	automationRuntime.Replay.SelectedIndex = selectedIndex
	automationRuntime.Replay.InventoryCount = inventoryCount
	automationRuntime.Replay.SelectedPath = selectedPath
	automationRuntime.Replay.SelectedName = selectedName
	automationRuntime.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	automationRuntimeMu.Unlock()
	if enteringReplayMenu {
		setAutomationScreen("replay_menu")
	}
}

func automationTeamModeName(mode TeamMode) string {
	switch mode {
	case TM_Single:
		return "single"
	case TM_Simul:
		return "simul"
	case TM_Turns:
		return "turns"
	case TM_Tag:
		return "tag"
	default:
		return fmt.Sprintf("unknown:%d", mode)
	}
}

func automationStateTypeName(stateType StateType) string {
	switch stateType {
	case ST_S:
		return "standing"
	case ST_C:
		return "crouching"
	case ST_A:
		return "airborne"
	case ST_L:
		return "lying"
	case ST_N:
		return "none"
	case ST_U:
		return "unchanged"
	default:
		return fmt.Sprintf("unknown:%d", stateType)
	}
}

func automationMoveTypeName(moveType MoveType) string {
	switch moveType {
	case MT_I:
		return "idle"
	case MT_A:
		return "attack"
	case MT_H:
		return "hit"
	case MT_U:
		return "unchanged"
	default:
		return fmt.Sprintf("unknown:%d", moveType)
	}
}

func automationFinishTypeName(finishType FinishType) string {
	switch finishType {
	case FT_NotYet:
		return "none"
	case FT_KO:
		return "ko"
	case FT_DKO:
		return "double_ko"
	case FT_TO:
		return "time_over"
	case FT_TODraw:
		return "time_over_draw"
	default:
		return fmt.Sprintf("unknown:%d", finishType)
	}
}

func automationWinTypeName(winType WinType) string {
	prefix := ""
	switch {
	case winType >= WT_CNormal:
		prefix = "clutch_"
		winType -= WT_CNormal - WT_Normal
	case winType >= WT_PNormal:
		prefix = "perfect_"
		winType -= WT_PNormal - WT_Normal
	}
	switch winType {
	case WT_Normal:
		return prefix + "normal"
	case WT_Special:
		return prefix + "special"
	case WT_Hyper:
		return prefix + "hyper"
	case WT_Cheese:
		return prefix + "cheese"
	case WT_Time:
		return prefix + "time"
	case WT_Throw:
		return prefix + "throw"
	case WT_Suicide:
		return prefix + "suicide"
	case WT_Teammate:
		return prefix + "teammate"
	default:
		return fmt.Sprintf("unknown:%d", winType)
	}
}

func automationGlobalSpecialFlagName(flag GlobalSpecialFlag) string {
	switch flag {
	case GSF_globalnoko:
		return "globalnoko"
	case GSF_globalnoshadow:
		return "globalnoshadow"
	case GSF_intro:
		return "intro"
	case GSF_nobardisplay:
		return "nobardisplay"
	case GSF_nobg:
		return "nobg"
	case GSF_nofg:
		return "nofg"
	case GSF_nokoslow:
		return "nokoslow"
	case GSF_nokosnd:
		return "nokosnd"
	case GSF_nomusic:
		return "nomusic"
	case GSF_roundnotover:
		return "roundnotover"
	case GSF_timerfreeze:
		return "timerfreeze"
	case GSF_camerafreeze:
		return "camerafreeze"
	case GSF_notimedisplay:
		return "notimedisplay"
	case GSF_roundfreeze:
		return "roundfreeze"
	case GSF_roundnotskip:
		return "roundnotskip"
	case GSF_skipfightdisplay:
		return "skipfightdisplay"
	case GSF_skipkodisplay:
		return "skipkodisplay"
	case GSF_skiprounddisplay:
		return "skiprounddisplay"
	case GSF_skipwindisplay:
		return "skipwindisplay"
	default:
		return fmt.Sprintf("unknown:%d", flag)
	}
}

func updateAutomationRuntimeState() {
	automationRuntimeMu.Lock()
	enteredFight := false
	automationRuntime.SystemScript = sys.cfg.Config.System
	automationRuntime.Common1Override = nativeCommon1Path
	automationRuntime.GameMode = sys.gameMode
	automationRuntime.GameRunning = sys.gameRunning
	automationRuntime.Paused = sys.paused
	automationRuntime.RenderWidth = sys.scrrect[2]
	automationRuntime.RenderHeight = sys.scrrect[3]
	automationRuntime.FPS = sys.gameFPS
	automationRuntime.Turbo = sys.turbo
	automationRuntime.MatchNumber = sys.matchNo
	automationRuntime.RoundNumber = sys.roundNo
	automationRuntime.RoundState = 0
	automationRuntime.GameTime = 0
	if sys.gameRunning {
		automationRuntime.RoundState = sys.roundState()
		automationRuntime.GameTime = sys.gameTime()
	}
	automationRuntime.MatchTime = sys.matchTime
	automationRuntime.CurrentRoundTime = sys.curRoundTime
	automationRuntime.InputConfig = automationInputConfig{
		ButtonAssist:               sys.cfg.Input.ButtonAssist,
		ButtonAssistScope:          "one-frame multi-button chord leniency; not motion simplification or auto-combo",
		SOCDResolution:             sys.cfg.Input.SOCDResolution,
		ControllerStickSensitivity: sys.cfg.Input.ControllerStickSensitivity,
		XInputTriggerSensitivity:   sys.cfg.Input.XinputTriggerSensitivity,
		UIRepeatDelay:              sys.cfg.Input.UiRepeatDelay,
		UIRepeatRate:               sys.cfg.Input.UiRepeatRate,
		PauseExitDelay:             sys.cfg.Input.PauseExitDelay,
	}
	automationRuntime.Replay.PlaybackActive = sys.replayFile != nil
	automationRuntime.Replay.RecordingActive = (sys.netConnection != nil && sys.netConnection.recording != nil) ||
		(sys.rollback.session != nil && sys.rollback.session.recording != nil)
	automationRuntime.Replay.LocalRecording = false
	automationRuntime.Replay.CompatibilityTip = "Replay headers are versioned and tied to synchronized settings/content; invalid or incompatible files are reported, never auto-loaded."
	automationRuntime.TeamModes = [2]string{
		automationTeamModeName(sys.tmode[0]),
		automationTeamModeName(sys.tmode[1]),
	}
	automationRuntime.TeamSizes = [2]int32{1, 1}
	for side := 0; side < 2; side++ {
		switch sys.tmode[side] {
		case TM_Simul, TM_Tag:
			automationRuntime.TeamSizes[side] = sys.numSimul[side]
		case TM_Turns:
			automationRuntime.TeamSizes[side] = sys.numTurns[side]
		}
	}
	automationRuntime.StageDefinition = ""
	automationRuntime.StageName = ""
	if sys.stage != nil {
		automationRuntime.StageDefinition = sys.stage.def
		automationRuntime.StageName = sys.stage.displayname
	}
	participants := make([]automationParticipant, 0, MaxPlayerNo)
	for slot := range sys.cgi {
		if sys.cgi[slot].def == "" {
			continue
		}
		participants = append(participants, automationParticipant{
			Slot:        slot + 1,
			Team:        slot%2 + 1,
			Definition:  sys.cgi[slot].def,
			Name:        sys.cgi[slot].name,
			DisplayName: sys.cgi[slot].displayname,
			Palette:     sys.cgi[slot].palno,
		})
	}
	automationRuntime.Participants = participants
	fighters := make([]automationFighter, 0, MaxPlayerNo)
	for slot := range sys.chars {
		if len(sys.chars[slot]) == 0 || sys.chars[slot][0] == nil {
			continue
		}
		character := sys.chars[slot][0]
		displayName := character.name
		if slot < len(sys.cgi) && sys.cgi[slot].displayname != "" {
			displayName = sys.cgi[slot].displayname
		}
		fighters = append(fighters, automationFighter{
			Slot:         slot + 1,
			Team:         character.teamside + 1,
			Name:         character.name,
			DisplayName:  displayName,
			Alive:        character.alive(),
			Control:      character.scf(SCF_ctrl),
			Life:         character.life,
			LifeMax:      character.lifeMax,
			Power:        character.getPower(),
			PowerMax:     character.powerMax,
			Position:     character.pos,
			Velocity:     character.vel,
			Facing:       character.facing,
			StateNumber:  character.ss.no,
			StateTime:    character.ss.time,
			StateType:    automationStateTypeName(character.ss.stateType),
			MoveType:     automationMoveTypeName(character.ss.moveType),
			Animation:    character.animNo,
			ReceivedHits: character.receivedHits,
			ReceivedDmg:  character.receivedDmg,
			Guarding:     character.inguarddist || character.inGuardState(),
			HitPause:     character.acttmp == -1,
			HelperCount:  Max(0, len(sys.chars[slot])-1),
		})
	}
	automationRuntime.Fighters = fighters
	helpers := make([]automationHelper, 0)
	for slot := range sys.chars {
		for index := 1; index < len(sys.chars[slot]); index++ {
			helper := sys.chars[slot][index]
			if helper == nil || helper.csf(CSF_destroy) {
				continue
			}
			helpers = append(helpers, automationHelper{
				RootSlot:    slot + 1,
				Index:       index,
				PlayerID:    helper.id,
				HelperID:    helper.helperId,
				ParentID:    helper.parentId,
				Team:        helper.teamside + 1,
				Name:        helper.name,
				Alive:       helper.alive(),
				Control:     helper.scf(SCF_ctrl),
				Position:    helper.pos,
				StateNumber: helper.ss.no,
				StateTime:   helper.ss.time,
				StateType:   automationStateTypeName(helper.ss.stateType),
				MoveType:    automationMoveTypeName(helper.ss.moveType),
				Animation:   helper.animNo,
			})
		}
	}
	automationRuntime.Helpers = helpers
	automationRuntime.GlobalAssertMask = uint32(sys.specialFlag)
	globalAsserts := make([]string, 0)
	for flag := GlobalSpecialFlag(1); flag <= GSF_skipwindisplay; flag <<= 1 {
		if sys.gsf(flag) {
			globalAsserts = append(globalAsserts, automationGlobalSpecialFlagName(flag))
		}
	}
	automationRuntime.GlobalAsserts = globalAsserts
	globalSources := make([]automationGlobalAssertSource, 0, len(sys.specialFlagSources))
	for _, source := range sys.specialFlagSources {
		if !sys.gsf(source.flag) {
			continue
		}
		globalSources = append(globalSources, automationGlobalAssertSource{
			Flag:        automationGlobalSpecialFlagName(source.flag),
			RootSlot:    source.playerNo + 1,
			Index:       source.helperIndex,
			PlayerID:    source.playerID,
			HelperID:    source.helperID,
			ParentID:    source.parentID,
			Team:        source.team + 1,
			Name:        source.name,
			StateNumber: source.stateNumber,
			StateTime:   source.stateTime,
		})
	}
	automationRuntime.GlobalSources = globalSources
	renderState := automationRenderState{
		Brightness:            sys.brightness,
		AllPalFXActive:        sys.allPalFX != nil && sys.allPalFX.enable,
		BackgroundPalFXActive: sys.bgPalFX != nil && sys.bgPalFX.enable,
		EnvironmentColorTime:  sys.envcol_time,
	}
	if sys.fightScreen.round != nil {
		renderState.ShutterTimer = sys.fightScreen.round.shutterTimer
		if sys.fightScreen.round.fadeIn != nil {
			renderState.FadeInActive = sys.fightScreen.round.fadeIn.isActive()
			renderState.FadeInRemaining = sys.fightScreen.round.fadeIn.timeRemaining
		}
		if sys.fightScreen.round.fadeOut != nil {
			renderState.FadeOutActive = sys.fightScreen.round.fadeOut.isActive()
			renderState.FadeOutRemaining = sys.fightScreen.round.fadeOut.timeRemaining
		}
	}
	automationRuntime.Render = renderState
	if automationRuntime.Screen == "loading" && len(fighters) > 0 {
		automationRuntime.Screen = "fight"
		enteredFight = true
	}
	automationRuntime.ComboCounts = sys.comboCount
	automationRuntime.WinnerTeam = 0
	if sys.finishType != FT_NotYet && sys.winTeam >= 0 {
		automationRuntime.WinnerTeam = sys.winTeam + 1
	}
	automationRuntime.FinishReason = automationFinishTypeName(sys.finishType)
	automationRuntime.WinTypes = [2]string{
		automationWinTypeName(sys.winType[0]),
		automationWinTypeName(sys.winType[1]),
	}
	automationRuntime.Camera = automationCamera{
		Position:       sys.cam.Pos,
		ScreenPosition: sys.cam.ScreenPos,
		Offset:         sys.cam.Offset,
		Scale:          sys.cam.Scale,
		MinScale:       sys.cam.MinScale,
		XMin:           sys.cam.XMin,
		XMax:           sys.cam.XMax,
		ZoomEnabled:    sys.cam.zoomEnabled(),
		View:           int(sys.cam.View),
	}
	automationRuntime.UpdatedAt = time.Now().Format(time.RFC3339Nano)
	automationRuntimeMu.Unlock()
	if enteredFight {
		recordAutomationEvent("screen", "fight")
	}
}

func automationJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func automationHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	automationJSON(w, http.StatusOK, map[string]any{
		"ok":               true,
		"protocol_version": automationProtocolVersion,
		"system_script":    sys.cfg.Config.System,
		"common1_override": nativeCommon1Path,
		"address":          automationHTTPAddress,
		"pid":              os.Getpid(),
		"started_at":       automationHTTPStartedAt.Format(time.RFC3339Nano),
	})
}

func automationCapabilitiesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	keys := make([]string, 0, len(StringToKeyLUT))
	for key := range StringToKeyLUT {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	automationJSON(w, http.StatusOK, map[string]any{
		"protocol_version": automationProtocolVersion,
		"transport":        "HTTP JSON over ephemeral IPv4 loopback",
		"binding":          "127.0.0.1 only",
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/health", "purpose": "Liveness, process identity, and protocol version"},
			{"method": "GET", "path": "/capabilities", "purpose": "Machine-readable control and observability contract"},
			{"method": "GET", "path": "/state", "purpose": "Latest semantic UI and match telemetry snapshot"},
			{"method": "GET", "path": "/events?after={id}", "purpose": "Ordered semantic screen, input, and screenshot events"},
			{"method": "GET", "path": "/replays", "purpose": "Read-only replay inventory and header compatibility metadata"},
			{"method": "POST", "path": "/input", "purpose": "Queue a validated input action on the engine main thread"},
			{"method": "GET", "path": "/screenshot", "purpose": "Capture the next rendered frame as PNG"},
		},
		"input": map[string]any{
			"actions":              []string{"tap", "pulse", "down", "up", "storyboard_cancel"},
			"keys":                 keys,
			"pulse_duration_ms":    50,
			"main_thread_queue":    true,
			"physical_layout_free": true,
		},
		"events": map[string]any{
			"kinds":          []string{"server", "screen", "input", "input_error", "screenshot", "screenshot_error"},
			"retained_count": 500,
			"ordered_by":     "id",
		},
		"state": map[string]any{
			"semantic_screen_from_lua":       true,
			"match_telemetry":                true,
			"loaded_participants":            true,
			"live_fighter_telemetry":         true,
			"live_helper_telemetry":          true,
			"live_global_assert_attribution": true,
			"live_render_readiness":          true,
			"winner_and_finish_reason":       true,
			"combo_counts":                   true,
			"camera_telemetry":               true,
			"stage_identity":                 true,
			"selection_cursor_identity":      true,
			"stage_cursor_identity":          true,
			"pause_menu_state":               true,
			"movelist_metadata":              true,
			"replay_state":                   true,
			"input_config":                   true,
			"render_metrics":                 true,
		},
		"limits": map[string]int{
			"input_queue":      cap(automationHTTPInput),
			"screenshot_queue": cap(automationHTTPScreenshot),
		},
	})
}

func automationStateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	automationRuntimeMu.RLock()
	state := automationRuntime
	automationRuntimeMu.RUnlock()
	automationJSON(w, http.StatusOK, state)
}

func automationEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
	automationEventMu.RLock()
	events := make([]automationEvent, 0, len(automationEvents))
	for _, event := range automationEvents {
		if event.ID > after {
			events = append(events, event)
		}
	}
	automationEventMu.RUnlock()
	automationJSON(w, http.StatusOK, map[string]any{"events": events})
}

func automationReplaysHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	replayRoot := filepath.Join(sys.baseDir, "save", "replays")
	entries, err := os.ReadDir(replayRoot)
	if err != nil {
		automationJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	items := make([]automationReplayInventoryItem, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".replay") {
			continue
		}
		path := filepath.Join(replayRoot, entry.Name())
		item := automationReplayInventoryItem{
			Name: entry.Name(),
			Path: filepath.ToSlash(filepath.Join("save", "replays", entry.Name())),
		}
		if info, infoErr := entry.Info(); infoErr == nil {
			item.Size = info.Size()
			item.ModifiedAt = info.ModTime().Format(time.RFC3339Nano)
		} else {
			item.Error = infoErr.Error()
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			item.Error = openErr.Error()
			items = append(items, item)
			continue
		}
		header, headerErr := readReplayHeader(file)
		_ = file.Close()
		if headerErr != nil {
			item.Error = headerErr.Error()
			items = append(items, item)
			continue
		}
		item.HeaderValid = true
		item.FormatVersion = header.FormatVersion
		item.SyncVersion = header.SyncVersion
		item.SyncCompatible = header.SyncVersion == syncConfigVersion
		item.StrictSettingCount = len(header.Strict)
		item.HostSettingCount = len(header.Host)
		item.ContentFingerprint = header.ContentFingerprint
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	automationJSON(w, http.StatusOK, map[string]any{
		"count":                     len(items),
		"items":                     items,
		"local_recording_supported": false,
		"mutation_endpoints":        []string{},
	})
}

func automationInputHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST required"})
		return
	}
	var body struct {
		Action string `json:"action"`
		Key    string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		automationJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	command := strings.TrimSpace(body.Action) + " " + strings.TrimSpace(body.Key)
	done := make(chan error, 1)
	select {
	case automationHTTPInput <- automationQueuedCommand{command: command, done: done}:
	case <-time.After(time.Second):
		automationJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "input queue full"})
		return
	}
	select {
	case err := <-done:
		if err != nil {
			automationJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		automationJSON(w, http.StatusOK, map[string]any{"ok": true, "command": command})
	case <-time.After(2 * time.Second):
		automationJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "engine did not process input"})
	}
}

func automationScreenshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		automationJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET required"})
		return
	}
	done := make(chan automationScreenshotResult, 1)
	select {
	case automationHTTPScreenshot <- automationScreenshotRequest{done: done}:
	case <-time.After(time.Second):
		automationJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "screenshot queue full"})
		return
	}
	select {
	case result := <-done:
		if result.err != nil {
			automationJSON(w, http.StatusInternalServerError, map[string]string{"error": result.err.Error()})
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(result.data)
	case <-time.After(5 * time.Second):
		automationJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "engine did not render a screenshot"})
	}
}

func startAutomationHTTPServer() error {
	discoveryFile, err := automationDiscoveryFilePath()
	if err != nil {
		return err
	}
	if discoveryFile == "" {
		return nil
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	automationHTTPAddress = "http://" + listener.Addr().String()
	discovery := map[string]any{
		"base_url":          automationHTTPAddress,
		"pid":               os.Getpid(),
		"protocol_version":  automationProtocolVersion,
		"capabilities_path": "/capabilities",
	}
	data, err := json.Marshal(discovery)
	if err != nil {
		_ = listener.Close()
		return err
	}
	f, err := os.OpenFile(discoveryFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("create automation discovery file: %w", err)
	}
	if _, err = f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		_ = listener.Close()
		return err
	}
	_ = f.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", automationHealthHandler)
	mux.HandleFunc("/capabilities", automationCapabilitiesHandler)
	mux.HandleFunc("/state", automationStateHandler)
	mux.HandleFunc("/events", automationEventsHandler)
	mux.HandleFunc("/replays", automationReplaysHandler)
	mux.HandleFunc("/input", automationInputHandler)
	mux.HandleFunc("/screenshot", automationScreenshotHandler)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 2 * time.Second}
	go func() { _ = server.Serve(listener) }()
	recordAutomationEvent("server", automationHTTPAddress)
	return nil
}

func pollAutomationHTTPInput() {
	for {
		select {
		case request := <-automationHTTPInput:
			err := executeAutomationCommand(request.command)
			if err != nil {
				recordAutomationEvent("input_error", request.command+": "+err.Error())
			} else {
				recordAutomationEvent("input", request.command)
			}
			request.done <- err
		default:
			return
		}
	}
}

func serviceAutomationScreenshotRequests() {
	requests := make([]automationScreenshotRequest, 0, len(automationHTTPScreenshot))
	for {
		select {
		case request := <-automationHTTPScreenshot:
			requests = append(requests, request)
		default:
			if len(requests) == 0 {
				return
			}
			data, err := captureScreenPNG()
			if err != nil {
				recordAutomationEvent("screenshot_error", err.Error())
			} else {
				recordAutomationEvent("screenshot", fmt.Sprintf("%d bytes", len(data)))
			}
			for _, request := range requests {
				request.done <- automationScreenshotResult{data: data, err: err}
			}
			return
		}
	}
}
