package server

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var TestTeams []Team = []Team{
	{bson.NewObjectId(), "blueteam", 1, "team1", "10.0.0.1", genPass("team1"), ""},
	{bson.NewObjectId(), "blueteam", 2, "team2", "10.0.0.2", genPass("team2"), ""},
}

func init() {
	SetupMongo(&DBSettings{URI: "mongodb://127.0.0.1:27017", DBName: "test"}, []string{})
	SetupScoringLoggers(&LogSettings{Level: "warn", Stdout: true})

	if DBSession() == nil {
		os.Exit(1)
	}

	cleanup()
}

func cleanup() {
	for _, c := range []*mgo.Collection{Teams(), Challenges(), DB().C("challenges")} {
		c.RemoveAll(nil)
	}
}

func genPass(s string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("Unable to create password from %q: %v", s, err)
		os.Exit(1)
	}
	return string(hash)
}

func TestDataGetTeams(t *testing.T) {
	var docs []interface{}
	for _, doc := range TestTeams {
		docs = append(docs, doc)
	}
	Teams().Insert(docs...)
	defer cleanup()

	for _, tt := range TestTeams {
		if tt.Group == "blueteam" {
			continue
		}

		var ok bool
		for _, dbTeam := range DataGetTeams() {
			if tt.Name == dbTeam.Name && tt.Number == dbTeam.Number {
				ok = true
			}
		}
		assert.True(t, ok, "Failed to retrieve blue team: %+v", tt)
	}
}

func TestDataAddTeams(t *testing.T) {
	assert.Nil(t, DataAddTeams(TestTeams))
	defer cleanup()

	dbTeams := []Team{}
	Teams().Find(nil).All(&dbTeams)
	for _, tt := range TestTeams {
		assert.Contains(t, dbTeams, tt, "Database did not contain the expected test team %+v", tt)
	}
}
