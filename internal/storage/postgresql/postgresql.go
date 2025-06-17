package postgresql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gxkxv/restapi-pet/internal/config"
	_ "github.com/lib/pq"
	"net/http"
)

type User struct {
	Name   string
	Age    int
	Gender string
	Nation string
}
type Storage struct {
	db *sql.DB
}

func New(cfg *config.Config) (*Storage, error) {
	const op = "storage.postgresql.New"

	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name, cfg.Database.SSLMode))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
		CREATE TABLE IF NOT EXISTS users (
			name TEXT,
			age INTEGER,
			gender TEXT,
			nation TEXT,
			PRIMARY KEY (name, age)
		)
	 `)

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

func (s *Storage) GetInfoFromURL(name string) error {
	person := User{Name: name}

	resp, err := http.Get(fmt.Sprintf("https://api.agify.io/?name=%s", name))
	if err != nil {
		return fmt.Errorf("get age: %w", err)
	}
	defer resp.Body.Close()

	var ageData struct {
		Age int `json:"age"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ageData); err != nil {
		return fmt.Errorf("decode age: %w", err)
	}
	person.Age = ageData.Age

	resp, err = http.Get(fmt.Sprintf("https://api.genderize.io/?name=%s", name))
	if err != nil {
		return fmt.Errorf("get gender: %w", err)
	}
	defer resp.Body.Close()

	var genderData struct {
		Gender string `json:"gender"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&genderData); err != nil {
		return fmt.Errorf("decode gender: %w", err)
	}
	person.Gender = genderData.Gender

	resp, err = http.Get(fmt.Sprintf("https://api.nationalize.io/?name=%s", name))
	if err != nil {
		return fmt.Errorf("get nationality: %w", err)
	}
	defer resp.Body.Close()

	var nationData struct {
		Country []struct {
			CountryID   string  `json:"country_id"`
			Probability float64 `json:"probability"`
		} `json:"country"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&nationData); err != nil {
		return fmt.Errorf("decode nationality: %w", err)
	}
	if len(nationData.Country) > 0 {
		person.Nation = nationData.Country[0].CountryID
	}

	stmt, err := s.db.Exec(fmt.Sprintf("INSERT INTO users(name,age,gender,nation) VALUES ('%s', %d, '%s', '%s')", person.Name, person.Age, person.Gender, person.Nation))
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	_ = stmt
	return nil
}
