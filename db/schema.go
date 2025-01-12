package db

type Position struct {
	WhiteBishops int
	BlackBishops int
	WhiteKnights int
	BlackKnights int
	WhitePawns   int
	BlackPawns   int
	WhiteQueens  int
	BlackQueens  int
	WhiteRooks   int
	BlackRooks   int
	MinEval      int
	MaxEval      int
}