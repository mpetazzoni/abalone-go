package game

// Game holds the full game state
type Game struct {
	Board         *Board
	Turn          Cell // Black or White
	CapturedBlack int  // black marbles pushed off (white's score)
	CapturedWhite int  // white marbles pushed off (black's score)
	Winner        Cell // Empty if game ongoing
}

// NewGame creates a new game with standard starting position
func NewGame() *Game {
	return &Game{
		Board: NewBoard(),
		Turn:  Black,
	}
}

// WinThreshold is the number of captures needed to win
const WinThreshold = 6

// CheckWinner updates the Winner field if someone has won
func (g *Game) CheckWinner() {
	if g.CapturedBlack >= WinThreshold {
		g.Winner = White
	} else if g.CapturedWhite >= WinThreshold {
		g.Winner = Black
	}
}

// IsOver returns true if the game has ended
func (g *Game) IsOver() bool {
	return g.Winner != Empty
}

// SwitchTurn switches to the other player
func (g *Game) SwitchTurn() {
	g.Turn = Opponent(g.Turn)
}
