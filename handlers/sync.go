package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Syncer struct {
	db     *sql.DB
	apiKey string
}

func NewSyncer(database *sql.DB, apiKey string) *Syncer {
	return &Syncer{db: database, apiKey: apiKey}
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
		homeID, homeOK := teamMap[m.HomeTeam.TLA]
		awayID, awayOK := teamMap[m.AwayTeam.TLA]
		if !homeOK || !awayOK {
			log.Printf("Sync: unknown team TLA home=%q away=%q, skipping match %d", m.HomeTeam.TLA, m.AwayTeam.TLA, m.ID)
			continue
		}

		_, err := s.db.Exec(`
			INSERT INTO matches (id, home_team_id, away_team_id, home_score, away_score, status, match_date, stage)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				home_score = excluded.home_score,
				away_score = excluded.away_score,
				status     = excluded.status,
				match_date = excluded.match_date,
				stage      = excluded.stage
		`, fmt.Sprintf("%d", m.ID), homeID, awayID,
			m.Score.FullTime.Home, m.Score.FullTime.Away,
			m.Status, m.UtcDate, m.Stage)
		if err != nil {
			log.Printf("Sync: failed to upsert match %d: %v", m.ID, err)
			continue
		}
		updated++
	}

	log.Printf("Sync: %d/%d matches upserted", updated, len(result.Matches))
}

func (s *Syncer) buildTeamMap() (map[string]int64, error) {
	rows, err := s.db.Query("SELECT id, code FROM teams")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := map[string]int64{}
	for rows.Next() {
		var id int64
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return nil, err
		}
		m[code] = id
	}
	return m, nil
}
