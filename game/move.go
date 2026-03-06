package game

import (
	"errors"
	"sort"
)

// Move represents a player's move
type Move struct {
	Marbles   []Hex
	Direction Hex
}

var (
	ErrInvalidMarbleCount = errors.New("must move 1, 2, or 3 marbles")
	ErrNotYourMarble      = errors.New("can only move your own marbles")
	ErrNotCollinear       = errors.New("marbles must be adjacent and collinear")
	ErrInvalidDirection   = errors.New("invalid direction")
	ErrMoveBlocked        = errors.New("move is blocked")
	ErrCannotPush         = errors.New("cannot push: no numerical superiority")
	ErrGameOver           = errors.New("game is over")
	ErrPushBlocked        = errors.New("push blocked: no space behind opponent marbles")
)

// isValidDirection checks if d is one of the 6 valid hex directions
func isValidDirection(d Hex) bool {
	for _, dir := range AllDirections {
		if d == dir {
			return true
		}
	}
	return false
}

// areCollinear checks if 2 or 3 marbles are adjacent and on the same line.
// Returns the axis direction (one of the 6 directions) and true if valid.
// For a single marble, always returns true with a zero direction.
func areCollinear(marbles []Hex) (Hex, bool) {
	if len(marbles) == 1 {
		return Hex{}, true
	}

	// For 2 marbles: check if they're adjacent (differ by exactly one direction)
	if len(marbles) == 2 {
		diff := Hex{marbles[1].Q - marbles[0].Q, marbles[1].R - marbles[0].R}
		if isValidDirection(diff) {
			return diff, true
		}
		neg := Hex{-diff.Q, -diff.R}
		if isValidDirection(neg) {
			return diff, true
		}
		return Hex{}, false
	}

	// For 3 marbles: sort by (q, r), check consecutive adjacency and same axis
	sorted := make([]Hex, 3)
	copy(sorted, marbles)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Q != sorted[j].Q {
			return sorted[i].Q < sorted[j].Q
		}
		return sorted[i].R < sorted[j].R
	})

	diff1 := Hex{sorted[1].Q - sorted[0].Q, sorted[1].R - sorted[0].R}
	diff2 := Hex{sorted[2].Q - sorted[1].Q, sorted[2].R - sorted[1].R}

	if diff1 != diff2 {
		return Hex{}, false
	}
	if !isValidDirection(diff1) {
		return Hex{}, false
	}

	return diff1, true
}

// isInlineMove checks if the direction is along the axis of the marbles
func isInlineMove(axis, dir Hex) bool {
	return dir == axis || dir == (Hex{-axis.Q, -axis.R})
}

// frontMarble returns the marble that is "in front" in the given direction.
// For an in-line move, this is the marble furthest in the direction of movement.
func frontMarble(marbles []Hex, dir Hex) Hex {
	best := marbles[0]
	for _, m := range marbles[1:] {
		// The front marble is the one where the dot product with dir is largest
		if m.Q*dir.Q+m.R*dir.R > best.Q*dir.Q+best.R*dir.R {
			best = m
		}
	}
	return best
}

// ApplyMove validates and applies a move to the game.
func ApplyMove(g *Game, marbles []Hex, dir Hex) error {
	if g.IsOver() {
		return ErrGameOver
	}

	if len(marbles) < 1 || len(marbles) > 3 {
		return ErrInvalidMarbleCount
	}

	if !isValidDirection(dir) {
		return ErrInvalidDirection
	}

	// Check all marbles belong to current player
	for _, m := range marbles {
		if !IsValid(m) || g.Board.Get(m) != g.Turn {
			return ErrNotYourMarble
		}
	}

	// Check collinearity for 2+ marbles
	axis, collinear := areCollinear(marbles)
	if len(marbles) > 1 && !collinear {
		return ErrNotCollinear
	}

	// Single marble: always treated as in-line (no broadside concept)
	if len(marbles) == 1 {
		dest := marbles[0].Neighbor(dir)
		if !IsValid(dest) {
			return ErrMoveBlocked
		}
		destCell := g.Board.Get(dest)
		if destCell == g.Turn {
			return ErrMoveBlocked // can't move onto own marble
		}
		if destCell == Opponent(g.Turn) {
			// Single marble cannot push
			return ErrCannotPush
		}
		// Simple move
		g.Board.Set(dest, g.Turn)
		g.Board.Set(marbles[0], Empty)
		g.SwitchTurn()
		return nil
	}

	// Determine if in-line or broadside
	if isInlineMove(axis, dir) {
		return applyInlineMove(g, marbles, dir)
	}
	return applyBroadsideMove(g, marbles, dir)
}

// applyInlineMove handles in-line moves (possibly with sumito pushing)
func applyInlineMove(g *Game, marbles []Hex, dir Hex) error {
	front := frontMarble(marbles, dir)
	next := front.Neighbor(dir)

	if IsValid(next) && g.Board.Get(next) == g.Turn {
		return ErrMoveBlocked // blocked by own marble
	}

	if !IsValid(next) {
		return ErrMoveBlocked // can't move own marble off board
	}

	if g.Board.Get(next) == Empty {
		// Simple in-line move: shift all marbles forward.
		// Because the marbles are collinear along the move direction,
		// we just place the player's marble at front+dir and clear the back.
		back := frontMarble(marbles, Hex{-dir.Q, -dir.R})
		g.Board.Set(front.Neighbor(dir), g.Turn)
		g.Board.Set(back, Empty)
		g.SwitchTurn()
		return nil
	}

	// Sumito: count opponent marbles in the push direction
	opp := Opponent(g.Turn)
	oppCount := 0
	scan := next
	for IsValid(scan) && g.Board.Get(scan) == opp {
		oppCount++
		scan = scan.Neighbor(dir)
	}

	if oppCount >= len(marbles) {
		return ErrCannotPush
	}

	// Check what's behind the last opponent marble
	// scan is now either off-board or on a non-opponent cell
	if IsValid(scan) && g.Board.Get(scan) != Empty {
		return ErrPushBlocked
	}

	// Execute push
	if IsValid(scan) {
		// Push into empty space
		g.Board.Set(scan, opp)
	} else {
		// Push off the board!
		if opp == Black {
			g.CapturedBlack++
		} else {
			g.CapturedWhite++
		}
	}

	// Shift player's marbles forward:
	// The front opponent position (next) becomes the player's color,
	// and the back of the player's column becomes empty.
	back := frontMarble(marbles, Hex{-dir.Q, -dir.R})
	g.Board.Set(next, g.Turn)
	g.Board.Set(back, Empty)

	g.CheckWinner()
	g.SwitchTurn()
	return nil
}

// applyBroadsideMove handles broadside (side-step) moves
func applyBroadsideMove(g *Game, marbles []Hex, dir Hex) error {
	// Each marble moves in dir; all destinations must be empty
	// (or occupied by one of the marbles being moved, which will vacate)
	for _, m := range marbles {
		dest := m.Neighbor(dir)
		if !IsValid(dest) {
			return ErrMoveBlocked
		}
		destCell := g.Board.Get(dest)
		if destCell != Empty {
			// Check if dest is one of the marbles being moved
			isMoving := false
			for _, other := range marbles {
				if dest == other {
					isMoving = true
					break
				}
			}
			if !isMoving {
				return ErrMoveBlocked
			}
		}
	}

	// Execute broadside: clear old positions, then set new ones
	for _, m := range marbles {
		g.Board.Set(m, Empty)
	}
	for _, m := range marbles {
		g.Board.Set(m.Neighbor(dir), g.Turn)
	}

	g.SwitchTurn()
	return nil
}
