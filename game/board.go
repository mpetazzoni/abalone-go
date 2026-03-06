package game

// Cell represents the state of a board cell
type Cell int

const (
	Empty Cell = iota
	Black
	White
)

// Hex represents axial coordinates on the hex board
type Hex struct {
	Q, R int
}

// Direction constants for the 6 hex directions
var (
	DirE  = Hex{1, 0}
	DirW  = Hex{-1, 0}
	DirNE = Hex{1, -1}
	DirSW = Hex{-1, 1}
	DirNW = Hex{0, -1}
	DirSE = Hex{0, 1}

	AllDirections = []Hex{DirE, DirW, DirNE, DirSW, DirNW, DirSE}
)

// Neighbor returns the adjacent hex in the given direction
func (h Hex) Neighbor(dir Hex) Hex {
	return Hex{h.Q + dir.Q, h.R + dir.R}
}

// Board represents the Abalone hex board
// Valid cells satisfy |q| <= 4, |r| <= 4, |q+r| <= 4 (61 cells total)
type Board struct {
	Cells map[Hex]Cell
}

// IsValid returns true if the hex is a valid board position
func IsValid(h Hex) bool {
	return abs(h.Q) <= 4 && abs(h.R) <= 4 && abs(h.Q+h.R) <= 4
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// NewBoard creates a board with the standard starting position
func NewBoard() *Board {
	b := &Board{Cells: make(map[Hex]Cell)}

	// Initialize all 61 valid cells as empty
	for q := -4; q <= 4; q++ {
		for r := -4; r <= 4; r++ {
			h := Hex{q, r}
			if IsValid(h) {
				b.Cells[h] = Empty
			}
		}
	}

	// Standard starting position:
	// Black occupies the top rows (r = -4, r = -3, and middle 3 of r = -2)
	// White occupies the bottom rows (r = 4, r = 3, and middle 3 of r = 2)

	// Black: full row at r=-4 (q=0..4) — 5 marbles
	for q := 0; q <= 4; q++ {
		h := Hex{q, -4}
		if IsValid(h) {
			b.Cells[h] = Black
		}
	}
	// Black: full row at r=-3 (q=-1..4) — 6 marbles
	for q := -1; q <= 4; q++ {
		h := Hex{q, -3}
		if IsValid(h) {
			b.Cells[h] = Black
		}
	}
	// Black: middle 3 at r=-2 (q=0..2) — 3 marbles
	for q := 0; q <= 2; q++ {
		b.Cells[Hex{q, -2}] = Black
	}

	// White: full row at r=4 (q=-4..0) — 5 marbles
	for q := -4; q <= 0; q++ {
		h := Hex{q, 4}
		if IsValid(h) {
			b.Cells[h] = White
		}
	}
	// White: full row at r=3 (q=-4..1) — 6 marbles
	for q := -4; q <= 1; q++ {
		h := Hex{q, 3}
		if IsValid(h) {
			b.Cells[h] = White
		}
	}
	// White: middle 3 at r=2 (q=-2..0) — 3 marbles
	for q := -2; q <= 0; q++ {
		b.Cells[Hex{q, 2}] = White
	}

	return b
}

// Get returns the cell state at the given position, or Empty if off-board
func (b *Board) Get(h Hex) Cell {
	if !IsValid(h) {
		return Empty // off-board treated as empty for logic purposes
	}
	return b.Cells[h]
}

// Set sets the cell state (only for valid positions)
func (b *Board) Set(h Hex, c Cell) {
	if IsValid(h) {
		b.Cells[h] = c
	}
}

// Clone returns a deep copy of the board
func (b *Board) Clone() *Board {
	nb := &Board{Cells: make(map[Hex]Cell, len(b.Cells))}
	for k, v := range b.Cells {
		nb.Cells[k] = v
	}
	return nb
}

// Opponent returns the opposite color
func Opponent(c Cell) Cell {
	if c == Black {
		return White
	}
	if c == White {
		return Black
	}
	return Empty
}
