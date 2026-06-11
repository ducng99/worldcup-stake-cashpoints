package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Syncer struct {
	db     *sql.DB
	apiKey string
	push   *PushService
}

func NewSyncer(database *sql.DB, apiKey string, push *PushService) *Syncer {
	return &Syncer{db: database, apiKey: apiKey, push: push}
}

type teamInfo struct {
	ID   int64
	Name string
}

type apiResponse struct {
	Matches []apiMatch `json:"matches"`
}

type apiMatch struct {
	ID       int      `json:"id"`
	UtcDate  string   `json:"utcDate"`
	Status   string   `json:"status"`
	Stage    string   `json:"stage"`
	HomeTeam apiTeam  `json:"homeTeam"`
	AwayTeam apiTeam  `json:"awayTeam"`
	Score    apiScore `json:"score"`
}

type apiTeam struct {
	TLA string `json:"tla"`
}

type apiScore struct {
	FullTime apiScorePair `json:"fullTime"`
}

type apiScorePair struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

func (s *Syncer) Sync() {
	log.Println("Syncing match data from football-data.org...")

	req, err := http.NewRequest("GET", "https://api.football-data.org/v4/competitions/WC/matches", nil)
	if err != nil {
		log.Printf("Sync: failed to create request: %v", err)
		return
	}
	req.Header.Set("X-Auth-Token", s.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Sync: request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Sync: API returned status %d", resp.StatusCode)
		return
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Sync: failed to decode response: %v", err)
		return
	}

	teamMap, err := s.buildTeamMap()
	if err != nil {
		log.Printf("Sync: failed to build team map: %v", err)
		return
	}

	updated := 0
	for _, m := range result.Matches {
		if m.HomeTeam.TLA == "" || m.AwayTeam.TLA == "" {
			continue
		}
		home, homeOK := teamMap[m.HomeTeam.TLA]
		away, awayOK := teamMap[m.AwayTeam.TLA]
		if !homeOK || !awayOK {
			log.Printf("Sync: unknown team TLA home=%q away=%q, skipping match %d", m.HomeTeam.TLA, m.AwayTeam.TLA, m.ID)
			continue
		}
		matchID := fmt.Sprintf("%d", m.ID)
		previousStatus, err := s.previousMatchStatus(matchID)
		if err != nil {
			log.Printf("Sync: failed to fetch previous match %d status: %v", m.ID, err)
			continue
		}

		_, err = s.db.Exec(`
			INSERT INTO matches (id, home_team_id, away_team_id, home_score, away_score, status, match_date, stage)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				home_score = excluded.home_score,
				away_score = excluded.away_score,
				status     = excluded.status,
				match_date = excluded.match_date,
				stage      = excluded.stage
		`, matchID, home.ID, away.ID,
			m.Score.FullTime.Home, m.Score.FullTime.Away,
			m.Status, m.UtcDate, m.Stage)
		if err != nil {
			log.Printf("Sync: failed to upsert match %d: %v", m.ID, err)
			continue
		}
		if s.push != nil && !isLiveStatus(previousStatus) && isLiveStatus(m.Status) {
			s.push.NotifyMatchStart(matchID, home.ID, away.ID, home.Name, away.Name)
		}
		updated++
	}
	if s.push != nil {
		if err := s.notifyLeaderboardChanges(); err != nil {
			log.Printf("Sync: failed to process leaderboard notifications: %v", err)
		}
	}

	log.Printf("Sync: %d/%d matches upserted", updated, len(result.Matches))
}

func (s *Syncer) buildTeamMap() (map[string]teamInfo, error) {
	rows, err := s.db.Query("SELECT id, code, name FROM teams")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := map[string]teamInfo{}
	for rows.Next() {
		var id int64
		var code, name string
		if err := rows.Scan(&id, &code, &name); err != nil {
			return nil, err
		}
		m[code] = teamInfo{ID: id, Name: name}
	}
	return m, nil
}

func (s *Syncer) previousMatchStatus(matchID string) (string, error) {
	var status string
	err := s.db.QueryRow("SELECT status FROM matches WHERE id = ?", matchID).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return status, err
}

func isLiveStatus(status string) bool {
	switch status {
	case "IN_PLAY", "PAUSED", "LIVE":
		return true
	default:
		return false
	}
}

func (s *Syncer) notifyLeaderboardChanges() error {
	entries, err := ComputeLeaderboard(s.db)
	if err != nil {
		return err
	}

	var existingCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM leaderboard_state").Scan(&existingCount); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	initializing := existingCount == 0

	for _, entry := range entries {
		var oldRank, oldPoints int
		err := s.db.QueryRow("SELECT rank, points FROM leaderboard_state WHERE user_id = ?", entry.UserID).Scan(&oldRank, &oldPoints)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if !initializing && err == nil && oldRank != entry.Rank {
			s.push.NotifyLeaderboardChange(entry.UserID, oldRank, entry.Rank, entry.Points)
		}
		_, err = s.db.Exec(`
			INSERT INTO leaderboard_state (user_id, rank, points, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
				rank = excluded.rank,
				points = excluded.points,
				updated_at = excluded.updated_at
		`, entry.UserID, entry.Rank, entry.Points, now)
		if err != nil {
			return err
		}
	}

	return nil
}
