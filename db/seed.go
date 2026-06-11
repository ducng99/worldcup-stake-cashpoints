package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type seedUser struct {
	Name  string   `json:"name"`
	Teams []string `json:"teams"`
}

type seedTeam struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func Seed(database *sql.DB) error {
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	allTeams, seedData, err := loadSeedData(DataDir())
	if err != nil {
		return err
	}

	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	teamIDs := map[string]int64{}
	for _, t := range allTeams {
		res, err := tx.Exec("INSERT INTO teams (name, code) VALUES (?, ?)", t.Name, t.Code)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		teamIDs[t.Code] = id
	}

	for _, u := range seedData {
		res, err := tx.Exec("INSERT INTO users (name) VALUES (?)", u.Name)
		if err != nil {
			return err
		}
		userID, _ := res.LastInsertId()

		for _, code := range u.Teams {
			teamID, ok := teamIDs[code]
			if !ok {
				return fmt.Errorf("seed user %q references unknown team code %q", u.Name, code)
			}
			if _, err := tx.Exec("INSERT INTO user_teams (user_id, team_id) VALUES (?, ?)", userID, teamID); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func loadSeedData(dataDir string) ([]seedTeam, []seedUser, error) {
	var teams []seedTeam
	if err := readJSON(filepath.Join(dataDir, "teams.json"), &teams); err != nil {
		return nil, nil, err
	}

	var users []seedUser
	if err := readJSON(filepath.Join(dataDir, "users.json"), &users); err != nil {
		return nil, nil, err
	}

	return teams, users, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed file %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("decode seed file %s: %w", path, err)
	}
	return nil
}
