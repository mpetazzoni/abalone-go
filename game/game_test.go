package game

import (
	"errors"
	"testing"
)

// --- Board Initialization Tests ---

func TestNewBoard_CellCount(t *testing.T) {
	b := NewBoard()
	if len(b.Cells) != 61 {
		t.Errorf("expected 61 cells, got %d", len(b.Cells))
	}
}

func TestNewBoard_MarbleCounts(t *testing.T) {
	b := NewBoard()
	var black, white, empty int
	for _, c := range b.Cells {
		switch c {
		case Black:
			black++
		case White:
			white++
		case Empty:
			empty++
		}
	}
	if black != 14 {
		t.Errorf("expected 14 black marbles, got %d", black)
	}
	if white != 14 {
		t.Errorf("expected 14 white marbles, got %d", white)
	}
	if empty != 33 {
		t.Errorf("expected 33 empty cells, got %d", empty)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		h    Hex
		want bool
	}{
		{Hex{0, 0}, true},
		{Hex{4, 0}, true},
		{Hex{0, 4}, true},
		{Hex{-4, 0}, true},
		{Hex{0, -4}, true},
		{Hex{4, -4}, true},   // |q+r|=0, valid
		{Hex{-4, 4}, true},   // |q+r|=0, valid
		{Hex{3, 2}, false},   // |q+r|=5 > 4
		{Hex{-3, -2}, false}, // |q+r|=5 > 4
		{Hex{5, 0}, false},   // |q|=5 > 4
		{Hex{0, 5}, false},   // |r|=5 > 4
	}
	for _, tt := range tests {
		got := IsValid(tt.h)
		if got != tt.want {
			t.Errorf("IsValid(%v) = %v, want %v", tt.h, got, tt.want)
		}
	}
}

func TestOpponent(t *testing.T) {
	if Opponent(Black) != White {
		t.Error("Opponent(Black) should be White")
	}
	if Opponent(White) != Black {
		t.Error("Opponent(White) should be Black")
	}
	if Opponent(Empty) != Empty {
		t.Error("Opponent(Empty) should be Empty")
	}
}

func TestBoardClone(t *testing.T) {
	b := NewBoard()
	c := b.Clone()

	// Modify clone, original should be unchanged
	c.Set(Hex{0, 0}, Black)
	if b.Get(Hex{0, 0}) != Empty {
		t.Error("Clone should be independent of original")
	}
}

// --- Helper to create a game with a custom board ---

// emptyGame creates a game with an empty board (all valid cells empty).
func emptyGame() *Game {
	g := &Game{
		Board: &Board{Cells: make(map[Hex]Cell)},
		Turn:  Black,
	}
	for q := -4; q <= 4; q++ {
		for r := -4; r <= 4; r++ {
			h := Hex{q, r}
			if IsValid(h) {
				g.Board.Cells[h] = Empty
			}
		}
	}
	return g
}

// --- Single Marble Move Tests ---

func TestSingleMarbleMove_Empty(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)

	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("origin should be empty after move")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("destination should be Black after move")
	}
	if g.Turn != White {
		t.Error("turn should switch to White")
	}
}

func TestSingleMarbleMove_BlockedByOwn(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black) // own marble blocking

	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if !errors.Is(err, ErrMoveBlocked) {
		t.Errorf("expected ErrMoveBlocked, got %v", err)
	}
}

func TestSingleMarbleMove_CannotPush(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, White) // opponent marble

	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if !errors.Is(err, ErrCannotPush) {
		t.Errorf("expected ErrCannotPush, got %v", err)
	}
}

func TestSingleMarbleMove_OffBoard(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{4, 0}, Black)

	err := ApplyMove(g, []Hex{{4, 0}}, DirE) // would go to (5,0), off board
	if !errors.Is(err, ErrMoveBlocked) {
		t.Errorf("expected ErrMoveBlocked, got %v", err)
	}
}

// --- Two-Marble In-line Move Tests ---

func TestTwoMarbleInline_Empty(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)

	// Move both east (in-line): front is (1,0), destination is (2,0)
	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("back marble should be cleared")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("middle position should be Black")
	}
	if g.Board.Get(Hex{2, 0}) != Black {
		t.Error("front destination should be Black")
	}
}

func TestTwoMarbleInline_West(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, Black)

	// Move west: front is (1,0), dest is (0,0)
	err := ApplyMove(g, []Hex{{1, 0}, {2, 0}}, DirW)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{2, 0}) != Empty {
		t.Error("back marble should be cleared")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("middle position should be Black")
	}
	if g.Board.Get(Hex{0, 0}) != Black {
		t.Error("front destination should be Black")
	}
}

// --- Three-Marble In-line Move Tests ---

func TestThreeMarbleInline_Empty(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, Black)

	// Move east
	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}, {2, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("back should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("(1,0) should be Black")
	}
	if g.Board.Get(Hex{2, 0}) != Black {
		t.Error("(2,0) should be Black")
	}
	if g.Board.Get(Hex{3, 0}) != Black {
		t.Error("(3,0) should be Black")
	}
}

// --- Broadside Move Tests ---

func TestBroadside_TwoMarbles(t *testing.T) {
	g := emptyGame()
	// Two marbles along E-W axis, move them NE (broadside)
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}}, DirNE) // NE = (1,-1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Empty {
		t.Error("(1,0) should be empty")
	}
	if g.Board.Get(Hex{1, -1}) != Black {
		t.Error("(1,-1) should be Black")
	}
	if g.Board.Get(Hex{2, -1}) != Black {
		t.Error("(2,-1) should be Black")
	}
}

func TestBroadside_ThreeMarbles(t *testing.T) {
	g := emptyGame()
	// Three marbles along E-W axis at r=0, move them SE (broadside)
	g.Board.Set(Hex{-1, 0}, Black)
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)

	err := ApplyMove(g, []Hex{{-1, 0}, {0, 0}, {1, 0}}, DirSE) // SE = (0,1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{-1, 0}) != Empty {
		t.Error("(-1,0) should be empty")
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Empty {
		t.Error("(1,0) should be empty")
	}
	if g.Board.Get(Hex{-1, 1}) != Black {
		t.Error("(-1,1) should be Black")
	}
	if g.Board.Get(Hex{0, 1}) != Black {
		t.Error("(0,1) should be Black")
	}
	if g.Board.Get(Hex{1, 1}) != Black {
		t.Error("(1,1) should be Black")
	}
}

func TestBroadside_Blocked(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{1, -1}, White) // blocking one destination

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}}, DirNW) // NW = (0,-1)
	// (0,0)->NW=(0,-1) ok, (1,0)->NW=(1,-1) blocked by White
	if !errors.Is(err, ErrMoveBlocked) {
		t.Errorf("expected ErrMoveBlocked, got %v", err)
	}
}

// --- Sumito (Pushing) Tests ---

func TestSumito_2v1(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, White) // opponent in push path

	// 2 Black push 1 White east
	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("(1,0) should be Black")
	}
	if g.Board.Get(Hex{2, 0}) != Black {
		t.Error("(2,0) should be Black")
	}
	if g.Board.Get(Hex{3, 0}) != White {
		t.Error("(3,0) should be White (pushed)")
	}
}

func TestSumito_3v1(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, Black)
	g.Board.Set(Hex{3, 0}, White)

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}, {2, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("(1,0) should be Black")
	}
	if g.Board.Get(Hex{2, 0}) != Black {
		t.Error("(2,0) should be Black")
	}
	if g.Board.Get(Hex{3, 0}) != Black {
		t.Error("(3,0) should be Black")
	}
	if g.Board.Get(Hex{4, 0}) != White {
		t.Error("(4,0) should be White (pushed)")
	}
}

func TestSumito_3v2(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{-1, 0}, Black)
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, White)
	g.Board.Set(Hex{3, 0}, White)
	// Push east: front=(1,0), next=(2,0)=W, (3,0)=W, scan=(4,0)=Empty
	// 3v2: legal. The chain shifts: (4,0)=W, (2,0)->Black, (-1,0)->Empty

	err := ApplyMove(g, []Hex{{-1, 0}, {0, 0}, {1, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{-1, 0}) != Empty {
		t.Error("(-1,0) should be empty")
	}
	if g.Board.Get(Hex{0, 0}) != Black {
		t.Error("(0,0) should be Black")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("(1,0) should be Black")
	}
	if g.Board.Get(Hex{2, 0}) != Black {
		t.Error("(2,0) should be Black")
	}
	if g.Board.Get(Hex{3, 0}) != White {
		t.Error("(3,0) should be White")
	}
	if g.Board.Get(Hex{4, 0}) != White {
		t.Error("(4,0) should be White (pushed)")
	}
}

func TestSumito_Blocked_EqualNumbers(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, White)
	g.Board.Set(Hex{3, 0}, White) // 2v2: not allowed

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}}, DirE)
	if !errors.Is(err, ErrCannotPush) {
		t.Errorf("expected ErrCannotPush for 2v2, got %v", err)
	}
}

func TestSumito_PushBlocked_ByOwnMarble(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, White)
	g.Board.Set(Hex{3, 0}, Black) // own marble behind opponent

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}}, DirE)
	if !errors.Is(err, ErrPushBlocked) {
		t.Errorf("expected ErrPushBlocked, got %v", err)
	}
}

// --- Push-off Tests ---

func TestPushOff_2v1(t *testing.T) {
	g := emptyGame()
	// Place white on the edge, push it off
	g.Board.Set(Hex{3, 0}, Black)
	g.Board.Set(Hex{4, 0}, White) // edge: next east is (5,0) off-board
	// Need 2 black to push
	g.Board.Set(Hex{2, 0}, Black)

	err := ApplyMove(g, []Hex{{2, 0}, {3, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{2, 0}) != Empty {
		t.Error("(2,0) should be empty")
	}
	if g.Board.Get(Hex{3, 0}) != Black {
		t.Error("(3,0) should be Black")
	}
	if g.Board.Get(Hex{4, 0}) != Black {
		t.Error("(4,0) should be Black")
	}
	if g.CapturedWhite != 1 {
		t.Errorf("expected 1 captured white, got %d", g.CapturedWhite)
	}
}

func TestPushOff_3v1_AtEdge(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{1, -4}, Black)
	g.Board.Set(Hex{2, -4}, Black)
	g.Board.Set(Hex{3, -4}, Black)
	g.Board.Set(Hex{4, -4}, White) // edge: next east is (5,-4) off-board

	err := ApplyMove(g, []Hex{{1, -4}, {2, -4}, {3, -4}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.CapturedWhite != 1 {
		t.Errorf("expected 1 captured white, got %d", g.CapturedWhite)
	}
	if g.Board.Get(Hex{1, -4}) != Empty {
		t.Error("(1,-4) should be empty")
	}
	if g.Board.Get(Hex{4, -4}) != Black {
		t.Error("(4,-4) should be Black")
	}
}

func TestPushOff_3v2_AtEdge(t *testing.T) {
	g := emptyGame()
	// 3 Black push 2 White, last White falls off
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, Black)
	g.Board.Set(Hex{3, 0}, White)
	g.Board.Set(Hex{4, 0}, White)
	// Push east: (4,0)->E=(5,0) off-board. One White captured, other shifts.

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}, {2, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.CapturedWhite != 1 {
		t.Errorf("expected 1 captured white, got %d", g.CapturedWhite)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("(1,0) should be Black")
	}
	if g.Board.Get(Hex{2, 0}) != Black {
		t.Error("(2,0) should be Black")
	}
	if g.Board.Get(Hex{3, 0}) != Black {
		t.Error("(3,0) should be Black")
	}
	if g.Board.Get(Hex{4, 0}) != White {
		t.Error("(4,0) should be White (remaining opponent)")
	}
}

// --- Win Condition ---

func TestWinCondition(t *testing.T) {
	g := emptyGame()
	g.CapturedWhite = 5 // Black has captured 5 whites already

	// Set up push-off scenario
	g.Board.Set(Hex{3, 0}, Black)
	g.Board.Set(Hex{4, 0}, White)
	g.Board.Set(Hex{2, 0}, Black)

	err := ApplyMove(g, []Hex{{2, 0}, {3, 0}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.CapturedWhite != 6 {
		t.Errorf("expected 6 captured white, got %d", g.CapturedWhite)
	}
	if g.Winner != Black {
		t.Error("Black should be the winner")
	}
	if !g.IsOver() {
		t.Error("game should be over")
	}
}

func TestMoveAfterGameOver(t *testing.T) {
	g := emptyGame()
	g.Winner = Black
	g.Board.Set(Hex{0, 0}, White)

	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if !errors.Is(err, ErrGameOver) {
		t.Errorf("expected ErrGameOver, got %v", err)
	}
}

// --- Wrong Turn and Ownership Tests ---

func TestWrongTurn(t *testing.T) {
	g := emptyGame()
	g.Turn = Black
	g.Board.Set(Hex{0, 0}, White) // White's marble, but it's Black's turn

	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if !errors.Is(err, ErrNotYourMarble) {
		t.Errorf("expected ErrNotYourMarble, got %v", err)
	}
}

func TestMovingOpponentMarble(t *testing.T) {
	g := emptyGame()
	g.Turn = White
	g.Board.Set(Hex{0, 0}, Black) // Black's marble, but it's White's turn

	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if !errors.Is(err, ErrNotYourMarble) {
		t.Errorf("expected ErrNotYourMarble, got %v", err)
	}
}

// --- Non-collinear Marbles ---

func TestNonCollinearMarbles(t *testing.T) {
	g := emptyGame()
	// Triangle shape: not collinear
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{0, 1}, Black)

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}, {0, 1}}, DirE)
	if !errors.Is(err, ErrNotCollinear) {
		t.Errorf("expected ErrNotCollinear, got %v", err)
	}
}

func TestNonAdjacentMarbles(t *testing.T) {
	g := emptyGame()
	// Two marbles with a gap
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{2, 0}, Black) // gap at (1,0)

	err := ApplyMove(g, []Hex{{0, 0}, {2, 0}}, DirE)
	if !errors.Is(err, ErrNotCollinear) {
		t.Errorf("expected ErrNotCollinear, got %v", err)
	}
}

// --- Invalid Marble Count ---

func TestInvalidMarbleCount_Zero(t *testing.T) {
	g := emptyGame()
	err := ApplyMove(g, []Hex{}, DirE)
	if !errors.Is(err, ErrInvalidMarbleCount) {
		t.Errorf("expected ErrInvalidMarbleCount, got %v", err)
	}
}

func TestInvalidMarbleCount_Four(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, 0}, Black)
	g.Board.Set(Hex{2, 0}, Black)
	g.Board.Set(Hex{3, 0}, Black)

	err := ApplyMove(g, []Hex{{0, 0}, {1, 0}, {2, 0}, {3, 0}}, DirE)
	if !errors.Is(err, ErrInvalidMarbleCount) {
		t.Errorf("expected ErrInvalidMarbleCount, got %v", err)
	}
}

// --- Invalid Direction ---

func TestInvalidDirection(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)

	err := ApplyMove(g, []Hex{{0, 0}}, Hex{2, 0}) // not a valid direction
	if !errors.Is(err, ErrInvalidDirection) {
		t.Errorf("expected ErrInvalidDirection, got %v", err)
	}
}

// --- Diagonal / NE-SW axis tests ---

func TestInline_NE_Axis(t *testing.T) {
	g := emptyGame()
	// Marbles along NE axis: (0,0), (1,-1), (2,-2)
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{1, -1}, Black)
	g.Board.Set(Hex{2, -2}, Black)

	// Move NE (front is (2,-2), dest (3,-3))
	err := ApplyMove(g, []Hex{{0, 0}, {1, -1}, {2, -2}}, DirNE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{1, -1}) != Black {
		t.Error("(1,-1) should be Black")
	}
	if g.Board.Get(Hex{2, -2}) != Black {
		t.Error("(2,-2) should be Black")
	}
	if g.Board.Get(Hex{3, -3}) != Black {
		t.Error("(3,-3) should be Black")
	}
}

func TestBroadside_NW_SE_Axis(t *testing.T) {
	g := emptyGame()
	// Two marbles along NW-SE axis: (0,0), (0,1)
	// Axis is SE = (0,1). Move them E (broadside).
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{0, 1}, Black)

	err := ApplyMove(g, []Hex{{0, 0}, {0, 1}}, DirE)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{0, 1}) != Empty {
		t.Error("(0,1) should be empty")
	}
	if g.Board.Get(Hex{1, 0}) != Black {
		t.Error("(1,0) should be Black")
	}
	if g.Board.Get(Hex{1, 1}) != Black {
		t.Error("(1,1) should be Black")
	}
}

// --- Turn Switching ---

func TestTurnSwitchesAfterMove(t *testing.T) {
	g := emptyGame()
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{0, 2}, White)

	// Black moves east
	err := ApplyMove(g, []Hex{{0, 0}}, DirE)
	if err != nil {
		t.Fatalf("Black move failed: %v", err)
	}
	if g.Turn != White {
		t.Error("should be White's turn")
	}

	// White moves west
	err = ApplyMove(g, []Hex{{0, 2}}, DirW)
	if err != nil {
		t.Fatalf("White move failed: %v", err)
	}
	if g.Turn != Black {
		t.Error("should be Black's turn")
	}
}

// --- areCollinear tests ---

func TestAreCollinear(t *testing.T) {
	tests := []struct {
		name    string
		marbles []Hex
		want    bool
	}{
		{"single", []Hex{{0, 0}}, true},
		{"two adjacent E", []Hex{{0, 0}, {1, 0}}, true},
		{"two adjacent NE", []Hex{{0, 0}, {1, -1}}, true},
		{"two adjacent SE", []Hex{{0, 0}, {0, 1}}, true},
		{"two non-adjacent", []Hex{{0, 0}, {2, 0}}, false},
		{"three in line E", []Hex{{0, 0}, {1, 0}, {2, 0}}, true},
		{"three in line NE", []Hex{{0, 0}, {1, -1}, {2, -2}}, true},
		{"three not collinear", []Hex{{0, 0}, {1, 0}, {0, 1}}, false},
		{"three collinear but gap", []Hex{{0, 0}, {1, 0}, {3, 0}}, false},
		{"three in reverse order", []Hex{{2, 0}, {0, 0}, {1, 0}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := areCollinear(tt.marbles)
			if got != tt.want {
				t.Errorf("areCollinear(%v) = %v, want %v", tt.marbles, got, tt.want)
			}
		})
	}
}

// --- NewGame test ---

func TestNewGame(t *testing.T) {
	g := NewGame()
	if g.Turn != Black {
		t.Error("Black should move first")
	}
	if g.CapturedBlack != 0 || g.CapturedWhite != 0 {
		t.Error("no captures at start")
	}
	if g.Winner != Empty {
		t.Error("no winner at start")
	}
	if g.IsOver() {
		t.Error("game should not be over at start")
	}
}

// --- Broadside where destination overlaps with moving marble ---

func TestBroadside_OverlappingPositions(t *testing.T) {
	g := emptyGame()
	// Two marbles along E-W axis: (0,0), (1,0)
	// Move them W (broadside would be perpendicular, but W is inline for this axis)
	// Let me use NW-SE axis instead: (0,0), (0,1)
	// Move W: (0,0)->(−1,0), (0,1)->(−1,1)
	g.Board.Set(Hex{0, 0}, Black)
	g.Board.Set(Hex{0, 1}, Black)

	err := ApplyMove(g, []Hex{{0, 0}, {0, 1}}, DirW)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if g.Board.Get(Hex{0, 0}) != Empty {
		t.Error("(0,0) should be empty")
	}
	if g.Board.Get(Hex{0, 1}) != Empty {
		t.Error("(0,1) should be empty")
	}
	if g.Board.Get(Hex{-1, 0}) != Black {
		t.Error("(-1,0) should be Black")
	}
	if g.Board.Get(Hex{-1, 1}) != Black {
		t.Error("(-1,1) should be Black")
	}
}
