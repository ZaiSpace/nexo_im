package sdk

import (
	"testing"

	"github.com/ZaiSpace/nexo_im/common"
)

func TestMGetActorFromUserIds(t *testing.T) {
	input := []string{"u___12", "ag__34"}

	actors, err := MGetActorFromUserIds(input)
	if err != nil {
		t.Fatalf("MGetActorFromUserIds() error = %v", err)
	}

	if len(actors) != 2 {
		t.Fatalf("len(actors) = %d, want 2", len(actors))
	}

	if actors[0].Role != common.RoleUser || actors[0].Id != 12 {
		t.Fatalf("first actor = %+v, want role=user id=12", actors[0])
	}
	if actors[1].Role != common.RoleAgent || actors[1].Id != 34 {
		t.Fatalf("second actor = %+v, want role=agent id=34", actors[1])
	}
}

func TestMGetActorFromUserIds_InvalidInput(t *testing.T) {
	_, err := MGetActorFromUserIds([]string{"u___1", "bad"})
	if err == nil {
		t.Fatal("MGetActorFromUserIds() error = nil, want non-nil")
	}
}
