package repository

// PlayerLinkExpr is the canonical player URL in SELECT lists (alias "p").
const PlayerLinkExpr = "COALESCE(p.link_v2, p.link)"

// PlayerLinkExprBare is the canonical player URL without a table alias.
const PlayerLinkExprBare = "COALESCE(link_v2, link)"
