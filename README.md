# Endgame Villain

Endgame Villain is a tool for finding drawn/even chess endgames from real Lichess games and analysis. These positions are categorized and can be used to play against friends on lichess, or against engines.

This tool is useful for people wishing to improve their endgame chops, beginners learning to play chess, and anyone who wants to become a villain in the endgame.

# Data Source

Lichess (🤍) provides a download of Stockfish evaluated positions: https://database.lichess.org/#evals

You may download this file yourself:

```
wget https://database.lichess.org/lichess_db_eval.jsonl.zst
unzstd lichess_db_eval.jsonl.zst
```
**Beware**, as of Dec 2024 this was a 7GB download, that uncompressed into a 39GB file!

Create a `.env` file in this directory and set `EVAL_FILE=/path/to/your/lichess_db_eval.jsonl`

# pprof

```
go tool pprof -http=: cpu.prof
```

To view the cpu profile in browser.