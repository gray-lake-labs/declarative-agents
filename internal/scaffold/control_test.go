package scaffold

import (
	"reflect"
	"strings"
	"testing"

	"github.com/gray-lake-labs/scrum-master/internal/beads"
	"github.com/gray-lake-labs/scrum-master/internal/program"
)

// StatesDocument round-trips: the beads parser reads back exactly the rows
// init writes, so control/states.md and the embedded default are one table
// (GH-130; srd041 R8, srd043 R1.3).
func TestStatesDocumentRoundTrips(t *testing.T) {
	rows, err := beads.ParseStates([]byte(StatesDocument(beads.Table)))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows, beads.Table) {
		t.Fatalf("parsed rows differ from the table:\n%v\n%v", rows, beads.Table)
	}
}

// The policy document lists the admission rules, and the policy reader
// takes them back, so removing one from the file disables it (GH-130).
func TestPolicyDocumentListsTheAdmissionRules(t *testing.T) {
	doc := PolicyDocument(program.Defaults())
	for _, name := range program.RuleNames() {
		if !strings.Contains(doc, "- "+name+"\n") {
			t.Errorf("policy document misses rule %s", name)
		}
	}
}
